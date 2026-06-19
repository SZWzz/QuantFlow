"""Replay buffer for experience replay in RL algorithms."""
from collections import deque
import random


class ReplayBuffer:
    """Fixed-capacity replay buffer for storing transitions."""

    def __init__(self, capacity=10000):
        self.buffer = deque(maxlen=capacity)

    def push(self, *args):
        """Store a transition (state, action, reward, next_state, done)."""
        self.buffer.append(args)

    def sample(self, batch_size):
        """Return a random sample of transitions."""
        return random.sample(self.buffer, min(batch_size, len(self.buffer)))

    def __len__(self):
        return len(self.buffer)
