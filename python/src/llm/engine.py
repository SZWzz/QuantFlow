"""LLMService — gRPC service for LLM inference with streaming."""

import json
import logging
import uuid
from typing import AsyncIterator

from src.proto import llm_pb2, llm_pb2_grpc
from src.llm.providers import get_provider, list_providers
from src.llm.prompt_template import PromptTemplate, PromptContext, ToolDef

logger = logging.getLogger(__name__)

# Available models registry (known models, queried at startup or hardcoded)
AVAILABLE_MODELS = [
    llm_pb2.LLMModelInfo(
        id="ollama/llama3.1:8b",
        provider="ollama",
        display_name="Llama 3.1 8B",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="openai/gpt-4o",
        provider="openai",
        display_name="GPT-4o",
        context_window=128000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="openai/gpt-4.1",
        provider="openai",
        display_name="GPT-4.1",
        context_window=1000000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="anthropic/claude-sonnet-4-6",
        provider="anthropic",
        display_name="Claude Sonnet 4.6",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="anthropic/claude-opus-4-8",
        provider="anthropic",
        display_name="Claude Opus 4.8",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="deepseek/deepseek-chat",
        provider="deepseek",
        display_name="DeepSeek-V3",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
]


class LLMService(llm_pb2_grpc.LLMServiceServicer):
    """gRPC service for LLM chat, model listing, and token counting."""

    async def Chat(self, request: llm_pb2.LLMChatRequest, context) -> AsyncIterator[llm_pb2.LLMChatResponse]:
        """Stream chat completions from the requested model."""
        stream_id = request.stream_id or str(uuid.uuid4())[:8]
        logger.info(f"Chat [{stream_id}]: model={request.model}, messages={len(request.messages)}, tools={len(request.tools)}")

        try:
            provider = get_provider(request.model)
        except ValueError as e:
            err = llm_pb2.LLMChatResponse()
            err.finish_reason = "error"
            err.delta_content = str(e)
            yield err
            return

        async for chunk in provider.chat(request, context):
            # Check if client cancelled
            if context.cancelled():
                logger.info(f"Chat [{stream_id}]: cancelled by client")
                break
            yield chunk

        logger.info(f"Chat [{stream_id}]: completed")

    async def ListModels(self, request: llm_pb2.LLMListModelsRequest, context) -> llm_pb2.LLMListModelsResponse:
        """Return all available models."""
        # Filter by available providers (only return models whose provider is registered)
        available_providers = set(list_providers())
        models = [m for m in AVAILABLE_MODELS if m.provider in available_providers]
        return llm_pb2.LLMListModelsResponse(models=models)

    async def CountTokens(self, request: llm_pb2.CountTokensRequest, context) -> llm_pb2.CountTokensResponse:
        """Estimate token count for messages using a simple heuristic."""
        total_chars = len(request.system_prompt or "")
        for msg in request.messages:
            total_chars += len(msg.content or "")
        # Rough estimate: 4 chars per token
        token_count = max(1, total_chars // 4)
        return llm_pb2.CountTokensResponse(token_count=token_count)
