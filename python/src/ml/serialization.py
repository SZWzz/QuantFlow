"""Model serialization: joblib for sklearn/xgboost/lightgbm, torch.save for PyTorch."""
import os
import logging
import joblib

logger = logging.getLogger(__name__)

# Default directory for model files
MODEL_DIR = os.environ.get("QUANTFLOW_MODEL_DIR", os.path.expanduser("~/.quantflow/models"))


def save_model(model, filepath: str) -> str:
    """Serialize a model to disk using joblib. Returns the absolute file path."""
    os.makedirs(os.path.dirname(filepath) or ".", exist_ok=True)

    if filepath.endswith(".pt"):
        _save_torch(model, filepath)
    else:
        if not filepath.endswith(".joblib"):
            filepath = filepath + ".joblib"
        joblib.dump(model, filepath)

    logger.info("model saved: %s (%d bytes)", filepath, os.path.getsize(filepath))
    return os.path.abspath(filepath)


def load_model(filepath: str):
    """Load a serialized model from disk."""
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"model file not found: {filepath}")

    if filepath.endswith(".pt"):
        return _load_torch(filepath)
    return joblib.load(filepath)


def _save_torch(model, filepath: str):
    try:
        import torch
        torch.save(model.state_dict(), filepath)
    except ImportError:
        raise ImportError("torch is required for PyTorch model serialization. Install with: pip install torch")


def _load_torch(filepath: str):
    try:
        import torch
    except ImportError:
        raise ImportError("torch is required for PyTorch model loading. Install with: pip install torch")
    return torch.load(filepath, weights_only=True)
