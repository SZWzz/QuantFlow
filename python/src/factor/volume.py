"""Volume factors — volume ratios, OBV, and liquidity measures."""
import numpy as np
import pandas as pd

from src.factor.registry import register, FactorMeta


@register(
    FactorMeta(
        name="volume_ratio_5d",
        category="volume",
        description="5-day volume ratio: volume / avg_volume_5d",
        default_params={"period": "5"},
    )
)
def volume_ratio_5d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 5))
    avg_vol = ohlcv["volume"].rolling(window=period, min_periods=period).mean()
    return ohlcv["volume"] / avg_vol.replace(0, np.nan)


@register(
    FactorMeta(
        name="volume_ratio_20d",
        category="volume",
        description="20-day volume ratio: volume / avg_volume_20d",
        default_params={"period": "20"},
    )
)
def volume_ratio_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    avg_vol = ohlcv["volume"].rolling(window=period, min_periods=period).mean()
    return ohlcv["volume"] / avg_vol.replace(0, np.nan)


@register(
    FactorMeta(
        name="obv",
        category="volume",
        description="On-Balance Volume: cumulative volume signed by price direction",
    )
)
def obv(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    sign = np.sign(ohlcv["close"].diff()).fillna(0)
    return (sign * ohlcv["volume"]).cumsum()


@register(
    FactorMeta(
        name="vwap_deviation",
        category="volume",
        description="Deviation from Volume-Weighted Average Price: (close - vwap) / vwap",
    )
)
def vwap_deviation(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    typical_price = (ohlcv["high"] + ohlcv["low"] + ohlcv["close"]) / 3
    cum_pv = (typical_price * ohlcv["volume"]).cumsum()
    cum_vol = ohlcv["volume"].cumsum()
    vwap = cum_pv / cum_vol.replace(0, np.nan)
    return (ohlcv["close"] - vwap) / vwap.replace(0, np.nan)


@register(
    FactorMeta(
        name="turnover_20d",
        category="volume",
        description="20-day average turnover rate proxy: volume / rolling_avg_volume",
        default_params={"period": "20"},
    )
)
def turnover_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    period = int(params.get("period", 20))
    avg_vol = ohlcv["volume"].rolling(window=period, min_periods=period).mean()
    return ohlcv["volume"] / avg_vol.replace(0, np.nan)
