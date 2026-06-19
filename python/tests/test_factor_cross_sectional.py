"""Tests for cross-sectional factor functions.

Covers: zscore_momentum_20d, rank_momentum_20d, zscore_volatility_20d,
        zscore_volume_ratio_5d, size_factor.
"""
import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def multi_symbol_ohlcv():
    """Synthetic multi-symbol OHLCV with 3 symbols, 60 days each."""
    np.random.seed(42)
    n_days = 60
    symbols = ["A", "B", "C"]

    frames = []
    for sym in symbols:
        # Each symbol has a different drift so their momentum differs
        drift = {"A": 0.1, "B": 0.05, "C": -0.02}[sym]
        close = 100 + np.arange(n_days) * drift + np.cumsum(np.random.randn(n_days) * 0.3)
        volume = np.random.randint(5000, 50000, n_days)
        dates = pd.date_range("2025-01-01", periods=n_days, freq="D")
        frames.append(pd.DataFrame({
            "date": dates,
            "symbol": sym,
            "open": close - np.random.randn(n_days) * 0.1,
            "high": close + np.abs(np.random.randn(n_days) * 0.2),
            "low": close - np.abs(np.random.randn(n_days) * 0.2),
            "close": close,
            "volume": volume,
        }))
    return pd.concat(frames, ignore_index=True)


@pytest.fixture
def single_symbol_ohlcv():
    """Single-symbol DataFrame (no 'symbol' column) to test fallback."""
    np.random.seed(42)
    n = 60
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    volume = np.random.randint(5000, 50000, n)
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": close + np.abs(np.random.randn(n) * 0.3),
        "low": close - np.abs(np.random.randn(n) * 0.3),
        "close": close,
        "volume": volume,
    })


class TestZScoreMomentum20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_returns_nan_for_single_symbol(self, single_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(single_symbol_ohlcv, {})
        assert result.isna().all()

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        """Z-scores should have mean ≈ 0 within each date group (ignoring NaNs)."""
        from src.factor.cross_sectional import zscore_momentum_20d
        result = zscore_momentum_20d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        # Check a few dates after the burn-in
        for date in df["date"].unique()[25:30]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestRankMomentum20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_rank_between_zero_and_one(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(multi_symbol_ohlcv, {})
        valid = result.dropna()
        assert (valid >= 0).all()
        assert (valid <= 1).all()

    def test_returns_nan_for_single_symbol(self, single_symbol_ohlcv):
        from src.factor.cross_sectional import rank_momentum_20d
        result = rank_momentum_20d(single_symbol_ohlcv, {})
        assert result.isna().all()


class TestZScoreVolatility20d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volatility_20d
        result = zscore_volatility_20d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volatility_20d
        result = zscore_volatility_20d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        for date in df["date"].unique()[25:30]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestZScoreVolumeRatio5d:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volume_ratio_5d
        result = zscore_volume_ratio_5d(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_zscore_mean_near_zero_per_date(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import zscore_volume_ratio_5d
        result = zscore_volume_ratio_5d(multi_symbol_ohlcv, {})
        df = multi_symbol_ohlcv.copy()
        df["z"] = result
        for date in df["date"].unique()[10:15]:
            group = df.loc[df["date"] == date, "z"].dropna()
            if len(group) > 0:
                assert abs(group.mean()) < 1e-10


class TestSizeFactor:
    def test_returns_series_with_multi_symbol(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import size_factor
        result = size_factor(multi_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(multi_symbol_ohlcv)

    def test_returns_series_with_single_symbol(self, single_symbol_ohlcv):
        """size_factor works on single-symbol DataFrames too (no symbol col)."""
        from src.factor.cross_sectional import size_factor
        result = size_factor(single_symbol_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(single_symbol_ohlcv)

    def test_values_are_finite_where_available(self, multi_symbol_ohlcv):
        from src.factor.cross_sectional import size_factor
        result = size_factor(multi_symbol_ohlcv, {})
        valid = result.dropna()
        assert np.isfinite(valid).all()
