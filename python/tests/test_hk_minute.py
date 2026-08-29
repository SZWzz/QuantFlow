"""Tests for HK minute data fetching via AKShare.

Note: _fetch_akshare_hk_minute is @lru_cache(maxsize=128) since 2026-07-30
(migrated from an unbounded dict + 60s TTL). Tests must call cache_clear()
to isolate cases, and cache semantics are now "LRU bound", not "60s TTL".
"""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parent.parent / "src"
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

from unittest.mock import patch, MagicMock
import pandas as pd
import pytest


@pytest.fixture(autouse=True)
def _clear_hk_minute_cache():
    from src.data.fetcher import _fetch_akshare_hk_minute

    _fetch_akshare_hk_minute.cache_clear()
    yield
    _fetch_akshare_hk_minute.cache_clear()


def test_fetch_akshare_hk_minute_empty_df():
    """Return empty list when DataFrame is empty."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    mock_df = pd.DataFrame()
    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_hk_hist_min_em.return_value = mock_df
        mock_import.return_value = mock_ak

        result = _fetch_akshare_hk_minute("00700")
        assert result == []


def test_fetch_akshare_hk_minute_parses_columns():
    """Parse Chinese column names to standard format."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    df = pd.DataFrame({
        "时间": ["2026-07-13 09:30:00", "2026-07-13 09:31:00"],
        "开盘": [100.0, 101.0],
        "收盘": [100.5, 101.5],
        "最高": [101.0, 102.0],
        "最低": [99.5, 100.5],
        "成交量": [10000, 15000],
        "成交额": [1005000, 1522500],
    })

    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_hk_hist_min_em.return_value = df
        mock_import.return_value = mock_ak

        result = _fetch_akshare_hk_minute("00700")
        assert len(result) == 2
        assert result[0]["symbol"] == "00700"
        assert result[0]["close"] == 100.5
        assert result[0]["volume"] == 10000
        assert result[0]["amount"] == 1005000
        assert result[0]["date"] == "2026-07-13 09:30:00"


def test_fetch_akshare_hk_minute_cache():
    """Second call for the same symbol hits the LRU cache (no re-import)."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    df = pd.DataFrame({
        "时间": ["2026-07-13 09:30:00"],
        "收盘": [100.5],
    })

    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_hk_hist_min_em.return_value = df
        mock_import.return_value = mock_ak

        first = _fetch_akshare_hk_minute("00700")
        second = _fetch_akshare_hk_minute("00700")

        # lru_cache: the underlying import happens only once per symbol
        assert mock_import.call_count == 1
        assert first == second
        assert len(second) == 1


def test_fetch_akshare_hk_minute_import_error():
    """Return empty list when akshare is not installed."""
    from src.data.fetcher import _fetch_akshare_hk_minute

    with patch("src.data.fetcher.importlib.import_module", side_effect=ImportError):
        result = _fetch_akshare_hk_minute("00700")
        assert result == []
