"""Anthropic provider — Claude models via Anthropic Messages API."""

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

ANTHROPIC_DEFAULT_URL = "https://api.anthropic.com"


class AnthropicProvider(LLMProvider):
    """Provider for Anthropic Claude models.

    Uses the Messages API with server-sent events (SSE) streaming.
    Tool use is translated between Anthropic's format and our unified proto.
    """

    def __init__(self, base_url: str = ANTHROPIC_DEFAULT_URL, api_key: str | None = None):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key or os.getenv("ANTHROPIC_API_KEY", "")
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=httpx.Timeout(120.0))
        return self._client

    async def chat(self, request: LLMChatRequest, context) -> AsyncIterator[LLMChatResponse]:
        """Stream chat via Anthropic /v1/messages."""
        if not self.api_key:
            resp = LLMChatResponse()
            resp.finish_reason = "error"
            resp.delta_content = "Error: ANTHROPIC_API_KEY not set."
            yield resp
            return

        client = await self._get_client()
        model_name = request.model.split("/", 1)[1] if "/" in request.model else request.model

        # Build Anthropic-format messages
        anthropic_messages = []
        for msg in request.messages:
            role = msg.role
            if role == "system":
                continue  # system prompt handled separately
            m = {"role": role, "content": []}
            if msg.content:
                m["content"].append({"type": "text", "text": msg.content})
            if msg.tool_calls:
                for tc in msg.tool_calls:
                    m["content"].append({
                        "type": "tool_use",
                        "id": tc.id,
                        "name": tc.name,
                        "input": json.loads(tc.arguments) if tc.arguments else {},
                    })
            anthropic_messages.append(m)

        body = {
            "model": model_name,
            "messages": anthropic_messages,
            "max_tokens": request.max_tokens or 4096,
            "stream": True,
        }

        if request.system_prompt:
            body["system"] = request.system_prompt

        if request.tools:
            body["tools"] = [
                {"name": t.name, "description": t.description, "input_schema": json.loads(t.parameters_json)}
                for t in request.tools
            ]

        if request.temperature:
            body["temperature"] = request.temperature

        try:
            async with client.stream(
                "POST",
                f"{self.base_url}/v1/messages",
                json=body,
                headers={
                    "x-api-key": self.api_key,
                    "anthropic-version": "2023-06-01",
                    "Content-Type": "application/json",
                },
            ) as response:
                if response.status_code == 401:
                    resp = LLMChatResponse()
                    resp.finish_reason = "error"
                    resp.delta_content = "Error: Invalid Anthropic API key."
                    yield resp
                    return
                response.raise_for_status()

                # Accumulate content and tool use across SSE events
                tool_use_buf: dict[int, dict] = {}
                async for line in response.aiter_lines():
                    if not line or not line.startswith("data: "):
                        continue
                    data_str = line[6:]
                    try:
                        event = json.loads(data_str)
                    except json.JSONDecodeError:
                        continue

                    resp = LLMChatResponse()

                    event_type = event.get("type", "")
                    if event_type == "content_block_delta":
                        delta = event.get("delta", {})
                        if delta.get("type") == "text_delta":
                            resp.delta_content = delta.get("text", "")
                        elif delta.get("type") == "input_json_delta":
                            idx = event.get("index", 0)
                            if idx not in tool_use_buf:
                                tool_use_buf[idx] = {"id": "", "name": "", "arguments": ""}
                            tool_use_buf[idx]["arguments"] += delta.get("partial_json", "")
                            tcd = ToolCallDelta()
                            tcd.index = idx
                            tcd.arguments_delta = delta.get("partial_json", "")
                            resp.tool_call_delta.CopyFrom(tcd)

                    elif event_type == "content_block_start":
                        block = event.get("content_block", {})
                        if block.get("type") == "tool_use":
                            idx = event.get("index", 0)
                            tool_use_buf[idx] = {
                                "id": block.get("id", ""),
                                "name": block.get("name", ""),
                                "arguments": "",
                            }
                            tcd = ToolCallDelta()
                            tcd.index = idx
                            tcd.id = block.get("id", "")
                            tcd.name = block.get("name", "")
                            resp.tool_call_delta.CopyFrom(tcd)

                    elif event_type == "message_delta":
                        usage = event.get("usage", {})
                        resp.prompt_tokens = usage.get("input_tokens", 0)
                        resp.completion_tokens = usage.get("output_tokens", 0)
                        # Only set finish_reason on message_delta
                        delta = event.get("delta", {})
                        resp.finish_reason = delta.get("stop_reason", "stop")

                    elif event_type == "message_stop":
                        if not resp.finish_reason:
                            resp.finish_reason = "stop"

                    yield resp
        except httpx.ConnectError as e:
            resp = LLMChatResponse()
            resp.finish_reason = "error"
            resp.delta_content = f"Error: Connection failed — {e}"
            yield resp


# Auto-register Anthropic on import
anthropic_key = os.getenv("ANTHROPIC_API_KEY", "")
anthropic_provider = AnthropicProvider(api_key=anthropic_key)
register_provider("anthropic", anthropic_provider)
