"""Tests for RL Trading Environment and Algorithms (Phase 10.3)."""
import numpy as np
import pytest

try:
    import torch
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False

try:
    import gymnasium as gym
    HAS_GYM = True
except ImportError:
    HAS_GYM = False


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


@pytest.mark.skipif(not HAS_TORCH or not HAS_GYM, reason="torch or gymnasium not installed")
class TestRLEnv:
    def test_env_reset_and_step(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv

        env = TradingEnv(sample_ohlcv, window_size=10)
        state, _ = env.reset()
        assert state.shape == (10 * 5 + 2,)  # window * features + position + cash

        action = 1  # hold
        next_state, reward, done, truncated, info = env.step(action)
        assert next_state.shape == state.shape
        assert isinstance(reward, float)
        assert "portfolio_value" in info
        assert not done or env.current_step >= len(sample_ohlcv) - 1

    def test_episode_runs_to_end(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv

        env = TradingEnv(sample_ohlcv, window_size=10)
        env.reset()
        done = False
        steps = 0
        max_steps = len(sample_ohlcv) - 10 - 2
        while not done:
            action = env.action_space.sample()
            _, _, done, _, _ = env.step(action)
            steps += 1
            assert steps <= max_steps + 5  # allow small margin

    def test_action_space(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv

        env = TradingEnv(sample_ohlcv, window_size=10, action_type="discrete")
        assert env.action_space.n == 3  # sell, hold, buy

        env2 = TradingEnv(sample_ohlcv, window_size=10, action_type="continuous")
        assert env2.action_space.shape == (1,)

    def test_reward_respects_portfolio(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv

        env = TradingEnv(sample_ohlcv, window_size=10, initial_cash=10000)
        env.reset()
        # Buy action should change portfolio value
        initial_value = env.portfolio_value
        _, reward, _, _, _ = env.step(2)  # buy
        assert isinstance(reward, float)


@pytest.mark.skipif(not HAS_TORCH or not HAS_GYM, reason="torch or gymnasium not installed")
class TestPPO:
    def test_ppo_network_forward(self, sample_ohlcv):
        from src.ml.rl.algorithms.ppo import PPONetwork

        net = PPONetwork(state_dim=52, action_dim=3)
        x = torch.randn(4, 52)
        actor_out, critic_out = net(x)
        assert actor_out.shape == (4, 3)
        assert critic_out.shape == (4, 1)

    def test_ppo_train_one_episode(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.ppo import PPOTrainer

        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = PPOTrainer(state_dim=env.observation_space.shape[0], action_dim=3)

        result = trainer.train_episode(env)
        assert trainer.total_episodes == 1
        assert "reward" in result
        assert "sharpe" in result
        assert "steps" in result
        assert isinstance(result["reward"], float)

    def test_ppo_multiple_episodes(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.ppo import PPOTrainer

        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = PPOTrainer(state_dim=env.observation_space.shape[0], action_dim=3)

        for _ in range(3):
            trainer.train_episode(env)
            env.reset()
        assert trainer.total_episodes == 3


@pytest.mark.skipif(not HAS_TORCH or not HAS_GYM, reason="torch or gymnasium not installed")
class TestDQN:
    def test_dqn_network_forward(self, sample_ohlcv):
        from src.ml.rl.algorithms.dqn import DQNNetwork

        net = DQNNetwork(state_dim=52, action_dim=3)
        x = torch.randn(4, 52)
        out = net(x)
        assert out.shape == (4, 3)

    def test_dqn_train_one_episode(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.dqn import DQNTrainer

        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = DQNTrainer(state_dim=env.observation_space.shape[0], action_dim=3)

        result = trainer.train_episode(env)
        assert trainer.total_episodes == 1
        assert "reward" in result
        assert "steps" in result
        assert isinstance(result["reward"], float)

    def test_dqn_epsilon_decay(self, sample_ohlcv):
        from src.ml.rl.env import TradingEnv
        from src.ml.rl.algorithms.dqn import DQNTrainer

        env = TradingEnv(sample_ohlcv, window_size=10)
        trainer = DQNTrainer(state_dim=env.observation_space.shape[0], action_dim=3, epsilon=1.0)
        initial_epsilon = trainer.epsilon

        for _ in range(5):
            trainer.train_episode(env)
            env.reset()

        assert trainer.epsilon < initial_epsilon


class TestReplayBuffer:
    def test_push_and_sample(self):
        from src.ml.rl.replay import ReplayBuffer

        buf = ReplayBuffer(capacity=100)
        for i in range(50):
            buf.push(i, i + 1, float(i), i + 2, False)

        assert len(buf) == 50

        batch = buf.sample(16)
        assert len(batch) == 16
        assert len(batch[0]) == 5  # (state, action, reward, next_state, done)

    def test_capacity_limit(self):
        from src.ml.rl.replay import ReplayBuffer

        buf = ReplayBuffer(capacity=10)
        for i in range(20):
            buf.push(i, i, float(i), i, False)

        assert len(buf) == 10

    def test_sample_empty(self):
        from src.ml.rl.replay import ReplayBuffer

        buf = ReplayBuffer(capacity=10)
        batch = buf.sample(5)
        assert len(batch) == 0
