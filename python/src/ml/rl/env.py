"""Trading environment for reinforcement learning (Gymnasium)."""
import numpy as np

try:
    import gymnasium as gym
    from gymnasium import spaces
    HAS_GYM = True
except ImportError:
    HAS_GYM = False
    gym = None
    spaces = None


class TradingEnv(gym.Env if HAS_GYM else object):
    """OHLCV trading environment with position, cash, and reward.

    Action space: Discrete(3) = {sell(-1), hold(0), buy(1)} or Continuous(-1, 1).
    Observation space: [flat_window * n_features + position + cash_pct].
    Reward: portfolio value change ratio.
    """

    def __init__(self, ohlcv: np.ndarray, window_size: int = 20,
                 action_type: str = "discrete", initial_cash: float = 10000):
        if not HAS_GYM:
            raise ImportError("gymnasium is required. Install with: pip install gymnasium")
        super().__init__()
        self.ohlcv = ohlcv.astype(np.float32)
        self.window_size = window_size
        self.initial_cash = initial_cash
        self.n_features = ohlcv.shape[1]

        state_dim = window_size * self.n_features + 2  # + position + cash_pct
        if action_type == "discrete":
            self.action_space = spaces.Discrete(3)  # -1, 0, 1
        else:
            self.action_space = spaces.Box(-1, 1, (1,), dtype=np.float32)
        self.observation_space = spaces.Box(-np.inf, np.inf, (state_dim,), dtype=np.float32)

        self.action_type = action_type
        self.reset()

    def reset(self, seed=None, options=None):
        super().reset(seed=seed)
        self.current_step = self.window_size
        self.position = 0.0
        self.cash = self.initial_cash
        self.portfolio_value = self.initial_cash
        self.prev_value = self.initial_cash
        return self._get_state(), {}

    def step(self, action):
        if self.action_type == "discrete":
            action_val = action - 1  # 0->sell(-1), 1->hold(0), 2->buy(1)
        else:
            action_val = float(action)

        prev_position = self.position
        self.position = np.clip(action_val, -1.0, 1.0)

        price = self.ohlcv[self.current_step, 0]
        prev_price = self.ohlcv[self.current_step - 1, 0] if self.current_step > 0 else price
        price_return = (price - prev_price) / prev_price if prev_price > 0 else 0.0

        # Correct portfolio update:
        # 1. Trade modifies cash based on position change
        # 2. Portfolio value = cash + position_value * (1 + price_return)
        portfolio_value_before = self.portfolio_value

        if prev_position != self.position:
            delta = self.position - prev_position
            trade_value = delta * portfolio_value_before
            trade_cost = abs(trade_value) * 0.001
            self.cash -= trade_value + trade_cost

        self.cash = max(self.cash, 0)

        # Compute new portfolio value after price change
        position_value = self.position * portfolio_value_before * (1 + price_return)
        self.portfolio_value = self.cash + position_value
        self.portfolio_value = max(self.portfolio_value, 0.0)

        reward = (self.portfolio_value - self.prev_value) / self.prev_value if self.prev_value > 0 else 0.0
        self.prev_value = self.portfolio_value
        self.current_step += 1

        done = self.current_step >= len(self.ohlcv) - 1
        truncated = False

        return self._get_state(), reward, done, truncated, {"portfolio_value": self.portfolio_value}

    def _get_state(self):
        start = self.current_step - self.window_size
        window = self.ohlcv[start:self.current_step]
        max_vals = np.abs(window).max(axis=0) + 1e-8
        flat_window = (window / max_vals).flatten()
        return np.concatenate([flat_window, [self.position, self.cash / self.initial_cash]]).astype(np.float32)
