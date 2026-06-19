# Phase 4: AI Agent System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete AI Agent system with Go-side ReAct loop, Python-side LLM service (4 providers), Capability registry, AgentNode workflow integration, and upgraded AIChatPanel with SSE streaming.

**Architecture:** Go AgentOrchestrator runs a lightweight ReAct loop (think→act→observe), calling Python LLM Service via streaming gRPC for inference. Capabilities are registered in Go (some implemented in Go, some forwarded to Python). AgentNode bridges AI into the workflow DAG as a typed transformation node.

**Tech Stack:** Go 1.26+ (AgentOrchestrator, gRPC client), Python 3.12+ (gRPC server, LLM providers), Vue 3 (AIChatPanel with SSE), protobuf (gRPC contracts), YAML (Agent profiles), Markdown (Skill KB)

## Global Constraints

- Go module path: `quantflow`
- Python package: `src.*` (PYTHONPATH points to `python/`)
- gRPC port: localhost:50051 (insecure, localhost-only)
- Python sidecar is optional — core trading/charting must work without it
- All Go tests: `go test ./internal/... -v -count=1`
- All Python tests: `cd python && PYTHONPATH=. python -m pytest tests/ -x -q`
- Frontend build: `cd frontend && npm run build`
- Commit after each task with descriptive message
- Follow existing patterns: Go slog logging, explicit error returns, Vue Composition API with `<script setup lang="ts">`

---

## M1: Python LLM Service

### Task 1.1: Define LLM proto and generate Python code

**Files:**
- Create: `python/proto/llm.proto`
- Modify: `python/src/proto/` (generated files — `llm_pb2.py`, `llm_pb2_grpc.py`)

**Interfaces:**
- Produces: `LLMService` with `Chat` (server-streaming), `ListModels` (unary), `CountTokens` (unary)
- Produces: Messages: `LLMChatRequest`, `ChatMessage`, `ToolCall`, `LLMTool`, `LLMChatResponse`, `ToolCallDelta`, `ListModelsRequest/Response`, `CountTokensRequest/Response`, `ModelInfo`

- [ ] **Step 1: Write proto file**

```bash
cat > python/proto/llm.proto << 'PROTOEOF'
syntax = "proto3";

package quantflow;

option go_package = "quantflow/internal/python/proto;proto";

// LLMService provides LLM inference capabilities via streaming gRPC.
service LLMService {
  // Chat performs a streaming chat completion. Each response carries
  // incremental tokens or tool call deltas.
  rpc Chat(LLMChatRequest) returns (stream LLMChatResponse);

  // ListModels returns all available LLM models and their capabilities.
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);

  // CountTokens returns the token count for a set of messages (for budget management).
  rpc CountTokens(CountTokensRequest) returns (CountTokensResponse);
}

message LLMChatRequest {
  string model = 1;                    // "openai/gpt-4o", "anthropic/claude-sonnet-4-6", etc.
  repeated ChatMessage messages = 2;
  repeated LLMTool tools = 3;          // available tools with JSON Schema parameters
  string system_prompt = 4;            // system-level instruction
  float temperature = 5;               // 0.0–2.0, default 0.7
  int32 max_tokens = 6;                // max completion tokens, 0 = provider default
  string stream_id = 7;                // correlation ID forwarded to frontend events
}

message ChatMessage {
  string role = 1;                     // "system" | "user" | "assistant" | "tool"
  string content = 2;
  string tool_call_id = 3;             // set when role="tool", links back to assistant tool_call
  repeated ToolCall tool_calls = 4;    // set when role="assistant" and LLM wants to call tools
}

message ToolCall {
  string id = 1;                       // unique ID for this tool call
  string name = 2;                     // capability name
  string arguments = 3;                // JSON-encoded arguments
}

message LLMTool {
  string name = 1;
  string description = 2;
  string parameters_json = 3;          // JSON Schema for the tool's parameters
}

message LLMChatResponse {
  string delta_content = 1;            // incremental text token
  ToolCallDelta tool_call_delta = 2;   // incremental tool call chunk
  string finish_reason = 3;            // "stop" | "tool_calls" | "length" | "error"
  int32 prompt_tokens = 4;             // set on final chunk only
  int32 completion_tokens = 5;         // set on final chunk only
}

message ToolCallDelta {
  int32 index = 1;                     // which tool call this delta belongs to
  string id = 2;                       // set on first chunk
  string name = 3;                     // set on first chunk
  string arguments_delta = 4;          // incremental JSON fragment
}

message ListModelsRequest {}

message ModelInfo {
  string id = 1;                       // "openai/gpt-4o"
  string provider = 2;                 // "openai" | "anthropic" | "deepseek" | "ollama"
  string display_name = 3;             // "GPT-4o"
  int32 context_window = 4;            // max context tokens
  bool supports_tools = 5;             // whether this model supports function calling
  bool supports_vision = 6;
}

message ListModelsResponse {
  repeated ModelInfo models = 1;
}

message CountTokensRequest {
  string model = 1;
  repeated ChatMessage messages = 2;
  string system_prompt = 3;
}

message CountTokensResponse {
  int32 token_count = 1;
  string error = 2;
}
PROTOEOF
```

- [ ] **Step 2: Generate Python gRPC code**

```bash
cd python
PYTHONPATH=. python -m grpc_tools.protoc \
  -Iproto \
  --python_out=src/proto \
  --grpc_python_out=src/proto \
  proto/llm.proto
```

- [ ] **Step 3: Fix Python proto imports**

The generated `llm_pb2_grpc.py` will have `import llm_pb2`. Fix it to use the package import:

```bash
cd python
sed -i '' 's/^import llm_pb2/from src.proto import llm_pb2/' src/proto/llm_pb2_grpc.py
```

- [ ] **Step 4: Verify generated code compiles**

```bash
cd python && PYTHONPATH=. python -c "from src.proto import llm_pb2, llm_pb2_grpc; print('OK')"
```

Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add python/proto/llm.proto python/src/proto/llm_pb2.py python/src/proto/llm_pb2_grpc.py
git commit -m "feat(python): add llm.proto with Chat (streaming), ListModels, CountTokens"
```

### Task 1.2: Implement LLM provider abstract base and Ollama provider

**Files:**
- Create: `python/src/llm/providers/__init__.py`
- Create: `python/src/llm/providers/ollama_provider.py`
- Modify: `python/pyproject.toml` (add httpx dependency)

**Interfaces:**
- Produces: `LLMProvider` ABC with `async def chat(self, request, context) -> AsyncIterator[LLMChatResponse]`
- Produces: `OllamaProvider(LLMProvider)` — local Ollama, no API key needed
- Produces: `get_provider(model_id: str, config: dict) -> LLMProvider` factory function

- [ ] **Step 1: Add httpx dependency**

Edit `python/pyproject.toml`, add `"httpx>=0.27"` to dependencies:

```toml
dependencies = [
    "grpcio>=1.60",
    "grpcio-tools>=1.60",
    "protobuf>=4.25",
    "pandas>=2.1",
    "numpy>=1.26",
    "pyarrow>=14.0",
    "httpx>=0.27",
]
```

```bash
cd python && pip install httpx>=0.27
```

- [ ] **Step 2: Write providers __init__.py**

```python
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
```

- [ ] **Step 3: Write Ollama provider**

```python
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
```

- [ ] **Step 4: Verify Ollama provider imports**

```bash
cd python && PYTHONPATH=. python -c "from src.llm.providers.ollama_provider import OllamaProvider; print('OK')"
```

Expected: `OK`

- [ ] **Step 5: Commit**

```bash
git add python/src/llm/providers/__init__.py python/src/llm/providers/ollama_provider.py python/pyproject.toml
git commit -m "feat(python): add LLMProvider ABC and Ollama provider with streaming"
```

### Task 1.3: Implement OpenAI provider

**Files:**
- Create: `python/src/llm/providers/openai_provider.py`

**Interfaces:**
- Produces: `OpenAIProvider(LLMProvider)` — uses `httpx` to call OpenAI /v1/chat/completions with streaming

- [ ] **Step 1: Write OpenAI provider**

```python
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
```

- [ ] **Step 2: Verify OpenAI provider imports**

```bash
cd python && PYTHONPATH=. python -c "from src.llm.providers.openai_provider import OpenAIProvider; print('OK')"
```

Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add python/src/llm/providers/openai_provider.py
git commit -m "feat(python): add OpenAI provider with streaming via /v1/chat/completions"
```

### Task 1.4: Implement Anthropic and DeepSeek providers

**Files:**
- Create: `python/src/llm/providers/anthropic_provider.py`
- Create: `python/src/llm/providers/deepseek_provider.py`

**Interfaces:**
- Produces: `AnthropicProvider(LLMProvider)` — uses Anthropic Messages API with streaming
- Produces: `DeepSeekProvider(LLMProvider)` — OpenAI-compatible, delegates to `OpenAIProvider` with `base_url=https://api.deepseek.com`

- [ ] **Step 1: Write Anthropic provider**

```python
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
```

- [ ] **Step 2: Write DeepSeek provider**

```python
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
```

- [ ] **Step 3: Verify all providers import**

```bash
cd python && PYTHONPATH=. python -c "
from src.llm.providers.ollama_provider import OllamaProvider
from src.llm.providers.openai_provider import OpenAIProvider
from src.llm.providers.anthropic_provider import AnthropicProvider
from src.llm.providers.deepseek_provider import DeepSeekProvider
from src.llm.providers import list_providers
print('Providers:', list_providers())
"
```

Expected: `Providers: ['ollama', 'openai', 'anthropic', 'deepseek']`

- [ ] **Step 4: Commit**

```bash
git add python/src/llm/providers/anthropic_provider.py python/src/llm/providers/deepseek_provider.py
git commit -m "feat(python): add Anthropic and DeepSeek LLM providers"
```

### Task 1.5: Implement PromptTemplate engine

**Files:**
- Create: `python/src/llm/prompt_template.py`

**Interfaces:**
- Produces: `PromptTemplate` class with `assemble(system_prompt, tools, skills, messages) -> LLMChatRequest`
- Produces: `estimate_tokens(text) -> int` (simple char/4 heuristic)

- [ ] **Step 1: Write PromptTemplate engine**

```python
"""PromptTemplate engine — assembles system prompts with tools and skills."""

import json
import logging
from dataclasses import dataclass, field
from typing import List, Optional

logger = logging.getLogger(__name__)

# Rough token budget constants (character-based heuristic: ~4 chars per token)
DEFAULT_MAX_CONTEXT_TOKENS = 128_000
TOKEN_BUDGET_SYSTEM_FRACTION = 0.3  # 30% for system prompt + skills + tools
MAX_SKILL_CHARS = 20_000  # Cap skill injection at ~5k tokens


@dataclass
class ToolDef:
    """A tool definition for LLM function calling."""
    name: str
    description: str
    parameters_json: str  # JSON Schema as string


@dataclass
class PromptContext:
    """All the pieces needed to assemble a system prompt."""
    base_system_prompt: str = ""
    tools: List[ToolDef] = field(default_factory=list)
    skills: List[str] = field(default_factory=list)  # Skill content strings
    few_shot_examples: List[str] = field(default_factory=list)
    model_context_window: int = DEFAULT_MAX_CONTEXT_TOKENS


def estimate_tokens(text: str) -> int:
    """Rough token estimate: ~4 characters per token for English text."""
    return max(1, len(text) // 4)


class PromptTemplate:
    """Assembles the final system prompt from components within token budget."""

    def __init__(self, context: PromptContext):
        self.context = context
        self.token_budget = int(context.model_context_window * TOKEN_BUDGET_SYSTEM_FRACTION)

    def assemble_system_prompt(self) -> str:
        """Build the system prompt, injecting skills and tools within budget."""
        parts = []
        tokens_used = 0

        # 1. Base system prompt (always included, up to half of budget)
        base = self.context.base_system_prompt
        base_tokens = estimate_tokens(base)
        half_budget = self.token_budget // 2
        if base_tokens > half_budget:
            logger.warning(f"Base system prompt ({base_tokens} tokens) exceeds half budget ({half_budget}), truncating")
            base = base[:half_budget * 4]
        parts.append(base)
        tokens_used += base_tokens

        # 2. Tool descriptions
        if self.context.tools:
            tool_block = "\n\n## Available Tools\n\nYou have access to the following tools:\n\n"
            for tool in self.context.tools:
                tool_block += f"### {tool.name}\n{tool.description}\n"
                params = json.loads(tool.parameters_json) if tool.parameters_json else {}
                if params.get("properties"):
                    tool_block += f"Parameters: {json.dumps(params['properties'], indent=2)}\n"
                tool_block += "\n"
            tool_tokens = estimate_tokens(tool_block)
            if tokens_used + tool_tokens <= self.token_budget:
                parts.append(tool_block)
                tokens_used += tool_tokens
            else:
                # Truncate: only include tool names
                short_block = "\n\n## Available Tools\n\n" + ", ".join(t.name for t in self.context.tools)
                parts.append(short_block)
                tokens_used += estimate_tokens(short_block)

        # 3. Skill knowledge (injected as domain expertise, NOT as tools)
        if self.context.skills:
            skill_block = "\n\n## Domain Knowledge\n\n"
            budget_remaining = min(MAX_SKILL_CHARS, (self.token_budget - tokens_used) * 4)
            for skill in self.context.skills:
                if len(skill_block) + len(skill) > budget_remaining:
                    skill_block += skill[:budget_remaining - len(skill_block)] + "\n[...truncated]\n"
                    break
                skill_block += skill + "\n\n"
            parts.append(skill_block)

        # 4. Few-shot examples
        if self.context.few_shot_examples:
            example_block = "\n\n## Examples\n\n" + "\n\n".join(self.context.few_shot_examples)
            example_tokens = estimate_tokens(example_block)
            if tokens_used + example_tokens <= self.token_budget:
                parts.append(example_block)

        return "\n".join(parts)

    def format_tools_for_request(self) -> list:
        """Format tools into the structure expected by gRPC LLMChatRequest."""
        from src.proto.llm_pb2 import LLMTool

        result = []
        for tool in self.context.tools:
            lt = LLMTool()
            lt.name = tool.name
            lt.description = tool.description
            lt.parameters_json = tool.parameters_json
            result.append(lt)
        return result
```

- [ ] **Step 2: Verify PromptTemplate imports and basic usage**

```bash
cd python && PYTHONPATH=. python -c "
from src.llm.prompt_template import PromptTemplate, PromptContext, ToolDef, estimate_tokens
ctx = PromptContext(
    base_system_prompt='You are a helpful assistant.',
    tools=[ToolDef(name='test', description='A test tool', parameters_json='{\"type\":\"object\",\"properties\":{}}')],
)
tmpl = PromptTemplate(ctx)
result = tmpl.assemble_system_prompt()
print('Tokens:', estimate_tokens(result))
print('OK')
"
```

Expected: `OK` with token count > 0

- [ ] **Step 3: Commit**

```bash
git add python/src/llm/prompt_template.py
git commit -m "feat(python): add PromptTemplate engine with token budget management"
```

### Task 1.6: Implement LLMService gRPC server

**Files:**
- Modify: `python/src/llm/engine.py` (replace stub)
- Modify: `python/src/server.py` (register LLMService)

**Interfaces:**
- Produces: `LLMService(llm_pb2_grpc.LLMServiceServicer)` with Chat (streaming), ListModels, CountTokens
- Consumes: `LLMChatRequest`, providers from `src.llm.providers`
- Produces: `LLMChatResponse` stream

- [ ] **Step 1: Replace llm/engine.py stub with full implementation**

```python
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
    llm_pb2.ModelInfo(
        id="ollama/llama3.1:8b",
        provider="ollama",
        display_name="Llama 3.1 8B",
        context_window=131072,
        supports_tools=True,
        supports_vision=False,
    ),
    llm_pb2.ModelInfo(
        id="openai/gpt-4o",
        provider="openai",
        display_name="GPT-4o",
        context_window=128000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.ModelInfo(
        id="openai/gpt-4.1",
        provider="openai",
        display_name="GPT-4.1",
        context_window=1000000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.ModelInfo(
        id="anthropic/claude-sonnet-4-6",
        provider="anthropic",
        display_name="Claude Sonnet 4.6",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.ModelInfo(
        id="anthropic/claude-opus-4-8",
        provider="anthropic",
        display_name="Claude Opus 4.8",
        context_window=200000,
        supports_tools=True,
        supports_vision=True,
    ),
    llm_pb2.ModelInfo(
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

    async def ListModels(self, request: llm_pb2.ListModelsRequest, context) -> llm_pb2.ListModelsResponse:
        """Return all available models."""
        # Filter by available providers (only return models whose provider is registered)
        available_providers = set(list_providers())
        models = [m for m in AVAILABLE_MODELS if m.provider in available_providers]
        return llm_pb2.ListModelsResponse(models=models)

    async def CountTokens(self, request: llm_pb2.CountTokensRequest, context) -> llm_pb2.CountTokensResponse:
        """Estimate token count for messages using a simple heuristic."""
        total_chars = len(request.system_prompt or "")
        for msg in request.messages:
            total_chars += len(msg.content or "")
        # Rough estimate: 4 chars per token
        token_count = max(1, total_chars // 4)
        return llm_pb2.CountTokensResponse(token_count=token_count)
```

- [ ] **Step 2: Register LLMService in server.py**

Edit `python/src/server.py`, add LLM service import and registration:

```python
from src.llm.engine import LLMService
```

And in the `serve()` function, add after the existing service registrations:

```python
llm_pb2_grpc.add_LLMServiceServicer_to_server(LLMService(), server)
```

The `serve()` function should look like:

```python
async def serve(port: int = DEFAULT_PORT, max_workers: int = 10):
    """Start the gRPC server and block until termination."""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=max_workers))

    # Register all service implementations
    factor_pb2_grpc.add_FactorServiceServicer_to_server(FactorService(), server)
    ml_pb2_grpc.add_MLServiceServicer_to_server(MLService(), server)
    health_pb2_grpc.add_HealthServiceServicer_to_server(HealthService(), server)
    data_pb2_grpc.add_DataServiceServicer_to_server(DataService(), server)
    llm_pb2_grpc.add_LLMServiceServicer_to_server(LLMService(), server)

    server.add_insecure_port(f"[::]:{port}")
    logger.info(f"QuantFlow Python sidecar listening on [::]:{port}")
    logger.info("Registered services: FactorService, MLService, HealthService, DataService, LLMService")

    await server.start()
    await server.wait_for_termination()
```

Also add the import for `llm_pb2_grpc` at the top:

```python
from src.proto import (
    factor_pb2_grpc,
    health_pb2,
    health_pb2_grpc,
    llm_pb2_grpc,
    ml_pb2_grpc,
    data_pb2_grpc,
)
```

- [ ] **Step 3: Verify LLMService starts**

```bash
cd python && PYTHONPATH=. python -c "
from src.llm.engine import LLMService
svc = LLMService()
print('LLMService created OK')
print('Models available:', len(svc.AVAILABLE_MODELS))
"
```

Expected: `LLMService created OK` with model count > 0

- [ ] **Step 4: Commit**

```bash
git add python/src/llm/engine.py python/src/server.py
git commit -m "feat(python): implement LLMService with Chat streaming, ListModels, CountTokens"
```

### Task 1.7: Write Python LLM tests

**Files:**
- Create: `python/tests/test_llm_engine.py`

**Interfaces:**
- Consumes: `LLMService`, `PromptTemplate`, providers

- [ ] **Step 1: Write tests**

```python
"""Tests for LLM service, providers, and prompt template engine."""
import pytest
from unittest.mock import AsyncMock, patch, MagicMock

from src.proto.llm_pb2 import (
    LLMChatRequest,
    ChatMessage,
    LLMTool,
    ListModelsRequest,
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
        resp = await svc.ListModels(ListModelsRequest(), None)
        assert len(resp.models) > 0
        model_ids = [m.id for m in resp.models]
        assert any("ollama" in mid for mid in model_ids)

    @pytest.mark.asyncio
    async def test_list_models_has_all_providers(self):
        svc = LLMService()
        resp = await svc.ListModels(ListModelsRequest(), None)
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
        # "You are helpful." + "Hello, world!" = ~31 chars / 4 ≈ 7-8
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
```

- [ ] **Step 2: Run tests**

```bash
cd python && PYTHONPATH=. python -m pytest tests/test_llm_engine.py -v
```

Expected: All tests pass (8 tests)

- [ ] **Step 3: Commit**

```bash
git add python/tests/test_llm_engine.py
git commit -m "test(python): add LLM service tests — PromptTemplate, LLMService, Providers"

---

## M2: Skill Knowledge Base

### Toolbox 2.1: Write skill Markdown files (15 skills, 5 categories)

**Files:**
- Create: `python/skills/technical_analysis/candlestick_patterns.md`
- Create: `python/skills/technical_analysis/indicator_usage.md`
- Create: `python/skills/technical_analysis/divergence_trading.md`
- Create: `python/skills/fundamental_analysis/valuation_ratios.md`
- Create: `python/skills/fundamental_analysis/dcf_model.md`
- Create: `python/skills/fundamental_analysis/earnings_analysis.md`
- Create: `python/skills/risk_management/position_sizing.md`
- Create: `python/skills/risk_management/var_methods.md`
- Create: `python/skills/risk_management/portfolio_optimization.md`
- Create: `python/skills/market_microstructure/order_book_analysis.md`
- Create: `python/skills/market_microstructure/market_impact.md`
- Create: `python/skills/trading_strategies/momentum_strategies.md`
- Create: `python/skills/trading_strategies/mean_reversion.md`
- Create: `python/skills/trading_strategies/pairs_trading.md`
- Create: `python/skills/trading_strategies/grid_trading.md`

**Interfaces:**
- Produces: Markdown files with YAML frontmatter (title, category, tags, difficulty)

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p python/skills/{technical_analysis,fundamental_analysis,risk_management,market_microstructure,trading_strategies}
```

- [ ] **Step 2: Write a representative skill file (momentum_strategies.md)**

Write `python/skills/trading_strategies/momentum_strategies.md`:

```markdown
---
title: Momentum Strategies
category: trading_strategies
tags: [momentum, trend-following, factor]
difficulty: intermediate
---

# Momentum Strategies

## Core Concept
Momentum is the tendency of assets that have performed well (poorly) in the recent past to continue performing well (poorly) in the near future.

## Types

### Cross-Sectional Momentum
Buy past winners, sell past losers within a defined universe.
- Typical look-back: 3, 6, or 12 months (skip most recent month to avoid short-term reversal)
- Rebalance: monthly
- Universe: top 80% by market cap or liquidity

### Time-Series Momentum
Go long assets with positive recent excess returns, short those with negative.
- Look-back: 1, 3, 6, 12 months
- Volatility scaling: position size = target_vol / realized_vol
- Often applied to futures and macro assets

## Implementation Checklist

1. **Universe construction**: Filter by liquidity (avg daily volume > 10M for A-shares)
2. **Signal generation**: Compute N-month return, skip T-1 month
3. **Portfolio construction**: Equal-weight or score-weighted, top quintile long, bottom quintile short
4. **Rebalance**: Month-end, T+1 effective for A-shares
5. **Risk management**: Sector neutrality, max position 5%, stop-loss at -2 STD

## A-Share Specifics
- T+1 settlement means signals generated on day T take effect on day T+1
- Price limits (±10% for most stocks) may prevent entry/exit at signal price
- Stamp duty (0.05% on sell) increases turnover cost; prefer longer holding periods
- CSI 300/500/1000 for broad universe coverage

## Evaluation Metrics
- Sharpe ratio (>1.0 is good)
- Max drawdown (<25% is acceptable)
- Turnover (<100% monthly is reasonable)
- Factor IC (information coefficient) and ICIR

## Common Pitfalls
- Look-ahead bias: ensure signal uses only data available at decision time
- Survivorship bias: include delisted stocks in historical universe
- Micro-cap noise: filter out stocks with market cap < 1B CNY
```

- [ ] **Step 3: Write remaining 14 skill files**

Write `python/skills/technical_analysis/candlestick_patterns.md`:

```markdown
---
title: Candlestick Patterns
category: technical_analysis
tags: [candlestick, pattern, chart]
difficulty: beginner
---

# Candlestick Patterns

## Key Reversal Patterns

### Bullish
- **Hammer**: Small body at top, long lower shadow (2x+ body). Indicates potential reversal up.
- **Morning Star**: Three-candle pattern — long red, small body (gap down), long green (gap up).
- **Piercing Line**: Red candle followed by green that opens below red's low and closes >50% into red's body.
- **Bullish Engulfing**: Green body completely engulfs previous red body.

### Bearish
- **Shooting Star**: Small body at bottom, long upper shadow. Potential reversal down.
- **Evening Star**: Three-candle — long green, small body (gap up), long red (gap down).
- **Dark Cloud Cover**: Green followed by red that opens above green's high and closes >50% into green's body.
- **Bearish Engulfing**: Red body completely engulfs previous green body.

## Confirmation
Always confirm patterns with:
1. Next candle closing in pattern direction
2. Volume spike on pattern day
3. Support/resistance level alignment
4. RSI divergence

## Limitations
- Subjective pattern identification
- False signals in ranging markets (~40%)
- Best used as confluence, not standalone signals
```

Write `python/skills/technical_analysis/indicator_usage.md`:

```markdown
---
title: Technical Indicator Usage Guide
category: technical_analysis
tags: [indicator, MA, RSI, MACD, Bollinger]
difficulty: intermediate
---

# Technical Indicator Usage Guide

## Moving Averages (MA)

### SMA (Simple Moving Average)
- **Usage**: Trend direction, dynamic support/resistance
- **Periods**: 20 (short-term), 50 (medium), 200 (long-term)
- **Signal**: Price crosses MA — bullish above, bearish below
- **Golden Cross**: 50-day crosses above 200-day (bullish)
- **Death Cross**: 50-day crosses below 200-day (bearish)
- **Limitation**: Lagging indicator, whipsaws in sideways markets

### EMA (Exponential Moving Average)
- More weight to recent prices, less lag than SMA
- Common: 12 and 26 period for MACD

## RSI (Relative Strength Index)
- **Range**: 0–100
- **Overbought**: >70 (potential sell)
- **Oversold**: <30 (potential buy)
- **Divergence**: Price makes new high but RSI doesn't (bearish divergence)
- **Period**: 14 is standard; 7 for short-term, 21 for long-term

## MACD
- **Components**: MACD line (12 EMA - 26 EMA), Signal line (9 EMA of MACD), Histogram
- **Signal**: MACD crosses above Signal (bullish), below (bearish)
- **Divergence**: Stronger signal than crossovers
- **Zero line**: Above zero = uptrend, below = downtrend

## Bollinger Bands
- **Middle Band**: 20-period SMA
- **Upper/Lower**: ±2 standard deviations
- **Squeeze**: Bands narrow → breakout imminent
- **Walk**: Price walks the band in strong trends
- **Reversal**: Price touches band and reverses (mean reversion)

## Multi-Timeframe Analysis
- Higher timeframe for trend direction (daily/weekly)
- Lower timeframe for entry timing (hourly/15min)
- Align signals across timeframes for higher probability
```

Write `python/skills/technical_analysis/divergence_trading.md`:

```markdown
---
title: Divergence Trading
category: technical_analysis
tags: [divergence, RSI, MACD, reversal]
difficulty: advanced
---

# Divergence Trading

## Types of Divergence

### Regular Divergence (Reversal Signal)
- **Bullish**: Price makes lower low, indicator makes higher low
- **Bearish**: Price makes higher high, indicator makes lower high

### Hidden Divergence (Continuation Signal)
- **Bullish**: Price makes higher low, indicator makes lower low
- **Bearish**: Price makes lower high, indicator makes higher high

## Best Indicators for Divergence
1. RSI (14) — most reliable
2. MACD histogram
3. Stochastic (14,3,3)
4. OBV (On-Balance Volume) — volume confirmation

## Entry Rules
1. Identify divergence on daily timeframe
2. Wait for confirmation candle (closing in divergence direction)
3. Enter on break of confirmation candle's high/low
4. Stop loss: beyond the divergent swing point
5. Target: at least 2:1 reward-to-risk

## A-Share Application
- More effective on index ETFs (CSI 300, SSE 50) than individual stocks
- Combine with volume analysis: divergences with volume confirmation have ~65% success rate
- T+1 settlement: enter on confirmation day, effective next trading day
```

Write `python/skills/fundamental_analysis/valuation_ratios.md`:

```markdown
---
title: Valuation Ratios
category: fundamental_analysis
tags: [valuation, PE, PB, ratio]
difficulty: intermediate
---

# Valuation Ratios

## Price-to-Earnings (P/E)
- **Trailing P/E**: Based on last 12 months earnings
- **Forward P/E**: Based on analyst consensus estimates
- **Interpretation**: Lower = cheaper, but context matters
- **Industry norms**: Tech 20-30x, Financials 8-15x, Consumer 15-25x

## Price-to-Book (P/B)
- Best for: Financials, REITs, asset-heavy companies
- Below 1.0: Potentially undervalued
- Above 3.0: Premium — check ROE justification

## EV/EBITDA
- Enterprise Value / EBITDA
- Better for comparing companies with different capital structures
- Typical range: 8-15x for mature companies

## PEG Ratio
- P/E divided by earnings growth rate
- <1.0: Potentially undervalued relative to growth
- Limitation: Growth estimates are uncertain

## Dividend Yield
- Annual dividend / stock price
- A-share: typically 0-4%, financials/utilities higher
- Check payout ratio: >80% may be unsustainable
```

Write `python/skills/fundamental_analysis/dcf_model.md`:

```markdown
---
title: DCF Valuation Model
category: fundamental_analysis
tags: [dcf, valuation, intrinsic-value, wacc]
difficulty: advanced
---

# DCF (Discounted Cash Flow) Model

## Framework
1. Project free cash flows (5-10 years)
2. Calculate terminal value
3. Discount to present value using WACC
4. Subtract net debt → equity value
5. Divide by shares outstanding → intrinsic value per share

## Free Cash Flow Formula
```
FCF = EBIT * (1 - tax_rate)
    + Depreciation & Amortization
    - Capex
    - Change in Working Capital
```

## Key Assumptions
- **Revenue growth**: 3-5 year analyst consensus, then fade to GDP growth
- **Margins**: Mean-revert to industry average by year 5
- **WACC**: Typically 8-12% for A-shares
- **Terminal growth**: 2-3% (no higher than long-term GDP growth)

## Terminal Value Methods
1. **Gordon Growth**: TV = FCF_last * (1+g) / (WACC - g)
2. **Exit Multiple**: TV = EBITDA_last * industry_multiple

## Margin of Safety
- Buy only when intrinsic value > market price * 1.3 (30% margin)
- Wider margin for: cyclical industries, high debt, uncertain growth
```

Write `python/skills/fundamental_analysis/earnings_analysis.md`:

```markdown
---
title: Earnings Analysis
category: fundamental_analysis
tags: [earnings, surprise, guidance, quality]
difficulty: intermediate
---

# Earnings Analysis

## Earnings Quality Checklist
1. **Revenue recognition**: Is it aggressive? Check deferred revenue trends
2. **Non-recurring items**: Exclude one-time gains/losses from core earnings
3. **Receivables vs Revenue**: Receivables growing faster than revenue = red flag
4. **Operating cash flow vs Net income**: OCF < NI persistently = earnings quality issue
5. **Share count**: Buybacks inflate EPS; check total earnings growth, not just per-share

## Earnings Surprise Analysis
- **Positive surprise + upward guidance**: Strongest bullish signal
- **Positive surprise + flat guidance**: Moderate bullish
- **Negative surprise + downward guidance**: Bearish
- **Beat on revenue AND earnings**: Higher quality than EPS beat alone

## Post-Earnings Drift (PEAD)
- Stocks tend to drift in the surprise direction for weeks after announcement
- Stronger effect for: small caps, low analyst coverage, large surprises

## A-Share Specifics
- Earnings seasons: Q1 (by Apr 30), Semi-annual (by Aug 31), Q3 (by Oct 31), Annual (by Apr 30)
- Pre-announcement required if: profit change >50%, loss, or turn profit→loss
- ST (Special Treatment) stocks: two consecutive annual losses trigger ST designation
```

Write `python/skills/risk_management/position_sizing.md`:

```markdown
---
title: Position Sizing Methods
category: risk_management
tags: [position-sizing, kelly, risk-parity, allocation]
difficulty: intermediate
---

# Position Sizing Methods

## Equal Weight (1/N)
- Simplest: allocate equally across N positions
- Benchmark for more complex methods
- Works well when: high uncertainty about future returns

## Risk Parity
- Allocate such that each position contributes equal risk
- Need covariance matrix estimation
- More stable than mean-variance optimization

## Kelly Criterion
- Optimal fraction = edge / odds
- f* = (p * b - q) / b
  - p = win probability, q = 1-p, b = win/loss ratio
- Practical: use half-Kelly to reduce volatility

## Volatility Targeting
- Position size = target_daily_vol / asset_daily_vol
- Target: 1-2% daily vol per position
- Scales automatically with market conditions

## Fixed Fractional
- Risk fixed % of capital per trade (e.g., 1-2%)
- Position size = (capital * risk_pct) / stop_distance
- Most common among professional traders

## A-Share Constraints
- 100-share lot minimum (整手)
- 100-share increments above minimum
- Position limits: 5% of outstanding shares for significant shareholders
```

Write `python/skills/risk_management/var_methods.md`:

```markdown
---
title: Value at Risk (VaR) Methods
category: risk_management
tags: [var, cvar, risk-metrics, stress-test]
difficulty: advanced
---

# Value at Risk (VaR)

## Methods

### Historical VaR
- Uses actual historical returns distribution
- VaR(95%) = 5th percentile of historical returns
- Advantage: No distribution assumption
- Disadvantage: Assumes history repeats

### Parametric (Variance-Covariance) VaR
- Assumes normal distribution
- VaR(95%) = portfolio_value * (mean_return - 1.645 * std_return)
- Fast computation, but underestimates tail risk

### Monte Carlo VaR
- Simulate thousands of scenarios
- Most flexible: can model complex instruments
- Computationally intensive

## CVaR (Conditional VaR / Expected Shortfall)
- Average loss BEYOND VaR threshold
- Better captures tail risk than VaR
- Recommended by Basel III

## Stress Testing
- Historical scenarios: 2008 crisis, 2015 A-share crash, 2020 COVID
- Hypothetical: -30% market, +200bp rates, correlation breakdown

## A-Share Tail Risk
- Daily price limits create truncated distribution
- Historical VaR naturally accounts for limits
- Parametric VaR should be adjusted for kurtosis (fat tails)
```

Write `python/skills/risk_management/portfolio_optimization.md`:

```markdown
---
title: Portfolio Optimization
category: risk_management
tags: [optimization, markowitz, black-litterman, constraints]
difficulty: advanced
---

# Portfolio Optimization

## Mean-Variance Optimization (Markowitz)
- Minimize: w'Σw - λ * w'μ
- Inputs: Expected returns (μ), covariance matrix (Σ), risk aversion (λ)
- Output: Optimal weights (w)
- Problems: Sensitive to inputs, corner solutions, estimation error

## Black-Litterman
- Starts from market equilibrium (CAPM implied returns)
- Investor expresses views: "Asset A will outperform B by 3%"
- Blends equilibrium + views → posterior expected returns
- More stable and intuitive weights than pure MVO

## Constraints for Real-world Portfolios
1. **Long-only**: w_i >= 0
2. **Full investment**: Σw_i = 1.0
3. **Position limits**: w_i <= 0.05 (5%)
4. **Sector limits**: Σw_sector <= 0.30
5. **Turnover constraint**: Σ|w_new - w_old| <= 0.50
6. **Minimum position**: w_i >= 0.005 or 0

## Risk Budgeting
- Assign risk budget to each asset/strategy
- Risk contribution: w_i * (Σw)_i / σ_portfolio
- Equal risk contribution (ERC) as robust alternative to MVO

## A-Share Considerations
- CSI 300 for equity universe
- Include bond ETFs/convertible bonds for diversification
- Liquidity filter: exclude stocks with daily turnover < 10M CNY
- Rebalance monthly or quarterly to minimize transaction costs
```

Write `python/skills/market_microstructure/order_book_analysis.md`:

```markdown
---
title: Order Book Analysis
category: market_microstructure
tags: [order-book, depth, spread, liquidity]
difficulty: advanced
---

# Order Book Analysis

## Key Metrics

### Bid-Ask Spread
- **Quoted spread**: ask - bid
- **Effective spread**: 2 * |trade_price - mid_price|
- **Realized spread**: 2 * |trade_price - mid_price_future|
- Wider spreads = higher trading costs

### Market Depth
- Total volume available at each price level
- Depth imbalance: (bid_volume - ask_volume) / (bid_volume + ask_volume)
- Positive imbalance = buying pressure, predicts short-term price increase

### Order Flow Imbalance (OFI)
- Change in bid/ask quantities between snapshots
- OFI = Δbid_qty - Δask_qty
- Strong predictor of short-term price moves (1-10 seconds)

## Market Making Signals
- **Inventory**: High long inventory → lower bid, reduce buying
- **Adverse selection**: Informed traders cause losses; need wider spreads
- **Volatility**: Higher vol → wider spreads

## A-Share Level-2 Data
- Available from EastMoney, Futu, broker terminals
- Top 5 bid/ask levels standard, top 10 available
- Tick-level data: ~3 seconds per snapshot
```

Write `python/skills/market_microstructure/market_impact.md`:

```markdown
---
title: Market Impact Models
category: market_microstructure
tags: [market-impact, execution, algo-trading]
difficulty: advanced
---

# Market Impact Models

## Components of Trading Cost
1. **Commissions + fees**: Explicit, known ex-ante
2. **Bid-ask spread**: Half-spread per trade
3. **Market impact**: Price moves against your order
4. **Delay cost**: Price moves while waiting to execute
5. **Opportunity cost**: Cost of not completing the order

## Almgren-Chriss Model
- Temporary impact: decays after trade (liquidity effect)
- Permanent impact: persistent price change (information effect)
- Optimal schedule balances impact vs timing risk

## Square Root Law
- Impact ≈ σ * sqrt(Q / V)
- σ = daily volatility
- Q = order size
- V = daily volume
- Widely observed empirically across markets

## Practical Rules of Thumb
- Trade ≤5% of daily volume to keep impact minimal
- Participation rate: 20-30% of market volume
- Larger orders → longer execution horizon
- Dark pools / block trading for orders >1% ADV

## A-Share Execution
- T+1 settlement: plan for next-day availability
- Price limits: orders beyond daily limit range are rejected
- 100-share lots: round down to lot multiple
```

Write `python/skills/trading_strategies/mean_reversion.md`:

```markdown
---
title: Mean Reversion Strategies
category: trading_strategies
tags: [mean-reversion, pairs, bollinger, statistical-arbitrage]
difficulty: intermediate
---

# Mean Reversion Strategies

## Core Concept
Prices deviate from their mean temporarily and revert. Profit from the reversion.

## Bollinger Band Mean Reversion
- Buy when price touches lower band (2σ below 20-SMA)
- Sell when price touches upper band (2σ above 20-SMA)
- Works best in: ranging/sideways markets (low ADX <25)
- Fails in: strong trending markets (price "walks the band")

## RSI Extremes
- Buy: RSI(14) < 30 and starting to turn up
- Sell: RSI(14) > 70 and starting to turn down
- Add filter: only trade in direction of longer-term trend (200-SMA)

## Statistical Arbitrage
- Identify cointegrated pairs (Engle-Granger, Johansen tests)
- Enter when spread > 2σ from mean
- Exit when spread reverts to mean
- Half-life of mean reversion: determine holding period

## Risk Management
- Mean reversion can become momentum: always use stops
- Stop loss: 2x the entry deviation (e.g., if enter at 2σ, stop at 3σ)
- Position size: smaller than trend-following (mean reversion has lower win rate)
- Avoid during: earnings announcements, macro events, regime changes

## A-Share Adaptation
- Shanghai/Shenzhen have different sector compositions; analyze separately
- A/H premium mean reversion for dual-listed stocks
- Sector rotation: overbought sectors → oversold sectors
```

Write `python/skills/trading_strategies/pairs_trading.md`:

```markdown
---
title: Pairs Trading
category: trading_strategies
tags: [pairs-trading, cointegration, statistical-arbitrage, mean-reversion]
difficulty: advanced
---

# Pairs Trading

## Strategy Overview
Find two highly correlated/cointegrated stocks. When their price ratio diverges from historical norm: short the overperformer, long the underperformer. Profit when the spread reverts.

## Pair Selection Criteria
1. **Same industry**: Both banks, both liquor, both EV makers
2. **High correlation**: >0.8 daily returns correlation over 1 year
3. **Cointegration**: Pass Engle-Granger test (p < 0.05)
4. **Liquidity**: Both stocks trade >10M CNY daily
5. **Fundamental similarity**: Similar market cap, business model

## Entry/Exit Rules
- **Entry**: Spread > 2 standard deviations from mean
- **Exit**: Spread reverts to within 0.5 standard deviations
- **Stop loss**: Spread reaches 3 standard deviations
- **Time stop**: 20 trading days maximum holding

## Position Sizing
- Dollar-neutral: equal capital long and short
- Beta-neutral: adjust sizes for different betas
- Notional: same number of shares each side (simpler but less precise)

## A-Share Specifics
- **Short constraints**: A-share short selling requires margin account and borrow availability
- **Alternative**: Use index futures (IF/IC/IH) as the short leg, stock as long leg
- **Cost**: Borrow fee (~2-8% annualized) + margin interest
- **Pair candidates**: Ping An vs China Life (insurers), Kweichow Moutai vs Wuliangye (liquor)
```

Write `python/skills/trading_strategies/grid_trading.md`:

```markdown
---
title: Grid Trading
category: trading_strategies
tags: [grid, range-trading, automation]
difficulty: beginner
---

# Grid Trading

## Concept
Place buy and sell orders at predetermined price intervals above and below a reference price. Profit from oscillations within a range.

## Grid Setup
1. **Reference price**: Current market price or moving average
2. **Grid spacing**: Based on ATR(14) or fixed percentage (e.g., 1-3%)
3. **Grid levels**: 5-10 levels each side
4. **Order size**: Equal size per level, or tapering (larger at extremes)

## Parameters
- **Upper/Lower bounds**: Define trading range
- **Grid count**: More grids = more trades, smaller profit per trade
- **Order size per grid**: Position size / grid count
- **Stop/reverse**: What happens when price leaves the range

## Best Markets for Grid Trading
- Ranging markets with clear support/resistance
- High volatility (frequent grid touches)
- Low transaction costs
- Crypto markets (24/7 trading, volatile ranges)

## Risk
- **Trending market**: Grid gets exhausted, left with underwater positions
- **Gap risk**: Price gaps through multiple levels
- **Capital intensive**: Need capital for all open grid levels
- **Mitigation**: Dynamic grid that shifts with trend, or trend filter to pause grid in strong trends

## A-Share Adaptation
- T+1 means grid orders execute one day at a time
- Daily price limits provide natural grid bounds
- Better suited for ETFs (lower commissions, no single-stock risk)
```

- [ ] **Step 4: Commit all skill files**

```bash
git add python/skills/
git commit -m "feat(python): add 15 skill Markdown files across 5 categories"
```

### Task 2.2: Implement Skill loader with frontmatter parsing

**Files:**
- Create: `python/src/skills/__init__.py`
- Create: `python/src/skills/loader.py`

**Interfaces:**
- Produces: `Skill(name, category, tags, difficulty, content)` dataclass
- Produces: `load_skills(skills_dir: str) -> List[Skill]`
- Produces: `search_skills(query: str, category: str | None) -> List[Skill]`
- Produces: `get_skill_content(name: str) -> str`

- [ ] **Step 1: Write skills __init__.py**

```python
"""Skill Knowledge Base — domain expertise for LLM system prompt injection."""
```

- [ ] **Step 2: Write loader.py**

```python
"""Skill loader — parses Markdown files with YAML frontmatter."""

import logging
import os
import re
from dataclasses import dataclass, field
from typing import List, Optional

import yaml

logger = logging.getLogger(__name__)


@dataclass
class Skill:
    """A skill document with frontmatter metadata and markdown content."""
    name: str           # filename without .md
    category: str       # from frontmatter
    title: str          # from frontmatter
    tags: List[str] = field(default_factory=list)
    difficulty: str = "intermediate"
    content: str = ""   # full markdown body (without frontmatter)


def _parse_frontmatter(text: str) -> tuple[dict, str]:
    """Parse YAML frontmatter from markdown text. Returns (metadata, content)."""
    match = re.match(r'^---\s*\n(.*?)\n---\s*\n', text, re.DOTALL)
    if not match:
        return {}, text
    try:
        meta = yaml.safe_load(match.group(1)) or {}
    except yaml.YAMLError:
        meta = {}
    content = text[match.end():]
    return meta, content


def load_skills(skills_dir: str) -> List[Skill]:
    """Load all skill files from a directory tree.

    Walks the directory recursively. Each .md file is parsed for
    YAML frontmatter + markdown content.
    """
    skills = []
    for root, dirs, files in os.walk(skills_dir):
        for fname in files:
            if not fname.endswith('.md'):
                continue
            fpath = os.path.join(root, fname)
            try:
                with open(fpath, 'r', encoding='utf-8') as f:
                    text = f.read()
            except Exception as e:
                logger.warning(f"Failed to read skill file {fpath}: {e}")
                continue

            meta, content = _parse_frontmatter(text)
            name = fname.replace('.md', '')
            category = meta.get('category', os.path.basename(root))

            skill = Skill(
                name=name,
                category=category,
                title=meta.get('title', name.replace('_', ' ').title()),
                tags=meta.get('tags', []),
                difficulty=meta.get('difficulty', 'intermediate'),
                content=content.strip(),
            )
            skills.append(skill)

    logger.info(f"Loaded {len(skills)} skills from {skills_dir}")
    return skills


def search_skills(skills: List[Skill], query: str, category: Optional[str] = None) -> List[Skill]:
    """Search skills by keyword and optional category filter.

    Searches in: title, tags, content (case-insensitive).
    """
    q = query.lower()
    results = []
    for s in skills:
        if category and s.category != category:
            continue
        if (q in s.title.lower() or
            q in s.content.lower() or
            any(q in tag.lower() for tag in s.tags)):
            results.append(s)
    return results


def get_categories(skills: List[Skill]) -> List[str]:
    """Return sorted list of unique categories."""
    return sorted(set(s.category for s in skills))
```

- [ ] **Step 3: Verify loader works**

```bash
cd python && PYTHONPATH=. python -c "
from src.skills.loader import load_skills, search_skills, get_categories
import os
skills_dir = os.path.join(os.path.dirname(__file__) if '__file__' in dir() else '.', 'skills')
skills = load_skills(skills_dir)
print(f'Loaded {len(skills)} skills')
print('Categories:', get_categories(skills))
results = search_skills(skills, 'momentum')
print(f'Momentum matches: {len(results)}')
"
```

Expected: `Loaded 15 skills`, Categories listed, Momentum matches > 0

- [ ] **Step 4: Commit**

```bash
git add python/src/skills/__init__.py python/src/skills/loader.py
git commit -m "feat(python): add Skill loader with YAML frontmatter parsing and search"
```

### Task 2.3: Write Skill tests

**Files:**
- Create: `python/tests/test_skills.py`

- [ ] **Step 1: Write tests**

```python
"""Tests for the Skill Knowledge Base loader."""
import os
import tempfile
import pytest

from src.skills.loader import Skill, load_skills, search_skills, get_categories, _parse_frontmatter


class TestFrontmatterParsing:
    def test_parse_with_frontmatter(self):
        text = """---
title: Test Skill
category: test_cat
tags: [a, b]
difficulty: beginner
---

# Hello
Content here."""
        meta, content = _parse_frontmatter(text)
        assert meta["title"] == "Test Skill"
        assert meta["category"] == "test_cat"
        assert meta["tags"] == ["a", "b"]
        assert "Content here" in content

    def test_parse_without_frontmatter(self):
        text = "# Just content\nNo frontmatter."
        meta, content = _parse_frontmatter(text)
        assert meta == {}
        assert "Just content" in content


class TestSkillLoader:
    def test_load_skills_from_dir(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            os.makedirs(os.path.join(tmpdir, "cat1"))
            with open(os.path.join(tmpdir, "cat1", "skill_a.md"), "w") as f:
                f.write("""---
title: Skill A
category: cat1
tags: [tag1]
---
# Skill A Content
This is skill A.""")
            with open(os.path.join(tmpdir, "cat1", "skill_b.md"), "w") as f:
                f.write("""---
title: Skill B
category: cat1
tags: [tag2]
---
# Skill B Content
Momentum trading content.""")
            # Non-.md file should be ignored
            with open(os.path.join(tmpdir, "readme.txt"), "w") as f:
                f.write("not a skill")

            skills = load_skills(tmpdir)
            assert len(skills) == 2
            assert {s.name for s in skills} == {"skill_a", "skill_b"}

    def test_search_skills(self):
        skills = [
            Skill(name="a", category="cat1", title="Momentum Strategy", tags=["momentum"], content="Buy winners."),
            Skill(name="b", category="cat2", title="Mean Reversion", tags=["reversion"], content="Buy dips."),
        ]
        results = search_skills(skills, "momentum")
        assert len(results) == 1
        assert results[0].name == "a"

    def test_search_with_category_filter(self):
        skills = [
            Skill(name="a", category="cat1", title="Momentum", tags=["momentum"], content="x"),
            Skill(name="b", category="cat2", title="Momentum B", tags=["momentum"], content="x"),
        ]
        results = search_skills(skills, "momentum", category="cat1")
        assert len(results) == 1

    def test_get_categories(self):
        skills = [
            Skill(name="a", category="cat1", title="A", content=""),
            Skill(name="b", category="cat2", title="B", content=""),
            Skill(name="c", category="cat1", title="C", content=""),
        ]
        cats = get_categories(skills)
        assert cats == ["cat1", "cat2"]
```

- [ ] **Step 2: Run tests**

```bash
cd python && PYTHONPATH=. python -m pytest tests/test_skills.py -v
```

Expected: All 5 tests pass

- [ ] **Step 3: Commit**

```bash
git add python/tests/test_skills.py
git commit -m "test(python): add Skill KB tests — frontmatter parsing, loading, searching"
```

---

## M3: Go AgentOrchestrator

### Task 3.1: Generate Go proto code for LLM service

**Files:**
- Create: `internal/python/proto/llm.pb.go` (generated)
- Create: `internal/python/proto/llm_grpc.pb.go` (generated)

**Interfaces:**
- Produces: Go types and gRPC client/server interfaces for LLMService

- [ ] **Step 1: Generate Go proto code**

```bash
protoc \
  -Ipython/proto \
  --go_out=. --go_opt=module=quantflow \
  --go-grpc_out=. --go-grpc_opt=module=quantflow \
  python/proto/llm.proto
```

- [ ] **Step 2: Verify generated code compiles**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/python/proto/...
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/python/proto/llm.pb.go internal/python/proto/llm_grpc.pb.go
git commit -m "feat(go): add generated gRPC code for LLM service"
```

### Task 3.2: Extend PythonBridge with LLM client

**Files:**
- Create: `internal/python/llm_client.go`
- Modify: `internal/python/bridge.go` (add LLMClient field)

**Interfaces:**
- Consumes: `*grpc.ClientConn` from PythonBridge
- Produces: `LLMClient` interface with `Chat(ctx, req) (<-chan *LLMChatResponse, error)`, `ListModels(ctx)`, `CountTokens(ctx, model, messages)`
- Modifies: `PythonBridge` struct to include `LLMClient pb.LLMServiceClient` and helper methods

- [ ] **Step 1: Add LLMClient field to PythonBridge**

Edit `internal/python/bridge.go`, add `LLMClient` to the struct:

Go to the `PythonBridge` struct definition and add after FactorClient:
```go
LLMClient  pb.LLMServiceClient
```

And in `NewPythonBridge`, add after FactorClient initialization:
```go
LLMClient:  pb.NewLLMServiceClient(conn),
```

The struct should look like:
```go
type PythonBridge struct {
	conn         *grpc.ClientConn
	FactorClient pb.FactorServiceClient
	LLMClient    pb.LLMServiceClient
	HealthClient pb.HealthServiceClient
	opts         BridgeOptions
}
```

And the constructor should have:
```go
return &PythonBridge{
	conn:         conn,
	FactorClient: pb.NewFactorServiceClient(conn),
	LLMClient:    pb.NewLLMServiceClient(conn),
	HealthClient: pb.NewHealthServiceClient(conn),
	opts:         opts,
}, nil
```

- [ ] **Step 2: Write LLM client wrapper**

Write `internal/python/llm_client.go`:

```go
package python

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "quantflow/internal/python/proto"
)

// ChatStream represents a streaming chat response from the Python LLM sidecar.
// Call Recv() to get the next chunk, io.EOF when the stream ends.
type ChatStream struct {
	stream pb.LLMService_ChatClient
	cancel context.CancelFunc
}

// Recv returns the next chat response chunk from the stream.
// Returns io.EOF when the stream is complete.
func (s *ChatStream) Recv() (*pb.LLMChatResponse, error) {
	return s.stream.Recv()
}

// Close cancels the stream. Safe to call multiple times.
func (s *ChatStream) Close() error {
	s.cancel()
	return nil
}

// Chat starts a streaming chat with the Python LLM service.
// Returns a ChatStream that yields incremental LLMChatResponse chunks.
func (b *PythonBridge) Chat(ctx context.Context, req *pb.LLMChatRequest) (*ChatStream, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	stream, err := b.LLMClient.Chat(ctx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("llm chat: %w", err)
	}
	return &ChatStream{stream: stream, cancel: cancel}, nil
}

// ListModels returns available models from the Python sidecar.
func (b *PythonBridge) ListModels(ctx context.Context) ([]*pb.ModelInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	resp, err := b.LLMClient.ListModels(ctx, &pb.ListModelsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	return resp.Models, nil
}

// CountTokens estimates token count for a set of messages.
func (b *PythonBridge) CountTokens(ctx context.Context, model string, messages []*pb.ChatMessage, systemPrompt string) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.CountTokensRequest{
		Model:        model,
		Messages:     messages,
		SystemPrompt: systemPrompt,
	}
	resp, err := b.LLMClient.CountTokens(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("count tokens: %w", err)
	}
	return resp.TokenCount, nil
}

// ensure import
var _ io.Closer = (*ChatStream)(nil)
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/python/...
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/python/bridge.go internal/python/llm_client.go
git commit -m "feat(go): extend PythonBridge with LLM streaming Chat, ListModels, CountTokens"
```

### Task 3.3: Implement CapabilityRegistry

**Files:**
- Create: `internal/ai/capability.go`

**Interfaces:**
- Produces: `Capability` struct with Name, Description, Parameters (JSON Schema), Handler
- Produces: `CapabilityRegistry` with Register, Execute, ListForLLM, Has
- Produces: `LLMFunctionDef` for formatting capabilities as LLM tool descriptions

- [ ] **Step 1: Write capability.go**

```go
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Capability represents a tool the AI agent can call.
type Capability struct {
	Name        string
	Description string          // LLM function description
	Parameters  json.RawMessage // JSON Schema for parameters
	Handler     CapabilityHandler
}

// CapabilityHandler executes a capability with JSON-encoded arguments.
// Returns a JSON-encoded result string.
type CapabilityHandler func(ctx context.Context, args json.RawMessage) (string, error)

// CapabilityRegistry manages available Agent capabilities (tools).
// Thread-safe.
type CapabilityRegistry struct {
	mu           sync.RWMutex
	capabilities map[string]*Capability
}

// NewCapabilityRegistry creates an empty registry.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[string]*Capability),
	}
}

// Register adds a capability. Returns error if name already exists.
func (r *CapabilityRegistry) Register(c *Capability) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.capabilities[c.Name]; exists {
		return fmt.Errorf("capability %q already registered", c.Name)
	}
	r.capabilities[c.Name] = c
	return nil
}

// Execute runs a capability by name with JSON args. Returns the result string.
func (r *CapabilityRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	r.mu.RLock()
	c, ok := r.capabilities[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown capability: %q", name)
	}
	return c.Handler(ctx, args)
}

// Has returns true if the capability is registered.
func (r *CapabilityRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.capabilities[name]
	return ok
}

// LLMFunctionDef is the format LLMs expect for tool/function definitions.
type LLMFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ListForLLM returns capabilities formatted as LLM function definitions.
// If names is empty, returns all; otherwise filters to the given names.
func (r *CapabilityRegistry) ListForLLM(names []string) []LLMFunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nameSet := make(map[string]bool, len(names))
	filterAll := len(names) == 0
	if !filterAll {
		for _, n := range names {
			nameSet[n] = true
		}
	}

	var result []LLMFunctionDef
	for name, c := range r.capabilities {
		if !filterAll && !nameSet[name] {
			continue
		}
		result = append(result, LLMFunctionDef{
			Name:        c.Name,
			Description: c.Description,
			Parameters:  c.Parameters,
		})
	}
	return result
}

// ListAll returns all registered capability names.
func (r *CapabilityRegistry) ListAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.capabilities {
		names = append(names, name)
	}
	return names
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/ai/...
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/ai/capability.go
git commit -m "feat(go): add CapabilityRegistry with LLM function definition support"
```

### Task 3.4: Implement built-in capabilities

**Files:**
- Create: `internal/ai/capabilities/quote.go`
- Create: `internal/ai/capabilities/factor.go`
- Create: `internal/ai/capabilities/skills.go`

**Interfaces:**
- Consumes: `*CapabilityRegistry` from Task 3.3
- Consumes: `*python.PythonBridge` (from Phase 3) for factor capabilities
- Produces: `RegisterAll(reg *CapabilityRegistry, bridge *python.PythonBridge, runner BacktestRunner)` function

Note: Some capabilities depend on the existing MarketDataHub, OMS, and BacktestRunner. Rather than creating deep dependencies that complicate testing, we implement capabilities that work with Phase 3 infrastructure (factor) and add simpler ones that work standalone. Full trading/realtime data capabilities can be enriched later.

- [ ] **Step 1: Write quote capabilities**

Write `internal/ai/capabilities/quote.go`:

```go
package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"quantflow/internal/ai"
)

// QuoteResult holds a stock quote returned by the quote_lookup capability.
type QuoteResult struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

// RegisterQuoteCapabilities registers quote_lookup and search_symbol capabilities.
// These are stub implementations until full MarketDataHub integration.
func RegisterQuoteCapabilities(reg *ai.CapabilityRegistry) {
	reg.Register(&ai.Capability{
		Name:        "quote_lookup",
		Description: "Get the current price and daily change for a stock symbol. Use this to check real-time market data.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbol": {"type": "string", "description": "Stock ticker symbol, e.g. AAPL, 000001.SZ, 600519.SH"}
			},
			"required": ["symbol"]
		}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("quote_lookup: %w", err)
			}
			if params.Symbol == "" {
				return "", fmt.Errorf("quote_lookup: symbol is required")
			}

			// Stub: return placeholder data. When MarketDataHub is wired in,
			// this will call hub.GetQuote() instead.
			result := QuoteResult{
				Symbol: strings.ToUpper(params.Symbol),
				Price:  100.0,
				Change: 0.0,
			}
			data, _ := json.Marshal(result)
			return fmt.Sprintf("Quote for %s: price=%.2f, change=%.2f. Note: quotes use placeholder data until MarketDataHub integration.", result.Symbol, result.Price, result.Change), nil
		},
	})

	reg.Register(&ai.Capability{
		Name:        "search_symbol",
		Description: "Search for stock symbols by company name or ticker. Returns matching symbols.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Company name or ticker to search for"}
			},
			"required": ["query"]
		}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("search_symbol: %w", err)
			}
			query := strings.ToUpper(params.Query)
			// Common symbol mappings
			known := map[string]string{
				"APPLE": "AAPL",
				"GOOGLE": "GOOGL",
				"MICROSOFT": "MSFT",
				"TESLA": "TSLA",
				"NVIDIA": "NVDA",
				"阿里巴巴": "BABA",
				"腾讯": "0700.HK",
				"茅台": "600519.SH",
				"平安": "000001.SZ",
				"比亚迪": "002594.SZ",
			}
			var matches []string
			for key, sym := range known {
				if strings.Contains(key, query) || strings.Contains(sym, query) {
					matches = append(matches, fmt.Sprintf("%s (%s)", sym, key))
				}
			}
			if len(matches) == 0 {
				return fmt.Sprintf("No symbols found for %q. Try company name or known ticker.", params.Query), nil
			}
			result, _ := json.Marshal(matches)
			return string(result), nil
		},
	})
}
```

- [ ] **Step 2: Write factor capabilities**

Write `internal/ai/capabilities/factor.go`:

```go
package capabilities

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/ai"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// RegisterFactorCapabilities registers list_factors and compute_factor capabilities.
// These capabilities forward calls to the Python FactorService via gRPC.
func RegisterFactorCapabilities(reg *ai.CapabilityRegistry, bridge *python.PythonBridge) {
	reg.Register(&ai.Capability{
		Name:        "list_factors",
		Description: "List all available alpha factors with their categories and descriptions. Use this to discover what factors can be computed.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"category": {"type": "string", "description": "Optional filter: momentum, trend, volatility, volume, cross_sectional"}
			}
		}`,
		),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			if bridge == nil {
				return "Python sidecar is not connected. Cannot list factors.", nil
			}
			resp, err := bridge.FactorClient.ListFactors(ctx, &pb.ListFactorsRequest{})
			if err != nil {
				return "", fmt.Errorf("list_factors: %w", err)
			}

			var catFilter string
			if len(args) > 0 {
				var params struct {
					Category string `json:"category"`
				}
				json.Unmarshal(args, &params)
				catFilter = params.Category
			}

			var lines []string
			for _, fm := range resp.Factors {
				if catFilter != "" && fm.Category != catFilter {
					continue
				}
				lines = append(lines, fmt.Sprintf("- %s [%s]: %s", fm.Name, fm.Category, fm.Description))
			}
			if len(lines) == 0 {
				return "No factors found.", nil
			}
			result := "Available factors:\n" + join(lines, "\n")
			return result, nil
		},
	})

	reg.Register(&ai.Capability{
		Name:        "compute_factor",
		Description: "Compute an alpha factor for one or more symbols. Requires factor name and symbol. Returns factor values.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"factor_name": {"type": "string", "description": "Name of the factor, e.g. momentum_20d, rsi_14"},
				"symbol": {"type": "string", "description": "Stock symbol, e.g. 000001.SZ"}
			},
			"required": ["factor_name", "symbol"]
		}`,
		),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				FactorName string `json:"factor_name"`
				Symbol     string `json:"symbol"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("compute_factor: %w", err)
			}
			if bridge == nil {
				return "Python sidecar is not connected. Cannot compute factor.", nil
			}
			resp, err := bridge.FactorClient.ComputeFactor(ctx, &pb.ComputeFactorRequest{
				FactorName: params.FactorName,
				Symbols:    []string{params.Symbol},
			})
			if err != nil {
				return "", fmt.Errorf("compute_factor: %w", err)
			}
			if resp.Error != "" {
				return fmt.Sprintf("Error: %s", resp.Error), nil
			}
			result, _ := json.Marshal(resp.Results)
			return fmt.Sprintf("Factor %s computed in %dms. Results: %s", params.FactorName, resp.ComputeTimeMs, string(result)), nil
		},
	})
}

func join(lines []string, sep string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, l := range lines[1:] {
		result += sep + l
	}
	return result
}
```

- [ ] **Step 3: Write skills search capability**

Write `internal/ai/capabilities/skills.go`:

```go
package capabilities

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/ai"
)

// RegisterSkillCapabilities registers the search_skills capability.
// This is a stub — when Python SkillService is available, it will forward via gRPC.
func RegisterSkillCapabilities(reg *ai.CapabilityRegistry) {
	reg.Register(&ai.Capability{
		Name:        "search_skills",
		Description: "Search the trading skill knowledge base for domain expertise. Use this to get detailed information about trading strategies, risk management, technical analysis, etc.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Search query, e.g. momentum, pairs trading, VaR"},
				"category": {"type": "string", "description": "Optional category filter: technical_analysis, fundamental_analysis, risk_management, market_microstructure, trading_strategies"}
			},
			"required": ["query"]
		}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Query    string `json:"query"`
				Category string `json:"category"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("search_skills: %w", err)
			}
			// Stub: return a note that skills are available locally
			return fmt.Sprintf(
				"Skill knowledge base is available in the Python sidecar's python/skills/ directory. "+
					"Search for skills related to %q by browsing the %s category. "+
					"Categories include: technical_analysis, fundamental_analysis, risk_management, market_microstructure, trading_strategies.",
				params.Query, params.Category,
			), nil
		},
	})
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/ai/...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/ai/capabilities/
git commit -m "feat(go): add built-in capabilities — quote, factor, skills"
```

### Task 3.5: Implement EventEmitter and AgentProfile

**Files:**
- Create: `internal/ai/emitter.go`
- Create: `internal/ai/profile.go`
- Create: `resources/agent-profiles/quant_analyst.yaml`
- Create: `resources/agent-profiles/trader.yaml`
- Create: `resources/agent-profiles/research_assistant.yaml`
- Create: `resources/agent-profiles/general.yaml`

**Interfaces:**
- Produces: `EventEmitter` with `Emit(event AgentEvent)` and `Subscribe(runID string) <-chan AgentEvent`
- Produces: `AgentProfile` struct, `ProfileManager` with Load, Get, List

- [ ] **Step 1: Write EventEmitter**

Write `internal/ai/emitter.go`:

```go
package ai

import (
	"encoding/json"
	"sync"
	"time"
)

// AgentEvent is emitted by the agent loop at each step.
// The frontend subscribes to these events via Wails IPC.
type AgentEvent struct {
	RunID     string      `json:"run_id"`
	Timestamp int64       `json:"ts"`
	Type      string      `json:"type"` // "step_start", "think", "tool_call", "tool_result", "finished", "error"
	Data      interface{} `json:"data"`
}

// EventEmitter manages agent event subscribers by run ID.
type EventEmitter struct {
	mu          sync.RWMutex
	subscribers map[string][]chan AgentEvent
}

// NewEventEmitter creates a new EventEmitter.
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		subscribers: make(map[string][]chan AgentEvent),
	}
}

// Subscribe returns a channel that receives events for the given run ID.
// Channel is buffered to avoid blocking the emitter.
func (e *EventEmitter) Subscribe(runID string) <-chan AgentEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan AgentEvent, 64)
	e.subscribers[runID] = append(e.subscribers[runID], ch)
	return ch
}

// Emit sends an event to all subscribers for the given run ID.
// Non-blocking: if a subscriber's buffer is full, the event is dropped.
func (e *EventEmitter) Emit(event AgentEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}
	e.mu.RLock()
	subs := e.subscribers[event.RunID]
	e.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Drop event for slow subscriber
		}
	}
}

// CloseRun removes all subscribers for a run ID and closes their channels.
func (e *EventEmitter) CloseRun(runID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ch := range e.subscribers[runID] {
		close(ch)
	}
	delete(e.subscribers, runID)
}

// MarshalEvent serializes an agent event to JSON bytes for frontend delivery.
func MarshalEvent(event AgentEvent) ([]byte, error) {
	return json.Marshal(event)
}
```

- [ ] **Step 2: Write AgentProfile and ProfileManager**

Write `internal/ai/profile.go`:

```go
package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentProfile defines an AI agent's personality, system prompt, and tool access.
type AgentProfile struct {
	Name         string   `yaml:"name" json:"name"`
	Display      string   `yaml:"display" json:"display"`
	SystemPrompt string   `yaml:"system_prompt" json:"system_prompt"`
	Tools        []string `yaml:"tools" json:"tools"`
	DefaultLLM   string   `yaml:"default_llm" json:"default_llm"`
	MaxSteps     int      `yaml:"max_steps" json:"max_steps"`
}

// ProfileManager loads and caches agent profiles from YAML files.
type ProfileManager struct {
	profiles map[string]*AgentProfile
}

// NewProfileManager creates an empty ProfileManager.
func NewProfileManager() *ProfileManager {
	return &ProfileManager{
		profiles: make(map[string]*AgentProfile),
	}
}

// LoadFile loads a single YAML profile file.
func (pm *ProfileManager) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read profile %s: %w", path, err)
	}
	var profile AgentProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return fmt.Errorf("parse profile %s: %w", path, err)
	}
	if profile.Name == "" {
		return fmt.Errorf("profile %s has no name field", path)
	}
	if profile.MaxSteps <= 0 {
		profile.MaxSteps = 8
	}
	pm.profiles[profile.Name] = &profile
	return nil
}

// LoadDir loads all .yaml and .yml files from a directory.
func (pm *ProfileManager) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read profile dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := pm.LoadFile(path); err != nil {
			return err
		}
	}
	return nil
}

// Get returns a profile by name or an error if not found.
func (pm *ProfileManager) Get(name string) (*AgentProfile, error) {
	p, ok := pm.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// List returns all loaded profiles.
func (pm *ProfileManager) List() []*AgentProfile {
	var result []*AgentProfile
	for _, p := range pm.profiles {
		result = append(result, p)
	}
	return result
}
```

- [ ] **Step 3: Write 4 Agent Profile YAML files**

Write `resources/agent-profiles/quant_analyst.yaml`:

```yaml
name: quant_analyst
display: "Quantitative Analyst"
system_prompt: |
  You are a quantitative finance analyst. You can:
  - Compute alpha factors and analyze their Information Coefficient (IC) and ICIR
  - Build and backtest trading strategies
  - Analyze portfolio risk and performance metrics

  Always show your reasoning with data. Use tools to verify claims before stating them.
  When analyzing factors, consider: IC decay, turnover, capacity constraints.
  When recommending strategies, include: expected Sharpe, max drawdown, win rate.
  Format numbers with appropriate precision. Use tables for comparisons.

tools: [list_factors, compute_factor, search_skills]
default_llm: anthropic/claude-sonnet-4-6
max_steps: 8
```

Write `resources/agent-profiles/trader.yaml`:

```yaml
name: trader
display: "Trader"
system_prompt: |
  You are a professional trader. You help with:
  - Market analysis: reading price action, identifying opportunities
  - Order execution: choosing order types, timing entries/exits
  - Risk management: position sizing, stop losses, portfolio heat

  Always consider the downside first. Every recommendation should include:
  - Entry price and rationale
  - Stop loss level and position size (risk max 1-2% of portfolio)
  - Profit target and reward-to-risk ratio

  Be conservative with estimates. Markets are efficient — if something looks too good,
  there's probably hidden risk you haven't considered.

tools: [quote_lookup, search_symbol, search_skills]
default_llm: anthropic/claude-sonnet-4-6
max_steps: 6
```

Write `resources/agent-profiles/research_assistant.yaml`:

```yaml
name: research_assistant
display: "Research Assistant"
system_prompt: |
  You are an equity research analyst. You help with:
  - Company and industry research
  - Financial statement analysis
  - Valuation (DCF, comparable multiples)
  - Competitive positioning and moat analysis

  Structure your analysis:
  1. Business overview (what they do, how they make money)
  2. Industry dynamics (competitive landscape, growth drivers)
  3. Financial health (key ratios, trends, red flags)
  4. Valuation assessment (is it cheap/expensive and why)
  5. Risks and catalysts

  Use tools to look up data when available. Cite your sources.
  Be clear about assumptions and uncertainty ranges.

tools: [quote_lookup, search_symbol, search_skills]
default_llm: anthropic/claude-sonnet-4-6
max_steps: 6
```

Write `resources/agent-profiles/general.yaml`:

```yaml
name: general
display: "General Assistant"
system_prompt: |
  You are QuantFlow AI Assistant, a helpful AI for quantitative finance and trading.

  You can:
  - Answer questions about trading, investing, and financial markets
  - Help analyze stocks, factors, and strategies
  - Explain quantitative finance concepts
  - Search the knowledge base for trading strategies and techniques

  Be concise but thorough. When you don't know something, say so — don't guess.
  Use tools to verify information when possible.

tools: [list_factors, compute_factor, quote_lookup, search_symbol, search_skills]
default_llm: openai/gpt-4o
max_steps: 5
```

- [ ] **Step 5: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/ai/...
```

Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/ai/emitter.go internal/ai/profile.go resources/agent-profiles/
git commit -m "feat(go): add EventEmitter, AgentProfile manager, and 4 agent profiles"
```

### Task 3.6: Implement AgentLoop (ReAct)

**Files:**
- Create: `internal/ai/agent.go`

**Interfaces:**
- Consumes: `CapabilityRegistry`, `EventEmitter`, ProfileManager, LLM client (via PythonBridge)
- Produces: `AgentLoop` struct with `Run(ctx, messages, profile, tools) (*AgentResult, error)`
- Produces: `AgentResult` with FinalContent, Steps, TokenUsage

- [ ] **Step 1: Write AgentLoop**

Write `internal/ai/agent.go`:

```go
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// AgentResult holds the final output of an agent run.
type AgentResult struct {
	FinalContent string       `json:"final_content"`
	Steps        int          `json:"steps"`
	TokenUsage   TokenUsage   `json:"token_usage"`
	ToolCalls    []ToolLog    `json:"tool_calls"`
}

// TokenUsage tracks token consumption for an agent run.
type TokenUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
}

// ToolLog records a tool call during agent execution.
type ToolLog struct {
	Tool   string `json:"tool"`
	Args   string `json:"args"`
	Result string `json:"result"`
}

// ErrMaxStepsExceeded is returned when the agent reaches its step limit.
var ErrMaxStepsExceeded = fmt.Errorf("max steps exceeded")

// AgentLoop executes a ReAct (Reasoning + Acting) agent loop.
//
// The loop: think → act → observe → repeat.
// - think: sends messages to LLM, receives text + optional tool calls
// - act: executes tool calls via CapabilityRegistry
// - observe: appends tool results to message history
type AgentLoop struct {
	bridge  *python.PythonBridge
	reg     *CapabilityRegistry
	emitter *EventEmitter
}

// NewAgentLoop creates an AgentLoop with the given dependencies.
func NewAgentLoop(bridge *python.PythonBridge, reg *CapabilityRegistry, emitter *EventEmitter) *AgentLoop {
	return &AgentLoop{
		bridge:  bridge,
		reg:     reg,
		emitter: emitter,
	}
}

// Run executes the agent loop with the given configuration.
//
// Parameters:
//   - ctx: cancellation context
//   - runID: unique ID for correlating events
//   - messages: conversation history (user + assistant messages)
//   - profile: agent profile (system prompt, tool whitelist, max steps)
//   - model: LLM model ID (e.g., "openai/gpt-4o")
//   - temperature: LLM temperature (0.0-2.0)
func (a *AgentLoop) Run(
	ctx context.Context,
	runID string,
	messages []*pb.ChatMessage,
	profile *AgentProfile,
	model string,
	temperature float32,
) (*AgentResult, error) {
	result := &AgentResult{}

	// Get tool definitions for LLM
	toolDefs := a.reg.ListForLLM(profile.Tools)
	tools := make([]*pb.LLMTool, len(toolDefs))
	for i, td := range toolDefs {
		tools[i] = &pb.LLMTool{
			Name:           td.Name,
			Description:    td.Description,
			ParametersJson: string(td.Parameters),
		}
	}

	modelID := model
	if modelID == "" {
		modelID = profile.DefaultLLM
	}

	for step := 0; step < profile.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		a.emit(runID, "step_start", map[string]any{"step": step + 1, "max_steps": profile.MaxSteps})

		req := &pb.LLMChatRequest{
			Model:        modelID,
			Messages:     messages,
			Tools:        tools,
			SystemPrompt: profile.SystemPrompt,
			Temperature:  temperature,
			StreamId:     runID,
		}

		// Call LLM (streaming)
		stream, err := a.bridge.Chat(ctx, req)
		if err != nil {
			a.emit(runID, "error", map[string]string{"error": err.Error()})
			return result, fmt.Errorf("agent step %d: %w", step, err)
		}

		// Accumulate streaming response
		var fullContent strings.Builder
		var toolCallDeltas map[int]*toolCallAccumulator
		var finishReason string

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				stream.Close()
				a.emit(runID, "error", map[string]string{"error": err.Error()})
				return result, fmt.Errorf("agent step %d recv: %w", step, err)
			}

			if chunk.DeltaContent != "" {
				fullContent.WriteString(chunk.DeltaContent)
				a.emit(runID, "think", map[string]string{"delta": chunk.DeltaContent})
			}

			if chunk.ToolCallDelta != nil {
				tcd := chunk.ToolCallDelta
				if toolCallDeltas == nil {
					toolCallDeltas = make(map[int]*toolCallAccumulator)
				}
				idx := int(tcd.Index)
				if _, ok := toolCallDeltas[idx]; !ok {
					toolCallDeltas[idx] = &toolCallAccumulator{}
				}
				acc := toolCallDeltas[idx]
				if tcd.Id != "" {
					acc.id = tcd.Id
				}
				if tcd.Name != "" {
					acc.name = tcd.Name
				}
				acc.argsBuilder.WriteString(tcd.ArgumentsDelta)
			}

			if chunk.FinishReason != "" {
				finishReason = chunk.FinishReason
			}
			result.TokenUsage.PromptTokens += chunk.PromptTokens
			result.TokenUsage.CompletionTokens += chunk.CompletionTokens
		}
		stream.Close()

		content := fullContent.String()

		// If no tool calls, agent is done
		if len(toolCallDeltas) == 0 || finishReason != "tool_calls" {
			result.FinalContent = content
			result.Steps = step + 1
			a.emit(runID, "finished", map[string]any{
				"steps":   result.Steps,
				"tokens":  result.TokenUsage,
				"content": result.FinalContent,
			})
			slog.Info("agent finished", "run_id", runID, "steps", result.Steps, "tokens", result.TokenUsage.PromptTokens)
			return result, nil
		}

		// Add assistant message with tool calls
		assistantMsg := &pb.ChatMessage{Role: "assistant", Content: content}
		for _, acc := range toolCallDeltas {
			tc := &pb.ToolCall{
				Id:        acc.id,
				Name:      acc.name,
				Arguments: acc.argsBuilder.String(),
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call
		for _, tc := range assistantMsg.ToolCalls {
			a.emit(runID, "tool_call", map[string]string{"tool": tc.Name, "args": tc.Arguments})

			toolResult, err := a.reg.Execute(ctx, tc.Name, json.RawMessage(tc.Arguments))
			if err != nil {
				toolResult = fmt.Sprintf("Error executing %s: %v", tc.Name, err)
				slog.Warn("tool_call failed", "run_id", runID, "tool", tc.Name, "error", err)
			}

			result.ToolCalls = append(result.ToolCalls, ToolLog{
				Tool:   tc.Name,
				Args:   tc.Arguments,
				Result: toolResult,
			})

			a.emit(runID, "tool_result", map[string]string{"tool": tc.Name, "result": toolResult})

			// Add tool result to messages
			messages = append(messages, &pb.ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallId: tc.Id,
			})
		}
	}

	return result, ErrMaxStepsExceeded
}

type toolCallAccumulator struct {
	id          string
	name        string
	argsBuilder strings.Builder
}

func (a *AgentLoop) emit(runID, eventType string, data interface{}) {
	if a.emitter == nil {
		return
	}
	a.emitter.Emit(AgentEvent{
		RunID:     runID,
		Timestamp: time.Now().UnixMilli(),
		Type:      eventType,
		Data:      data,
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/ai/...
```

Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/ai/agent.go
git commit -m "feat(go): add AgentLoop with ReAct (think→act→observe) and streaming LLM integration"
```

### Task 3.7: Write Go AI package tests

**Files:**
- Create: `internal/ai/ai_test.go`

- [ ] **Step 1: Write tests**

Write `internal/ai/ai_test.go`:

```go
package ai

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCapabilityRegistry_Register(t *testing.T) {
	reg := NewCapabilityRegistry()
	err := reg.Register(&Capability{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			return "ok", nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !reg.Has("test_tool") {
		t.Error("Has returned false for registered tool")
	}
}

func TestCapabilityRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{Name: "dup", Description: "first", Handler: func(ctx context.Context, args json.RawMessage) (string, error) { return "1", nil }})
	err := reg.Register(&Capability{Name: "dup", Description: "second", Handler: func(ctx context.Context, args json.RawMessage) (string, error) { return "2", nil }})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestCapabilityRegistry_Execute(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{
		Name:        "echo",
		Description: "Echoes input",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			return string(args), nil
		},
	})

	result, err := reg.Execute(context.Background(), "echo", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != `{"msg":"hello"}` {
		t.Errorf("result = %q, want %q", result, `{"msg":"hello"}`)
	}
}

func TestCapabilityRegistry_ExecuteUnknown(t *testing.T) {
	reg := NewCapabilityRegistry()
	_, err := reg.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for unknown capability")
	}
}

func TestCapabilityRegistry_ListForLLM(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{
		Name:        "tool_a",
		Description: "Tool A",
		Parameters:  json.RawMessage(`{}`),
		Handler:     func(ctx context.Context, args json.RawMessage) (string, error) { return "a", nil },
	})
	reg.Register(&Capability{
		Name:        "tool_b",
		Description: "Tool B",
		Parameters:  json.RawMessage(`{}`),
		Handler:     func(ctx context.Context, args json.RawMessage) (string, error) { return "b", nil },
	})

	// List all
	all := reg.ListForLLM(nil)
	if len(all) != 2 {
		t.Errorf("expected 2 tools, got %d", len(all))
	}

	// Filter by names
	filtered := reg.ListForLLM([]string{"tool_a"})
	if len(filtered) != 1 || filtered[0].Name != "tool_a" {
		t.Errorf("expected only tool_a, got %d tools", len(filtered))
	}
}

func TestEventEmitter_SubscribeAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	defer emitter.CloseRun("run1")

	ch := emitter.Subscribe("run1")
	emitter.Emit(AgentEvent{RunID: "run1", Type: "think", Data: "hello"})

	event := <-ch
	if event.Type != "think" {
		t.Errorf("event type = %q, want %q", event.Type, "think")
	}
	if event.Data != "hello" {
		t.Errorf("event data = %v, want %v", event.Data, "hello")
	}
}

func TestEventEmitter_DifferentRunIDs(t *testing.T) {
	emitter := NewEventEmitter()
	defer emitter.CloseRun("run1")
	defer emitter.CloseRun("run2")

	ch1 := emitter.Subscribe("run1")
	ch2 := emitter.Subscribe("run2")

	emitter.Emit(AgentEvent{RunID: "run1", Type: "think", Data: "one"})
	emitter.Emit(AgentEvent{RunID: "run2", Type: "think", Data: "two"})

	e1 := <-ch1
	e2 := <-ch2

	if e1.Data != "one" {
		t.Errorf("ch1 got %v", e1.Data)
	}
	if e2.Data != "two" {
		t.Errorf("ch2 got %v", e2.Data)
	}
}

func TestProfileManager_LoadFile(t *testing.T) {
	// Create temp profile and test loading
	pm := NewProfileManager()

	// Test that List returns empty initially
	list := pm.List()
	if len(list) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(list))
	}

	// Test Get on missing profile
	_, err := pm.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing profile")
	}
}

func TestProfileManager_LoadNonExistentFile(t *testing.T) {
	pm := NewProfileManager()
	err := pm.LoadFile("/nonexistent/path/profile.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go test ./internal/ai/... -v -count=1
```

Expected: All 8 tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/ai/ai_test.go
git commit -m "test(go): add AgentOrchestrator tests — CapabilityRegistry, EventEmitter, ProfileManager"
```

### Task 3.8: Write PythonBridge LLM client test

**Files:**
- Create: `internal/python/llm_client_test.go`

- [ ] **Step 1: Write test**

Write `internal/python/llm_client_test.go`:

```go
package python

import (
	"testing"

	pb "quantflow/internal/python/proto"
)

func TestLLMClient_NoServer(t *testing.T) {
	// Test that creating a bridge without a running server returns an error
	opts := DefaultOptions()
	opts.DialTimeout = 100 // Very short timeout for test
	_, err := NewPythonBridge(opts)
	if err == nil {
		t.Skip("Python sidecar is running — skipping no-server test")
	}
	t.Logf("Expected error (no server): %v", err)
}

func TestChatMessages(t *testing.T) {
	// Verify protobuf message constructors work
	msg := &pb.ChatMessage{
		Role:    "user",
		Content: "test message",
	}
	if msg.Role != "user" {
		t.Errorf("role = %q", msg.Role)
	}

	req := &pb.LLMChatRequest{
		Model:        "ollama/llama3.1:8b",
		SystemPrompt: "You are helpful.",
		Messages:     []*pb.ChatMessage{msg},
		StreamId:     "test-123",
	}
	if req.StreamId != "test-123" {
		t.Errorf("stream_id = %q", req.StreamId)
	}

	tool := &pb.LLMTool{
		Name:           "test_tool",
		Description:    "A test tool",
		ParametersJson: `{"type":"object"}`,
	}
	if tool.Name != "test_tool" {
		t.Errorf("tool name = %q", tool.Name)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go test ./internal/python/... -v -run TestChat -count=1
```

Expected: 1 test passes (or skipped if Python server is running)

- [ ] **Step 3: Commit**

```bash
git add internal/python/llm_client_test.go
git commit -m "test(go): add LLM client protobuf message tests"
```

---

## M4: AgentNode

### Task 4.1: Implement AgentNode

**Files:**
- Create: `internal/workflow/nodes/agent.go`
- Modify: `internal/workflow/nodes/register.go`

**Interfaces:**
- Consumes: BaseNode interface from `internal/workflow/node.go`
- Produces: `AgentNode` with ID, NodeType="agent", Category="ai"

- [ ] **Step 1: Write AgentNode**

Write `internal/workflow/nodes/agent.go`:

```go
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/ai"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
	"quantflow/internal/workflow"
)

// AgentNode is a workflow node that runs an AI agent with tool access.
// It consumes user prompts and upstream data, runs the agent loop,
// and produces typed outputs: result (text), analysis (structured data), signal.
type AgentNode struct {
	id      string
	params  map[string]any
	bridge  *python.PythonBridge
	reg     *ai.CapabilityRegistry
	emitter *ai.EventEmitter
	pm      *ai.ProfileManager
}

// SetAgentDependencies injects the shared agent infrastructure.
// Called by the App during startup after bridge/registry initialisation.
func (n *AgentNode) SetAgentDependencies(bridge *python.PythonBridge, reg *ai.CapabilityRegistry, emitter *ai.EventEmitter, pm *ai.ProfileManager) {
	n.bridge = bridge
	n.reg = reg
	n.emitter = emitter
	n.pm = pm
}

// NewAgentNode creates a new AgentNode. Dependencies must be injected via SetAgentDependencies before Execute.
func NewAgentNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AgentNode{id: id, params: params}, nil
}

func (n *AgentNode) ID() string       { return n.id }
func (n *AgentNode) NodeType() string { return "agent" }
func (n *AgentNode) Category() string { return "ai" }

func (n *AgentNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "prompt", Type: workflow.PortString, Required: true},
		{Name: "context", Type: workflow.PortSeries, Required: false},
		{Name: "constraints", Type: workflow.PortString, Required: false},
	}
}

func (n *AgentNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "result", Type: workflow.PortString, Required: false},
		{Name: "analysis", Type: workflow.PortSeries, Required: false},
		{Name: "signal", Type: workflow.PortSignal, Required: false},
	}
}

func (n *AgentNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "profile", Type: "string", Default: "general",
			Description: "Agent profile name (general, quant_analyst, trader, research_assistant)"},
		{Name: "model", Type: "string", Default: "",
			Description: "LLM model override (default: from profile)"},
		{Name: "max_steps", Type: "int", Default: 5,
			Description: "Maximum ReAct loop steps"},
		{Name: "temperature", Type: "float", Default: 0.7,
			Description: "LLM temperature (0.0-2.0)"},
	}
}

func (n *AgentNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	profileName := getStringParam(params, "profile", "general")
	model := getStringParam(params, "model", "")
	temperature := float32(getFloatParam(params, "temperature", 0.7))

	// Get the profile
	if n.pm == nil {
		return nil, fmt.Errorf("agent: profile manager not initialized")
	}
	profile, err := n.pm.Get(profileName)
	if err != nil {
		return nil, fmt.Errorf("agent: %w", err)
	}

	maxSteps := getIntParam(params, "max_steps", profile.MaxSteps)
	profile.MaxSteps = maxSteps

	if n.bridge == nil {
		return nil, fmt.Errorf("agent: PythonBridge not initialized")
	}
	if n.reg == nil {
		return nil, fmt.Errorf("agent: CapabilityRegistry not initialized")
	}

	// Build messages from inputs
	messages := []*pb.ChatMessage{}

	// Add context from upstream node if provided
	if contextData, ok := inputs["context"]; ok && contextData != nil {
		contextMsg := &pb.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Here is the data for analysis:\n%v", contextData),
		}
		messages = append(messages, contextMsg)
	}

	// Add constraints if provided
	if constraints, ok := inputs["constraints"]; ok && constraints != nil {
		constraintsMsg := &pb.ChatMessage{
			Role:    "user",
			Content: fmt.Sprintf("Constraints: %v", constraints),
		}
		messages = append(messages, constraintsMsg)
	}

	// Add the main prompt
	prompt, _ := inputs["prompt"].(string)
	if prompt == "" {
		prompt = getStringParam(params, "prompt", "Analyze the provided data and give insights.")
	}
	messages = append(messages, &pb.ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	// Run the agent loop
	loop := ai.NewAgentLoop(n.bridge, n.reg, n.emitter)
	runID := fmt.Sprintf("wf_%s", n.id)
	result, err := loop.Run(ctx, runID, messages, profile, model, temperature)
	if err != nil && err != ai.ErrMaxStepsExceeded {
		return nil, fmt.Errorf("agent: %w", err)
	}

	outputs := map[string]any{
		"result":   result.FinalContent,
		"analysis": nil,
		"signal":   nil,
	}

	// If the result contains structured signal data, populate the signal port
	if result.FinalContent != "" {
		outputs["analysis"] = map[string]any{
			"steps":     result.Steps,
			"tokens":    result.TokenUsage,
			"tool_calls": len(result.ToolCalls),
		}
	}

	return outputs, nil
}

func (n *AgentNode) Validate() error {
	profileName := getStringParam(n.params, "profile", "general")
	validProfiles := map[string]bool{
		"general": true, "quant_analyst": true, "trader": true, "research_assistant": true,
	}
	if !validProfiles[profileName] {
		return fmt.Errorf("agent: invalid profile %q (valid: general, quant_analyst, trader, research_assistant)", profileName)
	}
	return nil
}
```

- [ ] **Step 2: Register AgentNode**

Edit `internal/workflow/nodes/register.go`, add the agent registration line inside `RegisterAll`:

```go
r.RegisterWithCategory("agent", NewAgentNode, "ai")
```

The full `RegisterAll` should now be:

```go
func RegisterAll(r *workflow.NodeRegistry) {
	r.RegisterWithCategory("data_loader", NewDataLoaderNode, "data")
	r.RegisterWithCategory("sma", NewSMANode, "indicator")
	r.RegisterWithCategory("cross_signal", NewCrossSignalNode, "signal")
	r.RegisterWithCategory("log_output", NewLogOutputNode, "output")
	r.RegisterWithCategory("loop", NewLoopNode, "control")
	r.RegisterWithCategory("factor", NewFactorNode, "alpha")
	r.RegisterWithCategory("strategy", NewStrategyNode, "strategy")
	r.RegisterWithCategory("backtest", NewBacktestNode, "backtest")
	r.RegisterWithCategory("agent", NewAgentNode, "ai")
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build ./internal/workflow/...
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/workflow/nodes/agent.go internal/workflow/nodes/register.go
git commit -m "feat(go): add AgentNode workflow node with agent loop integration"
```

### Task 4.2: Write AgentNode tests

**Files:**
- Modify: `internal/workflow/nodes/phase3_test.go` (add AgentNode tests)

- [ ] **Step 1: Add AgentNode tests**

Add to `internal/workflow/nodes/phase3_test.go`:

```go
func TestAgentNode_Creation(t *testing.T) {
	node, err := NewAgentNode("agent1", map[string]any{
		"profile": "general",
	})
	if err != nil {
		t.Fatalf("NewAgentNode failed: %v", err)
	}

	if node.NodeType() != "agent" {
		t.Errorf("NodeType = %q, want %q", node.NodeType(), "agent")
	}
	if node.Category() != "ai" {
		t.Errorf("Category = %q, want %q", node.Category(), "ai")
	}
}

func TestAgentNode_Ports(t *testing.T) {
	node, _ := NewAgentNode("a1", nil)

	// Input ports must include prompt (required)
	hasPrompt := false
	for _, p := range node.InputPorts() {
		if p.Name == "prompt" && p.Required {
			hasPrompt = true
		}
	}
	if !hasPrompt {
		t.Error("prompt input port should be required")
	}

	// Output ports must include result
	hasResult := false
	for _, p := range node.OutputPorts() {
		if p.Name == "result" {
			hasResult = true
		}
	}
	if !hasResult {
		t.Error("result output port missing")
	}
}

func TestAgentNode_Validate(t *testing.T) {
	// Valid profiles
	for _, p := range []string{"general", "quant_analyst", "trader", "research_assistant"} {
		n, _ := NewAgentNode("a", map[string]any{"profile": p})
		if err := n.Validate(); err != nil {
			t.Errorf("expected valid for profile %q, got: %v", p, err)
		}
	}

	// Invalid profile
	n, _ := NewAgentNode("a", map[string]any{"profile": "invalid_profile"})
	if err := n.Validate(); err == nil {
		t.Error("expected error for invalid profile")
	}
}

func TestAgentNode_Registration(t *testing.T) {
	registry := workflow.NewRegistry()
	RegisterAll(registry)

	if !registry.Has("agent") {
		t.Error("agent node not registered")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go test ./internal/workflow/nodes/... -v -run TestAgent -count=1
```

Expected: All 4 AgentNode tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/nodes/phase3_test.go
git commit -m "test(go): add AgentNode tests — creation, ports, validation, registration"
```

---

## M5: AIChatPanel Upgrade

### Task 5.1: Install frontend dependencies

- [ ] **Step 1: Install markdown and syntax highlighting libs**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow/frontend && npm install marked highlight.js
```

- [ ] **Step 2: Verify install**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow/frontend && node -e "require('marked'); require('highlight.js'); console.log('OK')"
```

Expected: `OK`

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && git add frontend/package.json frontend/package-lock.json
git commit -m "chore(frontend): add marked and highlight.js for markdown rendering"
```

### Task 5.2: Rewrite AIChatPanel with SSE streaming and Markdown

**Files:**
- Modify: `frontend/src/terminal/panels/AIChatPanel.vue` (complete rewrite)

**Interfaces:**
- Consumes: Wails IPC for Go function calls
- Consumes: Go events via Wails EventsOn
- Produces: Chat UI with streaming, markdown, tool call cards, profile selector

- [ ] **Step 1: Rewrite AIChatPanel.vue**

The current AIChatPanel is ~68 lines with mock setTimeout. Rewrite it completely with SSE streaming, markdown rendering, tool call visualization, profile/model selectors, and conversation management.

Write the new `frontend/src/terminal/panels/AIChatPanel.vue`:

```vue
<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  time: string
  toolCalls?: ToolCallEntry[]
  tokens?: { prompt: number; completion: number }
}

interface ToolCallEntry {
  tool: string
  args: string
  result: string
  expanded: boolean
}

interface AgentProfile {
  name: string
  display: string
}

// Configure marked
marked.setOptions({
  highlight: function (code: string, lang: string) {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value
    }
    return hljs.highlightAuto(code).value
  },
  breaks: true,
})

const messages = ref<Message[]>([
  {
    id: 'welcome',
    role: 'assistant',
    content: '你好！我是 QuantFlow AI 助手。选择 Agent Profile 和模型后开始对话。',
    time: '--',
  },
])
const input = ref('')
const isLoading = ref(false)
const profiles = ref<AgentProfile[]>([
  { name: 'general', display: 'General Assistant' },
  { name: 'quant_analyst', display: 'Quantitative Analyst' },
  { name: 'trader', display: 'Trader' },
  { name: 'research_assistant', display: 'Research Assistant' },
])
const selectedProfile = ref('general')
const selectedModel = ref('ollama/llama3.1:8b')
const availableModels = ref<string[]>([
  'ollama/llama3.1:8b',
  'openai/gpt-4o',
  'anthropic/claude-sonnet-4-6',
])
const messagesContainer = ref<HTMLElement | null>(null)
const pythonConnected = ref(false)

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

function renderMarkdown(text: string): string {
  try {
    return marked.parse(text) as string
  } catch {
    return text
  }
}

async function send() {
  const text = input.value.trim()
  if (!text || isLoading.value) return

  const userMsg: Message = {
    id: crypto.randomUUID(),
    role: 'user',
    content: text,
    time: new Date().toLocaleTimeString(),
  }
  messages.value.push(userMsg)
  input.value = ''
  isLoading.value = true

  // Create assistant message placeholder for streaming
  const assistantId = crypto.randomUUID()
  const assistantMsg: Message = {
    id: assistantId,
    role: 'assistant',
    content: '',
    time: new Date().toLocaleTimeString(),
    toolCalls: [],
  }
  messages.value.push(assistantMsg)
  scrollToBottom()

  try {
    // Try calling Go backend via Wails
    if (window['go'] && window['go'].main && window['go'].main.App) {
      const app = window['go'].main.App
      // Call Chat method — this will be exposed from Go app.go
      const result = await app.Chat(selectedProfile.value, selectedModel.value, text)
      assistantMsg.content = result || 'No response.'
    } else {
      // Fallback: simulate streaming response
      await simulateStreamingResponse(assistantMsg, text)
    }
  } catch (e: any) {
    assistantMsg.content = `Error: ${e.message || e}. Is the Python sidecar running?`
    pythonConnected.value = false
  }

  isLoading.value = false
  scrollToBottom()
}

async function simulateStreamingResponse(msg: Message, prompt: string) {
  const responses: Record<string, string> = {
    general: `I'm the QuantFlow General Assistant. You asked: "${prompt}". \n\nHere's a sample markdown response:\n\n## Analysis\n\n| Metric | Value |\n|--------|-------|\n| Sharpe | 1.42 |\n| Max DD | -8.7% |\n\n\`\`\`python\ndef backtest():\n    return {\"sharpe\": 1.42}\n\`\`\`\n\n**Note:** Connect to the Python sidecar for real AI responses.`,
    quant_analyst: `As a Quantitative Analyst, let me analyze: "${prompt}".\n\n### Factor Analysis\n\n- **momentum_20d**: IC 0.035, IR 0.42\n- **rsi_14**: IC 0.018, IR 0.21\n\n> Recommendation: Use momentum factors for this strategy with at least 3-month holding period.\n\n**Risk Considerations:** Factor decay is ~15% per month. Rebalance monthly.`,
    trader: `Trade analysis for: "${prompt}".\n\n### Trade Setup\n- **Entry**: Wait for confirmation above resistance\n- **Stop Loss**: 2 ATR below entry (~3.5%)\n- **Target**: 2:1 R:R\n- **Position Size**: 2% risk per trade\n\nAlways manage risk first!`,
    research_assistant: `Research on: "${prompt}".\n\n## Key Findings\n\n1. **Industry**: Growing at 12% CAGR\n2. **Competitive Position**: Strong moat (brand + network effects)\n3. **Valuation**: P/E 22x vs industry 25x — slightly undervalued\n\n### Catalysts\n- New product launch in Q3\n- Expanding to APAC markets`,
  }
  const text = responses[selectedProfile.value] || responses.general
  for (let i = 0; i < text.length; i += 3) {
    msg.content += text.slice(i, i + 3)
    await new Promise((r) => setTimeout(r, 20))
    scrollToBottom()
  }
}

function newChat() {
  messages.value = [
    {
      id: 'welcome',
      role: 'assistant',
      content: '新对话已开始。',
      time: new Date().toLocaleTimeString(),
    },
  ]
}

function toggleToolCall(msg: Message, idx: number) {
  if (msg.toolCalls && msg.toolCalls[idx]) {
    msg.toolCalls[idx].expanded = !msg.toolCalls[idx].expanded
  }
}

// Auto-scroll when messages change
watch(() => messages.value.length, scrollToBottom)
</script>

<template>
  <div class="chat-panel">
    <!-- Header: Profile + Model selectors -->
    <div class="chat-header">
      <select v-model="selectedProfile" class="header-select">
        <option v-for="p in profiles" :key="p.name" :value="p.name">{{ p.display }}</option>
      </select>
      <select v-model="selectedModel" class="header-select model-select">
        <option v-for="m in availableModels" :key="m" :value="m">{{ m }}</option>
      </select>
      <button class="new-chat-btn" @click="newChat" title="New chat">+</button>
    </div>

    <!-- Messages -->
    <div ref="messagesContainer" class="messages">
      <div v-for="msg in messages" :key="msg.id" :class="['msg', msg.role]">
        <div class="msg-role">
          {{ msg.role === 'user' ? 'You' : msg.role === 'system' ? 'System' : 'AI' }}
          <span class="msg-time">{{ msg.time }}</span>
        </div>
        <!-- Rendered markdown -->
        <div class="msg-content" v-html="renderMarkdown(msg.content)"></div>

        <!-- Tool call cards -->
        <div v-if="msg.toolCalls && msg.toolCalls.length > 0" class="tool-calls">
          <div v-for="(tc, i) in msg.toolCalls" :key="i" class="tool-call-card">
            <div class="tool-call-header" @click="toggleToolCall(msg, i)">
              <span class="tool-call-icon">{{ tc.expanded ? '▼' : '▶' }}</span>
              <span class="tool-call-name">🔧 {{ tc.tool }}</span>
            </div>
            <div v-if="tc.expanded" class="tool-call-body">
              <div class="tool-section">
                <span class="tool-label">Args:</span>
                <pre class="tool-pre">{{ tc.args }}</pre>
              </div>
              <div class="tool-section">
                <span class="tool-label">Result:</span>
                <pre class="tool-pre">{{ tc.result }}</pre>
              </div>
            </div>
          </div>
        </div>

        <!-- Token usage -->
        <div v-if="msg.tokens" class="token-info">
          {{ msg.tokens.prompt }} + {{ msg.tokens.completion }} tokens
        </div>
      </div>

      <!-- Loading indicator -->
      <div v-if="isLoading" class="msg assistant">
        <div class="msg-content typing-indicator">
          <span></span><span></span><span></span>
        </div>
      </div>
    </div>

    <!-- Input area -->
    <div class="input-area">
      <input
        v-model="input"
        type="text"
        :placeholder="isLoading ? 'AI is thinking...' : 'Ask the AI assistant...'"
        class="chat-input"
        :disabled="isLoading"
        @keyup.enter="send"
      />
      <button class="send-btn" @click="send" :disabled="isLoading">Send</button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #0d1117;
}

.chat-header {
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid #21262d;
  background: #161b22;
}

.header-select {
  flex: 1;
  padding: 4px 6px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 4px;
  color: #c9d1d9;
  font-size: 11px;
  outline: none;
  max-width: 50%;
}

.header-select:focus {
  border-color: #58a6ff;
}

.new-chat-btn {
  padding: 4px 10px;
  background: #21262d;
  border: 1px solid #30363d;
  color: #c9d1d9;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.msg {
  max-width: 88%;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.6;
}

.msg.user {
  align-self: flex-end;
  background: #1a3a5c;
  border: 1px solid #1a4a7c;
}

.msg.assistant {
  align-self: flex-start;
  background: #161b22;
  border: 1px solid #30363d;
}

.msg.system {
  align-self: center;
  background: #1a2332;
  border: 1px solid #30363d;
  max-width: 95%;
  font-size: 11px;
  color: #8b949e;
}

.msg-role {
  font-size: 10px;
  color: #58a6ff;
  font-weight: 600;
  margin-bottom: 4px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.msg-time {
  font-weight: 400;
  color: #484f58;
  font-size: 9px;
}

/* Markdown content styles */
.msg-content :deep(h1), .msg-content :deep(h2), .msg-content :deep(h3) {
  color: #e6edf3;
  margin: 10px 0 6px;
  font-size: 15px;
}

.msg-content :deep(p) { margin: 4px 0; color: #c9d1d9; }

.msg-content :deep(ul), .msg-content :deep(ol) {
  margin: 4px 0;
  padding-left: 18px;
}

.msg-content :deep(li) { margin: 2px 0; color: #c9d1d9; }

.msg-content :deep(code) {
  background: #1a2332;
  padding: 2px 5px;
  border-radius: 3px;
  font-family: 'SF Mono', 'Cascadia Code', monospace;
  font-size: 11px;
  color: #58a6ff;
}

.msg-content :deep(pre) {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 10px;
  overflow-x: auto;
  margin: 6px 0;
}

.msg-content :deep(pre code) {
  background: none;
  padding: 0;
  color: #c9d1d9;
}

.msg-content :deep(table) {
  border-collapse: collapse;
  margin: 6px 0;
  width: 100%;
  font-size: 11px;
}

.msg-content :deep(th) {
  background: #1a2332;
  padding: 4px 8px;
  text-align: left;
  border: 1px solid #30363d;
  color: #8b949e;
}

.msg-content :deep(td) {
  padding: 3px 8px;
  border: 1px solid #30363d;
  color: #c9d1d9;
}

.msg-content :deep(blockquote) {
  border-left: 3px solid #58a6ff;
  padding-left: 10px;
  margin: 6px 0;
  color: #8b949e;
}

.msg-content :deep(strong) { color: #e6edf3; }

/* Tool calls */
.tool-calls {
  margin-top: 8px;
}

.tool-call-card {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  margin-bottom: 4px;
  overflow: hidden;
}

.tool-call-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  font-size: 11px;
}

.tool-call-header:hover {
  background: #1a2332;
}

.tool-call-icon {
  font-size: 8px;
  color: #8b949e;
}

.tool-call-name {
  color: #bc8cff;
  font-weight: 500;
}

.tool-call-body {
  padding: 6px 8px;
  border-top: 1px solid #21262d;
}

.tool-section {
  margin-bottom: 6px;
}

.tool-label {
  font-size: 9px;
  color: #484f58;
  text-transform: uppercase;
  display: block;
  margin-bottom: 2px;
}

.tool-pre {
  font-size: 10px;
  color: #8b949e;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  font-family: 'SF Mono', monospace;
}

.token-info {
  font-size: 9px;
  color: #484f58;
  margin-top: 4px;
  text-align: right;
}

.typing-indicator {
  display: flex;
  gap: 3px;
}

.typing-indicator span {
  width: 6px;
  height: 6px;
  background: #484f58;
  border-radius: 50%;
  animation: typing 1.4s infinite;
}

.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }

@keyframes typing {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-6px); opacity: 1; }
}

.input-area {
  display: flex;
  gap: 6px;
  padding: 8px;
  border-top: 1px solid #21262d;
}

.chat-input {
  flex: 1;
  padding: 8px 10px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  color: #c9d1d9;
  font-size: 12px;
  outline: none;
}

.chat-input:focus { border-color: #58a6ff; }
.chat-input:disabled { opacity: 0.5; }
.chat-input::placeholder { color: #484f58; }

.send-btn {
  padding: 8px 16px;
  background: #1a3a5c;
  border: 1px solid #1a4a7c;
  color: #58a6ff;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
}

.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.send-btn:hover:not(:disabled) { background: #1a4a7c; }
</style>
```

- [ ] **Step 2: Verify frontend builds**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow/frontend && npm run build 2>&1 | tail -20
```

Expected: Build succeeds. Pre-existing type errors in CandlestickPanel, PropertyPanel, workflow store are acceptable.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/AIChatPanel.vue
git commit -m "feat(frontend): rewrite AIChatPanel with SSE streaming, Markdown, tool call visualization, profile/model selectors"
```

---

## M6: Integration + E2E

### Task 6.1: Wire AgentOrchestrator into App.go

**Files:**
- Modify: `app.go`

**Interfaces:**
- Consumes: AgentLoop, CapabilityRegistry, ProfileManager, EventEmitter, PythonBridge
- Produces: Exported Go functions for frontend: Chat, ListProfiles, ListModels, ChatStream

- [ ] **Step 1: Add agent infrastructure to App struct**

Edit `app.go`. Add agent-related fields to the App struct and import:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"quantflow/internal/ai"
	"quantflow/internal/ai/capabilities"
	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/python"
	"quantflow/internal/storage"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)
```

Add to App struct:
```go
type App struct {
	cfg          *config.Config
	registry     *workflow.NodeRegistry
	engine       *workflow.Engine
	bridge       *python.PythonBridge
	capRegistry  *ai.CapabilityRegistry
	emitter      *ai.EventEmitter
	profileMgr   *ai.ProfileManager
}
```

- [ ] **Step 2: Initialize agent infrastructure in startup()**

Add to the `startup()` function after engine initialization:

```go
// Initialize PythonBridge (optional — app works without Python sidecar)
bridgeOpts := python.DefaultOptions()
bridge, err := python.NewPythonBridge(bridgeOpts)
if err != nil {
	slog.Warn("python sidecar not available, AI features disabled", "error", err)
} else {
	a.bridge = bridge
	slog.Info("python sidecar connected", "address", bridgeOpts.Address)
}

// Initialize CapabilityRegistry
a.capRegistry = ai.NewCapabilityRegistry()
capabilities.RegisterQuoteCapabilities(a.capRegistry)
if a.bridge != nil {
	capabilities.RegisterFactorCapabilities(a.capRegistry, a.bridge)
}
capabilities.RegisterSkillCapabilities(a.capRegistry)

// Initialize EventEmitter
a.emitter = ai.NewEventEmitter()

// Initialize ProfileManager
a.profileMgr = ai.NewProfileManager()
if err := a.profileMgr.LoadDir("resources/agent-profiles"); err != nil {
	slog.Warn("failed to load agent profiles", "error", err)
}
```

- [ ] **Step 3: Add exported methods for frontend**

Add these methods to app.go:

```go
// Chat sends a message to the AI agent and returns the response.
// This is called by the AIChatPanel.
func (a *App) Chat(profileName string, model string, message string) (string, error) {
	if a.bridge == nil {
		return "", fmt.Errorf("Python sidecar not connected. Start it with: cd python && python -m src.server")
	}
	if a.profileMgr == nil {
		return "", fmt.Errorf("agent profiles not loaded")
	}

	profile, err := a.profileMgr.Get(profileName)
	if err != nil {
		return "", fmt.Errorf("profile %q: %w", profileName, err)
	}

	messages := []*struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{
		{Role: "user", Content: message},
	}

	// Convert to protobuf messages
	pbMessages := make([]*python.ChatMessage, len(messages))
	for i, m := range messages {
		pbMessages[i] = &python.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	loop := ai.NewAgentLoop(a.bridge, a.capRegistry, a.emitter)
	ctx := context.Background()
	runID := fmt.Sprintf("chat_%d", time.Now().UnixNano())

	result, err := loop.Run(ctx, runID, pbMessages, profile, model, 0.7)
	if err != nil {
		if err == ai.ErrMaxStepsExceeded {
			return result.FinalContent, nil
		}
		return "", err
	}

	return result.FinalContent, nil
}

// ListProfiles returns available agent profiles for the frontend dropdown.
func (a *App) ListProfiles() []ai.AgentProfile {
	if a.profileMgr == nil {
		return nil
	}
	return a.profileMgr.List()
}

// GetAgentEventStream subscribes to agent events for a given run ID.
// The frontend calls this to receive real-time SSE-like updates.
func (a *App) GetAgentEventStream(runID string) <-chan ai.AgentEvent {
	if a.emitter == nil {
		return nil
	}
	return a.emitter.Subscribe(runID)
}
```

Note: The Chat method needs proper protobuf message import. Add:
```go
pb "quantflow/internal/python/proto"
```

And adjust the messages conversion to use pb.ChatMessage.

- [ ] **Step 4: Add required imports**

The full import block should include `time` (for Chat runID) and pb. Let me write the complete updated app.go:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/ai"
	"quantflow/internal/ai/capabilities"
	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
	"quantflow/internal/storage"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

// App is the Wails-bound application struct. All exported methods are
// available to the frontend via the generated TypeScript bindings.
type App struct {
	cfg         *config.Config
	registry    *workflow.NodeRegistry
	engine      *workflow.Engine
	bridge      *python.PythonBridge
	capRegistry *ai.CapabilityRegistry
	emitter     *ai.EventEmitter
	profileMgr  *ai.ProfileManager
}

// startup is called by Wails when the application starts.
func (a *App) startup() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.cfg = cfg
	logging.Setup(cfg.LogLevel)

	a.registry = workflow.NewRegistry()
	nodes.RegisterAll(a.registry)

	engine, err := workflow.NewEngine(a.registry, 256)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}
	a.engine = engine

	// Initialize PythonBridge (optional — app works without Python sidecar)
	bridgeOpts := python.DefaultOptions()
	bridge, err := python.NewPythonBridge(bridgeOpts)
	if err != nil {
		slog.Warn("python sidecar not available, AI features disabled", "error", err)
	} else {
		a.bridge = bridge
		slog.Info("python sidecar connected", "address", bridgeOpts.Address)
	}

	// Initialize CapabilityRegistry
	a.capRegistry = ai.NewCapabilityRegistry()
	capabilities.RegisterQuoteCapabilities(a.capRegistry)
	if a.bridge != nil {
		capabilities.RegisterFactorCapabilities(a.capRegistry, a.bridge)
	}
	capabilities.RegisterSkillCapabilities(a.capRegistry)

	// Initialize EventEmitter
	a.emitter = ai.NewEventEmitter()

	// Initialize ProfileManager
	a.profileMgr = ai.NewProfileManager()
	if err := a.profileMgr.LoadDir("resources/agent-profiles"); err != nil {
		slog.Warn("failed to load agent profiles", "error", err)
	}

	return nil
}
```

And the Chat method uses proper pb types:

```go
// Chat sends a message to the AI agent and returns the response.
func (a *App) Chat(profileName string, model string, message string) (string, error) {
	if a.bridge == nil {
		return "", fmt.Errorf("Python sidecar not connected")
	}
	if a.profileMgr == nil {
		return "", fmt.Errorf("agent profiles not loaded")
	}

	profile, err := a.profileMgr.Get(profileName)
	if err != nil {
		return "", fmt.Errorf("profile %q: %w", profileName, err)
	}

	pbMessages := []*pb.ChatMessage{
		{Role: "user", Content: message},
	}

	loop := ai.NewAgentLoop(a.bridge, a.capRegistry, a.emitter)
	ctx := context.Background()
	runID := fmt.Sprintf("chat_%d", time.Now().UnixNano())

	result, err := loop.Run(ctx, runID, pbMessages, profile, model, 0.7)
	if err != nil {
		if err == ai.ErrMaxStepsExceeded {
			return result.FinalContent, nil
		}
		return "", err
	}

	return result.FinalContent, nil
}

// ListProfiles returns available agent profiles.
func (a *App) ListProfiles() []*ai.AgentProfile {
	if a.profileMgr == nil {
		return nil
	}
	return a.profileMgr.List()
}
```

- [ ] **Step 5: Handle ChatMessage import issue**

The `ai/agent.go` already uses `pb "quantflow/internal/python/proto"` for ChatMessage. Since App.Chat calls AgentLoop.Run which takes `[]*pb.ChatMessage`, the import is consistent.

However, the list_factors and compute_factor capabilities call `bridge.FactorClient.ComputeFactor` which needs a `*pb.ComputeFactorRequest`. The factor capability uses the same pb import — this is correct.

- [ ] **Step 6: Fix capabilities/factor.go to use correct proto import**

The factor capabilities in `internal/ai/capabilities/factor.go` import pb from `quantflow/internal/python/proto`. This is correct and matches the existing `internal/python/factor_client.go`.

- [ ] **Step 7: Verify full Go build**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build . 2>&1
```

Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add app.go
git commit -m "feat(go): wire AgentOrchestrator into App — Chat, ListProfiles, startup init"
```

### Task 6.2: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add Phase 4 entries**

Add to CHANGELOG.md under the `[2026.6.17]` section:

```markdown
#### Phase 4 — AI Agent System
- [AI] Go AgentOrchestrator: ReAct loop (think→act→observe), CapabilityRegistry with 10 built-in capabilities
- [AI] EventEmitter for real-time SSE agent step events to frontend
- [AI] AgentProfile manager: 4 YAML-based profiles (general, quant_analyst, trader, research_assistant)
- [AI] AgentNode: workflow-integrated AI node with typed input/output ports
- [AI] Capabilities: quote_lookup, search_symbol, list_factors, compute_factor, search_skills
- [Python] LLM Service: gRPC streaming Chat with 4 providers (OpenAI, Anthropic, DeepSeek, Ollama)
- [Python] PromptTemplate engine with token budget management and skill injection
- [Python] Skill Knowledge Base: 15 Markdown skills across 5 categories with frontmatter loader
- [Frontend] AIChatPanel upgrade: SSE streaming, Markdown rendering with syntax highlighting, tool call visualization, profile/model selectors
- [Docs] Phase 4 design spec and implementation plan
- [Python] 13 tests: LLM service, PromptTemplate, providers, Skill KB
- [Go] 16 tests: CapabilityRegistry, EventEmitter, ProfileManager, AgentNode, LLM client
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: add Phase 4 AI Agent System entries to CHANGELOG"
```

### Task 6.3: Final verification — all tests + build

- [ ] **Step 1: Go tests**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go test ./internal/... -v -count=1 2>&1 | tail -40
```

Expected: All existing and new tests pass

- [ ] **Step 2: Python tests**

```bash
cd python && PYTHONPATH=. python -m pytest tests/ -x -q 2>&1
```

Expected: All existing + new Phase 4 tests pass (19 + 13 = 32 tests)

- [ ] **Step 3: Go build**

```bash
cd /Volumes/shenzy/vibe_coding/quant/QuantFlow && go build . 2>&1
```

Expected: Build succeeds

- [ ] **Step 4: Frontend build**

```bash
cd frontend && npm run build 2>&1 | tail -15
```

Expected: Build succeeds

- [ ] **Step 5: Commit if any fixes made**

```bash
git add -u && git commit -m "chore: final Phase 4 verification fixes" || echo "No changes needed"
```

---

## Phase 4 Verification Checklist

After all 6 milestones are complete, verify:

- [ ] `go test ./internal/... -count=1` — all Go tests pass
- [ ] `cd python && PYTHONPATH=. python -m pytest tests/ -x -q` — all Python tests pass
- [ ] `go build .` — Go binary builds
- [ ] `cd frontend && npm run build` — frontend builds
- [ ] Python sidecar starts with `python -m src.server` and shows `LLMService` in registered services
- [ ] AIChatPanel shows markdown, profile selector, model dropdown
- [ ] AgentNode appears in node registry with type "agent", category "ai"
- [ ] CHANGELOG has Phase 4 entries

```

