"""Tests for LLM provider instantiation and error handling.

Tests cover instantiation with/without API keys, default base_url correctness,
and graceful handling of missing keys — no network calls.
"""

from unittest.mock import patch


class TestOpenAIProvider:
    """OpenAIProvider — standard API key provider with configurable base_url."""

    def test_instantiation_default(self):
        from src.llm.providers.openai_provider import OpenAIProvider, OPENAI_DEFAULT_URL

        provider = OpenAIProvider()
        assert provider.base_url == OPENAI_DEFAULT_URL
        # api_key defaults to env var or empty string; don't depend on env here
        assert isinstance(provider.api_key, str)

    def test_instantiation_with_key(self):
        from src.llm.providers.openai_provider import OpenAIProvider

        provider = OpenAIProvider(api_key="sk-test-123")
        assert provider.api_key == "sk-test-123"

    def test_default_base_url(self):
        from src.llm.providers.openai_provider import OPENAI_DEFAULT_URL

        assert OPENAI_DEFAULT_URL == "https://api.openai.com"

    def test_empty_api_key_defaults_to_env(self):
        """When no api_key is given, fall back to OPENAI_API_KEY env var."""
        from src.llm.providers.openai_provider import OpenAIProvider

        with patch.dict("os.environ", {"OPENAI_API_KEY": "sk-env-key"}):
            provider = OpenAIProvider()
            assert provider.api_key == "sk-env-key"

    def test_empty_api_key_is_empty_string_when_no_env(self):
        """When neither arg nor env var provides a key, api_key is ''."""
        from src.llm.providers.openai_provider import OpenAIProvider

        with patch.dict("os.environ", {}, clear=True):
            provider = OpenAIProvider()
            assert provider.api_key == ""


class TestDeepSeekProvider:
    """DeepSeekProvider — subclass of OpenAIProvider with different defaults."""

    def test_instantiation_default(self):
        from src.llm.providers.deepseek_provider import DeepSeekProvider, DEEPSEEK_URL

        provider = DeepSeekProvider()
        assert provider.base_url == DEEPSEEK_URL
        assert isinstance(provider.api_key, str)

    def test_instantiation_with_key(self):
        from src.llm.providers.deepseek_provider import DeepSeekProvider

        provider = DeepSeekProvider(api_key="ds-test-key")
        assert provider.api_key == "ds-test-key"

    def test_default_base_url(self):
        from src.llm.providers.deepseek_provider import DEEPSEEK_URL

        assert DEEPSEEK_URL == "https://api.deepseek.com"

    def test_env_key_fallback(self):
        from src.llm.providers.deepseek_provider import DeepSeekProvider

        with patch.dict("os.environ", {"DEEPSEEK_API_KEY": "ds-env-key"}):
            provider = DeepSeekProvider()
            assert provider.api_key == "ds-env-key"


class TestOllamaProvider:
    """OllamaProvider — no API key, only base_url configuration."""

    def test_instantiation_default(self):
        from src.llm.providers.ollama_provider import OllamaProvider, OLLAMA_DEFAULT_URL

        provider = OllamaProvider()
        assert provider.base_url == OLLAMA_DEFAULT_URL
        # OllamaProvider does not accept or store an api_key
        assert not hasattr(provider, "api_key")

    def test_instantiation_with_url(self):
        from src.llm.providers.ollama_provider import OllamaProvider

        provider = OllamaProvider(base_url="http://custom:8080")
        assert provider.base_url == "http://custom:8080"

    def test_default_base_url(self):
        from src.llm.providers.ollama_provider import OLLAMA_DEFAULT_URL

        assert OLLAMA_DEFAULT_URL == "http://localhost:11434"


class TestAnthropicProvider:
    """AnthropicProvider — standard API key provider with configurable base_url."""

    def test_instantiation_default(self):
        from src.llm.providers.anthropic_provider import AnthropicProvider, ANTHROPIC_DEFAULT_URL

        provider = AnthropicProvider()
        assert provider.base_url == ANTHROPIC_DEFAULT_URL
        assert isinstance(provider.api_key, str)

    def test_instantiation_with_key(self):
        from src.llm.providers.anthropic_provider import AnthropicProvider

        provider = AnthropicProvider(api_key="sk-ant-test")
        assert provider.api_key == "sk-ant-test"

    def test_default_base_url(self):
        from src.llm.providers.anthropic_provider import ANTHROPIC_DEFAULT_URL

        assert ANTHROPIC_DEFAULT_URL == "https://api.anthropic.com"

    def test_env_key_fallback(self):
        from src.llm.providers.anthropic_provider import AnthropicProvider

        with patch.dict("os.environ", {"ANTHROPIC_API_KEY": "sk-ant-env"}):
            provider = AnthropicProvider()
            assert provider.api_key == "sk-ant-env"
