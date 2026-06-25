# Wire 5 Fake-Data Panels to Go Backend

## Motivation

审计发现 5 个面板使用硬编码假数据，但对应的 Go 后端能力已存在（或仅缺薄封装层）。修复后这些面板将展示真实数据。

## Scope

| # | Panel | 现状 | Go 后端 | 改动类型 |
|---|-------|------|---------|---------|
| 1 | **NotifyPanel** | 3 条硬编码通知 | `GetNotifications()` + `useNotifyStore` 已存在 | 仅前端 |
| 2 | **PositionPanel** | 3 笔硬编码持仓 | `GetPositions()` 已存在 | 仅前端 |
| 3 | **NewsPanel** | 5 条硬编码新闻 | news adapter 已存在，缺 Wails 绑定 | Go + 前端 |
| 4 | **BrokerStatusPanel** | 6 个硬编码券商状态 | `IsConnected()` 各 broker 已实现，缺聚合 | Go + 前端 |
| 5 | **SchedulePanel** | 2 个硬编码任务 | `schedule.Repo` CRUD 已实现，缺绑定 | Go + 前端 |

## Design

### 1. NotifyPanel → useNotifyStore

Store 已完全实现，面板只需替换硬编码数据为 store 调用。

### 2. PositionPanel → GetPositions()

Go 返回 `[]*trading.Position`（Symbol, Quantity, AvgPrice, MarketPrice, PnL, PnLPct）。

### 3. NewsPanel → 新增 GetNews(symbol, limit)

Go 内部调用 `EastMoneyNewsAdapter.FetchStockNews()`。
返回 `[]NewsItem{Title, Source, Time, URL, Symbol}`。

### 4. BrokerStatusPanel → 新增 GetBrokerStatuses()

遍历已注册 broker，调用 `IsConnected()`。
返回 `[]BrokerStatus{Name, Label, Market, Connected bool, Detail}`。

### 5. SchedulePanel → 新增 Schedule CRUD

- `ListScheduleTasks() []schedule.Task`
- `SaveScheduleTask(task) error`
- `DeleteScheduleTask(id) error`
- `ToggleScheduleTask(id, enabled) error`

## New/Modified Files

### Go
- `app.go` — 新增 7 个 Wails 绑定方法

### Frontend
- `NotifyPanel.vue` — 接入 useNotifyStore
- `PositionPanel.vue` — 接入 GetPositions
- `NewsPanel.vue` — 接入 GetNews
- `BrokerStatusPanel.vue` — 接入 GetBrokerStatuses
- `SchedulePanel.vue` — 接入 CRUD

## Acceptance Criteria

- [ ] NotifyPanel 展示来自 SQLite 的真实通知，可标记已读
- [ ] PositionPanel 展示真实持仓数据（含 PnL）
- [ ] NewsPanel 调用东方财富接口展示真实新闻
- [ ] BrokerStatusPanel 展示各券商真实连接状态
- [ ] SchedulePanel 完整 CRUD 定时任务
- [ ] `go build` + `npm run build` 通过

## Risks

- Schedule CRUD 依赖 `tasks` 表，需确保 migration 已创建
- News adapter 在非 A 股时段可能返回空，面板展示 empty state 即可
- Broker 连接检测首次可能较慢（网络超时），前端使用手动刷新
