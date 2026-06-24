# Phase 6: Frontend Panels + SSE Streaming + Pinia Stores

## Motivation

Phase 5 完成了 Go 后端全部服务（Broker、Portfolio、Notify、Schedule），但缺少对应的前端面板。用户无法在终端界面查看投资组合、管理通知、配置券商、设置定时任务。Phase 6 的目标是将 Phase 5 的后端能力完整对接到前端。

## Design

### 架构决策

- **独立面板模式**：7 个新面板各自独立，通过 DockView 自由停靠，与现有 10 个面板风格一致
- **直连 Wails IPC**：面板直接调用 `window.go.main.App.GetXxx()` 方法获取数据，setInterval 轮询刷新
- **SSE 推送通知**：NotifyPanel 通过 EventSource 订阅 Go 后端实时通知推送

### 数据流

```
┌─────────────────────────────────────────────────────────────────┐
│                      Frontend (Vue 3)                           │
│                                                                 │
│  Panel → window.go.main.App.GetXxx() → Go backend → JSON       │
│                                                                 │
│  PortfolioSummary ← GetPortfolioSummary/GetAllocation/GetPositions│
│  PositionDetail   ← GetPositions(symbol) + MarketDataHub       │
│  RiskDashboard    ← GetPortfolioSummary + 前端计算 RiskMetrics  │
│  TradeHistory     ← GetTrades/GetOrders                        │
│  SchedulePanel    ← ListScheduleTasks/Create/Delete             │
│  NotifyPanel      ← GetNotifications/MarkRead + SSE EventSource │
│  BrokerConfig     ← 本地配置表单 (SQLite via Wails IPC)         │
│                                                                 │
│  notifyStore: unreadCount, notifications[], SSE connection     │
│  portfolioStore: summary cache                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

### 面板一：PortfolioSummary

**文件**：`frontend/src/terminal/panels/PortfolioSummary.vue`

**功能**：组合概览仪表盘

**调用**：`GetPortfolioSummary()`, `GetPortfolioAllocation()`, `GetPositions()`

**布局**：
- **顶部 KPI 卡片行**（5 个）：Total Value, Cash Balance, Market Value, Total P&L, Day P&L
  - 盈亏正值绿色 `#3fb950`，负值红色 `#f85149`
- **中左（60% 宽）**：ECharts 权益曲线（30 天 daily_pnl 数据）
  - xAxis: date, yAxis: total_value, 填充渐变面积图
- **中右（40% 宽）**：ECharts 饼图（市场分布 by_market）
  - 颜色：A股=#d32f2f, 港股=#1976d2, 美股=#388e3c, 加密=#f57c00
- **底部**：持仓列表表格（Symbol, Market, Qty, Avg Price, Mkt Price, P&L, P&L%, Alloc%）
  - 点击持仓行跳转到 PositionDetail 面板

**刷新**：setInterval 10 秒轮询 `GetPortfolioSummary()`

**Props**：`panelId: string; params?: Record<string, any>`

---

### 面板二：PositionDetail

**文件**：`frontend/src/terminal/panels/PositionDetail.vue`

**功能**：单持仓深度分析

**调用**：`GetPositions()` + symbol 参数筛选

**布局**：
- **头部**：Symbol 大字体 + 市场标签（CN/HK/US/CRYPTO badge）+ 货币
- **KPI 行**（6 项）：Quantity, Avg Price, Market Price, Market Value, P&L ($), P&L (%), Alloc %
- **图表**：ECharts 迷你折线图（30 天 position_snapshots 价格 + 成本线标注）
- **底部**：相关交易记录（该品种的 Trade 列表）

**Props**：`panelId: string; params?: { symbol?: string }`

---

### 面板三：RiskDashboard

**文件**：`frontend/src/terminal/panels/RiskDashboard.vue`

**功能**：风控指标仪表盘

**调用**：`GetPortfolioSummary()` → 前端计算 RiskMetrics

**布局**：
- **仪表盘卡片**（2x3 网格，6 个 KPI）：
  - VaR (95%), CVaR (95%), Max Drawdown
  - Sharpe Ratio, Sortino Ratio, Annual Volatility
  - 每张卡片：指标名称 + 大数值 + 状态颜色（Sharpe > 1 绿色，< 0 红色）
- **图表**：ECharts 回撤曲线（从 daily_pnl 计算 peak→drawdown 序列）
  - xAxis: date, yAxis: drawdown%, 红色填充面积
- **风险分解**：Top 5 持仓的最大回撤贡献者列表

---

### 面板四：TradeHistory

**文件**：`frontend/src/terminal/panels/TradeHistory.vue`

**功能**：成交记录 + 订单历史

**调用**：`GetTrades()`, `GetOrders()`

**布局**：
- **顶部筛选栏**：
  - 状态筛选（All / Pending / Filled / Cancelled）
  - Symbol 搜索输入框
- **Tab 切换**：Trades | Orders
- **Trades 表格**：Date, Symbol, Side, Qty, Price, Total, OrderID
- **Orders 表格**：Placed At, Symbol, Side, Type, Qty, Filled, Price, Status
  - Status 带颜色标签：Pending=黄, Filled=绿, Cancelled=灰, Rejected=红
- **导出按钮**：CSV 导出（`exportCSV()` 工具函数）

---

### 面板五：SchedulePanel

**文件**：`frontend/src/terminal/panels/SchedulePanel.vue`

**功能**：定时任务管理

**调用**：`ListScheduleTasks()`, `CreateScheduleTask(name, cron, wfID)`, `DeleteScheduleTask(id)`

**布局**：
- **顶部**：新建任务按钮 + 任务数量计数
- **任务列表**：
  - 每行：Name, CronExpr（等宽字体）, Next Run（计算预览）, Last Status（✅/❌）
  - 右侧：Enabled 开关 + Delete 按钮
- **新建/编辑弹窗**（Modal）：
  - Name 输入框
  - Cron 表达式输入框 + 5 个快速预设按钮：
    - Every Hour: `0 * * * *`
    - Daily 9:00: `0 9 * * *`
    - Weekdays 9:25: `25 9 * * 1-5`
    - Every 5min: `*/5 * * * *`
    - Weekly: `0 9 * * 1`
  - 工作流选择器（从 `ListWorkflows()` 获取列表）
  - Timeout 输入（默认 1800 秒）

---

### 面板六：NotifyPanel

**文件**：`frontend/src/terminal/panels/NotifyPanel.vue`

**功能**：通知中心

**调用**：`GetNotifications(limit, offset)`, `MarkNotificationRead(id)`, SSE EventSource

**布局**：
- **顶部**：级别筛选标签（全部/交易/告警/信息/错误）
- **通知列表**：
  - 每行：级别图标 + 标题 + 正文 + 时间戳
  - 未读通知加粗白色，已读通知灰色
  - 点击通知 → 标记已读 + 跳转关联面板（如订单详情）
- **底部**：全部已读按钮 + 未读计数徽章

**SSE 连接**：
```typescript
const eventSource = new EventSource('/api/notifications/stream')
eventSource.onmessage = (event) => {
  const msg = JSON.parse(event.data)
  notifications.value.unshift(msg)
  unreadCount.value++
}
```

---

### 面板七：BrokerConfig

**文件**：`frontend/src/terminal/panels/BrokerConfig.vue`

**功能**：券商配置管理

**布局**：
- **券商选择**：下拉菜单（Binance / Futu）
- **Binance 配置**：
  - API Key 输入框（password 类型）
  - Secret Key 输入框（password 类型）
  - Testnet 开关
  - 连接测试按钮 → 调用 `Connect()` 并显示结果状态
- **Futu 配置**：
  - Host 输入（默认 localhost）
  - Port 输入（默认 11111）
  - 连接状态指示灯（红/绿圆点）
- **保存按钮**：所有配置存入 broker_config 表

---

### Pinia Store：notifyStore

**文件**：`frontend/src/stores/notify.ts`

- `notifications`: Notification[] — 通知列表
- `unreadCount`: number — 未读计数
- `filter`: string — 当前级别筛选
- `filteredNotifications`: computed — 筛选后的通知列表
- `fetchNotifications(limit, offset)`: 调用 GetNotifications
- `markRead(id)`: 标记已读
- `markAllRead()`: 全部已读
- `connectSSE()`: EventSource 连接 → 实时推送新通知

### Pinia Store：portfolioStore

**文件**：`frontend/src/stores/portfolio.ts`

- `summary`: PortfolioSummary | null — 组合摘要缓存
- `allocation`: Allocation | null — 资产配置
- `positions`: PositionDetail[] — 持仓列表
- `fetchSummary()`: 调用 GetPortfolioSummary
- `fetchAllocation()`: 调用 GetPortfolioAllocation
- `fetchPositions()`: 调用 GetPositions
- `startAutoRefresh()` / `stopAutoRefresh()`: 10 秒自动刷新

---

### OrderEntryPanel 增强

在 Symbol 和 Side 之间插入券商选择：
- `const broker = ref<'paper' | 'binance' | 'futu'>('paper')`
- 下拉框显示选项，默认 Paper（模拟交易）
- `placeOrder()` 传递 broker 参数到 `App.PlaceOrder()`

### 面板注册

在 `registry.ts` 中新增 7 行 `register()` 调用：
- `portfolio-summary`, `position-detail`, `risk-dashboard`, `trade-history`
- `schedule-panel`, `notify-panel`, `broker-config`

### CSV 导出工具

新建 `frontend/src/lib/export.ts`：
- `exportCSV(filename, headers, rows)`: 生成 CSV 文件并触发下载
- 用于 TradeHistory 导出功能

---

## File List

```
frontend/src/
├── terminal/panels/
│   ├── PortfolioSummary.vue     # NEW (~200 lines)
│   ├── PositionDetail.vue       # NEW (~150 lines)
│   ├── RiskDashboard.vue        # NEW (~180 lines)
│   ├── TradeHistory.vue         # NEW (~180 lines)
│   ├── SchedulePanel.vue        # NEW (~220 lines)
│   ├── NotifyPanel.vue          # NEW (~200 lines)
│   ├── BrokerConfig.vue         # NEW (~180 lines)
│   ├── OrderEntryPanel.vue      # MODIFIED: +broker selector (~20 lines)
│   └── registry.ts              # MODIFIED: +7 registrations
├── stores/
│   ├── notify.ts                # NEW (~60 lines)
│   └── portfolio.ts             # NEW (~50 lines)
└── lib/
    └── export.ts                # NEW (~30 lines)
```

## Acceptance Criteria

### 面板功能
- [ ] PortfolioSummary 显示总资产、持仓市值、盈亏 KPIs + 权益曲线 + 市场饼图
- [ ] PositionDetail 从 PortfolioSummary 点击进入，显示单持仓完整数据
- [ ] RiskDashboard 显示 6 个风控仪表盘卡片 + 回撤曲线图
- [ ] TradeHistory 显示成交/订单表格，支持状态筛选和 CSV 导出
- [ ] SchedulePanel 显示任务列表、新建/编辑弹窗、cron 预设、启停开关
- [ ] NotifyPanel 显示通知列表、级别筛选、标记已读、SSE 实时推送
- [ ] BrokerConfig 支持 Binance/Futu 配置表单和连接测试

### Store
- [ ] notifyStore 维护通知列表、未读计数、SSE 连接
- [ ] portfolioStore 维护组合摘要缓存和自动刷新

### 集成
- [ ] 所有面板通过 registry.ts 注册为异步组件
- [ ] OrderEntryPanel 增加券商选择下拉框
- [ ] 面板 dark theme 风格与现有面板一致
- [ ] `cd frontend && npx vue-tsc --noEmit` 无 TypeScript 错误

## Risks / Trade-offs

### 风险
1. **SSE 兼容性**：Wails WebView 可能不完全支持 EventSource。缓解方案：fallback 到短轮询（setInterval 3 秒）
2. **ECharts 性能**：多个图表同时渲染可能影响低配机器。缓解方案：图表懒加载（面板可见时才初始化 `v-if`）

### Trade-offs
1. **直连 IPC vs Store 中间层**：选择直连 IPC 模式，面板直接调用 Go 方法。优点：简单、与现有模式一致。缺点：同一数据可能被多个面板重复请求。通过 10 秒轮询间隔缓解。
2. **前端 vs 后端计算风控指标**：选择前端计算 RiskMetrics（从 daily_pnl 推导）。优点：Go 后端无需新增接口。缺点：历史数据量大时前端计算有压力。可后续移到 Go 后端。
