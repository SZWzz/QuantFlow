# Phase 4: AI Agent System — Design Spec

## Motivation

Phase 1-3 established the core infrastructure: workflow engine, trading engine, market data hub, Python gRPC sidecar with factor computation, and backtesting engine. However, the system lacks intelligent decision-making capability.

The AIChatPanel currently returns a hardcoded mock response. There is no AgentNode for workflow integration. The Python sidecar has an LLM stub but no actual LLM service. Phase 4 fills this gap by building a complete AI Agent system.

The goal: users can converse with AI in the chat panel OR embed AI as a typed workflow node that consumes upstream data and produces downstream signals/analysis — all within the same gRPC infrastructure established in Phase 3.

## Design

### Architecture Overview

```
Frontend (Vue 3)
┌─────────────────────────┐  ┌──────────────────────────────────┐
│ AIChatPanel (upgraded)  │  │ AgentNode (workflow canvas)       │
│ · SSE streaming render  │  │ · Agent Profile selector          │
│ · Markdown + code blocks│  │ · Input ports: prompt/context/data│
│ · Tool call visualize   │  │ · Output ports: result/analysis   │
│ · Conversation history  │  │ · Real-time step progress         │
└───────────┬─────────────┘  └────────────────┬─────────────────┘
            └────────────────┬────────────────┘
                             │ Wails IPC (Go func calls + SSE)
                             │
Go Backend
┌────────────────────────────┼─────────────────────────────────┐
│                 AgentOrchestrator                             │
│  ┌──────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ AgentLoop    │  │ CapabilityReg   │  │ EventEmitter    │  │
│  │ think→act→   │  │ · Go builtins   │  │ · SSE push      │  │
│  │ observe      │  │ · Python gRPC   │  │ · step event    │  │
│  │ maxSteps=10  │  │ · Workflow nodes│  │ · tool_call log │  │
│  │ cancel ctx   │  └─────────────────┘  │ · token usage   │  │
│  └──────┬───────┘                       └─────────────────┘  │
│         │                                                     │
│  ┌──────┴──────────────────────────────────────────────────┐  │
│  │              AgentProfile Manager                        │  │
│  │  · YAML loader  · Profile registry  · Tool whitelist    │  │
│  └─────────────────────────────────────────────────────────┘  │
│         │                                                     │
│  ┌──────┴──────────────┐  ┌──────────────────────────────┐   │
│  │ gRPC LLM Client     │  │ gRPC Skill Client (optional) │   │
│  │ · Chat(streaming)   │  │ · SearchSkills               │   │
│  │ · CountTokens       │  │ · GetSkillContent            │   │
│  └─────────────────────┘  └──────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
                             │ gRPC (localhost:50051)
Python Sidecar
┌────────────────────────────┼─────────────────────────────────┐
│  LLM Service (NEW)                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ OpenAI   │  │ Anthropic│  │ DeepSeek │  │ Ollama   │     │
│  │ Provider │  │ Provider │  │ Provider │  │ Provider │     │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ PromptTemplate Engine                                 │    │
│  │ · System prompt assembly  · Few-shot injection        │    │
│  │ · Tool description format · Token budget management   │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                               │
│  Skill Knowledge Base (NEW)                                   │
│  · skills/ directory — Markdown files organized by domain     │
│  · Inject relevant skills into system prompt (not tool call)  │
│  · Optional: FAISS vector index for semantic search           │
└──────────────────────────────────────────────────────────────┘
```

### Go Side: AgentOrchestrator

#### AgentLoop (Lightweight ReAct)

The ReAct loop runs in Go, not Python. Each iteration:
1. **Think**: Send messages + tool definitions to Python LLM via gRPC chat
2. **Act**: If LLM returns tool calls, execute them in Go's CapabilityRegistry
3. **Observe**: Append tool results to message history, loop

```go
type AgentLoop struct {
    maxSteps     int           // default 10
    llmClient    LLMClient     // gRPC to Python
    registry     *CapabilityRegistry
    emitter      *EventEmitter // SSE to frontend
}

func (a *AgentLoop) Run(ctx context.Context, messages []Message, tools []string) (*AgentResult, error) {
    for step := 0; step < a.maxSteps; step++ {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        resp := a.llmClient.Chat(ctx, messages, tools)
        a.emitter.Emit(StepEvent{Step: step, Type: "think", Content: resp.Content})

        if resp.ToolCalls == nil || len(resp.ToolCalls) == 0 {
            return a.parseFinalOutput(resp.Content), nil
        }

        for _, tc := range resp.ToolCalls {
            a.emitter.Emit(ToolCallEvent{Tool: tc.Name, Args: tc.Args})
            result := a.registry.Execute(ctx, tc.Name, tc.Args)
            messages = append(messages, Message{Role: "tool", Content: result, ToolCallID: tc.ID})
            a.emitter.Emit(ToolResultEvent{Tool: tc.Name, Result: result})
        }
    }
    return nil, ErrMaxStepsExceeded
}
```

Key design decisions:
- **Cancel propagation**: ctx passed from workflow engine, cancel stops the agent
- **Step limit**: configurable per AgentNode, prevents infinite loops
- **Event stream**: every step emits an event → frontend sees real-time progress

#### CapabilityRegistry

Unified tool registry. Tools can be implemented in Go or forwarded to Python:

```go
type Capability struct {
    Name        string
    Description string              // LLM function description
    Parameters  jsonschema.Schema    // JSON Schema for args
    Handler     func(ctx context.Context, params json.RawMessage) (any, error)
    Source      string              // "go" | "python" | "workflow"
}

type CapabilityRegistry struct {
    capabilities map[string]*Capability
}

// Register registers a capability. Duplicate names return error.
func (r *CapabilityRegistry) Register(c *Capability) error

// Execute runs a capability by name with JSON args. Returns stringified result.
func (r *CapabilityRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error)

// ListForLLM returns capabilities formatted as LLM function definitions.
func (r *CapabilityRegistry) ListForLLM(names []string) []LLMFunctionDef
```

Built-in capabilities (Phase 4 delivers these):

| Capability | Source | Description |
|------------|--------|-------------|
| `quote_lookup` | Go | Get real-time quote for a symbol |
| `search_symbol` | Go | Search symbols by name/code |
| `get_ohlcv` | Go | Fetch OHLCV bars from MarketDataHub |
| `list_factors` | Go→Python | List available alpha factors via gRPC FactorService |
| `compute_factor` | Go→Python | Compute a factor via gRPC FactorService |
| `run_backtest` | Go | Execute a backtest via BacktestRunner |
| `get_positions` | Go | Get current portfolio positions |
| `place_paper_order` | Go | Place a paper trading order via OMS |
| `get_workflow_status` | Go | Get workflow engine status |
| `search_skills` | Go→Python | Search the skill knowledge base |

#### EventEmitter (SSE)

Events are pushed to the frontend via Wails SSE (server-sent events):

```go
type AgentEvent struct {
    RunID     string      `json:"run_id"`
    Timestamp int64       `json:"ts"`
    Type      string      `json:"type"` // "step_start" | "think" | "tool_call" | "tool_result" | "finished" | "error"
    Data      interface{} `json:"data"`
}
```

The frontend subscribes to `agent:<runID>` events via a Wails event binding.

#### AgentProfile Manager

Profiles are YAML files in `resources/agent-profiles/`:

```yaml
# resources/agent-profiles/quant_analyst.yaml
name: quant_analyst
display: "Quantitative Analyst"
system_prompt: |
  You are a quantitative finance analyst. You can:
  - Compute alpha factors and analyze their IC/IR
  - Build and backtest trading strategies
  - Analyze portfolio risk and performance
  
  Always show your reasoning with data. Use tools to verify claims.
  When analyzing factors, consider: IC, decay, turnover, capacity.

tools: [quote_lookup, get_ohlcv, list_factors, compute_factor, run_backtest, get_positions]
default_llm: anthropic/claude-sonnet-4-6
max_steps: 8
```

```go
type AgentProfile struct {
    Name        string   `yaml:"name"`
    Display     string   `yaml:"display"`
    SystemPrompt string  `yaml:"system_prompt"`
    Tools       []string `yaml:"tools"`
    DefaultLLM  string   `yaml:"default_llm"`
    MaxSteps    int      `yaml:"max_steps"`
}

type ProfileManager struct {
    profiles map[string]*AgentProfile
}

func (pm *ProfileManager) Load(paths []string) error
func (pm *ProfileManager) Get(name string) (*AgentProfile, error)
func (pm *ProfileManager) List() []*AgentProfile
```

Initial profiles (4):

| Profile | Purpose | Key Tools |
|---------|---------|-----------|
| `quant_analyst` | Factor research + backtesting | list_factors, compute_factor, run_backtest |
| `trader` | Market analysis + order execution | quote_lookup, get_ohlcv, place_paper_order, get_positions |
| `research_assistant` | Stock research + fundamentals | quote_lookup, search_symbol, get_ohlcv |
| `general` | General-purpose chat, all tools | all |

### Python Side: LLM Service

#### Proto Definition: `python/proto/llm.proto`

```protobuf
service LLMService {
  rpc Chat(LLMChatRequest) returns (stream LLMChatResponse);
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
  rpc CountTokens(CountTokensRequest) returns (CountTokensResponse);
}

message LLMChatRequest {
  string model = 1;                  // "openai/gpt-4o", "anthropic/claude-sonnet-4-6", etc.
  repeated ChatMessage messages = 2;
  repeated LLMTool tools = 3;        // available tools (JSON Schema)
  string system_prompt = 4;
  float temperature = 5;
  int32 max_tokens = 6;
  string stream_id = 7;              // correlation ID for the stream
}

message ChatMessage {
  string role = 1;                   // "system" | "user" | "assistant" | "tool"
  string content = 2;
  string tool_call_id = 3;           // for role="tool" messages
  repeated ToolCall tool_calls = 4;  // for role="assistant" responses
}

message ToolCall {
  string id = 1;
  string name = 2;
  string arguments = 3;              // JSON string
}

message LLMChatResponse {
  string delta_content = 1;          // incremental token (streaming)
  ToolCallDelta tool_call_delta = 2; // incremental tool call
  string finish_reason = 3;          // "stop" | "tool_calls" | "length" | "error"
  int32 prompt_tokens = 4;
  int32 completion_tokens = 5;
}
```

#### Provider Pattern

```python
class LLMProvider(ABC):
    """Abstract provider interface."""
    @abstractmethod
    async def chat(self, request, context) -> AsyncIterator[LLMChatResponse]:
        ...

class OpenAIProvider(LLMProvider):
    def __init__(self, api_key: str): ...
    async def chat(self, request, context): ...

class AnthropicProvider(LLMProvider):
    def __init__(self, api_key: str): ...
    async def chat(self, request, context): ...

class DeepSeekProvider(LLMProvider):
    def __init__(self, api_key: str): ...
    async def chat(self, request, context): ...

class OllamaProvider(LLMProvider):
    def __init__(self, base_url: str = "http://localhost:11434"): ...
    async def chat(self, request, context): ...
```

Model naming convention: `provider/model` — e.g., `openai/gpt-4o`, `anthropic/claude-sonnet-4-6`, `deepseek/deepseek-chat`, `ollama/llama3.1:70b`

#### PromptTemplate Engine

Handles system prompt assembly:
- Base system prompt (from AgentProfile)
- Tool descriptions injected (formatted per provider conventions)
- Few-shot examples (optional, from profile)
- Skill knowledge injected (retrieved from Skill KB, capped at token budget)
- Token counting to stay within context window

### Python Side: Skill Knowledge Base

Skills are specialized domain knowledge stored as Markdown files. They are NOT tools — they are injected into the system prompt to give the LLM domain expertise.

```
python/skills/
├── technical_analysis/
│   ├── candlestick_patterns.md
│   ├── indicator_usage.md
│   └── divergence_trading.md
├── fundamental_analysis/
│   ├── valuation_ratios.md
│   ├── dcf_model.md
│   └── earnings_analysis.md
├── risk_management/
│   ├── position_sizing.md
│   ├── var_methods.md
│   └── portfolio_optimization.md
├── market_microstructure/
│   ├── order_book_analysis.md
│   └── market_impact.md
└── trading_strategies/
    ├── momentum_strategies.md
    ├── mean_reversion.md
    └── pairs_trading.md
```

Each skill file:
```markdown
---
title: Momentum Strategies
category: trading_strategies
tags: [momentum, trend, factor]
difficulty: intermediate
---

# Momentum Strategies

## Cross-Sectional Momentum
Buy past winners, sell past losers within a universe...

## Time-Series Momentum
Go long assets with positive recent returns, short those with negative...

## Implementation Considerations
- Look-back period selection (3, 6, 12 months common)
- Rebalance frequency (monthly typical)
- Transaction cost impact on turnover
- A-share specific: T+1 settlement means signals must be generated one day ahead
```

Initial skill set: ~15 skills across 5 categories (migrated/adapted from AstockPursue's 89 skills, selecting those relevant to A-share/US/HK/crypto markets).

### Go Side: gRPC LLM Client

Extends the existing PythonBridge with LLM client:

```go
// Added to PythonBridge struct:
LLMClient pb.LLMServiceClient

// LLMClient wraps the gRPC streaming Chat call:
func (b *PythonBridge) Chat(ctx context.Context, req *pb.LLMChatRequest) (<-chan *pb.LLMChatResponse, error)

// ListModels returns available models from the Python sidecar.
func (b *PythonBridge) ListModels(ctx context.Context) ([]string, error)

// CountTokens returns the token count for a message set.
func (b *PythonBridge) CountTokens(ctx context.Context, model string, messages []*pb.ChatMessage) (int32, error)
```

### Workflow Node: AgentNode

AgentNode is the bridge between AI and workflow — it's a typed transformation node:

```go
type AgentNode struct {
    id      string
    params  map[string]any
    bridge  *python.PythonBridge    // injected for LLM gRPC calls
    profile *AgentProfile           // loaded at node creation
}

// NodeType: "agent"
// Category: "ai"

// Input Ports:
//   prompt (string, required)   — user prompt/question
//   context (series, optional)  — upstream data/factors
//   constraints (string, optional) — constraints for the agent

// Output Ports:
//   result (string)         — final text response
//   analysis (series)       — structured analysis data
//   signal (series)         — trading signals (buy/sell indicators)

// Params:
//   profile        string   — AgentProfile name (default: "general")
//   model          string   — LLM model override (default: from profile)
//   max_steps      int      — ReAct loop max steps (default: from profile)
//   temperature    float    — LLM temperature (default: 0.7)
```

### Frontend: AIChatPanel Upgrade

Current state: mock response with `setTimeout`. Upgrade to:

1. **SSE Streaming**: Subscribe to `agent:<runID>` events. Render tokens as they arrive (typewriter effect).
2. **Markdown Rendering**: Use `marked` + `highlight.js` for code blocks. Tables for financial data.
3. **Tool Call Visualization**: Expandable cards showing what tool was called with what args, and the result.
4. **Conversation Management**: 
   - Save/load conversation history to SQLite
   - Clear conversation / new chat
   - Export conversation as Markdown
5. **Profile/Model Selector**: Dropdown to switch Agent Profile and LLM model.
6. **Token Usage Display**: Show prompt/completion tokens per message.

### Frontend: AgentNode Component (Workflow Canvas)

Custom node rendering for AgentNode on the vue-flow canvas:
- Shows agent profile name and model
- During execution: animated progress indicator, step counter
- On completion: expandable output preview

### Data Flow: Chat

```
User types "analyze AAPL momentum"
  → AIChatPanel sends via Wails IPC to Go
    → Go AgentOrchestrator.Run()
      → gRPC Chat(messages=[{system:"You are..."}, {user:"analyze AAPL..."}])
        → Python LLM Service → OpenAI/Anthropic API
      ← streaming tokens (SSE)
    → AgentOrchestrator emits each delta to frontend via SSE
  ← AIChatPanel renders with typewriter effect
```

### Data Flow: AgentNode in Workflow

```
[StockUniverseNode] → [OHLCVNode] → [AgentNode(profile=quant_analyst)]
                                           │
                                           ├── result → [ReportNode]
                                           ├── signal → [StrategyNode]
                                           └── analysis → [BacktestNode]

AgentNode execution:
1. Collect upstream inputs (OHLCV data, symbols)
2. Format as user message: "Analyze the following stocks: ... data: ..."
3. Run AgentLoop with profile's tools
4. Agent calls compute_factor → Go forwards to Python FactorService
5. Agent calls run_backtest → Go runs BacktestRunner locally
6. Agent produces final analysis → parsed into typed output ports
```

## Files to Create/Modify

### New Files

**Python:**
| File | Purpose |
|------|---------|
| `python/proto/llm.proto` | LLM service proto definition |
| `python/src/llm/engine.py` | LLM service implementation (was stub) |
| `python/src/llm/providers/__init__.py` | Provider registry |
| `python/src/llm/providers/openai_provider.py` | OpenAI provider |
| `python/src/llm/providers/anthropic_provider.py` | Anthropic provider |
| `python/src/llm/providers/deepseek_provider.py` | DeepSeek provider |
| `python/src/llm/providers/ollama_provider.py` | Ollama provider (local) |
| `python/src/llm/prompt_template.py` | Prompt assembly engine |
| `python/src/skills/__init__.py` | Skill KB loader |
| `python/src/skills/loader.py` | Load skills from filesystem |
| `python/skills/` (15+ md files) | Skill knowledge base |
| `python/tests/test_llm_engine.py` | LLM service tests |
| `python/tests/test_skills.py` | Skill KB tests |

**Go:**
| File | Purpose |
|------|---------|
| `internal/ai/agent.go` | AgentLoop (ReAct), AgentResult types |
| `internal/ai/capability.go` | Capability, CapabilityRegistry |
| `internal/ai/emitter.go` | EventEmitter, SSE push |
| `internal/ai/profile.go` | AgentProfile, ProfileManager, YAML loader |
| `internal/ai/llm_client.go` | gRPC LLM client wrapper |
| `internal/python/llm_client.go` | LLM methods on PythonBridge |
| `internal/workflow/nodes/agent.go` | AgentNode implementation |
| `internal/ai/capabilities/` | Built-in capability implementations |
| `internal/ai/capabilities/quote.go` | quote_lookup, search_symbol |
| `internal/ai/capabilities/ohlcv.go` | get_ohlcv |
| `internal/ai/capabilities/factor.go` | list_factors, compute_factor |
| `internal/ai/capabilities/backtest.go` | run_backtest |
| `internal/ai/capabilities/trading.go` | get_positions, place_paper_order |
| `resources/agent-profiles/quant_analyst.yaml` | Quant analyst profile |
| `resources/agent-profiles/trader.yaml` | Trader profile |
| `resources/agent-profiles/research_assistant.yaml` | Research assistant profile |
| `resources/agent-profiles/general.yaml` | General purpose profile |

**Frontend:**
| File | Purpose |
|------|---------|
| `frontend/src/terminal/panels/AIChatPanel.vue` | Rewrite with SSE, markdown, tool visualization |

### Modified Files

| File | Change |
|------|--------|
| `python/src/server.py` | Register LLMService + SkillService |
| `python/src/llm/engine.py` | Replace stub with full implementation |
| `internal/python/bridge.go` | Add LLMClient to PythonBridge struct |
| `internal/workflow/nodes/register.go` | Register AgentNode |
| `internal/python/proto/` | Add generated llm.pb.go, llm_grpc.pb.go |
| `app.go` | Expose AgentOrchestrator, Chat, ListProfiles to frontend |
| `frontend/src/terminal/panels/registry.ts` | AIChatPanel already registered — no change |
| `CHANGELOG.md` | Phase 4 entries |
| `go.mod` / `go.sum` | May need yaml.v3 for profile parsing |

## Acceptance Criteria

### M1: Python LLM Service
- [ ] Proto defined: `python/proto/llm.proto` with Chat(streaming), ListModels, CountTokens
- [ ] OpenAI provider implemented with streaming
- [ ] Ollama provider implemented (local, no API key needed)
- [ ] Anthropic provider implemented
- [ ] DeepSeek provider implemented
- [ ] PromptTemplate engine assembles system prompt + tools + skills correctly
- [ ] LLMService registered in server.py
- [ ] `python -m pytest tests/test_llm_engine.py` passes (≥5 tests)
- [ ] Generate Go proto code: `llm.pb.go`, `llm_grpc.pb.go`

### M2: Skill Knowledge Base
- [ ] 15+ skill Markdown files across 5 categories
- [ ] Skill loader with frontmatter parsing
- [ ] SearchSkills gRPC endpoint (keyword + category filter)
- [ ] Skills injectable into LLM system prompt
- [ ] `python -m pytest tests/test_skills.py` passes (≥3 tests)

### M3: Go AgentOrchestrator
- [ ] AgentLoop with ReAct (think→act→observe)
- [ ] CapabilityRegistry with 10 built-in capabilities
- [ ] EventEmitter with SSE push to frontend
- [ ] ProfileManager loads 4 YAML profiles
- [ ] PythonBridge extended with LLMClient (streaming Chat)
- [ ] `go test ./internal/ai/... -v` passes (≥8 tests)
- [ ] `go test ./internal/python/... -v -run TestLLM` passes

### M4: AgentNode
- [ ] AgentNode implements BaseNode interface
- [ ] AgentNode registered as "agent" in category "ai"
- [ ] Input ports: prompt (required), context, constraints
- [ ] Output ports: result, analysis, signal
- [ ] Profile selection via params
- [ ] Execute calls AgentOrchestrator.Run()
- [ ] `go test ./internal/workflow/nodes/... -v -run TestAgent` passes

### M5: AIChatPanel Upgrade
- [ ] SSE streaming: tokens appear incrementally (no full-reload on each message)
- [ ] Markdown rendering: code blocks with syntax highlighting, tables, lists
- [ ] Tool call visualization: expandable cards showing tool name + args + result
- [ ] Profile selector dropdown (4 profiles)
- [ ] Model selector (fetched from Python via ListModels)
- [ ] Conversation: new chat, clear, history persisted to sessionStore
- [ ] Token usage display per message
- [ ] Frontend build succeeds with no new errors

### M6: Integration + E2E
- [ ] Full chat flow: frontend → Go → Python LLM → streaming back
- [ ] AgentNode in workflow: OHLCV → Agent → output → downstream node
- [ ] All existing tests still pass (Go + Python + Frontend)
- [ ] Go build succeeds
- [ ] CHANGELOG updated with Phase 4 entries

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| **LLM API costs during testing** | Ollama (local) as primary test provider; mock provider for unit tests |
| **Streaming gRPC complexity** | gRPC server-streaming is well-supported in Python; Go client uses recv() loop |
| **Skill knowledge base grows too large for context window** | Token budget management in PromptTemplate engine; truncate/prioritize by relevance |
| **Tool call argument parsing errors** | Strict JSON Schema validation on both Go and Python sides before execution |
| **Agent infinite loops** | Hard max_steps limit per profile; context cancellation from workflow engine |
| **Python sidecar not running** | AIChatPanel shows clear "Python sidecar required" message; graceful fallback |
| **Provider API differences** | Each provider implements the same abstract interface; prompt assembly normalizes tool descriptions per provider convention |
| **Wails SSE support** | Wails v3 supports events; if SSE not available, fall back to polling on a Go function |
