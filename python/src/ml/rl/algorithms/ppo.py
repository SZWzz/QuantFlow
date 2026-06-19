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
    nn = None
    optim = None


class PPONetwork(nn.Module):
    """Actor-critic network with shared layers for PPO."""

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
    """PPO trainer with clipped surrogate objective.

    Trains one episode at a time and returns metrics dict.
    """

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
        """Run one episode and perform PPO update. Returns metrics dict."""
        state, _ = env.reset()
        states, actions, rewards, log_probs, values = [], [], [], [], []
        done = False

        # Collect trajectory
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

        # PPO update (multiple epochs)
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
