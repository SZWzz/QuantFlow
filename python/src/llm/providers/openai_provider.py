"""OpenAI provider — GPT-4o, GPT-4.1, o4-mini via OpenAI API."""

import json
import logging
import os
from typing import AsyncIterator

import httpx

from src.proto.llm_pb2 import (
    LLMChatRequest,
    LLMChatResponse,
    ToolCallDelta,
)
from src.llm.providers import LLMProvider, register_provider

logger = logging.getLogger(__name__)

OPENAI_DEFAULT_URL = "https://api.openai.com"


class OpenAIProvider(LLMProvider):
    """Provider for OpenAI compatible APIs (OpenAI, DeepSeek, etc.)."""

    def __init__(self, base_url: str = OPENAI_DEFAULT_URL, api_key: str | None = None):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key or os.getenv("OPENAI_API_KEY", "")
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=httpx.Timeout(120.0))
        return self._client

    async def chat(self, request: LLMChatRequest, context) -> AsyncIterator[LLMChatResponse]:
        """Stream chat via OpenAI /v1/chat/completions."""
        if not self.api_key:
            resp = LLMChatResponse()
            resp.finish_reason = "error"
            resp.delta_content = "Error: OPENAI_API_KEY not set."
            yield resp
            return

        client = await self._get_client()
        model_name = request.model.split("/", 1)[1] if "/" in request.model else request.model

        messages = []
        if request.system_prompt:
            messages.append({"role": "system", "content": request.system_prompt})
        for msg in request.messages:
            m = {"role": msg.role, "content": msg.content}
            if msg.tool_call_id:
                m["tool_call_id"] = msg.tool_call_id
            if msg.tool_calls:
                m["tool_calls"] = [
                    {"id": tc.id, "type": "function", "function": {"name": tc.name, "arguments": tc.arguments}}
                    for tc in msg.tool_calls
                ]
            messages.append(m)

        body = {
            "model": model_name,
            "messages": messages,
            "stream": True,
        }

        if request.tools:
            body["tools"] = [
                {"type": "function", "function": {"name": t.name, "description": t.description, "parameters": json.loads(t.parameters_json)}}
                for t in request.tools
            ]

        if request.temperature:
            body["temperature"] = request.temperature
        if request.max_tokens:
            body["max_tokens"] = request.max_tokens

        try:
            async with client.stream(
                "POST",
                f"{self.base_url}/v1/chat/completions",
                json=body,
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json",
                },
            ) as response:
                if response.status_code == 401:
                    resp = LLMChatResponse()
                    resp.finish_reason = "error"
                    resp.delta_content = "Error: Invalid API key."
                    yield resp
                    return
                response.raise_for_status()

                # Accumulate tool call deltas across chunks
                tool_call_buf: dict[int, dict] = {}
                async for line in response.aiter_lines():
                    if not line or not line.startswith("data: "):
                        continue
                    data_str = line[6:]
                    if data_str == "[DONE]":
                        continue
                    try:
                        chunk = json.loads(data_str)
                    except json.JSONDecodeError:
                        continue

                    resp = LLMChatResponse()
                    choice = chunk.get("choices", [{}])[0]
                    delta = choice.get("delta", {})

                    if "content" in delta and delta["content"]:
                        resp.delta_content = delta["content"]

                    if "tool_calls" in delta:
                        for tc in delta["tool_calls"]:
                            idx = tc.get("index", 0)
                            if idx not in tool_call_buf:
                                tool_call_buf[idx] = {"id": "", "name": "", "arguments": ""}
                            buf = tool_call_buf[idx]
                            if "id" in tc:
                                buf["id"] = tc["id"]
                            if tc.get("function", {}).get("name"):
                                buf["name"] = tc["function"]["name"]
                            if tc.get("function", {}).get("arguments"):
                                buf["arguments"] += tc["function"]["arguments"]

                            tcd = ToolCallDelta()
                            tcd.index = idx
                            tcd.id = buf["id"]
                            tcd.name = buf["name"]
                            tcd.arguments_delta = tc.get("function", {}).get("arguments", "")
                            resp.tool_call_delta.CopyFrom(tcd)

                    finish = choice.get("finish_reason")
                    if finish:
                        resp.finish_reason = finish

                    # Usage comes in the last chunk
                    usage = chunk.get("usage", {})
                    if usage:
                        resp.prompt_tokens = usage.get("prompt_tokens", 0)
                        resp.completion_tokens = usage.get("completion_tokens", 0)

                    yield resp
        except httpx.ConnectError as e:
            resp = LLMChatResponse()
            resp.finish_reason = "error"
            resp.delta_content = f"Error: Connection failed — {e}"
            yield resp


# Auto-register OpenAI on import
openai_key = os.getenv("OPENAI_API_KEY", "")
openai_provider = OpenAIProvider(api_key=openai_key)
register_provider("openai", openai_provider)
