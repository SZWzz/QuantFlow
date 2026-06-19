"""Volatility factors — price range, standard deviation, and risk measures."""
import numpy as np
import pandas as pd

from src.factor.registry import register, FactorMeta


@register(
    FactorMeta(
        name="atr_14",
        category="volatility",
        description="14-day Average True Range (Wilder's smoothing)",
        default_params={"period": "14"},
    )
)
def atr_14(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 14))
    high, low, close = ohlcv["high"], ohlcv["low"], ohlcv["close"].shift(1)
    tr = pd.concat(
        [high - low, (high - close).abs(), (low - close).abs()], axis=1
    ).max(axis=1)
    return tr.ewm(alpha=1 / period, adjust=False).mean()


@register(
    FactorMeta(
        name="volatility_20d",
        category="volatility",
        description="20-day annualized volatility (std of daily returns * sqrt(252))",
        default_params={"period": "20"},
    )
)
def volatility_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    returns = ohlcv["close"].pct_change()
    return returns.rolling(window=period, min_periods=period).std() * np.sqrt(252)


@register(
    FactorMeta(
        name="volatility_60d",
        category="volatility",
        description="60-day annualized volatility (long-term risk measure)",
        default_params={"period": "60"},
    )
)
def volatility_60d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 60))
    returns = ohlcv["close"].pct_change()
    return returns.rolling(window=period, min_periods=period).std() * np.sqrt(252)


@register(
    FactorMeta(
        name="bollinger_width_20",
        category="volatility",
        description="Bollinger Band width: (upper - lower) / middle, 2 std",
        default_params={"period": "20", "num_std": "2"},
    )
)
def bollinger_width_20(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    num_std = float(params.get("num_std", 2))
    close = ohlcv["close"]
    middle = close.rolling(window=period, min_periods=period).mean()
    std = close.rolling(window=period, min_periods=period).std()
    upper = middle + num_std * std
    lower = middle - num_std * std
    return (upper - lower) / middle


@register(
    FactorMeta(
        name="max_drawdown_60d",
        category="volatility",
        description="60-day maximum drawdown from peak",
        default_params={"period": "60"},
    )
)
def max_drawdown_60d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 60))
    close = ohlcv["close"]
    rolling_max = close.rolling(window=period, min_periods=period).max()
    return (close - rolling_max) / rolling_max
