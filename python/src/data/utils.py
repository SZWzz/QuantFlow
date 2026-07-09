"""Utility functions for data fetching and validation."""

import asyncio
import importlib
import json
import logging
import re
from datetime import datetime

logger = logging.getLogger(__name__)


def validate_dates(start_date: str, end_date: str) -> tuple[str, str]:
    """Validate and normalize date strings.

    Ensures start_date <= end_date, converts YYYYMMDD to
    YYYY-MM-DD format, and returns (start, end) pair.
    Raises ValueError on invalid dates or inverted range.
    """
    def _normalize(d: str) -> str:
        d = d.strip()
        if re.match(r"^\d{8}$", d):
            d = f"{d[:4]}-{d[4:6]}-{d[6:]}"
        try:
            datetime.strptime(d, "%Y-%m-%d")
        except ValueError:
            raise ValueError(f"invalid date: {d!r}")
        return d

    s = _normalize(start_date)
    e = _normalize(end_date)

    if s > e:
        raise ValueError(f"start_date ({s}) must be <= end_date ({e})")

    return s, e


def get_1m_bars(symbol: str, start_date: str, end_date: str, data_source: str = "akshare") -> list[dict]:
    """Fetch 1-minute bars for a symbol over a date range.

    Delegates to the appropriate data fetcher based on data_source.
    Returns a list of OHLCV dicts ordered by time ascending.
    Falls back to subprocess if direct import is unavailable.
    """
    start, end = validate_dates(start_date, end_date)

    if data_source == "akshare":
        return _fetch_akshare_1m(symbol, start, end)
    elif data_source == "mootdx":
        return _fetch_mootdx_1m(symbol, start, end)
    else:
        raise ValueError(f"unsupported data_source: {data_source!r}")


def _fetch_akshare_1m(symbol: str, start: str, end: str) -> list[dict]:
    """Fetch 1m bars via akshare stock_zh_a_hist_min_em."""
    try:
        ak = importlib.import_module("akshare")
    except ImportError:
        return _fetch_akshare_1m_subprocess(symbol, start, end)

    try:
        df = ak.stock_zh_a_hist_min_em(
            symbol=symbol,
            period="1",
            start_date=start,
            end_date=end,
            adjust="",
        )
    except Exception as exc:
        logger.warning("akshare 1m failed for %s: %s", symbol, exc)
        return _fetch_akshare_1m_subprocess(symbol, start, end)

    if df is None or (hasattr(df, "empty") and df.empty):
        return []

    df = df.copy()
    col_map = {}
    for c in df.columns:
        cl = str(c).strip().lower()
        if "时间" in cl and "日期" not in cl:
            col_map[c] = "time"
        elif "开盘" in cl:
            col_map[c] = "open"
        elif "收盘" in cl:
            col_map[c] = "close"
        elif "最高" in cl:
            col_map[c] = "high"
        elif "最低" in cl:
            col_map[c] = "low"
        elif "成交量" in cl:
            col_map[c] = "volume"
        elif "成交额" in cl:
            col_map[c] = "amount"
    df = df.rename(columns=col_map)

    if "close" not in df.columns:
        return []

    bars = []
    for _, row in df.iterrows():
        bar = {"symbol": symbol}
        dt = str(row.get("time", ""))
        if dt:
            bar["date"] = str(dt)[:19]
        for k in ("open", "high", "low", "close", "volume", "amount"):
            try:
                bar[k] = float(row.get(k, 0)) if row.get(k) is not None else 0.0
            except (ValueError, TypeError):
                bar[k] = 0.0
        bars.append(bar)

    bars.sort(key=lambda b: b.get("date", ""))
    return bars


def _fetch_akshare_1m_subprocess(symbol: str, start: str, end: str) -> list[dict]:
    """Fallback: fetch 1m bars via subprocess."""
    import subprocess
    import sys

    code = (
        "import akshare as ak; import json; "
        f"df = ak.stock_zh_a_hist_min_em(symbol='{symbol}', period='1', "
        f"start_date='{start}', end_date='{end}', adjust=''); "
        "if df is None or (hasattr(df, 'empty') and df.empty): print('[]'); "
        "else: print(df.to_json(orient='records', force_ascii=False))"
    )
    try:
        result = subprocess.run(
            [sys.executable, "-c", code],
            capture_output=True, text=True, timeout=60,
        )
        if result.returncode != 0:
            return []
        output = result.stdout.strip()
        return json.loads(output) if output else []
    except Exception:
        return []


def _fetch_mootdx_1m(symbol: str, start: str, end: str) -> list[dict]:
    """Fetch 1m bars via mootdx."""
    from src.data.fetcher import _fetch_mootdx_ohlcv

    bars = _fetch_mootdx_ohlcv([symbol], start, end, "1m")
    return bars
