import os
import tempfile
import pytest
import numpy as np

try:
    import xgboost as xgb
    HAS_XGB = True
except ImportError:
    HAS_XGB = False


@pytest.mark.skipif(not HAS_XGB, reason="xgboost not installed")
class TestSerialization:
    def test_save_and_load_xgboost(self):
        from src.ml.serialization import save_model, load_model

        X = np.random.randn(100, 5)
        y = np.random.randn(100)
        model = xgb.XGBRegressor(n_estimators=10)
        model.fit(X, y)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = save_model(model, tmpdir)
            assert os.path.exists(path)
            assert path.endswith(".joblib")

            loaded = load_model(path)
            preds = loaded.predict(X[:5])
            assert len(preds) == 5

    def test_save_model_creates_dir_if_not_exists(self):
        from src.ml.serialization import save_model
        import xgboost as xgb

        model = xgb.XGBRegressor(n_estimators=5)
        model.fit(np.random.randn(50, 3), np.random.randn(50))

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, "subdir", "model.joblib")
            saved = save_model(model, path)
            assert os.path.exists(saved)
