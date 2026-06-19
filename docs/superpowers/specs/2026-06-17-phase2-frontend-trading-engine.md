# QuantFlow Phase 2: Frontend + Trading Engine — Design Doc

> **Status**: Draft  
> **Date**: 2026-06-17  
> **Supersedes**: NEW_PROJECT_PROPOSAL.md §9 阶段 1.3-1.4 + 阶段 2.1-2.2  
> **Constraint**: Solo developer, 业余时间, builds on Phase 1 workflow engine  
> **Phase 1 delivers**: Pure-Go workflow engine, 5 nodes, SQLite persistence, CLI runner

## Motivation

Phase 1 交付了可 CLI 运行的工作流引擎，但它是一把没有刀柄的刀刃——用户看不到 DAG 拓扑，无法拖拽节点，无法实时查看行情，无法执行交易。Phase 2 的目标是**让引擎有脸有手**：

1. **脸**：Vue 3 前端，Terminal Mode（彭博式面板）+ Workflow Mode（vue-flow 画布）
2. **手**：Go 交易引擎 + 市场数据中枢，让策略不只是回测，能真正对接行情和下单

Phase 2 完成后，QuantFlow 将具备：用 vue-flow 编排策略 → 连接实时行情 → 模拟交易 → 在终端面板监控持仓/收益 的完整闭环。

## Design

### Overview

```
Phase 2: Frontend + Trading Engine (6 milestones, serial)
─────────────────────────────────────────────────────────
M1: Wails 骨架 + 前端地基    →  Wails v3 project, Vue 3 + Vite, Go ↔ JS bridge
M2: Terminal Mode 核心       →  CommandBar, DockView, 8 core panels
M3: Workflow Mode 核心       →  vue-flow canvas, NodePalette, PropertyPanel, ExecutionLog
M4: 双模式集成               →  Terminal ↔ Workflow 双向流动, Pinia stores
M5: 交易引擎                 →  Bar-by-bar pipeline, OMS, PaperEngine, OrderMatcher
M6: 市场数据中枢             →  Go channel Hub, 8 data adapters, L0→L1→L2 cache
─────────────────────────────────────────────────────────
Phase 3 (future): Python gRPC sidecar, factor computation, backtesting
```

### Architecture: How Phase 2 Extends Phase 1

```
Phase 1 (existing)                    Phase 2 (this spec)
─────────────────────                 ─────────────────────
app/main.go (standalone)      →       app/main.go (Wails entry)
app/internal/workflow/*       →       (unchanged, reused as-is)
app/internal/storage/*        →       (unchanged, reused as-is)
cmd/qf/main.go (CLI)          →       (kept, now coexists with GUI)
        —                       →       app/app.go (Wails-bound Go functions)
        —                       →       app/internal/trading/*
        —                       →       app/internal/market/*
        —                       →       frontend/* (Vue 3 + TypeScript)
```

**Key principle**: Phase 1 的 `workflow` 和 `storage` 包不做任何修改，Phase 2 只新增包和前端代码。CLI 工具 `qf` 保留，与 GUI 共存。

### Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                        Frontend (Vue 3)                           │
│                                                                    │
│  Terminal Mode          Workflow Mode          Shared             │
│  ┌─────────────────┐   ┌─────────────────┐   ┌──────────────┐   │
│  │ CommandBar      │   │ vue-flow Canvas │   │ ECharts      │   │
│  │ DockView        │   │ NodePalette     │   │ Monaco       │   │
│  │ Panels (8+)     │   │ PropertyPanel   │   │ Spreadsheet  │   │
│  │ PushPinBar      │   │ ExecutionLog    │   │ Theme        │   │
│  └────────┬────────┘   └────────┬────────┘   └──────┬───────┘   │
│           │                     │                    │            │
│           └─────────────────────┼────────────────────┘            │
│                                 │ Wails IPC (Go func calls)       │
└─────────────────────────────────┼──────────────────────────────────┘
                                  │
┌─────────────────────────────────┼──────────────────────────────────┐
│                     Go Backend (app/)                              │
│                                 │                                   │
│  ┌──────────────────────────────┼───────────────────────────┐     │
│  │                    app.go    │  Wails-bound functions     │     │
│  │  RunWorkflow()  ListNodes()  │  Subscribe()  PlaceOrder() │     │
│  └──────────┬──────────┬────────┴──────────┬────────────────┘     │
│             │           │                    │                      │
│  ┌──────────┴──┐  ┌─────┴──────────┐  ┌─────┴──────────┐         │
│  │ Workflow    │  │ MarketDataHub  │  │ Trading Engine │         │
│  │ Engine      │  │ (NEW)          │  │ (NEW)          │         │
│  │ (Phase 1)   │  │                │  │                │         │
│  │             │  │ Go channel     │  │ Bar-by-bar     │         │
│  │ DAG exec    │  │ pub/sub        │  │ pipeline       │         │
│  │ TopoSort    │  │ TTL cache      │  │ OMS            │         │
│  │ LRU cache   │  │ 8 adapters     │  │ PaperEngine    │         │
│  └──────┬──────┘  └──────┬─────────┘  └──────┬─────────┘         │
│         │                │                    │                    │
│  ┌──────┴────────────────┴────────────────────┴──────────┐       │
│  │              SQLite WAL (Phase 1 storage)              │       │
│  │  workflows | workflow_versions | execution_history     │       │
│  │  + trades (NEW) | + ohlcv_cache (NEW)                  │       │
│  └───────────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────────┘
```

### Pinia Stores (4 stores, shared across both modes)

```
terminalStore           workflowStore           dataStore           sessionStore
─────────────────────   ─────────────────────   ─────────────────   ─────────────────
layout: DockLayoutTree  canvas: {nodes,edges}   quotes: Map<sym,Q>  user: {token,profile}
activePanels: Map<>     runner: {status,prog}   ohlcv: Map<key,OH>  ui: {theme,density}
commandHistory: []      templates: []           subs: Map<topic,[]>  mode: term|workflow
pushPins: []            clipboard: {nodes}      sourceStatus: []     brokers: []
```

### Directory Structure — Phase 2 Additions

```
quantflow/
├── frontend/                       # ★ NEW: Vue 3 前端
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.ts                 # Vue 入口, Pinia, Wails runtime
│       ├── App.vue                 # 顶层路由 (Terminal | Workflow)
│       ├── terminal/               # Terminal Mode
│       │   ├── TerminalMode.vue    #   终端模式容器
│       │   ├── CommandBar.vue      #   命令栏 (Ctrl+K)
│       │   ├── DockView/           #   停靠面板系统
│       │   │   ├── DockView.vue    #     顶层容器
│       │   │   ├── DockContainer.vue  #  递归分割容器
│       │   │   ├── DockTab.vue     #     标签页 + 面板渲染
│       │   │   └── DockSplitter.vue #    分割条拖拽
│       │   ├── PushPinBar.vue      #   固定栏
│       │   ├── StatusBar.vue       #   状态栏
│       │   └── panels/             #   面板组件
│       │       ├── WatchlistPanel.vue
│       │       ├── QuoteDetailPanel.vue
│       │       ├── CandlestickPanel.vue
│       │       ├── OrderEntryPanel.vue
│       │       ├── PositionPanel.vue
│       │       ├── NewsPanel.vue
│       │       ├── AIChatPanel.vue
│       │       └── SystemMonitorPanel.vue
│       ├── workflow/               # Workflow Mode
│       │   ├── WorkflowMode.vue    #   工作流模式容器
│       │   ├── canvas/             #   vue-flow 画布
│       │   │   ├── WorkflowCanvas.vue
│       │   │   ├── CustomNode.vue  #     自定义节点渲染
│       │   │   └── ConnectionLine.vue
│       │   ├── NodePalette.vue     #   节点面板 (搜索+分类)
│       │   ├── PropertyPanel.vue   #   属性编辑面板
│       │   └── ExecutionLog.vue    #   实时执行日志
│       ├── stores/                 # Pinia stores
│       │   ├── terminal.ts
│       │   ├── workflow.ts
│       │   ├── data.ts
│       │   └── session.ts
│       ├── lib/                    # 共享工具
│       │   ├── wails.ts            #   Wails runtime 封装
│       │   ├── format.ts           #   数字/日期格式化
│       │   └── theme.ts            #   主题切换
│       └── types/                  # TypeScript 类型定义
│           ├── workflow.ts         #   Workflow, Node, Edge (与 Go 结构对齐)
│           ├── market.ts           #   Quote, OHLCVBar, Order
│           └── panel.ts            #   PanelDefinition
│
├── app/                            # Go 后端 (扩展)
│   ├── main.go                     # 改造: Wails 入口
│   ├── app.go                      # ★ NEW: Wails-bound Go functions
│   └── internal/
│       ├── trading/                # ★ NEW: 交易引擎
│       │   ├── engine.go           #   TradingEngine — bar-by-bar pipeline
│       │   ├── engine_test.go
│       │   ├── signal_adapter.go   #   Tick/Batch signal → order conversion
│       │   ├── signal_adapter_test.go
│       │   ├── risk_pipeline.go    #   风控管线 (止损/止盈/追踪止损)
│       │   ├── risk_pipeline_test.go
│       │   ├── oms.go              #   订单管理系统 (Order Management)
│       │   ├── oms_test.go
│       │   ├── paper_engine.go     #   模拟交易引擎
│       │   ├── paper_engine_test.go
│       │   ├── order_matcher.go    #   撮合引擎
│       │   ├── order_matcher_test.go
│       │   └── types.go            #   Order, Position, Trade, Signal 等类型
│       ├── market/                 # ★ NEW: 市场数据中枢
│       │   ├── hub.go              #   MarketDataHub — Go channel pub/sub
│       │   ├── hub_test.go
│       │   ├── cache.go            #   TTL 内存缓存 (sync.Map)
│       │   ├── cache_test.go
│       │   ├── normalize.go        #   数据归一化 (OHLCV/Quote/Tick schemas)
│       │   ├── normalize_test.go
│       │   ├── adapters/           #   数据源适配器
│       │   │   ├── adapter.go      #     Adapter 接口定义
│       │   │   ├── yahoo.go        #     Yahoo Finance
│       │   │   ├── yahoo_test.go
│       │   │   ├── eastmoney.go    #     EastMoney (A 股)
│       │   │   ├── eastmoney_test.go
│       │   │   └── binance.go      #     Binance (加密)
│       │   │   └── binance_test.go
│       │   └── sqlite_cache.go     #   L1 SQLite ohlcv_cache
│       └── workflow/               # (Phase 1, unchanged)
│
└── wails.json                      # ★ NEW: Wails 项目配置
```

---

## Milestone Details

### M1: Wails 骨架 + 前端地基

**What**: 让前端能启动、Go 能响应前端调用。最小可构建的 Wails + Vue 3 应用。

**Wails v3 Setup**:
- `wails init` 或手动搭建 Wails v3 项目结构
- `wails.json` 配置文件（应用名、窗口大小、前端路径）
- `app/main.go` 改造为 Wails 入口（创建窗口、挂载 App struct）
- `app/app.go` — 挂载到 Wails 的 Go struct，暴露函数给前端
- 前端 `index.html` + `main.ts` Vue 3 入口

**Vue 3 + Vite Scaffold**:
- TypeScript strict mode
- Pinia 安装 + 4 store 骨架
- vue-router (Terminal vs Workflow 顶层路由)
- ECharts (vue-echarts), Monaco Editor, vue-flow 依赖安装
- Tailwind CSS 或 Naive UI 组件库选择

**Go ↔ JS Bridge**:
```go
// app/app.go — Wails 暴露的函数签名示例
type App struct {
    engine   *workflow.Engine
    registry *workflow.NodeRegistry
    // ... trading, market 组件在后续 milestone 添加
}

// 前端可调用: Go 函数名 = JS camelCase 版本
func (a *App) RunWorkflow(jsonDef string) (*workflow.ExecutionResult, error)
func (a *App) ListNodes() []workflow.NodeMeta
func (a *App) ValidateWorkflow(jsonDef string) error
func (a *App) GetVersion() string
```

**New Dependencies**:
- Go: `github.com/wailsapp/wails/v3`
- JS: `vue`, `pinia`, `vue-router`, `@vue-flow/core`, `vue-echarts`, `echarts`, `monaco-editor`

**Verification**: `wails dev` 启动 → 浏览器看到 "QuantFlow Terminal" 标题 → 前端调用 `GetVersion()` 返回版本号

---

### M2: Terminal Mode 核心

**What**: 彭博式面板终端。CommandBar 键盘驱动 + DockView 停靠系统 + 首批面板。

**CommandBar** (`Ctrl+K`):
- 浮动搜索栏，模糊匹配面板名称/股票代码/命令
- 输入 "AAPL" → 打开 QuoteDetailPanel + CandlestickPanel
- 输入 "/wf my-strat" → 切换到 Workflow Mode
- 历史记录 (最近 20 条)，↑↓ 导航
- 键盘快捷键: `Ctrl+K` 激活, `Esc` 关闭

**DockView — 递归停靠系统**:
```
DockView
├── DockContainer (递归)
│   ├── direction: 'row' | 'column'
│   ├── children: (DockContainer | DockTab)[]
│   └── splitRatios: number[]  (拖拽调整)
└── DockTab
    ├── tabs: { name, component, icon, dirty }[]
    ├── activeTab: string
    └── actions: close | float | pin | tearOff
```

- 预置布局: `single` | `split-h` | `split-v` | `2x2` | `classic` | `trading`
- 面板可拖拽移动/停靠/浮动
- 分割条拖拽调整大小
- 布局保存到 `sessionStore` → 持久化到 SQLite

**首批 8 面板** (Minimal Viable Panel Set):

| 面板 | 功能 | 数据来源 |
|------|------|---------|
| `WatchlistPanel` | 自选列表，实时报价条 | dataStore.quotes |
| `QuoteDetailPanel` | 单股详细报价 (OHLCV, 成交量, 换手率) | dataStore.quotes |
| `CandlestickPanel` | K 线图 (ECharts)，支持 MA/BOLL 叠加 | dataStore.ohlcv |
| `OrderEntryPanel` | 下单面板 (symbol, side, qty, type) | → trading.PlaceOrder |
| `PositionPanel` | 当前持仓列表 (symbol, qty, avgPrice, P&L) | trading OMS |
| `NewsPanel` | 新闻流 (标题 + 时间, 按 symbol 过滤) | dataStore (future: adapter) |
| `AIChatPanel` | AI 对话面板 (流式 SSE) | future: gRPC Python |
| `SystemMonitorPanel` | CPU/内存/数据源状态/工作流状态 | Go runtime stats |

**面板协议** — 所有面板遵循统一接口:
```typescript
interface PanelDefinition {
    id: string
    name: string
    icon: string
    component: Component
    defaultLayout: { width: number, height: number }
    minSize: { width: number, height: number }
    subscribes?: string[]  // dataStore topics
    params?: Record<string, any>  // e.g., { symbol: 'AAPL' }
}
```

**PushPinBar + StatusBar**:
- PushPinBar: 固定面板/符号的快捷入口，跨布局持久
- StatusBar: 连接状态 | 券商在线数 | 数据流数 | 内存 | 离线模式指示

**Verification**: `wails dev` → Terminal Mode 完整渲染 → 打开 Watchlist + Candlestick → 切换布局 → CommandBar 搜索打开面板

---

### M3: Workflow Mode 核心

**What**: 可视化工作流画布。拖拽节点 → 连线 → 配置参数 → 执行 → 看日志。

**vue-flow Canvas**:
- 从 `workflowStore` 渲染 nodes + edges
- 自定义节点组件 `CustomNode.vue`:
  - 显示节点名称、类型图标、输入/输出端口圆点
  - 端口颜色按 PortType: ohlcv=绿, series=蓝, signal=红, any=灰
  - 执行状态指示: idle → running (脉冲动画) → success (绿边框) / failed (红边框)
- 拖拽从 NodePalette 添加新节点
- 连线: 端口到端口，类型不匹配时红色警告
- MiniMap + 缩放控制
- 画布状态 (viewport, node positions) 随 workflow 一同保存

**NodePalette**:
- 左侧面板，按 Category 分组 (data, indicator, signal, output, control)
- 搜索过滤
- 拖拽或双击添加到画布
- 显示每个节点的输入/输出端口信息
- 分类通过 `registry.ListAll()` 从 Go 获取

**PropertyPanel**:
- 右侧面板，选中节点时显示其参数表单
- 动态表单: 根据 `ParamSchema()` 生成 (int → number input, string → text input, string_array → tag editor)
- 修改即时反映到 `workflowStore`
- 显示节点 ID、NodeType、Category 元信息

**ExecutionLog**:
- 底部/侧边面板，显示工作流执行日志
- 实时流: `[node_id] ✓ success (123µs)` 或 `[node_id] ✗ error: ...`
- 按 layer 分组显示并行执行批次
- 支持复制日志、清空

**画布操作**:
- `F5` 执行完整工作流
- `Ctrl+Z` / `Ctrl+Shift+Z` 撤销/重做
- `Delete` 删除选中节点/连线
- 右键菜单: 复制节点 / 禁用节点 / 固定到终端

**Verification**: `wails dev` → 切换到 Workflow Mode → 从 Palette 拖入 DataLoader + SMA → 连线 → 配置参数 → 点击 Run → ExecutionLog 显示执行结果

---

### M4: 双模式集成

**What**: Terminal ↔ Workflow 双向流动 + Pinia stores 完整实现。

**Terminal → Workflow**:
- 任意面板右上角 `[⊕ 添加到工作流]` 按钮
- 将面板的当前状态 (symbol, params) 创建为对应的工作流节点
- 自动切换到 Workflow Mode，新节点已在画布中央
- 映射表: `WatchlistPanel → StockUniverseNode`, `CandlestickPanel → MarketDataNode`, `OrderEntryPanel → OrderNode`

**Workflow → Terminal**:
- 节点右键 → `[固定到终端]`
- 在 Terminal Mode 创建对应的监控面板
- 面板标题标注 "WF: <工作流名称>"
- 每次工作流执行结束，面板数据自动刷新

**Pinia Stores 实现**:

```typescript
// terminalStore — 终端状态
terminalStore = {
    layout: LayoutTree,           // DockView 布局树 (JSON serializable)
    activePanels: Map<string, PanelState>,  // 当前打开的面板实例
    commandHistory: string[],     // 最多 20 条
    pushPins: PushPin[],          // { id, label, type, payload }
    focusMode: boolean,
    // Actions
    openPanel(panelId, params): void,
    closePanel(instanceId): void,
    saveLayout(name): void,
    loadLayout(name): void,
}

// workflowStore — 工作流状态
workflowStore = {
    nodes: Node[],                // vue-flow nodes
    edges: Edge[],                // vue-flow edges
    viewport: Viewport,
    executionStatus: 'idle' | 'running' | 'completed' | 'failed',
    nodeStatuses: Map<string, NodeExecStatus>,
    runId: string | null,
    templates: WorkflowTemplate[],
    clipboard: { nodes: Node[] } | null,
    // Actions
    addNode(type, position): void,
    removeNode(id): void,
    addEdge(edge): void,
    runWorkflow(): Promise<void>,
    loadFromJSON(json): void,
    saveToJSON(): string,
}

// dataStore — 统一数据层
dataStore = {
    quotes: Map<string, QuoteSnapshot>,
    ohlcv: Map<string, OHLCVCache>,
    subscriptions: Map<string, Set<string>>,
    sourceStatus: DataSourceStatus[],
    // Actions
    subscribe(topic, panelId): () => void,  // returns unsubscribe
    fetchOHLCV(symbol, interval, range): Promise<OHLCVBar[]>,
}

// sessionStore — 用户 & UI 偏好
sessionStore = {
    user: { token?, profile? },
    ui: { theme: 'light'|'dark', density: 'compact'|'default'|'comfortable', language: 'zh'|'en', mode: 'terminal'|'workflow' },
    brokers: BrokerConnection[],
    // Actions
    setMode(mode): void,
    setTheme(theme): void,
    toggleMode(): void,  // Terminal ↔ Workflow
}
```

**Verification**: Terminal 面板 `[⊕]` → 切换到 Workflow 看到节点 → Run → 右键固定到终端 → 切回 Terminal 看到监控面板

---

### M5: 交易引擎

**What**: Bar-by-bar 事件驱动交易管线。信号 → 风控 → 订单管理 → 模拟撮合。

**Core Types**:
```go
// Signal — 策略节点产出的交易信号
type Signal struct {
    Symbol    string
    Direction string  // "buy" | "sell" | "hold"
    Quantity  float64
    Price     float64 // 0 = market
    Reason    string
    Timestamp time.Time
}

// Order — 订单管理系统中的订单
type Order struct {
    ID            string
    Symbol        string
    Side          string  // "buy" | "sell"
    OrderType     string  // "market" | "limit" | "stop"
    Quantity      float64
    Price         float64
    FilledQty     float64
    FilledAvgPrice float64
    Status        string  // "pending" | "partial" | "filled" | "cancelled" | "rejected"
    PlacedAt      time.Time
    FilledAt      *time.Time
}

// Position — 持仓
type Position struct {
    Symbol     string
    Quantity   float64
    AvgPrice   float64
    MarketPrice float64
    PnL        float64
    PnLPct     float64
}
```

**TradingEngine — Bar-by-Bar Pipeline**:
```go
type TradingEngine struct {
    oms         *OMS
    paperEngine *PaperEngine
    riskPipeline *RiskPipeline
    signalCh    chan Signal
}

// OnBar 是核心方法 — 每个 bar 事件驱动一次完整的交易管线
func (e *TradingEngine) OnBar(ctx context.Context, bar OHLCVBar) error {
    // 1. 风控管线 — 检查止损/止盈/回撤
    // 2. 信号处理 — 将 pending signals 转为 orders
    // 3. OMS — 订单状态机推进
    // 4. PaperEngine — 模拟撮合 (按 bar 的 OHLC 判断成交)
    // 5. 仓位更新 — 重新计算 P&L
}
```

**RiskPipeline**:
- 止损: `currentPrice <= stopLossPrice → close position`
- 止盈: `currentPrice >= takeProfitPrice → close position`
- 追踪止损: trailing stop 跟随最高价
- 最大回撤: `drawdown >= maxDrawdownPct → suspend trading`
- 单标的上限: `positionValue / portfolioValue >= maxSinglePositionPct → reject order`

**OMS (Order Management System)**:
- 订单状态机: `pending → partial → filled / cancelled / rejected`
- 订单簿: 按 symbol 分组，支持查询
- 成交记录: 不可变追加
- 仓位计算: 按 FIFO 匹配成交

**PaperEngine + OrderMatcher**:
```go
type OrderMatcher struct {
    orderBook map[string][]Order  // symbol → pending orders
}

// Match 用当前 bar 数据模拟撮合
// Market order → fill at bar.Open
// Limit order → fill if bar.Low <= limitPrice (buy) or bar.High >= limitPrice (sell)
func (m *OrderMatcher) Match(order Order, bar OHLCVBar) (filledQty float64, avgPrice float64)
```

**SQLite Migration (004_trading.sql)**:
```sql
CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    order_type TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL,
    filled_qty REAL DEFAULT 0,
    filled_avg_price REAL,
    status TEXT NOT NULL DEFAULT 'pending',
    placed_at INTEGER NOT NULL,
    filled_at INTEGER
);

CREATE TABLE trades (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL NOT NULL,
    timestamp INTEGER NOT NULL
);

CREATE TABLE positions (
    symbol TEXT PRIMARY KEY,
    quantity REAL NOT NULL,
    avg_price REAL NOT NULL,
    updated_at INTEGER NOT NULL
);
```

**Verification**: Go test 覆盖 Signal→Order→Match→Position 完整链路 → PaperEngine 用 CSV 数据回放验证 P&L 计算

---

### M6: 市场数据中枢

**What**: Go channel pub/sub 总线，类型安全，多数据源适配，三级缓存。

**MarketDataHub**:
```go
type MarketDataHub struct {
    topics      map[string]*topicBroker
    mu          sync.RWMutex
    adapters    map[string]Adapter
    l1Cache     *SQLiteOHLCVCache
}

// Subscribe 返回一个只读 channel + unsubscribe 函数
func (h *MarketDataHub) Subscribe(topic string, subID string) (<-chan MarketMessage, func()) {
    broker := h.getOrCreateBroker(topic)
    ch := make(chan MarketMessage, 64)
    broker.subscribers[subID] = ch
    // 有缓存数据立即推送
    if cached := broker.latest; cached != nil && !cached.Expired() {
        ch <- cached.msg
    }
    return ch, func() { broker.unsubscribe(subID); close(ch) }
}
```

**Topic 格式**: `market:<type>:<symbol>[:<interval>]`
- `market:quote:AAPL` → 实时报价
- `market:ohlcv:AAPL:1d` → 日 K 线
- `market:orderbook:BTCUSD` → 订单簿深度

**Adapter 接口**:
```go
type Adapter interface {
    Name() string
    Markets() []string            // ["US", "HK", "CN", "CRYPTO"]
    FetchQuote(ctx context.Context, symbol string) (*QuoteSnapshot, error)
    FetchOHLCV(ctx context.Context, symbol string, interval string, start, end time.Time) ([]OHLCVBar, error)
    Subscribe(ctx context.Context, topic string) (<-chan MarketMessage, error)
    HealthCheck(ctx context.Context) error
}
```

**首批 8 Adapters** (覆盖四大市场):

| Adapter | 市场 | 数据类型 |
|---------|------|---------|
| Yahoo | 美股/港股 | Quote + OHLCV |
| EastMoney | A 股 | Quote + OHLCV |
| AKShare | A 股 | Quote + OHLCV + 基本面 |
| TuShare | A 股 | Quote + OHLCV + 财务 |
| Futu | A/HK/US | Quote + OHLCV + 订单簿 |
| Sina | 港股 | Quote (免费) |
| Polygon | 美股 | Quote + OHLCV + 新闻 |
| Binance | 加密 | Quote + OHLCV + 深度 (WebSocket) |

**三级缓存**:
```
L0: sync.Map (内存)
    ├── Quote: TTL 5s
    ├── OHLCV: TTL 60s (最近 100 bars)
    └── 去重窗口: 100ms 合并 (coalesce)

L1: SQLite ohlcv_cache 表
    ├── 按 (symbol, interval, ts) 分区
    ├── TTL: 1m=2h, 5m=1d, 1h=7d, 1d=永久
    └── 自动淘汰: INSERT 时检查行数，超出阈值删最旧

L2: Adapter 远程获取
    └── L0 miss + L1 miss/stale → Adapter.Fetch*()
```

**数据归一化**:
```go
// Normalize 将不同数据源的原始数据转为统一的 QuantFlow 格式
func NormalizeQuote(raw map[string]any, schema string) (*QuoteSnapshot, error)
func NormalizeOHLCV(raw []map[string]any, schema string) ([]OHLCVBar, error)
```
支持的 schema: `OHLCV`, `QUOTE`, `TICK`，处理字段映射、时区转换、复权调整。

**Verification**: `go test ./internal/market/...` → Hub 订阅/退订 → Adapter mock 返回数据 → 缓存命中 L0 → L1 持久化 → 数据归一化

---

## Acceptance Criteria

- [ ] `wails dev` 启动完整桌面应用，Terminal Mode 默认显示
- [ ] CommandBar (`Ctrl+K`) 可搜索并打开面板
- [ ] DockView 支持拖拽停靠、分割条调整、布局保存/恢复 (single, split-h, 2x2)
- [ ] 8 个面板全部可渲染（即使部分使用 mock 数据）
- [ ] Workflow Mode 可拖拽节点、连线、配置参数
- [ ] `F5` 执行工作流，ExecutionLog 实时显示每层执行结果
- [ ] Terminal `[⊕]` → Workflow 创建节点；Workflow 右键 → Terminal 创建监控面板
- [ ] Go TradingEngine 完整管线: Signal → RiskPipeline → OMS → PaperEngine → Position
- [ ] PaperEngine 用 CSV 数据回放，P&L 计算正确
- [ ] MarketDataHub: 订阅/退订/推送，三级缓存命中逻辑正确
- [ ] 首批 8 个 Data Adapter 至少实现 HTTP/WS 连接 + 数据归一化
- [ ] 4 个 Pinia Store 测试覆盖
- [ ] SQLite migration 004 正确创建 orders/trades/positions 表
- [ ] Go 测试覆盖率 ≥ 70%
- [ ] 前端 vitest 测试覆盖核心组件
- [ ] `make build` 产出单二进制 (Go 后端 + 前端 assets 嵌入)

## Risks / Trade-offs

| Risk | Probability | Mitigation |
|------|------------|------------|
| Wails v3 API 不稳定 | 中 | 优先验证 Wails v3 基本功能；如不行退到 Wails v2 (稳定 3 年+) |
| vue-flow 大工作流性能 | 低 | Phase 1 已 benchmark 100 节点 <423µs；vue-flow 共享 xyflow 核心算法 |
| 自研 DockView 复杂度高 | 中 | 渐进: v1 固定 2×2 布局 → v2 拖拽重排 → v3 撕下浮动窗口 |
| 模拟撮合逻辑有 Bug | 中 | 用 Phase 1 的 SMA Cross 信号驱动 PaperEngine，对比手动计算验证 P&L |
| Adapter 上游 API 不稳定 | 高 | 每个 adapter 独立错误处理 + 降级；缓存层保证离线可用 |
| Go ↔ JS 类型同步 | 低 | 手动维护 TypeScript 类型定义；后续可考虑自动生成 |
| 前端依赖版本冲突 | 低 | 锁定关键依赖版本 (vue-flow, echarts, monaco-editor)；Pinia/vue-router 生态成熟 |
