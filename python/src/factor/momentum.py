"""Momentum factors — price trend and RSI family."""
import numpy as np
import pandas as pd

from src.factor.registry import register, FactorMeta


@register(
    FactorMeta(
        name="momentum_20d",
        category="momentum",
        description="20-day price momentum: close / close_20d_ago - 1",
        default_params={"period": "20"},
    )
)
def momentum_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    return ohlcv["close"].pct_change(period)


@register(
    FactorMeta(
        name="momentum_60d",
        category="momentum",
        description="60-day price momentum",
        default_params={"period": "60"},
    )
)
def momentum_60d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 60))
    return ohlcv["close"].pct_change(period)


@register(
    FactorMeta(
        name="momentum_120d",
        category="momentum",
        description="120-day price momentum (long-term trend)",
        default_params={"period": "120"},
    )
)
def momentum_120d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 120))
    return ohlcv["close"].pct_change(period)


@register(
    FactorMeta(
        name="momentum_5d_minus_20d",
        category="momentum",
        description="Short minus medium momentum: reversal/continuation signal",
    )
)
def momentum_5d_minus_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    return ohlcv["close"].pct_change(5) - ohlcv["close"].pct_change(20)


@register(
    FactorMeta(
        name="rsi_14",
        category="momentum",
        description="14-day Relative Strength Index (Wilder). >70=overbought, <30=oversold",
        default_params={"period": "14"},
    )
)
def rsi_14(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 14))
    delta = ohlcv["close"].diff()
    gain = delta.clip(lower=0).ewm(alpha=1 / period, adjust=False).mean()
    loss = (-delta.clip(upper=0)).ewm(alpha=1 / period, adjust=False).mean()

    # Compute RSI with edge case handling
    rsi = pd.Series(np.nan, index=ohlcv.index)
    mask_both = (gain > 0) & (loss > 0)
    rsi[mask_both] = 100.0 - (100.0 / (1.0 + gain[mask_both] / loss[mask_both]))
    rsi[(loss == 0) & (gain > 0)] = 100.0  # No losses → RSI = 100
    rsi[(gain == 0) & (loss > 0)] = 0.0    # No gains → RSI = 0
    return rsi
