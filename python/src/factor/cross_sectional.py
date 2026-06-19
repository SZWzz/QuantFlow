"""Cross-sectional factors — ranking, z-score, and relative strength."""
import numpy as np
import pandas as pd

from src.factor.registry import register, FactorMeta


@register(
    FactorMeta(
        name="zscore_momentum_20d",
        category="cross_sectional",
        description="Z-score of 20d momentum within a cross-section (for multi-symbol DataFrames)",
    )
)
def zscore_momentum_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    """Cross-sectional z-score of 20d momentum.

    When applied to a multi-symbol DataFrame with a 'symbol' column,
    computes per-day cross-sectional z-scores. For single-symbol DataFrames,
    returns NaN (cross-sectional factor requires multiple symbols).
    """
    if "symbol" not in ohlcv.columns:
        return pd.Series(np.nan, index=ohlcv.index)

    # Compute momentum per symbol
    ohlcv = ohlcv.copy()
    ohlcv["mom"] = ohlcv.groupby("symbol")["close"].transform(
        lambda x: x.pct_change(20)
    )

    # Cross-sectional z-score per date
    ohlcv["zscore"] = ohlcv.groupby("date")["mom"].transform(
        lambda x: (x - x.mean()) / x.std() if x.std() > 0 else 0
    )

    return ohlcv["zscore"]


@register(
    FactorMeta(
        name="rank_momentum_20d",
        category="cross_sectional",
        description="Cross-sectional percentile rank of 20d momentum (0-1)",
    )
)
def rank_momentum_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    """Cross-sectional percentile rank of 20d momentum."""
    if "symbol" not in ohlcv.columns:
        return pd.Series(np.nan, index=ohlcv.index)

    ohlcv = ohlcv.copy()
    ohlcv["mom"] = ohlcv.groupby("symbol")["close"].transform(
        lambda x: x.pct_change(20)
    )

    ohlcv["rank"] = ohlcv.groupby("date")["mom"].transform(
        lambda x: x.rank(pct=True) if len(x) > 1 else 0.5
    )

    return ohlcv["rank"]


@register(
    FactorMeta(
        name="zscore_volatility_20d",
        category="cross_sectional",
        description="Z-score of 20d volatility within a cross-section",
    )
)
def zscore_volatility_20d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    """Cross-sectional z-score of 20d annualized volatility."""
    if "symbol" not in ohlcv.columns:
        return pd.Series(np.nan, index=ohlcv.index)

    ohlcv = ohlcv.copy()
    ohlcv["vol"] = ohlcv.groupby("symbol")["close"].transform(
        lambda x: x.pct_change().rolling(20, min_periods=20).std() * np.sqrt(252)
    )

    ohlcv["zscore"] = ohlcv.groupby("date")["vol"].transform(
        lambda x: (x - x.mean()) / x.std() if x.std() > 0 else 0
    )

    return ohlcv["zscore"]


@register(
    FactorMeta(
        name="zscore_volume_ratio_5d",
        category="cross_sectional",
        description="Z-score of 5d volume ratio within a cross-section",
    )
)
def zscore_volume_ratio_5d(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    """Cross-sectional z-score of 5d volume ratio."""
    if "symbol" not in ohlcv.columns:
        return pd.Series(np.nan, index=ohlcv.index)

    ohlcv = ohlcv.copy()
    ohlcv["vol_ratio"] = ohlcv.groupby("symbol").apply(
        lambda g: g["volume"] / g["volume"].rolling(5, min_periods=5).mean().replace(0, np.nan)
    ).reset_index(level=0, drop=True)

    ohlcv["zscore"] = ohlcv.groupby("date")["vol_ratio"].transform(
        lambda x: (x - x.mean()) / x.std() if x.std() > 0 else 0
    )

    return ohlcv["zscore"]


@register(
    FactorMeta(
        name="size_factor",
        category="cross_sectional",
        description="Log of average daily turnover as a size/liquidity proxy",
    )
)
def size_factor(ohlcv: pd.DataFrame, params: dict) -> pd.Series:
    """Log of average daily turnover as a size/liquidity proxy."""
    if "symbol" not in ohlcv.columns:
        # Single symbol: just log volume
        avg_vol = ohlcv["volume"].rolling(window=20, min_periods=20).mean()
        return np.log(avg_vol.replace(0, np.nan))

    ohlcv = ohlcv.copy()
    ohlcv["avg_turnover"] = ohlcv.groupby("symbol")["volume"].transform(
        lambda x: x.rolling(20, min_periods=20).mean()
    )
    return np.log(ohlcv["avg_turnover"].replace(0, np.nan))
