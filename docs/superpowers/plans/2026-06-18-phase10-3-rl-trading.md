# Phase 10.3: RL Trading — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Build reinforcement learning trading: Gym trading environment + PPO/DQN/SAC algorithms (Python) + 3 Go workflow nodes (RLEnv/RLTrain/RLPredict) + RLMonitor frontend panel.

**Architecture:** Python RLEngine provides Gym environment and RL algorithms. RLTrain uses server-streaming gRPC to push per-episode progress. Go nodes orchestrate the pipeline: RLEnv configures the environment → RLTrain trains with live reward streaming → RLPredict outputs actions. Frontend RLMonitor shows real-time reward curves via SSE.

**Tech Stack:** Python 3.12+ (torch, gymnasium), Go 1.22+, Vue 3 + ECharts, gRPC streaming.

**Depends on:** Phase 10.1 (DeepEngine shares torch, MLService, gRPC streaming infrastructure).

## Global Constraints
- torch is optional — RLEngine degrades gracefully when torch not installed
- RLTrain gRPC is server-streaming: episodes pushed as they complete
- RL algorithms: PPO (primary), DQN (discrete), SAC (continuous)
- Action space: Discrete(3) = {sell, hold, buy} or Continuous(1) = position weight
- Reward: Sharpe ratio increment per episode
- State: [position, cash_pct, last N OHLCV bars, technical indicators]
- All code follows existing patterns

---

### Task 1: Python RLEngine — env.py + algorithms

**Files:**
- Create: `python/src/ml/rl/__init__.py`
- Create: `python/src/ml/rl/env.py` — TradingGymEnv
- Create: `python/src/ml/rl/algorithms/__init__.py`
- Create: `python/src/ml/rl/algorithms/ppo.py`
- Create: `python/src/ml/rl/algorithms/dqn.py`
- Create: `python/src/ml/rl/algorithms/sac.py`
- Create: `python/src/ml/rl/replay.py` — ReplayBuffer
- Create: `python/tests/test_rl_engine.py`

- [ ] **Step 1: Write test**

Write `python/tests/test_rl_engine.py`:

```python
import numpy as np
import pytest

try:
    import torch
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False


@pytest.fixture
def sample_ohlcv():
    np.random.seed(42)
    n = 200
    return np.column_stack([
        np.cumsum(np.random.randn(n) * 0.01) + 100,  # close
        np.cumsum(np.random.randn(n) * 0.01) + 99,   # open
        np.cumsum(np.random.randn(n) * 0.01) + 101,  # high
        np.cumsum(np.random.randn(n) * 0.01) + 98,   # low
        np.random.rand(n) * 1000,                     # volume
    ])


@pytest.mark.skipif(not HAS_TORCH, reason="torch not installed")
class TestRLEnv:
    def test_env_reset_and_step(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        
        env = TradingEnv(sample_ohlcv, window_size=10)
        state, _ = env.reset()
        assert state.shape == (10 * 5 + 2,)  # window * features + position + cash
        
        action = 1  # buy
        next_state, reward, done, truncated, info = env.step(action)
        assert next_state.shape == state.shape
        assert isinstance(reward, float)
        assert not done or env.current_step >= len(sample_ohlcv) - 1

    def test_action_space(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        
        env = TradingEnv(sample_ohlcv, window_size=10, action_type="discrete")
        assert env.action_space.n == 3  # sell, hold, buy
        
        env2 = TradingEnv(sample_ohlcv, window_size=10, action_type="continuous")
        assert env2.action_space.shape == (1,)


@pytest.mark.skipif(not HAS_TORCH, reason="torch not installed")
class TestPPO:
    def test_ppo_train_one_episode(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.ppo import PPOTrainer
        
        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = PPOTrainer(state_dim=env.observation_space.shape[0], action_dim=3)
        
        trainer.train_episode(env)
        assert trainer.total_episodes == 1


@pytest.mark.skipif(not HAS_TORCH, reason="torch not installed")
class TestDQN:
    def test_dqn_train_one_episode(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.dqn import DQNTrainer
        
        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = DQNTrainer(state_dim=env.observation_space.shape[0], action_dim=3)
        
        trainer.train_episode(env)
        assert trainer.total_episodes == 1
```

- [ ] **Step 2: Implement env.py**

```python
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
    """OHLCV trading environment with position, cash, and reward."""

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
        # Map action to position delta
        if self.action_type == "discrete":
            action_val = action - 1  # 0->sell(-1), 1->hold(0), 2->buy(1)
        else:
            action_val = float(action)

        prev_position = self.position
        self.position = np.clip(action_val, -1.0, 1.0)

        price = self.ohlcv[self.current_step, 0]
        prev_price = self.ohlcv[self.current_step - 1, 0]
        price_return = (price - prev_price) / prev_price if prev_price > 0 else 0.0

        # Update portfolio
        trade_cost = abs(self.position - prev_position) * self.portfolio_value * 0.001
        self.cash -= trade_cost
        self.portfolio_value = self.cash * (1 + self.position * price_return)
        self.cash = self.portfolio_value * (1 - abs(self.position))

        reward = (self.portfolio_value - self.prev_value) / self.prev_value
        self.prev_value = self.portfolio_value
        self.current_step += 1

        done = self.current_step >= len(self.ohlcv) - 1
        truncated = False

        return self._get_state(), reward, done, truncated, {"portfolio_value": self.portfolio_value}

    def _get_state(self):
        start = self.current_step - self.window_size
        window = self.ohlcv[start:self.current_step]
        flat_window = window.flatten() / (np.abs(window).max(axis=0) + 1e-8)
        return np.concatenate([flat_window, [self.position, self.cash / self.initial_cash]]).astype(np.float32)
```

- [ ] **Step 3: Implement PPO**

Write `python/src/ml/rl/algorithms/ppo.py`:

```python
"""PPO (Proximal Policy Optimization) for trading."""
import numpy as np
import logging

logger = logging.getLogger(__name__)

try:
    import torch
    import torch.nn as nn
    import torch.optim as optim
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False


class PPONetwork(nn.Module):
    def __init__(self, state_dim, action_dim, hidden_size=128):
        super().__init__()
        self.shared = nn.Sequential(
            nn.Linear(state_dim, hidden_size), nn.ReLU(),
            nn.Linear(hidden_size, hidden_size), nn.ReLU(),
        )
        self.actor = nn.Linear(hidden_size, action_dim)
        self.critic = nn.Linear(hidden_size, 1)

    def forward(self, x):
        shared = self.shared(x)
        return self.actor(shared), self.critic(shared)


class PPOTrainer:
    def __init__(self, state_dim, action_dim, lr=3e-4, gamma=0.99, clip_epsilon=0.2):
        if not HAS_TORCH:
            raise ImportError("torch is required. Install with: pip install torch")
        self.network = PPONetwork(state_dim, action_dim)
        self.optimizer = optim.Adam(self.network.parameters(), lr=lr)
        self.gamma = gamma
        self.clip_epsilon = clip_epsilon
        self.total_episodes = 0
        self.action_dim = action_dim

    def train_episode(self, env) -> dict:
        state, _ = env.reset()
        states, actions, rewards, log_probs, values = [], [], [], [], []
        done = False

        while not done:
            state_t = torch.FloatTensor(state).unsqueeze(0)
            with torch.no_grad():
                logits, value = self.network(state_t)
                probs = torch.softmax(logits, dim=-1)
                dist = torch.distributions.Categorical(probs)
                action = dist.sample()
                log_prob = dist.log_prob(action)

            next_state, reward, done, truncated, info = env.step(action.item())
            states.append(state)
            actions.append(action.item())
            rewards.append(reward)
            log_probs.append(log_prob.item())
            values.append(value.item())
            state = next_state

        # Compute returns and advantages
        returns = []
        G = 0
        for r in reversed(rewards):
            G = r + self.gamma * G
            returns.insert(0, G)
        returns = torch.FloatTensor(returns)
        values = torch.FloatTensor(values)
        advantages = returns - values
        advantages = (advantages - advantages.mean()) / (advantages.std() + 1e-8)

        # PPO update
        states_t = torch.FloatTensor(np.array(states))
        actions_t = torch.LongTensor(actions)
        old_log_probs = torch.FloatTensor(log_probs)

        for _ in range(4):  # PPO epochs
            logits, new_values = self.network(states_t)
            probs = torch.softmax(logits, dim=-1)
            dist = torch.distributions.Categorical(probs)
            new_log_probs = dist.log_prob(actions_t)

            ratio = torch.exp(new_log_probs - old_log_probs)
            surr1 = ratio * advantages
            surr2 = torch.clamp(ratio, 1 - self.clip_epsilon, 1 + self.clip_epsilon) * advantages
            actor_loss = -torch.min(surr1, surr2).mean()
            critic_loss = nn.MSELoss()(new_values.squeeze(), returns)
            loss = actor_loss + 0.5 * critic_loss

            self.optimizer.zero_grad()
            loss.backward()
            self.optimizer.step()

        self.total_episodes += 1
        total_reward = sum(rewards)
        sharpe = (np.mean(rewards) / (np.std(rewards) + 1e-8)) * np.sqrt(252) if len(rewards) > 1 else 0

        return {"episode": self.total_episodes, "reward": total_reward, "sharpe": sharpe, "steps": len(rewards)}
```

- [ ] **Step 4: Implement DQN + SAC (simplified)**

Write `python/src/ml/rl/algorithms/dqn.py` (simplified DQN) and `sac.py` (simplified SAC) following the same pattern as PPO but with DQN/SAC-specific logic. DQN uses epsilon-greedy + replay buffer + target network. SAC uses actor-critic with entropy bonus.

- [ ] **Step 5: Run tests, commit**

---

### Task 2: Wire RLEngine into MLService (RLTrain streaming)

**Files:**
- Modify: `python/src/ml/engine.py` — replace RLTrain/RLPredict stubs

- [ ] **Step 1: Implement streaming RLTrain**

Replace RLTrain stub:
```python
from typing import AsyncIterator

async def RLTrain(self, request, context) -> AsyncIterator[ml_pb2.RLTrainUpdate]:
    try:
        ohlcv = self._decode_arrow(request.ohlcv_data)
        ohlcv_np = ohlcv.to_pandas().values.astype(np.float32)
        
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.ppo import PPOTrainer
        
        env = TradingEnv(ohlcv_np, window_size=int(request.hyperparams.get("window_size", "20")))
        
        if request.algorithm == "ppo":
            trainer = PPOTrainer(env.observation_space.shape[0], env.action_space.n)
        elif request.algorithm == "dqn":
            from src.ml.rl.algorithms.dqn import DQNTrainer
            trainer = DQNTrainer(env.observation_space.shape[0], env.action_space.n)
        else:
            from src.ml.rl.algorithms.sac import SACTrainer
            trainer = SACTrainer(env.observation_space.shape[0], 1 if request.action_space == "continuous" else 3)
        
        total = request.total_episodes or 100
        for ep in range(total):
            result = trainer.train_episode(env)
            yield ml_pb2.RLTrainUpdate(
                episode=result["episode"], reward=result["reward"],
                sharpe=result["sharpe"], steps=result["steps"],
            )
    except ImportError as e:
        logger.warning("RLTrain: %s", e)
        return
    except Exception as e:
        logger.exception("RLTrain failed")
        return
```

- [ ] **Step 2: Add test, run, commit**

---

### Task 3: Go RL nodes (RLEnv + RLTrain + RLPredict)

**Files:**
- Create: `internal/workflow/nodes/rl_env.go`
- Create: `internal/workflow/nodes/rl_train.go`
- Create: `internal/workflow/nodes/rl_predict.go`
- Modify: `internal/workflow/nodes/register.go`

Implement 3 nodes following the same pattern as Phase 10.1 nodes. RLTrainNode consumes the streaming gRPC and outputs a reward_curve. Register all 3 as category "ml".

---

### Task 4: Frontend RLMonitor panel

**Files:**
- Create: `frontend/src/terminal/panels/RLMonitorPanel.vue`
- Modify: `frontend/src/terminal/panels/registry.ts`
- Modify: `frontend/src/stores/ml.ts` (add rlTrainingCurves state)

Panel shows: reward curve (real-time line), drawdown area, action distribution pie, position histogram. Controls: start/pause/save.

---

### Task 5: CHANGELOG

Add Phase 10.3 entries.
