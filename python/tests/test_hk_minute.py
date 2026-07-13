"""Tests for HK minute data fetching via AKShare."""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parent.parent / "src"
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

import json
from unittest.mock import patch, MagicMock
import pandas as pd
import pytest


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
    """Return cached result within 60s."""
    from src.data.fetcher import _fetch_akshare_hk_minute, _FETCH_AKSHARE_HK_MINUTE_CACHE

    _FETCH_AKSHARE_HK_MINUTE_CACHE["00700"] = [{"date": "09:30", "close": 100.0}]
    from src.data.fetcher import _FETCH_AKSHARE_HK_MINUTE_CACHE_TS
    import time
    _FETCH_AKSHARE_HK_MINUTE_CACHE_TS["00700"] = time.time()

    with patch("src.data.fetcher.importlib.import_module") as mock_import:
        result = _fetch_akshare_hk_minute("00700")
        mock_import.assert_not_called()
        assert len(result) == 1


def test_fetch_akshare_hk_minute_import_error():
    """Return empty list when akshare is not installed."""
    from src.data.fetcher import _fetch_akshare_hk_minute, _FETCH_AKSHARE_HK_MINUTE_CACHE, _FETCH_AKSHARE_HK_MINUTE_CACHE_TS

    _FETCH_AKSHARE_HK_MINUTE_CACHE.clear()
    _FETCH_AKSHARE_HK_MINUTE_CACHE_TS.clear()

    with patch("src.data.fetcher.importlib.import_module", side_effect=ImportError):
        result = _fetch_akshare_hk_minute("00700")
        assert result == []
