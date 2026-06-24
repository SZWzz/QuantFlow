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

        async def compute_one(factor_name):
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

        # Fan out: compute all factors concurrently.
        tasks = [compute_one(name) for name in request.factor_names]
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
