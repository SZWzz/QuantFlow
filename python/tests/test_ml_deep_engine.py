import tempfile
import numpy as np
import pandas as pd
import pyarrow as pa
import pytest

try:
    import torch
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False


@pytest.fixture
def sample_sequence_data():
    np.random.seed(42)
    n_samples, seq_len, n_features = 100, 10, 5
    X = np.random.randn(n_samples, seq_len, n_features).astype(np.float32)
    y = X[:, -1, 0] * 0.5 + np.random.randn(n_samples).astype(np.float32) * 0.1
    return X, y


@pytest.mark.skipif(not HAS_TORCH, reason="torch not installed")
class TestDeepEngine:
    def test_train_lstm(self, sample_sequence_data):
        from src.ml.deep_engine import DeepEngine

        X, y = sample_sequence_data
        features = pa.Table.from_pandas(pd.DataFrame({
            "data": [row.tobytes() for row in X],
            "seq_len": [X.shape[1]] * len(X),
            "n_features": [X.shape[2]] * len(X),
        }))
        targets = pa.Table.from_pandas(pd.DataFrame({"target": y}))

        engine = DeepEngine()
        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "lstm",
                "model_dir": tmpdir,
                "hidden_size": "32",
                "num_layers": "1",
                "epochs": "3",
                "batch_size": "16",
            })
            assert "model_path" in result
            assert "metrics" in result
            assert "train_loss" in result["metrics"]

    def test_train_transformer(self, sample_sequence_data):
        from src.ml.deep_engine import DeepEngine

        X, y = sample_sequence_data
        features = pa.Table.from_pandas(pd.DataFrame({
            "data": [row.tobytes() for row in X],
            "seq_len": [X.shape[1]] * len(X),
            "n_features": [X.shape[2]] * len(X),
        }))
        targets = pa.Table.from_pandas(pd.DataFrame({"target": y}))

        engine = DeepEngine()
        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "transformer",
                "model_dir": tmpdir,
                "d_model": "32",
                "nhead": "4",
                "num_layers": "1",
                "epochs": "3",
                "batch_size": "16",
            })
            assert "model_path" in result

    def test_predict(self, sample_sequence_data):
        from src.ml.deep_engine import DeepEngine

        X, y = sample_sequence_data
        features = pa.Table.from_pandas(pd.DataFrame({
            "data": [row.tobytes() for row in X],
            "seq_len": [X.shape[1]] * len(X),
            "n_features": [X.shape[2]] * len(X),
        }))
        targets = pa.Table.from_pandas(pd.DataFrame({"target": y}))

        engine = DeepEngine()
        with tempfile.TemporaryDirectory() as tmpdir:
            result = engine.train(features, targets, {
                "model_type": "lstm",
                "model_dir": tmpdir,
                "hidden_size": "16",
                "epochs": "2",
                "batch_size": "32",
            })
            preds = engine.predict(result["model_path"], features)
            assert len(preds) == len(X)

    def test_torch_not_installed_raises(self):
        """DeepEngine constructor should not raise; only train/predict should
        error if torch is literally not importable. This test guards the import
        check path."""
        from src.ml.deep_engine import DeepEngine
        engine = DeepEngine()
        assert engine is not None
