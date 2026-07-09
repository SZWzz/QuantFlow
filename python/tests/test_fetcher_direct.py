import pytest
from unittest.mock import patch, MagicMock
import pandas as pd
import asyncio

import src.data.fetcher as _fetcher_mod


@pytest.mark.asyncio
async def test_handle_akshare_direct():
    """Verify that _handle_akshare can call akshare functions directly via importlib."""
    from src.data.fetcher import _handle_akshare

    mock_df = pd.DataFrame({"col1": [1, 2, 3]})

    with patch("importlib.import_module") as mock_import:
        mock_ak = MagicMock()
        mock_ak.stock_zh_a_hist.return_value = mock_df
        mock_import.return_value = mock_ak

        result = await _handle_akshare("stock_zh_a_hist", symbol="600519", period="daily")
        assert result == [{"col1": 1}, {"col1": 2}, {"col1": 3}]
        mock_ak.stock_zh_a_hist.assert_called_once_with(symbol="600519", period="daily")


@pytest.mark.asyncio
async def test_handle_akshare_fallback():
    """Test fallback to subprocess when direct import fails."""
    from src.data.fetcher import _handle_akshare

    with patch.object(_fetcher_mod, '_akshare_module', None):
        with patch("src.data.fetcher._run_akshare_subprocess", return_value=[{"col": 1}]) as mock_sub:
            with patch("importlib.import_module", side_effect=ImportError("not found")):
                result = await _handle_akshare("stock_zh_a_hist", symbol="600519")
                mock_sub.assert_called_once()
                assert result == [{"col": 1}]


@pytest.mark.asyncio
async def test_handle_akshare_empty_dataframe():
    """Test handling of empty DataFrame result."""
    from src.data.fetcher import _handle_akshare

    empty_df = pd.DataFrame()
    with patch.object(_fetcher_mod, '_akshare_module', None):
        with patch("importlib.import_module") as mock_import:
            mock_ak = MagicMock()
            mock_ak.stock_zh_a_hist.return_value = empty_df
            mock_import.return_value = mock_ak

            result = await _handle_akshare("stock_zh_a_hist", symbol="600519")
            assert result == []
