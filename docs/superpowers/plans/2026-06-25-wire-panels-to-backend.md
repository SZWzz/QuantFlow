# Wire 5 Fake-Data Panels to Go Backend — Implementation Plan

## Task 1: PositionPanel → GetPositions (仅前端)

**文件**: `frontend/src/terminal/panels/PositionPanel.vue`

改动：
- 删除硬编码 `positions` ref（3 笔 AAPL/GOOGL/NVDA 假数据）
- 新增 `loadPositions()` 调用 `go.main.App.GetPositions()`
- `onMounted` 触发加载
- Position 接口对齐 Go 返回结构（Symbol, Quantity, AvgPrice, MarketPrice, PnL, PnLPct）

**Commit**: `[Frontend] PositionPanel: wire to GetPositions()`

---

## Task 2: NotifyPanel → useNotifyStore (仅前端)

**文件**: `frontend/src/terminal/panels/NotifyPanel.vue`

改动：
- 删除硬编码 notifications + 本地 markRead/markAllRead/filter
- 引入 `useNotifyStore`，`onMounted(() => store.fetchNotifications())`
- 模板绑定 store 的 computed/actions

**Commit**: `[Frontend] NotifyPanel: wire to useNotifyStore`

---

## Task 3: NewsPanel → GetNews (Go + 前端)

### 3a. Go 绑定

**文件**: `app.go`

新增 `NewsItem` struct + `GetNews(symbol, limit int) ([]NewsItem, error)`。
内部调用已有的 `a.newsAdpt.FetchStockNews()`。

### 3b. 前端

**文件**: `frontend/src/terminal/panels/NewsPanel.vue`

替换硬编码 items → `GetNews(symbol, 20)`，新增 loading/empty state。

**Commit**: `[Go][Frontend] NewsPanel: add GetNews binding + wire`

---

## Task 4: BrokerStatusPanel → GetBrokerStatuses (Go + 前端)

### 4a. Go 绑定

**文件**: `app.go`

新增 `BrokerStatus` struct + `GetBrokerStatuses() []BrokerStatus`。
包含 Paper Trading (always connected) + Alpaca broker (IsConnected check)。
后续可扩展 Binance/Futu/IBKR/OKX。

### 4b. 前端

**文件**: `frontend/src/terminal/panels/BrokerStatusPanel.vue`

替换硬编码 brokers → `GetBrokerStatuses()` + 刷新按钮。

**Commit**: `[Go][Frontend] BrokerStatusPanel: add GetBrokerStatuses + wire`

---

## Task 5: SchedulePanel → Schedule CRUD (Go + 前端)

### 5a. Go 绑定

**文件**: `app.go`

App struct 新增 `scheduler *schedule.Scheduler` 字段。
ServiceStartup 中初始化（传入 `a.db`）。
新增 4 个 Wails 绑定：
- `ListScheduleTasks() ([]schedule.Task, error)`
- `SaveScheduleTask(task schedule.Task) error`
- `DeleteScheduleTask(id string) error`
- `ToggleScheduleTask(id string, enabled bool) error`

### 5b. 前端

**文件**: `frontend/src/terminal/panels/SchedulePanel.vue`

替换硬编码 tasks → 完整 CRUD（增删改查 + 启停 toggle）。

**Commit**: `[Go][Frontend] SchedulePanel: add CRUD bindings + wire`

---

## Task 6: 构建验证

- `go vet ./...`
- `cd frontend && npm run build`
- `go build -o build/quantflow .`
- 更新 CHANGELOG

**Commit**: `[Chore] CHANGELOG: wire 5 panels to Go backend`
