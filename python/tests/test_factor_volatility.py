import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def sample_ohlcv():
    """Generate 100 bars of synthetic OHLCV data."""
    np.random.seed(42)
    n = 100
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    high = close + np.abs(np.random.randn(n) * 0.3)
    low = close - np.abs(np.random.randn(n) * 0.3)
    volume = np.random.rand(n) * 10000 + 5000
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": high,
        "low": low,
        "close": close,
        "volume": volume,
    })


class TestATR:
    def test_atr_returns_series(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result = atr_14(sample_ohlcv, {"period": "14"})
        assert isinstance(result, pd.Series)
        assert len(result) == len(sample_ohlcv)

    def test_atr_values_non_negative(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result = atr_14(sample_ohlcv, {"period": "14"})
        valid = result.dropna()
        assert (valid >= 0).all(), "ATR values should be non-negative"

    def test_atr_custom_period(self, sample_ohlcv):
        from src.factor.volatility import atr_14
        result5 = atr_14(sample_ohlcv, {"period": "5"})
        result20 = atr_14(sample_ohlcv, {"period": "20"})
        assert result5.dropna().std() != result20.dropna().std()


class TestVolatility:
    def test_volatility_20d_returns_series(self, sample_ohlcv):
        from src.factor.volatility import volatility_20d
        result = volatility_20d(sample_ohlcv, {"period": "20"})
        assert isinstance(result, pd.Series)

    def test_volatility_60d_smoother_than_20d(self, sample_ohlcv):
        from src.factor.volatility import volatility_20d, volatility_60d
        v20 = volatility_20d(sample_ohlcv, {"period": "20"})
        v60 = volatility_60d(sample_ohlcv, {"period": "60"})
        valid_idx = v60.dropna().index
        if len(valid_idx) > 10:
            assert v60.loc[valid_idx].diff().std() < v20.loc[valid_idx].diff().std() * 1.5
