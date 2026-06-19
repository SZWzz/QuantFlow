import tempfile
import numpy as np
import pandas as pd
import pyarrow as pa
import pytest

try:
    import xgboost as xgb
    HAS_XGB = True
except ImportError:
    HAS_XGB = False

try:
    import lightgbm as lgb
    HAS_LGB = True
except ImportError:
    HAS_LGB = False


@pytest.fixture
def sample_data():
    np.random.seed(42)
    X = np.random.randn(200, 5)
    y = X[:, 0] * 0.5 + X[:, 1] * (-0.3) + np.random.randn(200) * 0.1
    feature_table = pa.Table.from_pandas(pd.DataFrame(X, columns=[f"f_{i}" for i in range(5)]))
    target_table = pa.Table.from_pandas(pd.DataFrame({"target": y}))
    return feature_table, target_table


@pytest.mark.skipif(not HAS_XGB, reason="xgboost not installed")
class TestTreeEngine:
    def test_train_xgboost_regression(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, targets = sample_data

        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "xgboost",
                "model_dir": tmpdir,
                "n_estimators": "50",
                "max_depth": "3",
                "target_type": "regression",
            })

            assert "model_path" in result
            assert "metrics" in result
            assert "train_rmse" in result["metrics"]
            assert result["metrics"]["train_rmse"] > 0

    def test_train_xgboost_classification(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, _ = sample_data
        y = np.random.choice([0, 1], size=200)
        targets = pa.Table.from_pandas(pd.DataFrame({"target": y}))

        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "xgboost",
                "model_dir": tmpdir,
                "n_estimators": "30",
                "target_type": "classification",
            })

            assert "metrics" in result
            assert "train_accuracy" in result["metrics"]

    def test_predict(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, targets = sample_data

        with tempfile.TemporaryDirectory() as tmpdir:
            train_result = engine.train(features, targets, {
                "model_type": "xgboost",
                "model_dir": tmpdir,
                "n_estimators": "30",
            })
            preds = engine.predict(train_result["model_path"], features)
            assert len(preds) == 200

    def test_evaluate(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, targets = sample_data

        with tempfile.TemporaryDirectory() as tmpdir:
            train_result = engine.train(features, targets, {
                "model_type": "xgboost",
                "model_dir": tmpdir,
                "n_estimators": "30",
            })
            evaluation = engine.evaluate(train_result["model_path"], features, targets)
            assert "mse" in evaluation
            assert "mae" in evaluation

    def test_feature_importance(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, targets = sample_data

        with tempfile.TemporaryDirectory() as tmpdir:
            train_result = engine.train(features, targets, {
                "model_type": "xgboost",
                "model_dir": tmpdir,
                "n_estimators": "30",
            })
            fi = engine.feature_importance(train_result["model_path"], features)
            assert len(fi) == 5
            assert all("feature" in f and "importance" in f for f in fi)


@pytest.mark.skipif(not HAS_LGB, reason="lightgbm not installed")
class TestLightGBMEngine:
    def test_train_lightgbm(self, sample_data):
        from src.ml.tree_engine import TreeEngine

        engine = TreeEngine()
        features, targets = sample_data

        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "lightgbm",
                "model_dir": tmpdir,
                "n_estimators": "30",
                "target_type": "regression",
            })

            assert "train_rmse" in result["metrics"]
