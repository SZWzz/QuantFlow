"""Integration tests for the FactorService via gRPC."""
import pytest
import pandas as pd
import numpy as np
import pyarrow as pa
import pyarrow.ipc as ipc
import grpc

from src.proto import factor_pb2, factor_pb2_grpc, health_pb2, health_pb2_grpc

# Import all factor modules to trigger registration
from src.factor import momentum, trend, volatility, volume, cross_sectional  # noqa: F401
from src.factor.registry import list_factors


def make_ohlcv_arrow(prices: list, symbol: str = "000001.SZ") -> bytes:
    """Create Arrow IPC bytes for a simple OHLCV DataFrame."""
    df = pd.DataFrame(
        {
            "symbol": [symbol] * len(prices),
            "date": pd.date_range("2024-01-01", periods=len(prices), freq="D"),
            "open": prices,
            "high": [p * 1.02 for p in prices],
            "low": [p * 0.98 for p in prices],
            "close": prices,
            "volume": [10000] * len(prices),
        }
    )
    table = pa.Table.from_pandas(df)
    sink = pa.BufferOutputStream()
    writer = ipc.new_stream(sink, table.schema)
    writer.write_table(table)
    writer.close()
    return sink.getvalue().to_pybytes()


class TestFactorRegistry:
    def test_at_least_25_factors(self):
        """Ensures we have the minimum number of factors registered."""
        factors = list_factors()
        assert len(factors) >= 25, f"Expected >= 25 factors, got {len(factors)}"

    def test_all_categories_present(self):
        """Verify all 5 factor categories have at least one factor."""
        factors = list_factors()
        categories = {f.category for f in factors}
        expected = {"momentum", "trend", "volatility", "volume", "cross_sectional"}
        assert expected.issubset(categories), f"Missing categories: {expected - categories}"


class TestFactorServiceIntegration:
    """Integration tests requiring a running Python sidecar."""

    @pytest.fixture(scope="class")
    def channel(self):
        """Connect to the running gRPC server."""
        try:
            ch = grpc.insecure_channel("localhost:50051")
            grpc.channel_ready_future(ch).result(timeout=2)
            return ch
        except grpc.FutureTimeoutError:
            pytest.skip("Python gRPC sidecar not running on localhost:50051")

    def test_health_ping(self, channel):
        """Health check should return healthy."""
        stub = health_pb2_grpc.HealthServiceStub(channel)
        resp = stub.Ping(health_pb2.PingRequest())
        assert resp.healthy
        assert resp.version != "", "version should not be empty"

    def test_list_factors(self, channel):
        """ListFactors should return 25+ factors with metadata."""
        stub = factor_pb2_grpc.FactorServiceStub(channel)
        resp = stub.ListFactors(factor_pb2.ListFactorsRequest())
        assert len(resp.factors) >= 25
        # Verify each factor has required fields
        for f in resp.factors:
            assert f.name
            assert f.category
            assert f.description

    def test_compute_momentum_20d(self, channel):
        """Compute momentum_20d for a simple price series."""
        stub = factor_pb2_grpc.FactorServiceStub(channel)

        prices = [100 + i for i in range(30)]
        arrow_data = make_ohlcv_arrow(prices, "000001.SZ")

        resp = stub.ComputeFactor(
            factor_pb2.ComputeFactorRequest(
                factor_name="momentum_20d",
                symbols=["000001.SZ"],
                start_date="2024-01-01",
                end_date="2024-01-30",
                ohlcv_data=arrow_data,
            )
        )
        assert resp.error == "", f"Factor error: {resp.error}"
        assert len(resp.results) == 1
        assert resp.results[0].symbol == "000001.SZ"
        assert resp.compute_time_ms > 0
        assert len(resp.results[0].values) > 0

    def test_compute_rsi(self, channel):
        """Compute RSI and verify values are in [0, 100]."""
        stub = factor_pb2_grpc.FactorServiceStub(channel)

        np.random.seed(42)
        prices = (100 + np.cumsum(np.random.randn(50))).tolist()
        arrow_data = make_ohlcv_arrow(prices, "600519.SH")

        resp = stub.ComputeFactor(
            factor_pb2.ComputeFactorRequest(
                factor_name="rsi_14",
                symbols=["600519.SH"],
                ohlcv_data=arrow_data,
            )
        )
        assert resp.error == "", f"Factor error: {resp.error}"
        valid = [v for v in resp.results[0].values if not np.isnan(v)]
        assert all(0 <= v <= 100 for v in valid)

    def test_compute_factor_batch(self, channel):
        """ComputeFactorBatch should compute multiple factors at once."""
        stub = factor_pb2_grpc.FactorServiceStub(channel)

        prices = [100 + i for i in range(30)]
        arrow_data = make_ohlcv_arrow(prices, "000001.SZ")

        resp = stub.ComputeFactorBatch(
            factor_pb2.ComputeFactorBatchRequest(
                factor_names=["momentum_20d", "sma_5", "sma_20"],
                symbols=["000001.SZ"],
                ohlcv_data=arrow_data,
            )
        )
        assert resp.total_compute_time_ms > 0
        assert len(resp.factor_responses) == 3
        for fr in resp.factor_responses:
            assert fr.error == "", f"Factor {fr.factor_name} error: {fr.error}"

    def test_unknown_factor_returns_error(self, channel):
        """An unknown factor name should return an error string, not crash."""
        stub = factor_pb2_grpc.FactorServiceStub(channel)
        resp = stub.ComputeFactor(
            factor_pb2.ComputeFactorRequest(
                factor_name="nonexistent_factor_xyz",
                symbols=["000001.SZ"],
            )
        )
        assert resp.error != ""
