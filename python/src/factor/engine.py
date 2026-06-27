"""FactorService gRPC implementation."""
import asyncio
import io
import time
import logging

import pandas as pd
import pyarrow as pa
import pyarrow.ipc as ipc

from src.proto import factor_pb2, factor_pb2_grpc
from src.factor.registry import list_factors, compute

logger = logging.getLogger(__name__)


class FactorService(factor_pb2_grpc.FactorServiceServicer):
    """gRPC service for factor computation."""

    async def ComputeFactor(self, request, context):
        t0 = time.time()
        try:
            from src.factor.registry import _compute_funcs, is_cross_sectional
            if request.factor_name not in _compute_funcs:
                return factor_pb2.ComputeFactorResponse(
                    factor_name=request.factor_name,
                    error=f"Unknown factor: {request.factor_name}",
                )

            # Decode Arrow IPC bytes → pandas DataFrame
            if request.ohlcv_data:
                reader = ipc.open_stream(request.ohlcv_data)
                table = reader.read_all()
                df = table.to_pandas()
            else:
                df = pd.DataFrame()

            results = []

            if df.empty:
                # No data — return empty results for each requested symbol
                for symbol in request.symbols:
                    results.append(factor_pb2.FactorResult(symbol=symbol))
            elif is_cross_sectional(request.factor_name) and "symbol" in df.columns:
                # P0-3 fix: cross-sectional factors need the FULL multi-symbol panel.
                # Compute once on the complete DataFrame, then slice per symbol.
                values = compute(request.factor_name, df, dict(request.params))
                df_with_vals = df.copy()
                df_with_vals["_factor_val"] = values.values

                for symbol in request.symbols:
                    symbol_vals = df_with_vals[df_with_vals["symbol"] == symbol]["_factor_val"]
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=df_with_vals[df_with_vals["symbol"] == symbol]["date"].astype(str).tolist()
                            if "date" in df_with_vals.columns else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in symbol_vals.tolist()],
                        )
                    )
            else:
                # Time-series factor: compute per symbol (original behavior)
                for symbol in request.symbols:
                    if "symbol" in df.columns:
                        symbol_df = df[df["symbol"] == symbol]
                    else:
                        symbol_df = df
                    if symbol_df.empty:
                        continue
                    values = compute(request.factor_name, symbol_df, dict(request.params))
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=values.index.astype(str).tolist()
                            if hasattr(values, "index") else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in values.tolist()],
                        )
                    )

            elapsed_ms = int((time.time() - t0) * 1000)
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                results=results,
                compute_time_ms=elapsed_ms,
            )
        except Exception as e:
            logger.exception(f"ComputeFactor failed: {e}")
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                error=str(e),
            )

    async def ComputeFactorBatch(self, request, context):
        t0 = time.time()

        # Pre-decode OHLCV data once (was re-parsed O(N) times in the old loop).
        if request.ohlcv_data:
            reader = ipc.open_stream(request.ohlcv_data)
            table = reader.read_all()
            df = table.to_pandas()
        else:
            df = pd.DataFrame()

        def compute_one(factor_name):
            """Compute a single factor using the pre-decoded DataFrame."""
            from src.factor.registry import _compute_funcs, is_cross_sectional
            if factor_name not in _compute_funcs:
                return factor_pb2.ComputeFactorResponse(
                    factor_name=factor_name,
                    error=f"Unknown factor: {factor_name}",
                )
            results = []

            if df.empty:
                for symbol in request.symbols:
                    results.append(factor_pb2.FactorResult(symbol=symbol))
            elif is_cross_sectional(factor_name) and "symbol" in df.columns:
                # P0-3 fix: full panel for cross-sectional
                values = compute(factor_name, df, dict(request.params))
                df_with_vals = df.copy()
                df_with_vals["_factor_val"] = values.values
                for symbol in request.symbols:
                    sub = df_with_vals[df_with_vals["symbol"] == symbol]
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=sub["date"].astype(str).tolist() if "date" in sub.columns else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in sub["_factor_val"].tolist()],
                        )
                    )
            else:
                for symbol in request.symbols:
                    symbol_df = df[df["symbol"] == symbol] if "symbol" in df.columns else df
                    if symbol_df.empty:
                        continue
                    values = compute(factor_name, symbol_df, dict(request.params))
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=values.index.astype(str).tolist() if hasattr(values, "index") else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in values.tolist()],
                        )
                    )
            return factor_pb2.ComputeFactorResponse(
                factor_name=factor_name,
                results=results,
            )

        # Fan out: compute all factors concurrently via thread pool.
        tasks = [asyncio.to_thread(compute_one, name) for name in request.factor_names]
        responses = await asyncio.gather(*tasks)

        elapsed_ms = int((time.time() - t0) * 1000)
        return factor_pb2.ComputeFactorBatchResponse(
            factor_responses=list(responses),
            total_compute_time_ms=elapsed_ms,
        )

    async def ListFactors(self, request, context):
        factors = list_factors()
        return factor_pb2.ListFactorsResponse(
            factors=[
                factor_pb2.FactorMeta(
                    name=f.name,
                    category=f.category,
                    description=f.description,
                    default_params=f.default_params,
                )
                for f in factors
            ]
        )


# ── Factor Analysis Utilities ──────────────────────────────────────────────────

import numpy as np
from pandas import Series, DataFrame
from typing import List, Dict, Tuple, Optional


def compute_ic(factor_values: Series, forward_returns: Series) -> float:
    """Compute Information Coefficient (rank correlation) between factor and forward returns."""
    common = factor_values.dropna().index.intersection(forward_returns.dropna().index)
    if len(common) < 5:
        return 0.0
    return factor_values.loc[common].rank().corr(forward_returns.loc[common].rank())


def quantile_backtest(
    factor_values: Series, returns: Series, n_quantiles: int = 5
) -> Dict[str, List[float]]:
    """分层回测: group assets by factor quantile, compute avg return per group per period.

    Returns dict with keys 'q1'..'qN', 'spread', 'cum_pnl'."""
    common = factor_values.dropna().index.intersection(returns.dropna().index)
    if len(common) < n_quantiles:
        return {"q1": [], "spread": [], "cum_pnl": []}

    fv = factor_values.loc[common]
    ret = returns.loc[common]
    fv_clean = fv.replace([np.inf, -np.inf], np.nan).dropna()
    ret_clean = ret.replace([np.inf, -np.inf], np.nan)
    common_clean = fv_clean.index.intersection(ret_clean.index)
    if len(common_clean) < n_quantiles:
        return {"q1": [], "spread": [], "cum_pnl": []}

    fv = fv.loc[common_clean]
    ret = ret.loc[common_clean]
    labels = [f"q{i+1}" for i in range(n_quantiles)]
    quantiles = pd.qcut(fv, n_quantiles, labels=labels, duplicates="drop")
    result: Dict[str, List[float]] = {}
    for q in labels:
        mask = quantiles == q
        if mask.any():
            result[q] = ret[mask].values.tolist()
        else:
            result[q] = []

    if result[labels[0]] and result[labels[-1]]:
        top = np.mean(result[labels[-1]])
        bottom = np.mean(result[labels[0]])
        result["spread"] = [top - bottom]
    else:
        result["spread"] = []

    result["cum_pnl"] = [(np.mean(r) if r else 0.0) for r in result.values() if r]
    return result


def ic_decay(
    factor_values: Series, forward_returns: DataFrame, periods: List[int] = None
) -> List[float]:
    """IC decay over multiple forward periods."""
    if periods is None:
        periods = [1, 5, 10, 20]
    decay = []
    for p in periods:
        col = f"ret_{p}d" if f"ret_{p}d" in forward_returns.columns else forward_returns.columns[0]
        fwd = forward_returns[col] if col in forward_returns.columns else forward_returns.iloc[:, 0]
        decay.append(compute_ic(factor_values, fwd))
    return decay


def winsorize(series: Series, limits: Tuple[float, float] = (0.01, 0.99)) -> Series:
    """Winsorize (clamp) extreme values at percentile limits."""
    lower = series.quantile(limits[0])
    upper = series.quantile(limits[1])
    return series.clip(lower, upper)


def neutralize(
    factor: Series, industry: Series, market_cap: Series
) -> Series:
    """Industry + market-cap neutralization via OLS residuals.

    Regresses factor ~ industry_dummies + log(market_cap), returns residuals."""
    import statsmodels.api as sm
    common = factor.dropna().index.intersection(industry.dropna().index).intersection(market_cap.dropna().index)
    if len(common) < 10:
        return factor

    df = DataFrame({
        "factor": factor.loc[common],
        "industry": industry.loc[common].astype(str),
        "log_mcap": np.log(market_cap.loc[common].replace(0, np.nan)).fillna(0),
    })
    industry_dummies = pd.get_dummies(df["industry"], drop_first=True).astype(float)
    X = sm.add_constant(industry_dummies)
    X["log_mcap"] = df["log_mcap"].values
    X = X.replace([np.inf, -np.inf], np.nan).fillna(0)
    model = sm.OLS(df["factor"].values, X.values).fit()
    residuals = model.resid
    result = Series(residuals, index=common)
    return result.reindex(factor.index).fillna(0.0)


def standardize(series: Series) -> Series:
    """Z-score standardization: (x - mean) / std."""
    mu = series.mean()
    sigma = series.std()
    if sigma == 0:
        return series - mu
    return (series - mu) / sigma
