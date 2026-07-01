"""LLM provider registry — maps model IDs to provider instances."""

from abc import ABC, abstractmethod
from typing import AsyncIterator, Dict

from src.proto.llm_pb2 import LLMChatRequest, LLMChatResponse


class LLMProvider(ABC):
    """Abstract base for LLM providers. Each provider handles one API family."""

    @abstractmethod
    async def chat(self, request: LLMChatRequest, context) -> AsyncIterator[LLMChatResponse]:
        """Stream chat completions for the given request.

        Args:
            request: The gRPC chat request with messages and tools.
            context: gRPC ServicerContext for cancellation detection.

        Yields:
            LLMChatResponse messages with incremental deltas.
        """
        ...


# Global registry: provider_name -> provider instance
_providers: Dict[str, "LLMProvider"] = {}


def register_provider(name: str, provider: "LLMProvider") -> None:
    """Register a provider instance under its name (e.g., 'ollama', 'openai')."""
    _providers[name] = provider


def get_provider(model_id: str) -> "LLMProvider":
    """Resolve a model ID to its provider instance.

    Model ID format: "provider/model_name" (e.g., "openai/gpt-4o", "ollama/llama3.1").
    """
    provider_name = model_id.split("/")[0]
    if provider_name not in _providers:
        raise ValueError(f"Unknown provider: {provider_name}. Registered: {list(_providers.keys())}")
    return _providers[provider_name]


def list_providers() -> list[str]:
    """Return names of all registered providers."""
    return list(_providers.keys())


# Import all built-in provider modules to trigger auto-registration.
# Each provider module calls register_provider() at module level.
# These imports are placed after the ABC/registry definitions to avoid
# circular import issues — Python's partial module initialization means
# LLMProvider and register_provider are available when the submodules import them.
from src.llm.providers import ollama_provider  # noqa: F401, E402
from src.llm.providers import openai_provider  # noqa: F401, E402
from src.llm.providers import anthropic_provider  # noqa: F401, E402
from src.llm.providers import deepseek_provider  # noqa: F401, E402
from src.llm.providers import compatible_providers  # noqa: F401, E402
