import numpy as np
import pandas as pd
import pytest


@pytest.fixture
def sample_ohlcv():
    np.random.seed(42)
    n = 100
    close = 100 + np.cumsum(np.random.randn(n) * 0.5)
    high = close + np.abs(np.random.randn(n) * 0.3)
    low = close - np.abs(np.random.randn(n) * 0.3)
    volume = np.random.randint(5000, 50000, n)
    return pd.DataFrame({
        "open": close - np.random.randn(n) * 0.1,
        "high": high, "low": low, "close": close, "volume": volume,
    })


class TestVolumeRatio:
    def test_volume_ratio_5d_returns_series(self, sample_ohlcv):
        from src.factor.volume import volume_ratio_5d
        result = volume_ratio_5d(sample_ohlcv, {"period": "5"})
        assert isinstance(result, pd.Series)

    def test_volume_ratio_20d(self, sample_ohlcv):
        from src.factor.volume import volume_ratio_20d
        result = volume_ratio_20d(sample_ohlcv, {"period": "20"})
        assert len(result) == len(sample_ohlcv)


class TestOBV:
    def test_obv_returns_series(self, sample_ohlcv):
        from src.factor.volume import obv
        result = obv(sample_ohlcv, {})
        assert isinstance(result, pd.Series)
        assert len(result) == len(sample_ohlcv)


class TestVWAP:
    def test_vwap_deviation_returns_series(self, sample_ohlcv):
        from src.factor.volume import vwap_deviation
        result = vwap_deviation(sample_ohlcv, {})
        assert isinstance(result, pd.Series)
