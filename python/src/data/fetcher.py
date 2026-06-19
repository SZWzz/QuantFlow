"""DataService gRPC implementation — Python-side data fetching via mootdx, akshare, tushare.

Data is returned as JSON-encoded bytes for simplicity.
"""

import json
import logging
import threading

from src.proto import data_pb2, data_pb2_grpc

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Mootdx (TDX TCP) support
# ---------------------------------------------------------------------------

_HAS_MOOTDX = False
try:
    from mootdx.quotes import Quotes
    from mootdx import config as mootdx_config
    _HAS_MOOTDX = True
except ImportError:
    pass

_FREQ_MAP: dict[str, str] = {
    "1D": "day",
    "1W": "week",
    "1M": "mon",
    "1m": "1m",
    "5m": "5m",
    "15m": "15m",
    "30m": "30m",
    "1H": "1h",
}

_CHUNK_SIZE = 800  # Max bars per TDX request


def _normalize_code(symbol: str) -> str:
    """Normalize a symbol to 6-digit plain code for mootdx."""
    s = (symbol or "").strip().upper()
    for suffix in (".SH", ".SZ", ".BJ", ".SS"):
        if s.endswith(suffix):
            s = s[:-3]
            break
    for prefix in ("SH", "SZ", "BJ"):
        if s.startswith(prefix) and len(s) > 2:
            s = s[2:]
            break
    return s.strip()


def _init_mootdx_client():
    """Configure best IP and return a Quotes client (from astockpursue ref)."""
    mootdx_config.setup()
    bestip_hq = mootdx_config.get("BESTIP", {}).get("HQ", "")
    if not bestip_hq:
        server_list = mootdx_config.get("SERVER", {}).get("HQ", [])
        if server_list:
            mootdx_config.set("BESTIP", {"HQ": server_list[0][1:]})
    return Quotes.factory(market="std")


# Cached mootdx Quotes client — setup()/factory() probe TDX servers and are expensive,
# so we build the client once and reuse it across fetches. Guarded by a lock because
# mootdx client thread-safety is undocumented and gRPC may dispatch concurrent requests.
_mootdx_client_lock = threading.Lock()
_mootdx_client = None


def _get_mootdx_client():
    """Return the cached mootdx Quotes client, building it once on first use.

    Uses double-checked locking: the fast path (client already built) takes no lock.
    On a cached client we do NOT auto-refresh here — callers reset via
    _reset_mootdx_client() after a failure so the next fetch rebuilds it.
    """
    global _mootdx_client
    if _mootdx_client is not None:
        return _mootdx_client
    with _mootdx_client_lock:
        if _mootdx_client is None:
            _mootdx_client = _init_mootdx_client()
    return _mootdx_client


def _reset_mootdx_client():
    """Drop the cached client so the next _get_mootdx_client() rebuilds it.

    Call this after a TDX operation fails (broken connection, bad server), so a
    stale client is not reused indefinitely.
    """
    global _mootdx_client
    with _mootdx_client_lock:
        _mootdx_client = None


def _fetch_mootdx_ohlcv(symbols: list[str], start_date: str, end_date: str, interval: str) -> list[dict]:
    """Fetch OHLCV bars via mootdx (TDX TCP protocol) with pagination.

    Returns a list of bar dicts with keys: symbol, date, open, high, low, close, volume.
    """
    import pandas as pd

    freq = _FREQ_MAP.get(interval)
    if freq is None:
        raise ValueError(
            f"mootdx: unsupported interval {interval!r}. "
            f"Supported: {list(_FREQ_MAP.keys())}"
        )

    client = _get_mootdx_client()
    start_ts = pd.Timestamp(start_date)
    all_bars: list[dict] = []

    for symbol in symbols:
        plain = _normalize_code(symbol)
        symbol_frames: list[pd.DataFrame] = []
        chunk_start = 0
        max_chunks = 100

        for _ in range(max_chunks):
            try:
                raw = client.bars(
                    symbol=plain,
                    frequency=freq,
                    start=chunk_start,
                    offset=_CHUNK_SIZE,
                )
            except Exception as exc:
                logger.warning("mootdx bars failed for %s at offset %d: %s", plain, chunk_start, exc)
                _reset_mootdx_client()
                break

            if raw is None or len(raw) == 0:
                break

            df = pd.DataFrame(raw)
            col_map = {}
            if "datetime" in df.columns:
                col_map["datetime"] = "trade_date"
            if "vol" in df.columns:
                col_map["vol"] = "volume"
            df = df.rename(columns=col_map)

            if "trade_date" not in df.columns:
                break

            df["trade_date"] = pd.to_datetime(df["trade_date"])
            symbol_frames.append(df)

            if df["trade_date"].min() <= start_ts:
                break
            if len(raw) < _CHUNK_SIZE:
                break
            chunk_start += _CHUNK_SIZE

        if not symbol_frames:
            continue

        result = pd.concat(symbol_frames, ignore_index=True)
        result = result.drop_duplicates(subset=["trade_date"])
        result = result.set_index("trade_date").sort_index()
        result = result.loc[start_date:end_date]

        for idx, row in result.iterrows():
            bar = {"symbol": symbol, "date": idx.strftime("%Y-%m-%d")}
            for col in ("open", "high", "low", "close", "volume"):
                if col in result.columns:
                    val = row[col]
                    bar[col] = float(val) if pd.notna(val) else 0.0
            all_bars.append(bar)

    if not all_bars:
        raise ValueError(f"mootdx: no data for {symbols} in [{start_date}, {end_date}]")

    return all_bars


def _fetch_mootdx_quote(symbols: list[str]) -> list[dict]:
    """Fetch minute-line (分时图) for today via mootdx and extract quote summary.

    Returns a list of quote dicts with keys: symbol, last, open, high, low, volume.
    """
    import pandas as pd

    client = _get_mootdx_client()
    quotes: list[dict] = []

    for symbol in symbols:
        plain = _normalize_code(symbol)
        try:
            raw = client.minute(symbol=plain)
        except Exception as exc:
            logger.warning("mootdx minute failed for %s: %s", plain, exc)
            _reset_mootdx_client()
            continue

        if raw is None or len(raw) == 0:
            continue

        df = pd.DataFrame(raw)
        col_map = {}
        for c in df.columns:
            cl = str(c).strip().lower()
            if cl in ("price", "close"):
                col_map[c] = "price"
            elif cl in ("volume", "vol"):
                col_map[c] = "volume"
            elif cl in ("amount", "amt"):
                col_map[c] = "amount"
        df = df.rename(columns=col_map)

        if "price" not in df.columns:
            continue

        prices = df["price"].dropna()
        if len(prices) == 0:
            continue

        last = float(prices.iloc[-1])
        open_p = float(prices.iloc[0])
        high = float(prices.max())
        low = float(prices.min())
        vol = float(df["volume"].sum()) if "volume" in df.columns else 0.0

        quotes.append({
            "symbol": symbol,
            "last": last,
            "open": open_p,
            "high": high,
            "low": low,
            "volume": vol,
        })

    if not quotes:
        raise ValueError(f"mootdx: no minute data for {symbols}")

    return quotes


# ---------------------------------------------------------------------------
# gRPC Service
# ---------------------------------------------------------------------------


class DataService(data_pb2_grpc.DataServiceServicer):
    """gRPC service for Python-side data fetching.

    Routes to mootdx, akshare, tushare, etc. based on the `source` field.
    """

    async def FetchData(self, request, context):
        source = request.source
        data_type = request.data_type
        symbols = list(request.symbols)
        start_date = request.start_date
        end_date = request.end_date
        # params carries per-source options (e.g. mootdx K-line interval).
        params = dict(request.params) if request.params else {}

        if not symbols:
            return data_pb2.FetchDataResponse(error="no symbols provided")

        try:
            if source == "mootdx":
                return await self._handle_mootdx(data_type, symbols, start_date, end_date, params)
            else:
                return data_pb2.FetchDataResponse(
                    error=f"DataService: source '{source}' not implemented. Supported: mootdx"
                )
        except Exception as exc:
            logger.exception("DataService.FetchData failed (source=%s, type=%s)", source, data_type)
            return data_pb2.FetchDataResponse(error=str(exc))

    async def _handle_mootdx(self, data_type, symbols, start_date, end_date, params):
        if not _HAS_MOOTDX:
            return data_pb2.FetchDataResponse(
                error="mootdx not installed. Install with: pip install mootdx"
            )

        if data_type == "ohlcv":
            # Interval must come from request.params (set by the Go adapter),
            # not be hardcoded — otherwise a 1W/1m/5m request silently returns
            # daily bars. Default to "1D" only when the caller omits it.
            interval = params.get("interval", "1D")
            bars = _fetch_mootdx_ohlcv(symbols, start_date, end_date, interval)
        elif data_type == "quote":
            bars = _fetch_mootdx_quote(symbols)
        else:
            return data_pb2.FetchDataResponse(
                error=f"mootdx: unsupported data_type '{data_type}'. Supported: ohlcv, quote"
            )

        return data_pb2.FetchDataResponse(
            data=json.dumps(bars, ensure_ascii=False).encode("utf-8"),
            source="mootdx",
        )
