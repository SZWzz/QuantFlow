# Phase 5: Broker Integration + Portfolio & Risk + Notification + Scheduler

## Motivation

Phase 1-4 建立了 QuantFlow 的核心骨架：工作流引擎、交易引擎（模拟）、市场数据中枢、Python 因子/回测、AI 智能体。但终端尚不具备**实盘交易能力**——无法连接真实券商下单、无法管理实盘持仓、无法推送交易通知、无法定时自动执行任务。

Phase 5 的目标是将 QuantFlow 从「研究回测工具」升级为「可实盘交易的量化终端」。

## Design

### 整体架构

```
┌──────────────────────────────────────────────────────────────────┐
│                         Frontend (Vue 3)                          │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────┐ ┌─────────────┐ │
│  │PortfolioSummary│ │RiskDashboard│ │TradeHistory│ │SchedulePanel│ │
│  │PositionDetail │ │              │ │           │ │NotifyPanel  │ │
│  └──────┬───────┘ └──────┬───────┘ └─────┬─────┘ └──────┬──────┘ │
│         │                │               │              │         │
│         └────────────────┼───────────────┼──────────────┘         │
│                          │ Wails IPC                              │
└──────────────────────────┼───────────────────────────────────────┘
                           │
┌──────────────────────────┼───────────────────────────────────────┐
│                     Go Backend                                     │
│                           │                                        │
│  ┌────────────────────────┼────────────────────────────────────┐  │
│  │                   App (Wails)                                │  │
│  │  PortfolioSummary / PlaceOrder / GetNotifications / ...     │  │
│  └────────┬───────────┬───────────┬──────────────┬─────────────┘  │
│           │            │            │               │              │
│  ┌────────┴──┐  ┌─────┴─────┐  ┌──┴──────────┐  ┌─┴──────────┐  │
│  │ Trading   │  │ Portfolio │  │ Schedule     │  │ Notify     │  │
│  │ Engine    │  │ Service   │  │ Engine       │  │ Manager    │  │
│  │           │  │           │  │              │  │            │  │
│  │ OMS ──────┼──┤           │  │ robfig/cron  │  │ Telegram   │  │
│  │  │        │  │           │  │   ↓          │  │ InApp      │  │
│  │  ├─Paper  │  │           │  │ Workflow     │  │            │  │
│  │  └─Broker │  │           │  │ Executor     │  │            │  │
│  └─────┬─────┘  └─────┬─────┘  └──────┬───────┘  └──────┬─────┘  │
│        │               │               │                  │        │
│  ┌─────┴───────────────┴───────────────┴──────────────────┴───┐  │
│  │                    Data Layer                                │  │
│  │  SQLite WAL: schedule_tasks, daily_pnl, position_snapshots, │  │
│  │              notifications, broker_config                   │  │
│  └─────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 模块一：Broker 接口与适配器

#### Broker 接口 (`internal/trading/broker.go`)

```go
type Broker interface {
    // 连接管理
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    IsConnected() bool
    Name() string

    // 订单管理
    SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error)
    CancelOrder(ctx context.Context, orderID string) error
    ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error

    // 查询
    GetOrders(ctx context.Context) ([]*Order, error)
    GetPositions(ctx context.Context) ([]*Position, error)
    GetAccount(ctx context.Context) (*AccountInfo, error)

    // 事件回调
    OnOrderUpdate(func(order *Order))
    OnTradeUpdate(func(trade *Trade))
}

type AccountInfo struct {
    BrokerName    string  `json:"broker_name"`
    TotalValue    float64 `json:"total_value"`
    CashBalance   float64 `json:"cash_balance"`
    MarginBalance float64 `json:"margin_balance"`
    BuyingPower   float64 `json:"buying_power"`
    Currency      string  `json:"currency"`
}

type BrokerOrderResult struct {
    BrokerOrderID string      `json:"broker_order_id"`
    Status        OrderStatus `json:"status"`
    Message       string      `json:"message"`
}
```

#### OMS 集成

OMS 增加可选 Broker 字段：

- `broker == nil` → 模拟模式，走 PaperEngine 逻辑（现有行为不变）
- `broker != nil` → 实盘模式，PlaceOrder/CancelOrder 路由到券商
- 券商通过 `OnOrderUpdate` 回调更新 OMS 订单状态
- PaperEngine 的所有风控检查（RiskPipeline）在提交券商前执行

#### FutuBroker (`internal/trading/brokers/futu.go`)

- 连接方式：TCP 直连本地 FutuOpenD（默认 `localhost:11111`）
- 协议：Protobuf over TCP（富途官方协议）
- 覆盖市场：A 股（沪/深）、港股、美股
- 特性：KeepAlive 心跳、断线自动重连、连接状态监控
- 前置依赖：用户需自行安装 FutuOpenD 网关

#### BinanceBroker (`internal/trading/brokers/binance.go`)

- 连接方式：REST API + WebSocket 用户数据流
- 认证：HMAC SHA256 签名（API Key + Secret）
- 覆盖市场：现货 + USDT 永续合约
- 特性：listenKey 管理、WebSocket 保活、速率限制

#### 配置存储

```sql
-- Migration 006: broker_config
CREATE TABLE broker_config (
    broker_name TEXT PRIMARY KEY,   -- "futu" | "binance"
    enabled INTEGER DEFAULT 1,
    config_json TEXT NOT NULL,      -- JSON: api_key, secret, endpoint, etc.
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

---

### 模块二：调度系统 (`internal/schedule/`)

#### 核心类型

```go
type Scheduler struct {
    cron    *cron.Cron
    repo    *ScheduleRepo
    engine  WorkflowExecutor       // 接口：执行指定工作流
    notify  notifier               // 接口：任务结果通知
}

type ScheduleTask struct {
    ID            string     `json:"id"`
    Name          string     `json:"name"`
    CronExpr      string     `json:"cron_expr"`
    WorkflowID    string     `json:"workflow_id"`
    Enabled       bool       `json:"enabled"`
    TimeoutSec    int        `json:"timeout_sec"`     // 默认 1800 (30min)
    LastRunAt     *time.Time `json:"last_run_at"`
    LastRunStatus string     `json:"last_run_status"`
    CreatedAt     time.Time  `json:"created_at"`
}
```

#### 功能特性

- **CRUD**：创建/编辑/删除/启停定时任务
- **工作流触发**：通过 `WorkflowExecutor` 接口触发工作流执行
- **超时保护**：context.WithTimeout 控制，默认 30 分钟
- **重叠保护**：sync.Mutex 确保同一任务不会并发执行
- **结果通知**：执行完成后可选推送 Telegram/应用内通知

#### WorkflowExecutor 接口

```go
type WorkflowExecutor interface {
    Execute(ctx context.Context, workflowID string) (executionID string, err error)
    GetStatus(ctx context.Context, executionID string) (status string, err error)
}
```

#### 数据库

```sql
-- Migration 007: schedule
CREATE TABLE schedule_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    timeout_sec INTEGER DEFAULT 1800,
    last_run_at TEXT,
    last_run_status TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

#### 前端面板：SchedulePanel

- 任务列表（名称、cron 表达式、下次执行时间、上次状态、启停开关）
- 新建/编辑任务对话框（cron 表达式选择器 + 工作流选择器）
- 手动触发按钮 + 执行历史查看

---

### 模块三：通知系统 (`internal/notify/`)

#### 核心接口

```go
type Level string

const (
    LevelInfo    Level = "info"
    LevelWarning Level = "warn"
    LevelError   Level = "error"
    LevelTrade   Level = "trade"
)

type Message struct {
    Level    Level             `json:"level"`
    Title    string            `json:"title"`
    Body     string            `json:"body"`
    Metadata map[string]string `json:"metadata"`
}

type Notifier interface {
    Send(ctx context.Context, msg *Message) error
    Name() string
}
```

#### NotificationMgr

```go
type NotificationMgr struct {
    notifiers []Notifier
    repo      *NotifyRepo
    eventCh   chan *Message         // 缓冲 256
}

// 广播通知到所有已注册渠道
func (m *NotificationMgr) Send(msg *Message) { m.eventCh <- msg }

// 查询历史
func (m *NotificationMgr) GetHistory(limit, offset int) ([]*Notification, error)
```

#### 通知触发源

| 触发源 | 内容示例 | 级别 |
|--------|---------|------|
| OMS 订单成交 | "AAPL 限价单买入 100 股已成交 @185.30" | trade |
| OMS 订单拒绝 | "TSLA 市价单被拒绝：风控-单笔金额超限" | error |
| 调度任务完成 | "早盘筛选已完成，选出 3 只标的" | info |
| 调度任务失败 | "因子计算失败：Python sidecar 无响应" | error |
| 风控告警 | "NVDA 单日跌幅 5.2% 触发止损线" | warn |
| 券商连接 | "富途连接断开，正在重连（第 3 次）" | warn |
| 系统事件 | "QuantFlow 启动完成" | info |

#### TelegramNotifier

- 通过 Telegram Bot API (`https://api.telegram.org/bot<token>/sendMessage`)
- 配置：Bot Token + Chat ID（存储在 broker_config 表）
- 支持 MarkdownV2 格式
- 交易通知带 InlineKeyboard 按钮（查看持仓/撤单）
- 异步发送，失败重试 3 次

#### InAppNotifier

- 写入 SQLite 通知队列
- 前端通过 SSE 订阅实时通知
- StatusBar 显示未读通知数量徽章
- 新通知弹出 Toast（3 秒自动消失）

#### 数据库

```sql
-- Migration 008: notifications
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    metadata TEXT DEFAULT '{}',
    is_read INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX idx_notifications_unread ON notifications(is_read, created_at);
```

#### 前端面板：NotifyPanel

- 通知中心：按时间倒序排列，支持按级别筛选
- 未读/已读状态 + 全部已读按钮
- 点击通知跳转到关联面板（如订单详情）

---

### 模块四：投资组合与风控 (`internal/portfolio/`)

#### PortfolioService

```go
type PortfolioService struct {
    oms    *trading.OMS
    market *market.MarketDataHub
    repo   *PortfolioRepo
}

func (p *PortfolioService) GetSummary() *PortfolioSummary
func (p *PortfolioService) GetPositions() []*PositionDetail
func (p *PortfolioService) GetPnLHistory(days int) []*DailyPnL
func (p *PortfolioService) GetAllocation() *Allocation
func (p *PortfolioService) RecordDailySnapshot() error    // 每日收盘后调用
```

#### 扩展 Position 类型

```go
type PositionDetail struct {
    *trading.Position
    Market       string  `json:"market"`       // CN / HK / US / CRYPTO
    Currency     string  `json:"currency"`     // CNY / HKD / USD / USDT
    DayPnL       float64 `json:"day_pnl"`
    DayPnLPct    float64 `json:"day_pnl_pct"`
    CostBasis    float64 `json:"cost_basis"`
    AllocPct     float64 `json:"alloc_pct"`    // 占组合百分比
}
```

#### RiskMetrics

```go
type RiskMetrics struct {
    Var95         float64 `json:"var_95"`
    CVaR95        float64 `json:"cvar_95"`
    MaxDrawdown   float64 `json:"max_drawdown"`
    MaxDDStart    string  `json:"max_dd_start"`
    MaxDDEnd      string  `json:"max_dd_end"`
    SharpeRatio   float64 `json:"sharpe_ratio"`
    SortinoRatio  float64 `json:"sortino_ratio"`
    CalmarRatio   float64 `json:"calmar_ratio"`
    TotalExposure float64 `json:"total_exposure"`
    Leverage      float64 `json:"leverage"`
    DailyVol      float64 `json:"daily_volatility"`
    AnnualVol     float64 `json:"annual_volatility"`
}
```

#### 数据库

```sql
-- Migration 009: portfolio
CREATE TABLE daily_pnl (
    date TEXT NOT NULL,
    total_value REAL NOT NULL,
    cash REAL NOT NULL,
    market_value REAL NOT NULL,
    pnl REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    PRIMARY KEY (date)
);

CREATE TABLE position_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol TEXT NOT NULL,
    date TEXT NOT NULL,
    quantity REAL NOT NULL,
    avg_price REAL NOT NULL,
    market_price REAL NOT NULL,
    pnl REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    UNIQUE(symbol, date)
);

CREATE INDEX idx_daily_pnl_date ON daily_pnl(date);
CREATE INDEX idx_position_snapshots_date ON position_snapshots(date);
```

#### 前端面板

| 面板 | 功能 | 图表组件 |
|------|------|---------|
| **PortfolioSummary** | 总资产、当日盈亏、持仓列表、资产配置 | ECharts 权益曲线 + 饼图 |
| **PositionDetail** | 单持仓详情（成本、市值、盈亏、分配比例） | 持仓盈亏走势 |
| **RiskDashboard** | VaR/CVaR/最大回撤/夏普比率仪表盘 | 回撤曲线 + 风险分解 |
| **TradeHistory** | 成交/订单记录，支持筛选/导出 | 交易分布 |

---

### 模块五：新增工作流节点

#### 交易类 (`internal/workflow/nodes/`)

| 节点 | 注册名 | 输入 | 输出 | 类别 |
|------|--------|------|------|------|
| PlaceOrderNode | `place_order` | symbol, side, order_type, qty, price, stop_price | order_id, status | trading |
| CancelOrderNode | `cancel_order` | order_id | success | trading |
| PositionQueryNode | `position_query` | broker, symbol(可选) | positions[] | trading |
| OrderQueryNode | `order_query` | broker, status(可选) | orders[] | trading |

#### 通知类

| 节点 | 注册名 | 输入 | 输出 | 类别 |
|------|--------|------|------|------|
| NotifyNode | `notify` | level, title, body, channels[] | success | notify |
| AlertNode | `alert` | symbol, condition, threshold, message | triggered, value | notify |

#### 调度类

| 节点 | 注册名 | 输入 | 输出 | 类别 |
|------|--------|------|------|------|
| ScheduleNode | `schedule` | cron_expr, workflow_id | task_id | schedule |
| WaitNode | `wait` | duration_sec | — | schedule |

#### 投资组合/风控类

| 节点 | 注册名 | 输入 | 输出 | 类别 |
|------|--------|------|------|------|
| PortfolioSummaryNode | `portfolio_summary` | — | summary | portfolio |
| RiskMetricsNode | `risk_metrics` | positions[], benchmark(可选) | metrics | risk |
| AllocationNode | `allocation` | positions[] | by_market, by_sector | portfolio |

---

### 数据流总览

```
用户操作 → 前端 Panel → Wails IPC → Go App Method → Domain Service → SQLite / Broker

下单流程:
OrderEntryPanel → App.PlaceOrder() → OMS.PlaceOrder()
  ├── broker==nil → PaperEngine.OnSignal()
  └── broker!=nil → Broker.SubmitOrder() → 券商 API
       └── Broker.OnOrderUpdate() → OMS.FillOrder() → NotificationMgr.Send("trade")

通知流程:
Broker.OnTradeUpdate() → OMS → NotificationMgr.Send()
  ├── TelegramNotifier.Send() → Telegram Bot API
  └── InAppNotifier.Send() → SQLite → SSE → StatusBar/NotifyPanel

调度流程:
SchedulePanel → App.CreateTask() → Scheduler.AddTask() → robfig/cron
  └── cron 触发 → Scheduler.executeTask() → WorkflowEngine.Execute()
       └── 完成 → NotificationMgr.Send("info"/"error")

投资组合数据流:
MarketDataHub(实时行情) → Position.MarketPrice 更新 → PortfolioService.GetSummary()
  └── 每日 15:30 → RecordDailySnapshot() → daily_pnl 表
```

### New/Modified Files

#### Go Backend

```
internal/trading/
├── broker.go              # NEW: Broker 接口定义
├── brokers/
│   ├── futu.go            # NEW: FutuBroker 实现
│   ├── futu_test.go       # NEW
│   ├── binance.go         # NEW: BinanceBroker 实现
│   └── binance_test.go    # NEW
├── oms.go                 # MODIFIED: 增加 broker 字段和路由逻辑
└── risk_pipeline.go       # MODIFIED: 扩展风控配置

internal/schedule/
├── scheduler.go           # NEW: 调度引擎
├── scheduler_test.go      # NEW
├── repo.go                # NEW: SQLite CRUD
└── types.go               # NEW: ScheduleTask

internal/notify/
├── manager.go             # NEW: NotificationMgr
├── manager_test.go        # NEW
├── telegram.go            # NEW: TelegramNotifier
├── inapp.go               # NEW: InAppNotifier
├── repo.go                # NEW: SQLite CRUD
└── types.go               # NEW: Message, Notifier 接口

internal/portfolio/
├── service.go             # NEW: PortfolioService
├── service_test.go        # NEW
├── risk.go                # NEW: RiskService + RiskMetrics 计算
├── repo.go                # NEW: SQLite CRUD
└── types.go               # NEW: PortfolioSummary, Allocation

internal/storage/migrations/
├── 006_broker_config.sql  # NEW
├── 007_schedule.sql       # NEW
├── 008_notifications.sql  # NEW
└── 009_portfolio.sql      # NEW

internal/workflow/nodes/
├── place_order.go         # NEW
├── cancel_order.go        # NEW
├── position_query.go      # NEW
├── order_query.go         # NEW
├── notify.go              # NEW
├── alert.go               # NEW
├── schedule.go            # NEW
├── wait.go                # NEW
├── portfolio_summary.go   # NEW
├── risk_metrics.go        # NEW
├── allocation.go          # NEW
└── phase5_test.go         # NEW: 集成测试

app/
└── app.go                 # MODIFIED: 添加 Broker/Portfolio/Schedule/Notify 相关导出方法
```

#### Frontend

```
frontend/src/
├── terminal/panels/
│   ├── PortfolioSummary.vue     # NEW
│   ├── PositionDetail.vue       # NEW
│   ├── RiskDashboard.vue        # NEW
│   ├── TradeHistory.vue         # NEW
│   ├── SchedulePanel.vue        # NEW
│   ├── NotifyPanel.vue          # NEW
│   └── BrokerConfig.vue         # NEW
├── stores/
│   ├── portfolio.ts             # NEW: Pinia store
│   └── notify.ts                # NEW: Pinia store
└── workflow/
    └── canvas/                   # MODIFIED: 添加新节点类型到 NodePalette
```

#### Dependencies

```
go get github.com/robfig/cron/v3     # 调度引擎
go get github.com/gorilla/websocket   # Binance WebSocket (已有或新增)
go get google.golang.org/protobuf     # 富途 Protobuf
```

## Acceptance Criteria

### Broker 接口与适配器
- [ ] Broker 接口定义完整，包含 Connect/Disconnect/SubmitOrder/CancelOrder/ModifyOrder/GetOrders/GetPositions/GetAccount/回调
- [ ] OMS 支持通过 SetBroker() 切换模拟/实盘模式
- [ ] BinanceBroker 通过 REST API 成功连接、查询账户、下单、撤单（测试网 testnet.binance.vision）
- [ ] FutuBroker 通过 FutuOpenD 成功连接、查询账户、下单、撤单
- [ ] 订单状态变更通过 OnOrderUpdate 回调正确更新 OMS
- [ ] 风控检查（RiskPipeline）在提交券商前执行
- [ ] 单元测试覆盖 Broker 接口的所有方法（mock broker）

### 调度系统
- [ ] Scheduler 支持 CRUD 定时任务
- [ ] robfig/cron 正确解析和执行 cron 表达式
- [ ] 任务触发后正确执行指定工作流
- [ ] 超时保护：context.WithTimeout 生效
- [ ] 重叠保护：同一任务不并发执行
- [ ] SchedulePanel 前端面板功能完整

### 通知系统
- [ ] NotificationMgr 支持多 Notifier 注册和广播
- [ ] TelegramNotifier 通过 Bot API 成功发送消息（含 MarkdownV2 格式）
- [ ] InAppNotifier 写入 SQLite 并通过 SSE 推送到前端
- [ ] OMS 订单状态变更自动触发交易通知
- [ ] 调度任务完成/失败自动触发通知
- [ ] NotifyPanel 通知中心功能完整（筛选、已读、跳转）

### 投资组合与风控
- [ ] PortfolioService.GetSummary() 正确计算总资产、持仓市值、盈亏
- [ ] PortfolioService.GetAllocation() 正确计算市场/行业分布
- [ ] DailyPnL 每日快照正确记录和查询
- [ ] RiskMetrics 计算正确：VaR(历史模拟法)、CVaR、MaxDD、Sharpe、Sortino
- [ ] PortfolioSummary、RiskDashboard、TradeHistory 面板功能完整

### 新工作流节点
- [ ] 所有 11 个新节点注册到 NodeRegistry
- [ ] PlaceOrderNode 正确下单并返回订单状态
- [ ] CancelOrderNode 正确撤单
- [ ] NotifyNode 正确发送通知到指定渠道
- [ ] AlertNode 正确判断条件并触发告警
- [ ] ScheduleNode 正确创建定时任务
- [ ] 单元测试覆盖所有新节点的 Execute 方法

### 整体
- [ ] `go test ./...` 全部通过
- [ ] `go vet ./...` 无告警
- [ ] CHANGELOG 更新 Phase 5 条目
- [ ] 版本日期更新到 2026.6.17

## Risks / Trade-offs

### 风险

1. **富途 FutuOpenD 前置依赖**：富途需要用户安装并运行 FutuOpenD 网关程序。缓解方案：在 BrokerConfig 面板中明确说明前置依赖，提供下载链接和配置指南；连接失败时给出清晰的错误提示。

2. **Binance 网络限制**：Binance API 在中国大陆可能无法直接访问。缓解方案：在 BrokerConfig 中提供代理配置选项；优先支持 testnet 进行功能验证。

3. **实盘交易安全**：下单操作不可逆。缓解方案：所有下单路径强制执行 RiskPipeline 检查（单笔金额上限、日内交易次数、止损强制）；前端做二次确认弹窗；OMS 记录完整审计日志。

4. **调度系统可靠性**：robfig/cron 是进程内调度，应用关闭后任务不执行。缓解方案：启动时检查并执行遗漏的任务（可选）；文档中说明此限制；后续可扩展系统级调度器。

5. **富途协议复杂度**：富途使用自定义 Protobuf 协议，不是标准 REST API。缓解方案：优先实现 BinanceBroker（REST API 简单），积累经验后再做 FutuBroker；富途协议使用其官方 proto 文件。

### Trade-offs

1. **Broker 接口设计**：首期只做 Futu 和 Binance，但接口预留了扩展空间（Order/Position 使用通用类型，券商特定字段通过 Metadata 传递）。后续添加 Alpaca/OKX/Bybit 时无需改接口。

2. **通知渠道**：首期只做 Telegram + 应用内，不做 Email/钉钉/飞书。理由：Telegram Bot API 免费且最稳定；应用内通知直达用户；其他渠道可按需添加（接口已支持多 Notifier 注册）。

3. **调度系统选型**：选择 robfig/cron 完整调度而非简单 interval 定时器。理由：用户需要「每个交易日上午 9:25 执行」这种精确时间点触发，interval 无法表达。

4. **投资组合 vs 回测结果**：PortfolioService 和 BacktestRunner 有部分重叠（都计算 Sharpe/MaxDD 等指标）。方案：PortfolioService 聚焦实盘持仓的实时计算；BacktestRunner 聚焦历史模拟。两者复用部分计算逻辑但不耦合。
