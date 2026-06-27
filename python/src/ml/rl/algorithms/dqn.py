"""DQN (Deep Q-Network) for discrete-action trading."""
import numpy as np
import random
from collections import deque

try:
    import torch
    import torch.nn as nn
    import torch.optim as optim
    HAS_TORCH = True
except ImportError:
    HAS_TORCH = False


class DQNNetwork(nn.Module):
    """Q-network: state → Q-values for each action."""

    def __init__(self, state_dim, action_dim, hidden_size=128):
        super().__init__()
        self.net = nn.Sequential(
            nn.Linear(state_dim, hidden_size), nn.ReLU(),
            nn.Linear(hidden_size, hidden_size), nn.ReLU(),
            nn.Linear(hidden_size, action_dim),
        )

    def forward(self, x):
        return self.net(x)


class DQNTrainer:
    """DQN trainer with epsilon-greedy exploration and experience replay."""

    def __init__(self, state_dim, action_dim, lr=1e-3, gamma=0.99,
                 epsilon=1.0, epsilon_min=0.01, epsilon_decay=0.995):
        if not HAS_TORCH:
            raise ImportError("torch is required. Install with: pip install torch")
        self.network = DQNNetwork(state_dim, action_dim)
        self.target = DQNNetwork(state_dim, action_dim)
        self.target.load_state_dict(self.network.state_dict())
        self.optimizer = optim.Adam(self.network.parameters(), lr=lr)
        self.gamma = gamma
        self.epsilon = epsilon
        self.epsilon_min = epsilon_min
        self.epsilon_decay = epsilon_decay
        self.memory = deque(maxlen=10000)
        self.total_episodes = 0
        self.action_dim = action_dim
        self.state_dim = state_dim

    def train_episode(self, env) -> dict:
        """Run one episode with epsilon-greedy, replay training. Returns metrics dict."""
        state, _ = env.reset()
        total_reward = 0
        steps = 0
        done = False

        while not done:
            # Epsilon-greedy action selection
            if random.random() < self.epsilon:
                action = random.randrange(self.action_dim)
            else:
                with torch.no_grad():
                    action = self.network(torch.FloatTensor(state).unsqueeze(0)).argmax().item()

            next_state, reward, done, _, _ = env.step(action)
            self.memory.append((state, action, reward, next_state, done))
            state = next_state
            total_reward += reward
            steps += 1

            # Train on a minibatch
            if len(self.memory) >= 32:
                batch = random.sample(self.memory, 32)
                bs, ba, br, bns, bd = zip(*batch)
                bs_t = torch.FloatTensor(np.array(bs))
                br_t = torch.FloatTensor(br)

                with torch.no_grad():
                    next_q = self.target(torch.FloatTensor(np.array(bns))).max(1)[0]
                    targets = br_t + self.gamma * next_q * (1 - torch.FloatTensor(bd))

                current_q = self.network(bs_t).gather(1, torch.LongTensor(ba).unsqueeze(1)).squeeze()
                loss = nn.MSELoss()(current_q, targets)

                self.optimizer.zero_grad()
                loss.backward()
                self.optimizer.step()

        # Decay epsilon and update target network periodically
        self.epsilon = max(self.epsilon_min, self.epsilon * self.epsilon_decay)
        if self.total_episodes % 10 == 0:
            self.target.load_state_dict(self.network.state_dict())

        self.total_episodes += 1

        # Zero-division protection: guard against near-zero std
        reward_std = np.std([total_reward])
        if abs(total_reward) < 1e-10 or reward_std < 1e-10:
            sharpe = 0.0
        else:
            sharpe = (np.mean([total_reward]) / (reward_std + 1e-8)) * np.sqrt(252)

        return {"episode": self.total_episodes, "reward": total_reward, "sharpe": sharpe, "steps": steps}

    def predict(self, obs, deterministic=True):
        """Predict action from observation.

        Args:
            obs: numpy array of shape (batch, state_dim)
            deterministic: if True, use argmax; otherwise sample from Q-values

        Returns:
            action: numpy array of shape (batch,)
            q_values: unused (for interface compatibility)
        """
        self.network.eval()
        with torch.no_grad():
            q_values = self.network(torch.FloatTensor(obs))
            if deterministic:
                action = q_values.argmax(dim=1).cpu().numpy()
            else:
                action = torch.multinomial(torch.softmax(q_values, dim=-1), 1).squeeze(1).cpu().numpy()
        self.network.train()
        return action, None
