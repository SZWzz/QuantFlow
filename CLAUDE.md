# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Project spec**: See [../NEW_PROJECT_PROPOSAL.md](../NEW_PROJECT_PROPOSAL.md) for the full architecture and design rationale.

## Project Identity

**QuantFlow Terminal** — 双模式量化金融终端，融合彭博式面板终端 + 可视化工作流编排。

- **技术栈**: Go 1.22+ (Wails v3 桌面壳) / Vue 3 + TypeScript (前端) / SQLite WAL (存储) / Python 3.12+ (gRPC sidecar)
- **目标市场**: A 股 / 港股 / 美股 / 加密（无印度市场）
- **许可证**: AGPL-3.0

## Non-Negotiable Rules

### 1. Spec-Before-Code — 写代码前必须有 Spec 文档

**Every non-trivial code change MUST be preceded by a spec document.** "Non-trivial" means anything beyond:
- Single-line bug fixes (typo, wrong variable name)
- Formatting/whitespace changes
- Adding a single obvious test case

Spec documents live in `docs/specs/` and follow this naming convention:
```
docs/specs/YYYY-MM-DD-<slug>.md
```

A minimal spec must answer:
```markdown
# <Title>

## Motivation
Why is this change needed? What problem does it solve?

## Design
How does it work? Include:
- Data flow (Go struct → SQLite → Vue component or similar)
- New/modified files
- API changes (Go exported functions, gRPC proto changes, Pinia store changes)

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Risks / Trade-offs
What could go wrong? What alternatives were considered?
```

For large features (>5 files changed), the spec should also include a **component interaction diagram** and a **migration plan** if schema changes are needed.

### 1a. Plan-After-Spec — Spec 后必须有实施计划，再执行

**Every spec MUST be followed by an implementation plan before any code is written.** The canonical workflow for any feature phase is:

```
Spec (设计文档) → Plan (实施计划) → Execute (执行)
```

Implementation plans live in `docs/superpowers/plans/` and follow this naming convention:
```
docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md
```

Each plan must:
- Break the feature into bite-sized tasks (each task is one action, 2-5 minutes)
- Specify exact file paths to create/modify
- Include complete code in every step (no "TBD" or "implement later")
- Follow TDD: write failing test → implement → pass → commit
- End each task with a commit

**Execution via subagent-driven-development:** After the plan is approved, dispatch each task as an independent subagent with review gates between tasks. The subagent workflow is: implementer → review (spec compliance + code quality) → fix (if needed) → re-review → next task.

**Multi-phase features:** For features spanning multiple sub-phases (e.g., Phase 10 = 10.1 + 10.2 + 10.3 + 10.4), each sub-phase gets its own plan file. Execute sub-phases sequentially, evaluating each before starting the next.

### 2. Changelog Maintenance — 每次修改必须维护 CHANGELOG

**Every change — no exceptions — must be recorded in `CHANGELOG.md`.** Follow [Keep a Changelog](https://keepachangelog.com/) conventions:

```markdown
## [YYYY.M.D] - YYYY-MM-DD

### Added
- [Scope] Description of new feature

### Changed
- [Scope] Description of change to existing behavior

### Fixed
- [Scope] Description of bug fix

### Removed
- [Scope] Description of removed feature
```

Scopes: `[Terminal]`, `[Workflow]`, `[Engine]`, `[Broker]`, `[MarketData]`, `[AI]`, `[Frontend]`, `[Storage]`, `[Python]`, `[Docs]`

Each entry must be a complete sentence describing **what** changed and **why**.

### 3. Version Date Check — 提交前检查版本日期

Before every commit and push, verify the version matches today's date. The version is defined in:
1. `frontend/package.json` — `"version"` field (format: `YYYY.M.D`)
2. `README.md` — version badge
3. `CHANGELOG.md` — latest version header

Update all three to match today's date if stale.

### 4. Documentation Requirements for Critical Code

**Critical code MUST carry documentation.** "Critical" means:

1. **Financial correctness** — P&L calculations, position sizing, commission/slippage models, market rules (T+1, price limits, stamp duty)
2. **Data integrity** — look-ahead bias prevention, data alignment, survivorship-bias guards
3. **Concurrency** — goroutine/channel synchronization, shared mutable state, mutex ordering
4. **Security** — credential encryption, API key storage, input sanitization
5. **Market-specific logic** — A-share settlement rules, HK Stock Connect, crypto funding rates

Minimum documentation:
- **Package-level docstring**: Purpose, key abstractions, cross-references
- **Exported function/type docstrings**: One-line summary + behavior
- **Inline comments**: Explain *why* for non-obvious decisions (magic numbers, ordering constraints, workarounds)

## Build, Test, and Run Commands

```bash
# Development (requires Go 1.22+, Node 20+, Python 3.12+)
wails dev                           # Live-reload dev mode

# Build
wails build                         # Production build

# Backend tests
cd app && go test ./... -v -count=1

# Frontend tests
cd frontend && npx vitest run

# Python sidecar tests
cd python && python -m pytest tests/ -x -q

# Full check before commit
cd app && go vet ./... && go test ./...
cd frontend && npx vue-tsc --noEmit && npx vitest run
cd python && python -m pytest tests/ -x -q
```

## Architecture Overview

### Dual-Mode Frontend

```
Terminal Mode (default)          Workflow Mode (Ctrl+W)
┌─────────────────────────┐      ┌─────────────────────────┐
│ CommandBar (Ctrl+K)      │      │ vue-flow Canvas          │
│ DockView (停靠面板系统)    │◄────►│ Node Palette (16 类)      │
│ 50+ Panels              │      │ Property Panel          │
│ PushPinBar / StatusBar  │      │ Execution Log           │
└─────────────────────────┘      └─────────────────────────┘
         │        ▲                         │        ▲
         └────────┼─────────────────────────┘        │
                  │  Shared: dataStore + sessionStore │
                  │  Go backend via Wails IPC          │
```

### Go Backend Layers

```
Presentation:   Wails App (Go functions exposed to frontend)
Orchestration:   Workflow Engine (Kahn + goroutine DAG executor)
Domain:          Trading Engine / MarketDataHub / AI Agent / Portfolio
Data:            SQLite WAL (single file) + sync.Map caches
External:        Python gRPC sidecar (ML/factors/LLM)
                 Broker HTTP/WS adapters
                 100+ Data source adapters
```

### Key Design Decisions (from proposal)

- **SQLite only** — no PostgreSQL, no Redis. Desktop app, single user. WAL mode provides sufficient concurrency.
- **Python as sidecar** — gRPC boundary. Python does ML/compute, Go does orchestration. Python is optional (core features work without it).
- **Wails not Tauri** — Go shell, same language as backend, `go build` compiles everything.
- **Vue 3 not React** — vue-flow is the official Vue port of xyflow; Pinia is more structured than Zustand.
- **Dual-mode not workflow-only** — Bloomberg terminal users need instant data access, not drag-and-drop workflows.

### Market Focus

| Market | Settlement | Key Rules | Data Sources | Brokers |
|--------|-----------|-----------|-------------|---------|
| A 股 | T+1 | 涨跌停 ±10%/±20%, 印花税 0.05% | EastMoney, AKShare, TuShare | 富途, 长桥, 华泰 |
| 港股 | T+2 | 港股通, 交收 T+2 | Futu, 新浪 | 富途, 长桥, IBKR |
| 美股 | T+2 | PDT规则, wash sale | Yahoo, Polygon, Alpaca | Alpaca, IBKR, Tradier |
| 加密 | 即时 | 永续资金费率, 强平 | Binance, OKX | Binance, OKX, Bybit |

### Directory Structure

```
quantflow/
├── app/                    # Go backend (Wails app)
│   ├── main.go             #   Wails entry point
│   ├── app.go              #   Exported Go functions
│   └── internal/
│       ├── workflow/       #   Workflow engine
│       ├── trading/        #   Trading engine + brokers
│       ├── market/         #   MarketDataHub + adapters
│       ├── ai/             #   AI agent orchestrator
│       ├── portfolio/      #   Portfolio + risk
│       ├── research/       #   Equity research + sentiment
│       ├── notify/         #   Notification engine
│       ├── schedule/       #   Cron scheduler
│       ├── auth/           #   JWT + local lock
│       ├── storage/        #   SQLite (WAL mode)
│       └── python/         #   gRPC bridge to Python
├── python/                 # Python gRPC sidecar
│   └── src/
│       ├── factor/         #   Factor computation
│       ├── ml/             #   ML (Qlib, PyTorch, RL)
│       ├── llm/            #   LLM inference
│       └── data/           #   Data fetching scripts
├── frontend/               # Vue 3 frontend
│   └── src/
│       ├── terminal/       #   Terminal Mode components
│       │   ├── panels/     #     50+ Bloomberg-style panels
│       │   └── DockView/   #     Docking panel system
│       ├── workflow/       #   Workflow Mode components
│       │   └── canvas/     #     vue-flow canvas
│       ├── stores/         #   Pinia (4 stores)
│       └── lib/            #   i18n, theme, format
├── docs/
│   └── specs/              #   Spec documents (one per change)
├── resources/              #   Icons, templates, agent profiles
├── CHANGELOG.md
└── README.md
```

## Development Workflow

### Before Writing Code

1. Create spec doc: `docs/specs/YYYY-MM-DD-<slug>.md`
2. If the change introduces a new Go package, a new Vue component, or a new Pinia store — diagram the data flow in the spec
3. If the change modifies SQLite schema — include the migration SQL in the spec

### While Writing Code

1. Follow existing patterns (naming, error handling, logging)
2. Go: `slog` for logging, explicit error returns, no panic in library code
3. Vue: Composition API with `<script setup lang="ts">`, Pinia for state
4. Document critical code as you write it (see rule 4 above)
5. Add tests: Go table-driven tests, Vue component tests with vitest

### After Writing Code

1. Run the full check commands above
2. Update `CHANGELOG.md` with all changes
3. Update version date if needed (see rule 3)
4. Commit with a descriptive message referencing the spec doc

## Known Patterns to Follow

### Go: Workflow Node Implementation

```go
// Every node type registers itself via init() or explicit registration
type MyNode struct {
    BaseNode
    Param1 string `json:"param1"`
}

func (n *MyNode) NodeType() string { return "my_node" }
func (n *MyNode) Category() string { return "data" }

func (n *MyNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    // 1. Validate inputs against port definitions
    // 2. Execute logic
    // 3. Return typed outputs
}
```

### Vue: Panel Implementation

```vue
<script setup lang="ts">
// Each panel subscribes to data topics via the dataStore
// Each panel has a [+] button to add itself to a workflow
import { useDataStore } from '@/stores/data'

const props = defineProps<{ symbol: string }>()
const dataStore = useDataStore()

// Subscribe to real-time quotes
const unsub = dataStore.subscribe(`market:quote:${props.symbol}`)
onUnmounted(() => unsub())
</script>
```

> **⚠️ 禁用 `window.confirm()` / `window.alert()`** — Wails v3 的 webview 禁用了同步原生对话框:`confirm()` 直接返回 `false` 且不弹窗,`alert()` 是 no-op。任何 `if (!confirm(...)) return` 守卫会静默中止(典型受害者:所有删除/清空按钮)。
> - 用 `frontend/src/lib/wails.ts` 的异步助手:`confirmDialog(msg)`(返回 `Promise<boolean>`)、`alertDialog(msg)`。
> - **必须 `await`** — polyfill 把 `window.confirm` 也改成了 Promise 版,不带 `await` 的 `if (confirm(...))` 会拿到 truthy Promise 直接放行(等于不确认就执行)。
> - 底层 API:`Dialogs.Question({Title, Message, Buttons:[{Label:"确定",IsDefault:true},{Label:"取消",IsCancel:true}]})` → resolve 被点按钮的 Label(`"确定"` = 确认);`Dialogs.Info(...)` 用于提示。
> - 列表/查询类不受影响(它们不碰 confirm)——若某操作"加载正常但删除/保存无效",优先怀疑 confirm 守卫,而非 Go 绑定签名。

### SQLite: Schema Migration

```go
// Migrations are numbered sequentially, never modified after deployment
var migrations = []struct {
    Version int
    SQL     string
}{
    {1, `CREATE TABLE workflows (...)`},
    {2, `ALTER TABLE workflows ADD COLUMN tags TEXT DEFAULT '[]'`},
}
```

### Python: gRPC Service

```python
# Each service is a separate gRPC endpoint
# Services receive protobuf messages and return protobuf responses
# Services do NOT access SQLite directly — all storage goes through Go
class FactorService(factor_pb2_grpc.FactorServiceServicer):
    async def ComputeFactors(self, request, context):
        # Compute using pandas, return Arrow-encoded DataFrame
        ...
```

## Recurring Reminders

- **消去印度市场** — 本项目不包含任何印度券商/交易所/数据源。市场聚焦: A股 > 港股 > 美股 > 加密。
- **SQLite 是唯一数据库** — 永远不要引入 PostgreSQL/Redis 依赖。进程内缓存用 `sync.Map` 或 channel。
- **Terminal 和 Workflow 共享底层** — 新面板必须同时考虑是否有对应的 workflow node，反之亦然。
- **Python 是可选 sidecar** — 核心交易/行情/工作流功能必须纯 Go 可用。Python 只用于 ML/因子/LLM。
- **禁用 `window.confirm()` / `window.alert()`** — Wails v3 webview 把它们禁用了(`confirm` 直接返回 `false`)。面板里的确认/提示一律用 `@/lib/wails` 的 `confirmDialog` / `alertDialog` 并 `await`。详见上文「Vue: Panel Implementation」。
