"""Tests for momentum factors."""
import pandas as pd
import numpy as np
import pytest
from src.factor.momentum import momentum_20d, momentum_60d, rsi_14


def make_ohlcv(prices: list) -> pd.DataFrame:
    """Create a minimal OHLCV DataFrame with given close prices."""
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


class TestMomentum:
    def test_momentum_20d_positive(self):
        """20d momentum should be positive when price trends up."""
        df = make_ohlcv([100 + i for i in range(30)])  # 100 → 129
        result = momentum_20d(df, {"period": "20"})
        # close[-1]=129, close[-21]=109, return = 129/109 - 1 ≈ 0.1835
        assert result.iloc[-1] == pytest.approx(129 / 109 - 1, 0.01)

    def test_momentum_20d_negative(self):
        """20d momentum should be negative when price trends down."""
        df = make_ohlcv([130 - i for i in range(30)])  # 130 → 101
        result = momentum_20d(df, {"period": "20"})
        assert result.iloc[-1] < 0

    def test_momentum_early_nan(self):
        """Momentum should be NaN before period days."""
        df = make_ohlcv([100] * 30)
        result = momentum_20d(df, {"period": "20"})
        assert pd.isna(result.iloc[0])
        assert pd.isna(result.iloc[19])
        assert not pd.isna(result.iloc[20])


class TestRSI:
    def test_rsi_all_up(self):
        """RSI for steadily increasing prices should approach 100."""
        df = make_ohlcv([100 + i * 0.5 for i in range(50)])  # More data for RSI convergence
        result = rsi_14(df, {"period": "14"})
        assert result.iloc[-1] > 90  # Almost all gains

    def test_rsi_all_down(self):
        """RSI for steadily decreasing prices should approach 0."""
        df = make_ohlcv([120 - i for i in range(20)])
        result = rsi_14(df, {"period": "14"})
        assert result.iloc[-1] < 10  # Almost all losses

    def test_rsi_range(self):
        """RSI should always be in [0, 100]."""
        np.random.seed(42)
        prices = 100 + np.cumsum(np.random.randn(100) * 2)
        df = make_ohlcv(prices.tolist())
        result = rsi_14(df, {"period": "14"})
        valid = result.dropna()
        assert (valid >= 0).all()
        assert (valid <= 100).all()
