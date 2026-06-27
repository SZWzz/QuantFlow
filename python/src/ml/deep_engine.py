"""DeepEngine: LSTM and Transformer time-series prediction (PyTorch, optional)."""
import os
import time
import logging
import numpy as np
import pyarrow as pa

logger = logging.getLogger(__name__)

_HAS_TORCH = False
_nn_Module = object  # fallback base class when PyTorch is not installed
try:
    import torch
    import torch.nn as nn
    from torch.utils.data import DataLoader, TensorDataset
    _HAS_TORCH = True
    _nn_Module = nn.Module
except ImportError:
    pass


class _LSTMPredictor(_nn_Module):
    def __init__(self, input_size, hidden_size, num_layers):
        if not _HAS_TORCH:
            raise ImportError("PyTorch not installed")
        super().__init__()
        self.lstm = nn.LSTM(input_size, hidden_size, num_layers, batch_first=True)
        self.fc = nn.Linear(hidden_size, 1)

    def forward(self, x):
        out, _ = self.lstm(x)
        return self.fc(out[:, -1, :]).squeeze(-1)


class _TransformerPredictor(_nn_Module):
    def __init__(self, input_size, d_model, nhead, num_layers):
        if not _HAS_TORCH:
            raise ImportError("PyTorch not installed")
        super().__init__()
        self.input_proj = nn.Linear(input_size, d_model)
        encoder_layer = nn.TransformerEncoderLayer(d_model=d_model, nhead=nhead, batch_first=True)
        self.transformer = nn.TransformerEncoder(encoder_layer, num_layers=num_layers)
        self.fc = nn.Linear(d_model, 1)

    def forward(self, x):
        x = self.input_proj(x)
        out = self.transformer(x)
        return self.fc(out[:, -1, :]).squeeze(-1)


class DeepEngine:
    """Trains deep learning models for time-series prediction (LSTM, Transformer)."""

    def _check_torch(self):
        if not _HAS_TORCH:
            raise ImportError(
                "torch is required for DeepEngine. Install with: pip install torch"
            )

    def train(self, features: pa.Table, targets: pa.Table, params: dict) -> dict:
        self._check_torch()
        start = time.time()
        model_type = params.get("model_type", "lstm")
        model_dir = params.get("model_dir", "/tmp/quantflow_models")

        # Reconstruct numpy arrays from byte-packed Arrow table
        X = np.array([np.frombuffer(row, dtype=np.float32).reshape(
            features.column("seq_len")[i].as_py(),
            features.column("n_features")[i].as_py()
        ) for i, row in enumerate(features.column("data").to_pylist())])
        y = targets.column("target").to_numpy().astype(np.float32)

        X_t = torch.tensor(X)
        y_t = torch.tensor(y)

        # Build model
        if model_type == "lstm":
            hidden_size = int(params.get("hidden_size", 64))
            num_layers = int(params.get("num_layers", 2))
            model = _LSTMPredictor(X.shape[2], hidden_size, num_layers)
        elif model_type == "transformer":
            d_model = int(params.get("d_model", 64))
            nhead = int(params.get("nhead", 4))
            num_layers = int(params.get("num_layers", 2))
            model = _TransformerPredictor(X.shape[2], d_model, nhead, num_layers)
        else:
            raise ValueError(f"unsupported deep model type: {model_type}")

        # Train
        epochs = int(params.get("epochs", 10))
        batch_size = int(params.get("batch_size", 32))
        dataset = TensorDataset(X_t, y_t)
        loader = DataLoader(dataset, batch_size=batch_size, shuffle=False)

        optimizer = torch.optim.Adam(model.parameters(), lr=float(params.get("learning_rate", 0.001)))
        loss_fn = nn.MSELoss()

        model.train()
        final_loss = 0.0
        for epoch in range(epochs):
            epoch_loss = 0.0
            for batch_X, batch_y in loader:
                optimizer.zero_grad()
                preds = model(batch_X)
                loss = loss_fn(preds, batch_y)
                loss.backward()
                optimizer.step()
                epoch_loss += loss.item()
            final_loss = epoch_loss / len(loader)
            logger.debug("epoch %d/%d loss=%.6f", epoch + 1, epochs, final_loss)

        # Save
        filepath = os.path.join(model_dir, f"{model_type}_{int(time.time())}.pt")
        os.makedirs(model_dir, exist_ok=True)
        torch.save(model.state_dict(), filepath)

        elapsed_ms = int((time.time() - start) * 1000)
        return {
            "model_path": filepath,
            "metrics": {"train_loss": final_loss},
            "train_time_ms": elapsed_ms,
        }

    def predict(self, model_path: str, features: pa.Table) -> pa.Array:
        self._check_torch()

        X = np.array([np.frombuffer(row, dtype=np.float32).reshape(
            features.column("seq_len")[i].as_py(),
            features.column("n_features")[i].as_py()
        ) for i, row in enumerate(features.column("data").to_pylist())])

        # Determine model type from file path
        if "lstm" in model_path.lower():
            # Infer architecture from saved state_dict keys
            state = torch.load(model_path, weights_only=True)
            # Reconstruct model
            hidden_dim = state["fc.weight"].shape[1]
            num_lstm_layers = sum(1 for k in state if k.startswith("lstm.weight_ih_l"))
            model = _LSTMPredictor(X.shape[2], hidden_dim, num_lstm_layers)
            model.load_state_dict(state)
        else:
            state = torch.load(model_path, weights_only=True)
            d_model = state["fc.weight"].shape[1]
            nhead = 4  # default, could be stored in metadata
            num_tf_layers = sum(1 for k in state if "transformer.layers" in k and k.endswith("self_attn.in_proj_weight"))
            if num_tf_layers == 0:
                num_tf_layers = 1
            model = _TransformerPredictor(X.shape[2], d_model, nhead, num_tf_layers)
            model.load_state_dict(state)

        model.eval()
        X_t = torch.tensor(X)
        with torch.no_grad():
            preds = model(X_t).numpy()

        return pa.array(preds.tolist())
