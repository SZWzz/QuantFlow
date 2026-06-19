"""SAC (Soft Actor-Critic) stub — full implementation coming in Phase 10.3.1."""

try:
    import torch
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False


class SACTrainer:
    """SAC trainer stub. Full implementation planned for Phase 10.3.1."""

    def __init__(self, state_dim, action_dim, **kwargs):
        if not HAS_TORCH:
            raise ImportError("torch is required. Install with: pip install torch")
        self.total_episodes = 0

    def train_episode(self, env) -> dict:
        raise NotImplementedError("SAC implementation coming in Phase 10.3.1")
