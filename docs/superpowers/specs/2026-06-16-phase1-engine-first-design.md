# QuantFlow Phase 1: Engine-First — Design Doc

> **Status**: Approved  
> **Date**: 2026-06-16  
> **Supersedes**: NEW_PROJECT_PROPOSAL.md §9 阶段 1 (原方案)  
> **Constraint**: Solo developer, 业余时间, Engine-first priority  

## Motivation

NEW_PROJECT_PROPOSAL.md 的原始阶段 1 假设 3-4 人团队在 12 周内并行推进 Go 骨架 + 工作流引擎 + Terminal UI + Workflow UI。单人业余场景下此方案不可行：

1. **上下文切换成本过高** — 同时接触 Wails、Vue 3、vue-flow、Go workflow 四个陌生领域
2. **前端先行的返工** — 在没有后端数据时做的 UI mock，后期对接真实引擎时大量重写
3. **核心价值交付过晚** — 工作流引擎（QuantFlow 的核心差异点）到第 4 周才开始

本设计将阶段 1 重构为 **Engine-First**：纯 Go 工作流引擎，CLI 驱动，零前端依赖。M5 完成时即具备完整 CLI 运行能力。

## Design

### Overview

```
Phase 1: Engine-First (5 milestones, serial)
─────────────────────────────────────────────
M1: 项目地基       →  Go module, config, SQLite
M2: 节点系统       →  BaseNode, Registry, 5 impl nodes
M3: DAG 执行引擎   →  Kahn, goroutine parallel, channel passing  ★ CORE
M4: 持久化与恢复   →  SQLite storage, versioning, breakpoint
M5: CLI + 测试     →  qf CLI, 80% coverage, benchmarks
─────────────────────────────────────────────
Phase 2 (future): Wails shell, vue-flow UI, data sources
```

### Milestone Details

#### M1: Go 项目地基

**What**: 最小可构建骨架，无业务逻辑。

- Go module: `quantflow`
- Directory layout:
  ```
  app/
  ├── main.go                 # 入口（当前仅打印启动信息）
  ├── internal/
  │   ├── config/             # Viper + YAML 配置管理
  │   ├── storage/            # SQLite WAL 连接 + 迁移框架
  │   └── logging/            # slog 封装
  └── go.mod / go.sum
  ```
- SQLite: WAL 模式，迁移框架（`var migrations []struct{Version int; SQL string}`）
- Makefile: `make build`, `make test`, `make lint`
- CI 就绪：`go vet`, `go test ./...`

**Verification**: `go build && go test ./...` 全绿

**Files**: `go.mod`, `main.go`, `internal/config/`, `internal/storage/`, `internal/logging/`, `Makefile`

---

#### M2: 节点系统

**What**: 工作流引擎的原子单元 — 节点接口、端口类型、注册机制。

**Core types**:

```go
// PortType defines the data type flowing through ports
type PortType string
const (
    PortOHLCV    PortType = "ohlcv"
    PortSeries   PortType = "series"    // []float64
    PortSignal   PortType = "signal"    // buy/sell/hold + confidence
    PortString   PortType = "string"
    PortAny      PortType = "any"
)

// PortDefinition describes an input or output port
type PortDefinition struct {
    Name     string
    Type     PortType
    Required bool
}

// BaseNode is the interface every node must implement
type BaseNode interface {
    ID() string
    NodeType() string      // "data_loader", "sma", "cross_signal", etc.
    Category() string      // "data", "indicator", "signal", "output", "control"
    InputPorts() []PortDefinition
    OutputPorts() []PortDefinition
    ParamSchema() []ParamDef
    Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error)
    Validate() error       // validate params against schema
}

// NodeRegistry manages node type registration
type NodeRegistry struct { ... }
func (r *NodeRegistry) Register(constructor NodeConstructor)
func (r *NodeRegistry) Create(nodeType string, id string, params map[string]any) (BaseNode, error)
func (r *NodeRegistry) ListByCategory(category string) []NodeMeta
```

**First 5 node implementations** (1 per category):

| Node | Category | Purpose |
|------|----------|---------|
| DataLoaderNode | data | Load OHLCV from SQLite/CSV → emits `ohlcv` |
| SMANode | indicator | SMA(N) over input series → emits `series` |
| CrossSignalNode | signal | Fast/Slow MA cross → emits `signal` |
| LogOutputNode | output | Print inputs to stdout/log |
| LoopNode | control | Iterate over array input, execute sub-DAG |

**Design decisions**:
- Port types are string enums, not Go types — avoids generics complexity while maintaining type safety via validation
- Nodes use `map[string]any` for data passing — pragmatic for a heterogeneous DAG. Strong typing can be layered on later
- `NodeRegistry` uses factory pattern — each node type registered via `init()` or explicit call

**Verification**: Register → Create → Validate → Execute (in isolation) unit tests pass

**Files**: `internal/workflow/node.go`, `internal/workflow/registry.go`, `internal/workflow/nodes/*.go`

---

#### M3: DAG 执行引擎 ★ 核心交付

**What**: 加载一个有向无环图，按拓扑顺序分层并行执行节点。

**Core types**:

```go
// Workflow represents an executable DAG
type Workflow struct {
    ID      string
    Name    string
    Nodes   []NodeInstance   // node instances with wiring
    Edges   []Edge           // connections between ports
}

type NodeInstance struct {
    ID       string
    NodeType string
    Params   map[string]any
}

type Edge struct {
    FromNode   string
    FromPort   string
    ToNode     string
    ToPort     string
}

// Engine executes workflows
type Engine struct {
    registry  *NodeRegistry
    cache     *lru.Cache   // node result cache
}

func (e *Engine) Execute(ctx context.Context, wf *Workflow) (*ExecutionResult, error)
func (e *Engine) Validate(wf *Workflow) error  // DAG cycle check + port type match
```

**Algorithm**:

1. **Validate**: Detect cycles (DFS), verify all port connections are type-compatible
2. **Topo sort**: Kahn's algorithm → layers (nodes at same depth can run in parallel)
3. **Execute by layer**: For each layer, launch goroutines for all nodes. Each node receives input via channel from upstream nodes. Collect results, feed to next layer.
4. **Error handling**: If any node fails, cancel context → all sibling goroutines stop. Return partial results + error.
5. **Caching**: LRU cache keyed by (nodeID, inputHash) — re-running same workflow skips unchanged nodes.

**CLI runner** (minimal):

```bash
$ qf run workflow.json
[qf] Validating...
[qf] Topo sort: 3 layers, 5 nodes
[qf] Layer 1: [data_loader:load_aapl] ✓ (120ms)
[qf] Layer 2: [sma:ma_fast] ✓ (5ms) | [sma:ma_slow] ✓ (4ms)
[qf] Layer 3: [cross_signal:signal] ✓ (2ms) | [log_output:print] ✓ (0ms)
[qf] Done. 5/5 nodes succeeded.
```

**Concurrency model**:
- Each layer's nodes execute in parallel via `errgroup`
- Inter-layer data flow: upstream node writes to `sync.Map`, downstream reads
- Context cancellation propagates through all goroutines
- Single `Engine` instance is goroutine-safe (multiple workflows can run concurrently)

**Verification**: CLI `qf run examples/sma_cross.json` produces expected output

**Files**: `internal/workflow/engine.go`, `internal/workflow/dag.go`, `internal/workflow/cache.go`, `cmd/qf/main.go`

---

#### M4: 持久化与恢复

**What**: 工作流可保存到 SQLite，可回读，可版本管理。

**SQLite schema**:

```sql
CREATE TABLE workflows (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE workflow_versions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT NOT NULL REFERENCES workflows(id),
    version     INTEGER NOT NULL,
    graph_json  TEXT NOT NULL,          -- full Workflow JSON
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(workflow_id, version)
);

CREATE TABLE execution_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id TEXT NOT NULL,
    version     INTEGER NOT NULL,
    started_at  TEXT,
    finished_at TEXT,
    status      TEXT NOT NULL,          -- 'running' | 'success' | 'failed'
    result_json TEXT,                    -- ExecutionResult JSON
    error_msg   TEXT
);
```

**Operations**:
- `SaveWorkflow(wf *Workflow) error` — upsert workflows row, insert new workflow_versions row
- `LoadWorkflow(id string, version *int) (*Workflow, error)` — latest or specific version
- `ListWorkflows() ([]WorkflowMeta, error)`
- `SaveExecution(workflowID string, version int, result *ExecutionResult) error`

**Breakpoint recovery**:
- Before executing each node, save its input to `execution_checkpoints` table
- On resume after crash, skip nodes with matching (nodeID, inputHash) in checkpoints
- Checkpoints are per-execution-run, cleaned up on successful completion

**Verification**: Save workflow → restart app → load → execute → verify same result

**Files**: `internal/storage/workflow_repo.go`, `internal/storage/migrations/002_workflows.sql`

---

#### M5: CLI 工具 + 测试覆盖

**What**: 完整的开发者 CLI 和测试套件。

**CLI** (via `cmd/qf/main.go`):

```bash
qf run <file>            # Load and execute a workflow JSON file
qf run --id <uuid>       # Load from DB and execute
qf nodes                  # List all registered node types by category
qf validate <file>        # Validate workflow JSON without executing
qf version                # Print version
```

**Test suite**:

| Layer | Target | Tool |
|-------|--------|------|
| Unit | All exported functions | `go test` |
| Integration | Full DAG execution with real nodes | `go test -tags=integration` |
| Bench | 100-node DAG execution time | `go test -bench=.` |
| Race | Goroutine safety | `go test -race` |

**Coverage target**: 80%+ for `internal/workflow/` package

**Example workflows** (shipped with repo):

```
examples/
├── sma_cross.json         # DataLoader → SMA(fast) + SMA(slow) → CrossSignal → LogOutput
├── multi_asset.json       # Loop[3 assets] → SMA → Merge → LogOutput
└── error_handling.json    # Demonstrates node failure + partial results
```

**Verification**: `make test && make bench` 全绿, coverage report shows 80%+

**Files**: `cmd/qf/main.go`, `internal/workflow/*_test.go`, `examples/*.json`

### Out of Scope for Phase 1

| Item | Reason | Future Phase |
|------|--------|-------------|
| Wails v3 桌面壳 | 纯 Go 引擎不需要桌面壳来运行 | Phase 2 (UI) |
| DockView 停靠面板 | 前端组件，依赖 Wails | Phase 2 |
| vue-flow 画布 | 可视化编辑器 | Phase 2 |
| Terminal Mode 面板 | 行情/交易面板，依赖数据源 | Phase 2+ |
| Python gRPC sidecar | ML/因子/LLM 计算 | Phase 3 |
| 真实行情连接 | 数据源适配器 | Phase 2 |
| 券商集成 | 交易执行 | Phase 2 |
| Notification | 通知渠道 | Phase 4 |

### Data Flow (M3 in action)

```
workflow.json
    │
    ▼
┌────────────────────────────────────────────┐
│  Engine.Execute(ctx, workflow)              │
│                                             │
│  1. Validate(workflow)                      │
│     ├── Cycle check (DFS)                   │
│     └── Port type matching                  │
│                                             │
│  2. TopoSort(nodes, edges) → layers[]       │
│     e.g. [[DataLoader], [SMA_fast, SMA_slow],│
│            [CrossSignal, LogOutput]]        │
│                                             │
│  3. For each layer:                         │
│     errgroup.Go(func() {                    │
│       inputs := collectFromUpstream(sync.Map)│
│       outputs, err := node.Execute(inputs)  │
│       storeToUpstream(nodeID, outputs)      │
│     })                                      │
│                                             │
│  4. Return ExecutionResult{Status, Nodes}   │
└────────────────────────────────────────────┘
```

### Component Interaction

```
┌──────────────────────────────────────────────────┐
│                   cmd/qf (CLI)                     │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │ run      │  │ nodes    │  │ validate       │  │
│  └────┬─────┘  └────┬─────┘  └───────┬────────┘  │
│       │              │                │            │
│       └──────────────┼────────────────┘            │
│                      │                             │
│              ┌───────┴───────┐                     │
│              │  Engine API   │                     │
│              └───────┬───────┘                     │
└──────────────────────┼────────────────────────────┘
                       │
┌──────────────────────┼────────────────────────────┐
│         internal/workflow/ (engine package)        │
│                      │                             │
│  ┌───────────────────┼───────────────────┐        │
│  │ Engine                                │        │
│  │  ├── Validate(dag)                    │        │
│  │  ├── Execute(dag) → goroutine layers  │        │
│  │  └── cache: LRU(nodeID×input → output)│        │
│  └───────┬───────────┬───────────────────┘        │
│          │            │                            │
│  ┌───────┴──┐  ┌──────┴────────┐                  │
│  │ Registry │  │ nodes/        │                  │
│  │ · List   │  │ · data_loader │                  │
│  │ · Create │  │ · sma         │                  │
│  └──────────┘  │ · cross_signal│                  │
│                │ · log_output  │                  │
│                │ · loop        │                  │
│                └───────────────┘                  │
│                                                   │
└──────────────────────┬────────────────────────────┘
                       │
┌──────────────────────┼────────────────────────────┐
│       internal/storage/                            │
│  ┌──────────────────────┐  ┌──────────────────┐   │
│  │ WorkflowRepo         │  │ Migrations       │   │
│  │ · Save/Load/List     │  │ · 001_init       │   │
│  │ · SaveVersion        │  │ · 002_workflows  │   │
│  │ · SaveExecution      │  │ · 003_checkpoints│   │
│  └──────────────────────┘  └──────────────────┘   │
└───────────────────────────────────────────────────┘
```

## Acceptance Criteria

- [ ] M1: `go build && go test ./...` 全绿
- [ ] M2: NodeRegistry 可注册/列举/创建所有 5 种节点，单元测试覆盖
- [ ] M3: `qf run examples/sma_cross.json` 输出正确的执行日志和结果
- [ ] M4: 工作流保存到 SQLite → 重启 → 加载 → 执行，结果一致。断点恢复可用。
- [ ] M5: `make test` 80%+ 覆盖率，`make bench` 无性能回归。3 个 example 全部可运行。
- [ ] `go vet ./...` 零警告
- [ ] `go test -race ./...` 零竞态
- [ ] CHANGELOG.md 记录所有变更

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| `map[string]any` 失去类型安全 | Port type validation at graph construction time catches mismatches. Strong typing can be added later with generics if needed |
| 不引入前端导致后期集成困难 | Engine 是纯 Go 包，暴露干净的 Go API。Wails 绑定只需调用 Go 函数——与前端无耦合 |
| LRU 缓存命中率低 | 先实现最简单的 LRU，M3 验证实际命中率后再决定是否需要 content-addressed cache |
| 单人业余时间不可预测 | 里程碑驱动而非日历驱动——不设期限，按顺序推进 |

## Alternatives Considered

1. **Proposal 原方案（4 线并行）**: 被否 — 不适合 solo 场景。UI 和引擎同时推进导致上下文切换过多。
2. **Thin Vertical Slice（每层只做一点）**: 被否 — 频繁切换 Go/Vue/Python 语言，每层深度不够。
3. **Shell-First（先做 Wails UI）**: 被否 — 用户明确表示"能跑的工作流引擎"是核心目标。
4. **Engine-First（本方案）**: 选用 — 纯 Go，零前端，最快交付核心价值。
