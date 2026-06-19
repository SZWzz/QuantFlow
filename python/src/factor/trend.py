"""Trend factors — moving averages, MACD, and directional indicators."""
import numpy as np
import pandas as pd

from src.factor.registry import register, FactorMeta


def _sma(series: pd.Series, period: int) -> pd.Series:
    """Simple Moving Average."""
    return series.rolling(window=period, min_periods=period).mean()


def _ema(series: pd.Series, period: int) -> pd.Series:
    """Exponential Moving Average."""
    return series.ewm(span=period, adjust=False).mean()


@register(
    FactorMeta(
        name="sma_5",
        category="trend",
        description="5-day Simple Moving Average",
        default_params={"period": "5"},
    )
)
def sma_5(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 5))
    return _sma(ohlcv["close"], period)


@register(
    FactorMeta(
        name="sma_20",
        category="trend",
        description="20-day Simple Moving Average (monthly trend)",
        default_params={"period": "20"},
    )
)
def sma_20(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    return _sma(ohlcv["close"], period)


@register(
    FactorMeta(
        name="sma_60",
        category="trend",
        description="60-day Simple Moving Average (quarterly trend)",
        default_params={"period": "60"},
    )
)
def sma_60(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 60))
    return _sma(ohlcv["close"], period)


@register(
    FactorMeta(
        name="sma_5_minus_sma_20",
        category="trend",
        description="Short-term trend minus medium-term: golden/death cross signal",
    )
)
def sma_5_minus_sma_20(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    close = ohlcv["close"]
    return (_sma(close, 5) - _sma(close, 20)) / close


@register(
    FactorMeta(
        name="macd_12_26_9",
        category="trend",
        description="MACD (12/26/9): DIF - DEA signal line",
    )
)
def macd_12_26_9(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    close = ohlcv["close"]
    dif = _ema(close, 12) - _ema(close, 26)
    dea = _ema(dif, 9)
    return dif - dea  # MACD histogram
