"""DeepSeek provider — DeepSeek-V3, DeepSeek-R1 via OpenAI-compatible API."""

import os

from src.llm.providers.openai_provider import OpenAIProvider
from src.llm.providers import register_provider

DEEPSEEK_URL = "https://api.deepseek.com"


class DeepSeekProvider(OpenAIProvider):
    """Provider for DeepSeek models via OpenAI-compatible API.

    DeepSeek's API is fully OpenAI-compatible, so we inherit OpenAIProvider
    and just change the base_url and default API key.
    """

    def __init__(self, base_url: str = DEEPSEEK_URL, api_key: str | None = None):
        super().__init__(base_url=base_url, api_key=api_key or os.getenv("DEEPSEEK_API_KEY", ""))


# Auto-register DeepSeek on import
deepseek_key = os.getenv("DEEPSEEK_API_KEY", "")
deepseek_provider = DeepSeekProvider(api_key=deepseek_key)
register_provider("deepseek", deepseek_provider)
