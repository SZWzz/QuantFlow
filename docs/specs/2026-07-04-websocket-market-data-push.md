# WebSocket Market Data Push

## Motivation

当前行情面板（自选股）通过 `setInterval` + Wails IPC 轮询获取数据，存在以下问题：

- **无效请求**：非交易时段（周末/休市）每 10s 发起一次 Wails IPC 调用，尽管后端已加 `lastQuote` 缓存，网络层仍有 IPC 开销
- **延迟高**：轮询间隔固定 10s，无法获得接近实时的行情变化
- **多面板 N 倍放大**：3 个行情面板同时打开 = 3 个 10s 轮询
- **扩展性差**：新面板若需实时数据，只能复制同样的轮询模式

WebSocket 推送方案统一用后端主动推送替代前端轮询，一次行情更新推送给所有订阅面板。

## Design

### 数据流

```
                        Go Backend
  ┌─────────────────────────────────────────────────────────────┐
  │                                                             │
  │  QuotePoller (goroutine, ticker 5s)                        │
  │   订阅列表: { "CN:600519", "US:AAPL", ... }                 │
  │     │ 每 tick 遍历订阅, 调 FetchQuoteWithFallback           │
  │     │ 成功 → MarketDataHub.Publish("market:quote:K", data) │
  │     ▼                                                       │
  │  MarketDataHub (in-process pub/sub)                         │
  │     │                                                       │
  │     ▼                                                       │
  │  MarketDataBridge (goroutine)                               │
  │    订阅 MarketDataHub 所有 topic                             │
  │    收到 MarketMessage → ws.Hub.Broadcast(topic, data)       │
  │     │                                                       │
  │     ▼                                                       │
  │  ws.Hub (WebSocket hub, per-topic client sets)              │
  │     │ 序列化 {Topic, Data} 为 JSON                           │
  │     │ 非阻塞发送到每个匹配 client.send                       │
  └─────┼───────────────────────────────────────────────────────┘
        │ WebSocket (coder/websocket, Wails HTTP server)
        │ ws://localhost:PORT/ws/market
        ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                        Frontend                             │
  │                                                             │
  │  useWebSocket composable                                    │
  │    connect("ws://.../ws/market")                            │
  │      → 发送 {type:"subscribe", topics:["market:quote:*"]}   │
  │    onMessage("market:quote:CN:600519", handler)             │
  │      → handler(quoteSnapshot)                               │
  │      → 更新 WatchlistPanel.quotes[sym]                     │
  │                                                             │
  │  WatchlistPanel                                             │
  │    移除: setInterval + refreshAll()                          │
  │    新增: useWebSocket().onMessage() 订阅                   │
  │    初始数据: 挂载时调一次 GetQuote (Wails IPC)               │
  └─────────────────────────────────────────────────────────────┘
```

### 关键设计决策

| 决策 | 选项 | 选择理由 |
|------|------|---------|
| WebSocket 挂载方式 | Wails v3 Service.Route | 同一端口，无需额外 HTTP server |
| WS 库 | `coder/websocket`（已有） | gorilla/websocket 无依赖，coder 已被 Wails 引入 |
| 轮询驱动模式 | QuotePoller 后台 goroutine | 简单可靠，适配器接口无需改造 |
| 订阅管理 | WS 协议 `{type:"subscribe"}` | 复用现有 `internal/ws` 客户端协议 |
| 首次数据 | 挂载时 Wails IPC GetQuote | 即时展示，不等 poller 下一 tick |
| 轮询间隔 | 5s | 用户体验 vs 资源消耗的平衡点 |
| 推拉分离 | 拉（IPC）保留，推（WS）做增量 | 向后兼容，非实时面板无需改动 |

### 新增文件

**`internal/market/poller.go`** — QuotePoller

```go
type QuotePoller struct {
    hub   *MarketDataHub
    reg   *AdapterRegistry
    mu    sync.RWMutex
    subs  map[string]bool  // "market:symbol" → subscribed
    close chan struct{}
}

// Subscribe(market, symbol string) — 添加到订阅列表, 若为空则启动 ticker
// Unsubscribe(market, symbol string) — 从订阅列表移除, 若为空则停止 ticker
// Run(ctx) — 主循环, 每 5s tick 遍历 subs 并 fetch+pubish
```

**`internal/market/bridge.go`** — MarketDataBridge

```go
type MarketDataBridge struct {
    marketHub *MarketDataHub
    wsHub     *ws.Hub
}

// Run(ctx) — subscribes to MarketDataHub wildcard, forwards to ws.Hub.Broadcast
```

**`internal/ws/service.go`** — MarketWSService (Wails service wrapper)

```go
// MarketWSService wraps ws.Hub as a Wails service (http.Handler)
type MarketWSService struct {
    Hub *ws.Hub
}

func (s *MarketWSService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ws.ServeWS(w, r, s.Hub)
}
```

### 修改文件

| 文件 | 改动 |
|------|------|
| `main.go` | 添加 `MarketWSService` 注册，Route: `/ws/market` |
| `app.go` | 存储 `MarketDataHub` 实例（替换 `_ = market.NewHub()`） |
| `app.go` | 启动 `QuotePoller` + `MarketDataBridge` |
| `internal/ws/handler.go` | `ServeWS` 改为接受 hub 参数（不再依赖全局 `DefaultHub`） |
| `internal/ws/hub.go` | 移除 `DefaultHub` 全局变量和 `init()` |
| `frontend/.../WatchlistPanel.vue` | 替换轮询为 `useWebSocket` 订阅 |

### 不修改的文件

- `internal/ws/client.go` — 不变，读写逻辑已完善
- 其他面板文件（Candlestick、Depth 等）— 本次只改 WatchlistPanel 试点

### WebSocket 协议

**前端 → 后端**（JSON，通过 WS 连接发送）：

```json
{"type": "subscribe", "topics": ["market:quote:CN:600519"]}
{"type": "unsubscribe", "topics": ["market:quote:CN:600519"]}
```

**后端 → 前端**（JSON，通过 WS 接收）：

```json
{"topic": "market:quote:CN:600519", "data": {"symbol":"600519","last":1890.5,...}}
```

协议复用 `internal/ws/client.go` 已实现的 `subscribeMessage` 结构。

### QuotePoller 订阅生命周期

```
WS 连接建立 → 前端 onMessage("subscribe", topics) → QuotePoller.Subscribe(...)
WS 连接断开 → hub.unregister → QuotePoller 检测并 Unsubscribe 该 client 的所有 topic
订阅列表为空 → QuotePoller 停止 ticker (select case ←close)
新订阅加入 → QuotePoller 启动 ticker (若已停止)
```

### 错误处理

- QuotePoller 单个 symbol fetch 失败 → 仅 log，不 panic，不影响其他 symbol
- MarketDataBridge 转发失败（ws.Hub 关闭）→ log 并重试
- WS 连接断开 → 前端 `useWebSocket` 自动重连（指数退避），重连后重新 subscribe
- 后端重启 → 前端重连后重新 subscribe → QuotePoller 重新开始轮询

### 测试策略

- `poller_test.go`: Subscribe/Unsubscribe, ticker 启停, fetch 成功/失败分支
- `bridge_test.go`: MarketMessage → ws.Broadcast 正确性
- `WatchlistPanel.spec.ts`: WS 消息触发行情更新

## Acceptance Criteria

- [ ] WatchlistPanel 不发起 `setInterval` 轮询
- [ ] WatchlistPanel 挂载时通过一次 Wails IPC 获取初始行情
- [ ] WatchlistPanel 通过 WS 接收实时推送，数据正确显示
- [ ] 非交易时段 QuotePoller 不 fetch（走 `IsTradingHours` / `lastQuote` 缓存）
- [ ] WS 连接断开后自动重连并恢复订阅
- [ ] 关闭 WatchlistPanel 后取消订阅，QuotePoller 停止对应 symbol 轮询
- [ ] 后端构建通过，所有已有测试通过
- [ ] CHANGELOG 更新

## Risks / Trade-offs

- **QuotePoller 增加后台开销**：即使用户只有一个面板打开 5 个 symbol，后台仍需每 5s fetch 5 次。可以后续改为 ticker 间隔自适应（有活跃报价时缩短，休市时拉长）。
- **WS 端点暴露**：Wails HTTP server 仅本地可访问（localhost），无安全风险。若将来需要远程访问，需加 token 认证。
- **已有 `init()` 全局 hub 移除**：`DefaultHub` 和 `init()` 是隐式初始化的旧模式，改为显式创建和注入，需要确保无其他代码依赖这个全局变量。
- **试点范围最小化**：只改 WatchlistPanel，但 `internal/ws/handler.go` 的签名变化会影响未来使用。保持向后兼容：`ServeWS` 接受 hub 参数，提供一个重载或 wrapper 用默认实例。
