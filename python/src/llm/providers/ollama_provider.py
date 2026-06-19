"""Ollama provider — local LLM via Ollama API (http://localhost:11434)."""

import json
import logging
from typing import AsyncIterator

import httpx

from src.proto.llm_pb2 import (
    LLMChatRequest,
    LLMChatResponse,
    ToolCallDelta,
)
from src.llm.providers import LLMProvider, register_provider

logger = logging.getLogger(__name__)

OLLAMA_DEFAULT_URL = "http://localhost:11434"


class OllamaProvider(LLMProvider):
    """Provider for Ollama — local LLM inference, no API key required.

    Uses Ollama's /api/chat endpoint with streaming.
    Tool calling is supported in Ollama 0.3+ via the native tools API.
    """

    def __init__(self, base_url: str = OLLAMA_DEFAULT_URL):
        self.base_url = base_url.rstrip("/")
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=httpx.Timeout(120.0))
        return self._client

    async def chat(self, request: LLMChatRequest, context) -> AsyncIterator[LLMChatResponse]:
        """Stream chat via Ollama /api/chat."""
        client = await self._get_client()
        model_name = request.model.split("/", 1)[1] if "/" in request.model else request.model

        # Build Ollama-format messages
        ollama_messages = []
        if request.system_prompt:
            ollama_messages.append({"role": "system", "content": request.system_prompt})
        for msg in request.messages:
            ollama_msg = {"role": msg.role, "content": msg.content}
            if msg.tool_calls:
                ollama_msg["tool_calls"] = [
                    {"function": {"name": tc.name, "arguments": json.loads(tc.arguments)}}
                    for tc in msg.tool_calls
                ]
            ollama_messages.append(ollama_msg)

        body = {
            "model": model_name,
            "messages": ollama_messages,
            "stream": True,
        }

        if request.tools:
            body["tools"] = [
                {"type": "function", "function": {"name": t.name, "description": t.description, "parameters": json.loads(t.parameters_json)}}
                for t in request.tools
            ]

        if request.temperature:
            body["options"] = {"temperature": request.temperature}

        try:
            async with client.stream("POST", f"{self.base_url}/api/chat", json=body) as response:
                response.raise_for_status()
                async for line in response.aiter_lines():
                    if not line:
                        continue
                    try:
                        chunk = json.loads(line)
                    except json.JSONDecodeError:
                        continue

                    resp = LLMChatResponse()

                    if "message" in chunk:
                        msg = chunk["message"]
                        if "content" in msg and msg["content"]:
                            resp.delta_content = msg["content"]
                        if "tool_calls" in msg:
                            for tc in msg["tool_calls"]:
                                tcd = ToolCallDelta()
                                func = tc.get("function", {})
                                tcd.name = func.get("name", "")
                                tcd.arguments_delta = json.dumps(func.get("arguments", {}))
                                resp.tool_call_delta.CopyFrom(tcd)

                    if chunk.get("done", False):
                        resp.finish_reason = "stop"
                        if "prompt_eval_count" in chunk:
                            resp.prompt_tokens = chunk["prompt_eval_count"]
                        if "eval_count" in chunk:
                            resp.completion_tokens = chunk["eval_count"]

                    yield resp
        except httpx.ConnectError:
            resp = LLMChatResponse()
            resp.finish_reason = "error"
            resp.delta_content = "Error: Cannot connect to Ollama. Is it running? (ollama serve)"
            yield resp


# Auto-register Ollama on import
register_provider("ollama", OllamaProvider())
