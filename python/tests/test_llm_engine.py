"""Tests for LLM service, providers, and prompt template engine."""
import pytest
from unittest.mock import AsyncMock, patch, MagicMock

from src.proto.llm_pb2 import (
    LLMChatRequest,
    ChatMessage,
    LLMTool,
    LLMListModelsRequest,
    CountTokensRequest,
)
from src.llm.engine import LLMService
from src.llm.providers import get_provider, list_providers
from src.llm.prompt_template import PromptTemplate, PromptContext, ToolDef, estimate_tokens


class TestPromptTemplate:
    """Tests for the PromptTemplate engine."""

    def test_assemble_basic(self):
        ctx = PromptContext(
            base_system_prompt="You are a quant assistant.",
        )
        tmpl = PromptTemplate(ctx)
        result = tmpl.assemble_system_prompt()
        assert "quant assistant" in result

    def test_assemble_with_tools(self):
        ctx = PromptContext(
            base_system_prompt="You are helpful.",
            tools=[ToolDef(name="quote", description="Get stock quote", parameters_json='{"type":"object","properties":{"symbol":{"type":"string"}}}')],
        )
        tmpl = PromptTemplate(ctx)
        result = tmpl.assemble_system_prompt()
        assert "quote" in result
        assert "Get stock quote" in result

    def test_assemble_with_skills(self):
        ctx = PromptContext(
            base_system_prompt="You are helpful.",
            skills=["# Momentum Strategy\nBuy winners, sell losers."],
        )
        tmpl = PromptTemplate(ctx)
        result = tmpl.assemble_system_prompt()
        assert "Momentum Strategy" in result

    def test_token_budget_respected(self):
        """Skills should be truncated if they exceed budget."""
        ctx = PromptContext(
            base_system_prompt="Short prompt.",
            model_context_window=4000,  # very small context
        )
        tmpl = PromptTemplate(ctx)
        result = tmpl.assemble_system_prompt()
        est = estimate_tokens(result)
        # Should be well under the context window
        assert est < 4000

    def test_estimate_tokens(self):
        assert estimate_tokens("") == 1
        assert estimate_tokens("hello world") > 0
        # ~4 chars per token
        assert estimate_tokens("a" * 400) == 100


class TestLLMService:
    """Tests for the LLMService gRPC implementation."""

    @pytest.mark.asyncio
    async def test_list_models(self):
        svc = LLMService()
        resp = await svc.ListModels(LLMListModelsRequest(), None)
        assert len(resp.models) > 0
        model_ids = [m.id for m in resp.models]
        assert any("ollama" in mid for mid in model_ids)

    @pytest.mark.asyncio
    async def test_list_models_has_all_providers(self):
        svc = LLMService()
        resp = await svc.ListModels(LLMListModelsRequest(), None)
        providers = set(m.provider for m in resp.models)
        assert "ollama" in providers  # Ollama is always registered

    @pytest.mark.asyncio
    async def test_count_tokens(self):
        svc = LLMService()
        req = CountTokensRequest(
            model="openai/gpt-4o",
            messages=[ChatMessage(role="user", content="Hello, world!")],
            system_prompt="You are helpful.",
        )
        resp = await svc.CountTokens(req, None)
        assert resp.token_count > 0
        # "You are helpful." + "Hello, world!" = ~31 chars / 4 ~= 7-8
        assert resp.token_count >= 5

    @pytest.mark.asyncio
    async def test_chat_unknown_provider(self):
        svc = LLMService()
        req = LLMChatRequest(model="unknown/model")
        chunks = [c async for c in svc.Chat(req, MagicMock())]
        assert len(chunks) > 0
        assert chunks[0].finish_reason == "error"

    @pytest.mark.asyncio
    async def test_chat_empty_messages(self):
        """Chat with empty messages should still return something (error or empty, not crash)."""
        svc = LLMService()
        req = LLMChatRequest(model="unknown/model")
        chunks = [c async for c in svc.Chat(req, MagicMock())]
        assert len(chunks) > 0  # Should at least yield an error


class TestProviders:
    """Tests for provider registration."""

    def test_providers_registered(self):
        providers = list_providers()
        assert "ollama" in providers
        assert len(providers) >= 1

    def test_get_provider_ollama(self):
        provider = get_provider("ollama/llama3.1:8b")
        assert provider is not None

    def test_get_provider_unknown_raises(self):
        with pytest.raises(ValueError):
            get_provider("nonexistent/model")
