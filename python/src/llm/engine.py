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
    # ── Google Gemini ──
    llm_pb2.LLMModelInfo(
        id="google/gemini-2.5-flash",
        provider="google",
        display_name="Gemini 2.5 Flash",
        context_window=1000000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="google/gemini-2.5-pro",
        provider="google",
        display_name="Gemini 2.5 Pro",
        context_window=1000000,
        supports_tools=True,
        supports_vision=True,
    ),
    # ── Mistral AI ──
    llm_pb2.LLMModelInfo(
        id="mistral/mistral-large-2506",
        provider="mistral",
        display_name="Mistral Large",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="mistral/mistral-3.1-small",
        provider="mistral",
        display_name="Mistral 3.1 Small",
        context_window=32768,
        supports_tools=True,
        supports_vision=False,
    ),
    # ── Groq ──
    llm_pb2.LLMModelInfo(
        id="groq/llama-3.3-70b",
        provider="groq",
        display_name="Llama 3.3 70B",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="groq/deepseek-r1-671b",
        provider="groq",
        display_name="DeepSeek R1 671B",
        context_window=131072,
        supports_tools=False,
        supports_vision=False,
    ),
    # ── SiliconFlow ──
    llm_pb2.LLMModelInfo(
        id="siliconflow/deepseek-v3",
        provider="siliconflow",
        display_name="DeepSeek-V3",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="siliconflow/deepseek-r1",
        provider="siliconflow",
        display_name="DeepSeek-R1",
        context_window=131072,
        supports_tools=False,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="siliconflow/qwen-2.5-72b",
        provider="siliconflow",
        display_name="Qwen 2.5 72B",
        context_window=131072,
        supports_tools=True,
        supports_vision=True,
    ),
    # ── Zhipu AI (智谱) ──
    llm_pb2.LLMModelInfo(
        id="zhipu/glm-5",
        provider="zhipu",
        display_name="GLM-5",
        context_window=131072,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="zhipu/glm-5-flash",
        provider="zhipu",
        display_name="GLM-5 Flash",
        context_window=131072,
        supports_tools=True,
        supports_vision=True,
    ),
    # ── OpenRouter ──
    llm_pb2.LLMModelInfo(
        id="openrouter/openai/gpt-4o",
        provider="openrouter",
        display_name="GPT-4o (OpenRouter)",
        context_window=128000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="openrouter/anthropic/claude-opus-4",
        provider="openrouter",
        display_name="Claude Opus 4 (OpenRouter)",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="openrouter/deepseek/deepseek-chat",
        provider="openrouter",
        display_name="DeepSeek-V3 (OpenRouter)",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    # ── OpenCode Zen ──
    llm_pb2.LLMModelInfo(
        id="opencode/gpt-5.5",
        provider="opencode",
        display_name="GPT 5.5",
        context_window=272000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="opencode/gpt-5.4",
        provider="opencode",
        display_name="GPT 5.4",
        context_window=272000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="opencode/claude-sonnet-4-6",
        provider="opencode",
        display_name="Claude Sonnet 4.6",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.LLMModelInfo(
        id="opencode/deepseek-v4-flash",
        provider="opencode",
        display_name="DeepSeek V4 Flash",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.LLMModelInfo(
        id="opencode/qwen3.7-plus",
        provider="opencode",
        display_name="Qwen 3.7 Plus",
        context_window=131072,
        supports_tools=True,
        supports_vision=True,
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
        """Estimate token count using model-specific tokenizer when available."""
        total_text = (request.system_prompt or "")
        for msg in request.messages:
            total_text += (msg.content or "")

        try:
            import tiktoken
            model = request.model or "gpt-4"
            try:
                encoding = tiktoken.encoding_for_model(model)
            except KeyError:
                encoding = tiktoken.get_encoding("cl100k_base")
            token_count = len(encoding.encode(total_text))
        except ImportError:
            # Fallback: approximate (chars / 4 for English, ~1.5 for CJK)
            ascii_count = sum(1 for c in total_text if ord(c) < 128)
            unicode_count = len(total_text) - ascii_count
            token_count = max(1, ascii_count // 4 + unicode_count)

        return llm_pb2.CountTokensResponse(token_count=token_count)
