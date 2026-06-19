"""Tests for the DataService gRPC fetcher — mootdx (TDX) routing.

These tests assert the gRPC service forwards the requested K-line interval
through to mootdx (rather than silently returning daily bars for every
interval) and that the interval map is respected. No network calls and no
mootdx install required — the mootdx client is mocked.
"""

from unittest.mock import MagicMock, patch

import pytest

from src.data import fetcher
from src.proto import data_pb2


@pytest.fixture(autouse=True)
def _reset_mootdx_cache():
    """Ensure each test starts with no cached mootdx client.

    The fetcher caches the Quotes client module-wide; without a reset, a mock
    client pinned by one test leaks into the next and assertions on call_args
    see a stale mock. Cache-state tests call _reset_mootdx_client() explicitly
    too — this fixture just guarantees isolation for the rest.
    """
    fetcher._reset_mootdx_client()
    yield
    fetcher._reset_mootdx_client()


def _make_bars_request(interval: str) -> data_pb2.FetchDataRequest:
    """Build an OHLCV FetchDataRequest the way the Go MootdxAdapter does."""
    req = data_pb2.FetchDataRequest()
    req.source = "mootdx"
    req.data_type = "ohlcv"
    req.symbols.extend(["600519.SH"])
    req.start_date = "2026-06-10"
    req.end_date = "2026-06-12"
    req.params["interval"] = interval
    return req


def _fake_bars_payload() -> list[dict]:
    """One chunk (< _CHUNK_SIZE) of TDX-shaped bars within the request window.

    mootdx's client.bars() returns a list of dicts whose keys are lowercase
    ('datetime', 'open', ... 'vol'); the fetcher renames datetime→trade_date
    and vol→volume. Keep dates inside [start_date, end_date] so the slice
    result is non-empty and the function does not raise "no data".
    """
    return [
        {"datetime": "2026-06-10", "open": 1600.0, "high": 1610.0, "low": 1595.0, "close": 1605.0, "vol": 12000},
        {"datetime": "2026-06-11", "open": 1605.0, "high": 1620.0, "low": 1600.0, "close": 1618.0, "vol": 9800},
        {"datetime": "2026-06-12", "open": 1618.0, "high": 1630.0, "low": 1612.0, "close": 1625.0, "vol": 11000},
    ]


@pytest.mark.asyncio
async def test_handle_mootdx_ohlcv_forwards_interval_to_mootdx():
    """The interval from request.params must reach mootdx's frequency= arg.

    Regression guard: _handle_mootdx used to hardcode "1D", so a 1W request
    silently returned daily bars. With the fix, "1W" maps to frequency="week".
    """
    fake_client = MagicMock()
    fake_client.bars.return_value = _fake_bars_payload()

    with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client), \
         patch.object(fetcher, "_HAS_MOOTDX", True):
        svc = fetcher.DataService()
        resp = await svc.FetchData(_make_bars_request("1W"), context=None)

    assert not resp.error, f"expected data, got error: {resp.error}"
    # The single most important assertion: bars() was called with the mapped
    # frequency, not the hardcoded "day".
    fake_client.bars.assert_called_once()
    _, kwargs = fake_client.bars.call_args
    assert kwargs["frequency"] == "week", (
        f"interval not forwarded: expected frequency='week' for interval '1W', "
        f"got {kwargs.get('frequency')!r}"
    )


@pytest.mark.asyncio
async def test_handle_mootdx_ohlcv_interval_map_table():
    """Each supported interval maps to the expected mootdx frequency token."""
    cases = {
        "1D": "day",
        "1W": "week",
        "1M": "mon",
        "1m": "1m",
        "5m": "5m",
        "15m": "15m",
        "30m": "30m",
        "1H": "1h",
    }
    for interval, expected_freq in cases.items():
        # Reset the cache each iteration so this iteration's fake_client is the
        # one actually used (the cache otherwise reuses the first iteration's).
        fetcher._reset_mootdx_client()
        fake_client = MagicMock()
        fake_client.bars.return_value = _fake_bars_payload()

        with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client), \
             patch.object(fetcher, "_HAS_MOOTDX", True):
            svc = fetcher.DataService()
            resp = await svc.FetchData(_make_bars_request(interval), context=None)

        assert not resp.error, f"[{interval}] expected data, got error: {resp.error}"
        _, kwargs = fake_client.bars.call_args
        assert kwargs["frequency"] == expected_freq, (
            f"[{interval}] expected frequency={expected_freq!r}, got {kwargs.get('frequency')!r}"
        )


@pytest.mark.asyncio
async def test_handle_mootdx_ohlcv_default_interval_when_params_omitted():
    """When the caller omits interval, the fetcher defaults to daily (1D)."""
    req = data_pb2.FetchDataRequest()
    req.source = "mootdx"
    req.data_type = "ohlcv"
    req.symbols.extend(["600519.SH"])
    req.start_date = "2026-06-10"
    req.end_date = "2026-06-12"
    # no params set

    fake_client = MagicMock()
    fake_client.bars.return_value = _fake_bars_payload()

    with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client), \
         patch.object(fetcher, "_HAS_MOOTDX", True):
        svc = fetcher.DataService()
        resp = await svc.FetchData(req, context=None)

    assert not resp.error, f"expected data, got error: {resp.error}"
    _, kwargs = fake_client.bars.call_args
    assert kwargs["frequency"] == "day"


@pytest.mark.asyncio
async def test_handle_mootdx_ohlcv_unsupported_interval_errors():
    """An unsupported interval must raise a clear error, not silently fall back."""
    fake_client = MagicMock()  # should never be called

    with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client), \
         patch.object(fetcher, "_HAS_MOOTDX", True):
        svc = fetcher.DataService()
        resp = await svc.FetchData(_make_bars_request("2D"), context=None)

    assert resp.error, "expected an error for unsupported interval '2D'"
    assert "unsupported interval" in resp.error
    fake_client.bars.assert_not_called()


# ---------------------------------------------------------------------------
# mootdx Quotes client caching
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_mootdx_client_is_cached_across_fetches():
    """_get_mootdx_client builds the client once and reuses it across fetches.

    Regression guard: previously _init_mootdx_client() ran on every fetch, redoing
    the expensive setup()/factory() TDX probe each time.
    """
    fake_client = MagicMock()
    fake_client.bars.return_value = _fake_bars_payload()

    fetcher._reset_mootdx_client()  # start from a clean cache
    try:
        with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client) as init_mock, \
             patch.object(fetcher, "_HAS_MOOTDX", True):
            svc = fetcher.DataService()
            await svc.FetchData(_make_bars_request("1D"), context=None)
            await svc.FetchData(_make_bars_request("1D"), context=None)
            await svc.FetchData(_make_bars_request("1D"), context=None)
        assert init_mock.call_count == 1, (
            f"client should be built once across 3 fetches, built {init_mock.call_count} times"
        )
    finally:
        fetcher._reset_mootdx_client()


@pytest.mark.asyncio
async def test_mootdx_client_reset_after_failure_then_recovers():
    """A bars() failure resets the cache; the next fetch rebuilds and recovers."""
    bad = MagicMock()
    bad.bars.side_effect = RuntimeError("tdx connection reset")
    good = MagicMock()
    good.bars.return_value = _fake_bars_payload()

    fetcher._reset_mootdx_client()
    try:
        # First fetch: bad client → exception caught → cache reset. Second fetch:
        # cache empty → rebuild with good client → success.
        with patch.object(fetcher, "_init_mootdx_client", side_effect=[bad, good]), \
             patch.object(fetcher, "_HAS_MOOTDX", True):
            svc = fetcher.DataService()
            first = await svc.FetchData(_make_bars_request("1D"), context=None)
            second = await svc.FetchData(_make_bars_request("1D"), context=None)

        # First fetch produced no bars (bad client raised), so the service returns an
        # error string rather than data — that's the documented "no data" behavior.
        assert first.error, "expected the failed fetch to surface an error"
        assert not second.error, (
            f"expected recovery after client reset, got error: {second.error}"
        )
    finally:
        fetcher._reset_mootdx_client()

