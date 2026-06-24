"""Tests for TreeEngine train/val split (P0-4).

Regression: previously _train_xgboost/_train_lightgbm computed metrics
on the same X used for training (train_rmse/train_accuracy), with no
validation set — causing overfitting and data leakage for time-series.
"""
import numpy as np
import pyarrow as pa
import pytest

from src.ml.tree_engine import TreeEngine


def _make_features(n=300, seed=42):
    rng = np.random.default_rng(seed)
    X = rng.normal(0, 1, size=(n, 5)).astype(np.float64)
    y = (X[:, 0] * 0.5 + X[:, 1] * 0.3 + rng.normal(0, 0.1, n)).astype(np.float64)
    return pa.Table.from_arrays([pa.array(X[:, i]) for i in range(5)], names=[f"f{i}" for i in range(5)]), \
           pa.Table.from_arrays([pa.array(y)], names=["target"])


def test_train_returns_validation_metrics():
    """train() must return val_rmse/val_mae (not just train_*)."""
    features, targets = _make_features(300)
    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "regression",
    })
    metrics = result["metrics"]
    assert "val_rmse" in metrics, f"missing val_rmse in metrics: {metrics}"
    assert "val_mae" in metrics, f"missing val_mae in metrics: {metrics}"


def test_train_validation_metrics_differ_from_train():
    """val metrics should differ from train metrics (proves a real split)."""
    features, targets = _make_features(300)
    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "regression",
    })
    metrics = result["metrics"]
    # val_rmse should generally be >= train_rmse (overfitting), and must differ
    assert "train_rmse" in metrics and "val_rmse" in metrics
    assert metrics["train_rmse"] != metrics["val_rmse"], (
        f"train_rmse == val_rmse, split likely not applied: {metrics}"
    )


def test_train_classification_returns_val_accuracy():
    """Classification target must return val_accuracy."""
    rng = np.random.default_rng(0)
    X = rng.normal(0, 1, size=(200, 4))
    y = (X[:, 0] + rng.normal(0, 0.5, 200) > 0).astype(np.int64)
    features = pa.Table.from_arrays([pa.array(X[:, i]) for i in range(4)], names=[f"f{i}" for i in range(4)])
    targets = pa.Table.from_arrays([pa.array(y)], names=["target"])

    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "classification",
    })
    assert "val_accuracy" in result["metrics"], f"missing val_accuracy: {result['metrics']}"
