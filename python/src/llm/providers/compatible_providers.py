"""OpenAI-compatible third-party LLM providers.

These providers all speak the OpenAI /v1/chat/completions protocol, so we reuse
OpenAIProvider with different default base URLs and API key env vars:
  - Google Gemini (generativelanguage.googleapis.com)
  - Mistral AI (api.mistral.ai)
  - Groq (api.groq.com)
  - SiliconFlow (api.siliconflow.cn)
  - Zhipu AI / 智谱 (open.bigmodel.cn)
  - OpenRouter (openrouter.ai)
"""

import os

from src.llm.providers.openai_provider import OpenAIProvider
from src.llm.providers import register_provider

for name, env_key, default_url in [
    ("google",      "GOOGLE_API_KEY",        "https://generativelanguage.googleapis.com"),
    ("mistral",     "MISTRAL_API_KEY",       "https://api.mistral.ai"),
    ("groq",        "GROQ_API_KEY",          "https://api.groq.com/openai/v1"),
    ("siliconflow", "SILICONFLOW_API_KEY",   "https://api.siliconflow.cn/v1"),
    ("zhipu",       "ZHIPU_API_KEY",         "https://open.bigmodel.cn/api/paas/v4"),
    ("openrouter",  "OPENROUTER_API_KEY",    "https://openrouter.ai/api/v1"),
]:
    api_key = os.getenv(env_key, "")
    provider = OpenAIProvider(base_url=default_url, api_key=api_key)
    register_provider(name, provider)
