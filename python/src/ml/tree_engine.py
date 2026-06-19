"""TreeEngine: XGBoost and LightGBM model training, prediction, and evaluation."""
import os
import time
import logging
import numpy as np
import pyarrow as pa

from src.ml.serialization import save_model, load_model

logger = logging.getLogger(__name__)


class TreeEngine:
    """Trains and evaluates tree-based models (XGBoost, LightGBM)."""

    def train(self, features: pa.Table, targets: pa.Table, params: dict) -> dict:
        """Train a tree model.

        Args:
            features: Arrow Table of feature matrix.
            targets: Arrow Table with 'target' column.
            params: dict with keys: model_type, model_dir, n_estimators, max_depth,
                    learning_rate, target_type, and any model-specific hyperparams.

        Returns:
            dict with keys: model_path, metrics (train_rmse, train_mae, ...), train_time_ms.
        """
        start = time.time()
        model_type = params.get("model_type", "xgboost")
        model_dir = params.get("model_dir", "/tmp/quantflow_models")
        target_type = params.get("target_type", "regression")

        X = features.to_pandas().values.astype(np.float64)
        y = targets.column("target").to_numpy().astype(np.float64 if target_type == "regression" else np.int64)

        if model_type == "xgboost":
            model, metrics = self._train_xgboost(X, y, params, target_type)
            ext = ".joblib"
        elif model_type == "lightgbm":
            model, metrics = self._train_lightgbm(X, y, params, target_type)
            ext = ".joblib"
        else:
            raise ValueError(f"unsupported tree model type: {model_type}")

        filepath = os.path.join(model_dir, f"{model_type}_{int(time.time())}{ext}")
        model_path = save_model(model, filepath)

        elapsed_ms = int((time.time() - start) * 1000)
        logger.info("TreeEngine trained %s in %dms, metrics=%s", model_type, elapsed_ms, metrics)

        return {
            "model_path": model_path,
            "metrics": metrics,
            "train_time_ms": elapsed_ms,
        }

    def predict(self, model_path: str, features: pa.Table) -> pa.Array:
        """Generate predictions from a saved model."""
        model = load_model(model_path)
        X = features.to_pandas().values.astype(np.float64)
        preds = model.predict(X)
        return pa.array(preds.tolist())

    def evaluate(self, model_path: str, features: pa.Table, targets: pa.Table) -> dict:
        """Compute evaluation metrics."""
        model = load_model(model_path)
        X = features.to_pandas().values.astype(np.float64)
        y_true = targets.column("target").to_numpy()

        y_pred = model.predict(X)

        mse = float(np.mean((y_true - y_pred) ** 2))
        mae = float(np.mean(np.abs(y_true - y_pred)))
        rmse = float(np.sqrt(mse))

        metrics = {"mse": mse, "mae": mae, "rmse": rmse}

        # For classification, compute accuracy
        if hasattr(model, "predict_proba"):
            y_pred_class = np.argmax(model.predict_proba(X), axis=1) if model.n_classes_ > 2 else (model.predict_proba(X)[:, 1] > 0.5).astype(int)
            accuracy = float(np.mean(y_pred_class == y_true))
            metrics["accuracy"] = accuracy

        return metrics

    def feature_importance(self, model_path: str, features: pa.Table) -> list[dict]:
        """Return feature importance rankings."""
        model = load_model(model_path)
        if not hasattr(model, "feature_importances_"):
            return []

        names = features.column_names
        importances = model.feature_importances_
        ranked = sorted(zip(names, importances), key=lambda x: x[1], reverse=True)
        return [{"feature": name, "importance": float(imp)} for name, imp in ranked]

    def _train_xgboost(self, X: np.ndarray, y: np.ndarray, params: dict, target_type: str):
        import xgboost as xgb

        n_estimators = int(params.get("n_estimators", 100))
        max_depth = int(params.get("max_depth", 6))
        learning_rate = float(params.get("learning_rate", 0.1))

        if target_type == "classification":
            model = xgb.XGBClassifier(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, eval_metric="logloss"
            )
            model.fit(X, y)
            y_pred = model.predict(X)
            accuracy = float(np.mean(y_pred == y))
            metrics = {"train_accuracy": accuracy}
        else:
            model = xgb.XGBRegressor(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42
            )
            model.fit(X, y)
            y_pred = model.predict(X)
            rmse = float(np.sqrt(np.mean((y - y_pred) ** 2)))
            mae = float(np.mean(np.abs(y - y_pred)))
            metrics = {"train_rmse": rmse, "train_mae": mae}

        return model, metrics

    def _train_lightgbm(self, X: np.ndarray, y: np.ndarray, params: dict, target_type: str):
        import lightgbm as lgb

        n_estimators = int(params.get("n_estimators", 100))
        max_depth = int(params.get("max_depth", -1))
        learning_rate = float(params.get("learning_rate", 0.1))

        if target_type == "classification":
            model = lgb.LGBMClassifier(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, verbose=-1
            )
            model.fit(X, y)
            y_pred = model.predict(X)
            accuracy = float(np.mean(y_pred == y))
            metrics = {"train_accuracy": accuracy}
        else:
            model = lgb.LGBMRegressor(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, verbose=-1
            )
            model.fit(X, y)
            y_pred = model.predict(X)
            rmse = float(np.sqrt(np.mean((y - y_pred) ** 2)))
            mae = float(np.mean(np.abs(y - y_pred)))
            metrics = {"train_rmse": rmse, "train_mae": mae}

        return model, metrics
