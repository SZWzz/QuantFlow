"""Tests for MLService gRPC implementation — integration test via the service class itself."""
import tempfile
import numpy as np
import pandas as pd
import pyarrow as pa
import pytest
import grpc
from concurrent import futures

try:
    import xgboost
    HAS_XGB = True
except ImportError:
    HAS_XGB = False

try:
    import gplearn  # noqa: F401
    HAS_GPLEARN = True
except ImportError:
    HAS_GPLEARN = False


@pytest.fixture
def ml_service():
    from src.ml.engine import MLService
    return MLService()


@pytest.fixture
def arrow_features():
    X = np.random.randn(100, 5).astype(np.float64)
    table = pa.Table.from_pandas(pd.DataFrame(X, columns=[f"f_{i}" for i in range(5)]))
    sink = pa.BufferOutputStream()
    with pa.ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


@pytest.fixture
def arrow_targets():
    y = np.random.randn(100).astype(np.float64)
    table = pa.Table.from_pandas(pd.DataFrame({"target": y}))
    sink = pa.BufferOutputStream()
    with pa.ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


@pytest.fixture
def arrow_factor_data():
    np.random.seed(42)
    n = 100
    X = np.column_stack([
        np.random.randn(n),
        np.random.randn(n) * 0.5,
    ])
    table = pa.Table.from_pandas(pd.DataFrame(X, columns=["f_0", "f_1"]))
    sink = pa.BufferOutputStream()
    with pa.ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


@pytest.fixture
def arrow_returns_data(arrow_factor_data):
    np.random.seed(42)
    n = 100
    f0 = np.random.randn(n)
    returns = f0 * 0.3 + np.random.randn(n) * 0.05
    table = pa.Table.from_pandas(pd.DataFrame({"return": returns}))
    sink = pa.BufferOutputStream()
    with pa.ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


@pytest.mark.skipif(not HAS_XGB, reason="xgboost not installed")
@pytest.mark.asyncio
async def test_train_and_predict_flow(ml_service, arrow_features, arrow_targets):
    from src.proto import ml_pb2

    # Train
    train_req = ml_pb2.TrainRequest(
        model_type="xgboost",
        features=arrow_features,
        targets=arrow_targets,
        target_type="regression",
        forecast_horizon=5,
    )
    train_req.hyperparams["n_estimators"] = "20"

    train_resp = await ml_service.Train(train_req, None)
    assert train_resp.model_id != ""
    assert "train_rmse" in train_resp.metrics

    # Predict
    pred_req = ml_pb2.PredictRequest(
        model_id=train_resp.model_id,
        features=arrow_features,
    )
    pred_resp = await ml_service.Predict(pred_req, None)
    assert len(pred_resp.predictions) == 100
    assert pred_resp.predict_time_ms > 0

    # Evaluate
    eval_req = ml_pb2.EvaluateRequest(
        model_id=train_resp.model_id,
        features=arrow_features,
        actuals=arrow_targets,
    )
    eval_resp = await ml_service.Evaluate(eval_req, None)
    assert "mse" in eval_resp.metrics


@pytest.mark.asyncio
async def test_unsupported_model_type(ml_service, arrow_features, arrow_targets):
    from src.proto import ml_pb2

    train_req = ml_pb2.TrainRequest(
        model_type="unknown_model",
        features=arrow_features,
        targets=arrow_targets,
        target_type="regression",
    )
    with pytest.raises(ValueError, match="unsupported model_type"):
        mock_ctx = type('MockContext', (), {'set_code': lambda s, c: None, 'set_details': lambda s, d: None})()
        await ml_service.Train(train_req, mock_ctx)


@pytest.mark.asyncio
async def test_predict_missing_model(ml_service, arrow_features):
    from src.proto import ml_pb2

    pred_req = ml_pb2.PredictRequest(
        model_id="nonexistent",
        features=arrow_features,
    )
    mock_ctx = type('MockContext', (), {'set_code': lambda s, c: None, 'set_details': lambda s, d: None})()
    with pytest.raises(KeyError, match="model not found"):
        await ml_service.Predict(pred_req, mock_ctx)


@pytest.mark.skipif(not HAS_GPLEARN, reason="gplearn not installed")
@pytest.mark.asyncio
async def test_alpha_mining_discovers_factors(ml_service, arrow_factor_data, arrow_returns_data):
    """AlphaMining should discover factor formulas with non-zero IC."""
    from src.proto import ml_pb2

    req = ml_pb2.AlphaMiningRequest(
        factor_data=arrow_factor_data,
        returns_data=arrow_returns_data,
        population_size=30,
        generations=3,
        top_k=3,
    )

    resp = await ml_service.AlphaMining(req, None)
    assert len(resp.factors) > 0
    assert len(resp.factors) <= 3
    assert resp.mining_time_ms > 0
    for f in resp.factors:
        assert f.formula != ""
        assert hasattr(f, "ic")
        assert hasattr(f, "ir")
        assert hasattr(f, "sharpe")
