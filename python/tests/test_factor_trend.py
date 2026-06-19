"""Tests for trend factors."""
import pandas as pd
import numpy as np
import pytest
from src.factor.trend import sma_5, sma_20, sma_5_minus_sma_20, macd_12_26_9


def make_ohlcv(prices: list) -> pd.DataFrame:
    n = len(prices)
    return pd.DataFrame(
        {
            "open": prices,
            "high": [p * 1.01 for p in prices],
            "low": [p * 0.99 for p in prices],
            "close": prices,
            "volume": [1000] * n,
        }
    )


class TestSMA:
    def test_sma_flat(self):
        """SMA of constant prices equals the constant."""
        df = make_ohlcv([100] * 30)
        result = sma_5(df, {"period": "5"})
        assert result.iloc[-1] == 100.0

    def test_sma_20_approximates_mean(self):
        """SMA should equal the arithmetic mean over the window."""
        df = make_ohlcv(list(range(1, 31)))  # 1..30
        result = sma_20(df, {"period": "20"})
        # Last 20 values: 11..30, mean = 20.5
        assert result.iloc[-1] == pytest.approx(20.5, 0.01)

    def test_sma_5_minus_sma_20_golden_cross(self):
        """When short MA crosses above long MA, diff becomes positive."""
        # Create data where sma_5 crosses above sma_20
        prices = [100] * 20 + [110] * 10  # Spike at end
        df = make_ohlcv(prices)
        result = sma_5_minus_sma_20(df, {})
        assert result.iloc[-1] > 0  # Short > Long after spike


class TestMACD:
    def test_macd_flat_zero(self):
        """MACD should be near zero for flat prices."""
        df = make_ohlcv([100] * 100)
        result = macd_12_26_9(df, {})
        valid = result.dropna()
        assert (valid.abs() < 1e-6).all()

    def test_macd_output_not_empty(self):
        """MACD should produce non-NaN values after warmup."""
        np.random.seed(42)
        prices = 100 + np.cumsum(np.random.randn(200))
        df = make_ohlcv(prices.tolist())
        result = macd_12_26_9(df, {})
        assert not pd.isna(result.iloc[-1])
