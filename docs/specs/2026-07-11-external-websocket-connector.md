# 外部数据源 WebSocket 连接框架 — 加密市场先行

## Motivation

当前 QuantFlow 的数据管道是**全 HTTP 轮询**：

```
外部 REST API (5s 轮询) → QuotePoller → wsHub → 前端
                                            ↑ 只这一段是 WS
```

真正的外部 WebSocket 推模式从未实现。这导致：

1. **加密市场延迟高**：Binance/OKX/Gate.io 原生支持 WSS 推送（<100ms 延迟），但系统用 5s HTTP 轮询获取，浪费了数据源能力
2. **无泛化 WS 连接管理**：面向多个交易所的 WebSocket 连接（重连、ping/pong、订阅管理）无共享抽象，每个适配器需要自己实现
3. **无法验证推模式架构**：A 股不支持 WS，加密是唯一能验证全链路推模式的市场。如果不先在加密验证，未来即使有 A 股 Level-2 也不知道框架是否可靠

## Design

### 架构

```
┌─────────────────┐     WebSocket      ┌──────────────────────┐
│ Binance WSS      │────wss://stream───→│                      │
│ OKX WSS          │────wss://ws───────→│  wsconn.Manager      │
│ Gate.io WSS      │────wss://api──────→│  (连接生命周期管理)   │
└─────────────────┘                     │                      │
                                        │  每个交易所 1 个连接   │
                                        │  自动重连 + 心跳      │
                                        │  订阅管理 + 消息路由   │
                                        └────────┬─────────────┘
                                                 │ marketHub.Publish
                                                 ▼
                                        ┌──────────────────────┐
                                        │   MarketDataHub       │
                                        │   (进程内 pub/sub)     │
                                        └────────┬─────────────┘
                                                 │ wsHub.Broadcast
                                                 ▼
                                        ┌──────────────────────┐
                                        │   前端 WebSocket       │
                                        │   (已有 useWebSocket)  │
                                        └──────────────────────┘
```

### 新增文件

**`internal/market/wsconn/manager.go`** — 共享连接管理器

```go
package wsconn

// Manager manages WebSocket connections to external data sources.
// Each connection is a goroutine with auto-reconnect, heartbeat, and
// subscription tracking.
type Manager struct {
    conns   map[string]*Conn    // exchange → connection
    hub     *market.MarketDataHub
    mu      sync.RWMutex
    close   chan struct{}
}

// Conn represents a single WebSocket connection to an exchange.
type Conn struct {
    exchange string         // "binance", "okx", "gateio"
    url      string
    conn     *websocket.Conn
    subs     map[string]bool // subscribed channels
    mu       sync.Mutex
    close    chan struct{}
}

func NewManager(hub *market.MarketDataHub) *Manager
func (m *Manager) Connect(exchange, url string) error
func (m *Manager) Subscribe(exchange, channel string) error
func (m *Manager) Unsubscribe(exchange, channel string) error
func (m *Manager) Disconnect(exchange string)
func (m *Manager) Close()
```

**`internal/market/adapters/binance_ws.go`** — Binance WS 适配

```go
package adapters

// BinanceWSAdapter provides WebSocket-based real-time data for Binance.
// Uses wsconn.Manager for connection lifecycle.
type BinanceWSAdapter struct {
    connMgr *wsconn.Manager
}

func (a *BinanceWSAdapter) Name() string     { return "binance_ws" }
func (a *BinanceWSAdapter) Markets() []string { return []string{"CRYPTO"} }

// Connect establishes a WebSocket connection to Binance streams.
func (a *BinanceWSAdapter) Connect(ctx context.Context) error {
    return a.connMgr.Connect("binance", "wss://stream.binance.com:9443/ws")
}

// SubscribeChannels subscribes to ticker and kline channels for given symbols.
func (a *BinanceWSAdapter) SubscribeChannels(symbols []string) error {
    for _, sym := range symbols {
        // Binance uses lowercase symbol
        s := strings.ToLower(sym)
        if err := a.connMgr.Subscribe("binance", s+"@ticker"); err != nil {
            return err
        }
        if err := a.connMgr.Subscribe("binance", s+"@kline_1m"); err != nil {
            return err
        }
    }
    return nil
}
```

**`internal/market/adapters/okx_ws.go`** — OKX WS 适配

**`internal/market/adapters/gateio_ws.go`** — Gate.io WS 适配

### 修改文件

| 文件 | 改动 |
|------|------|
| `internal/market/adapter.go` | 新增可选接口 `WSProvider { Connect(ctx) error; SubscribeChannels(symbols []string) error }` |
| `internal/market/registry.go` | 新增 `GetWSProvider(market string) (WSProvider, bool)` |
| `internal/market/hub.go` | 适配器 WS 数据直接 `Publish` 到 hub（无修改，hub 已支持任意来源发布） |
| `internal/market/poller.go` | `pollOnce` 中检查是否有活跃 WS 连接，有则跳过该 symbol 的 HTTP 轮询 |
| `app_startup.go` | 启动时 `wsconn.Manager` 初始化 + 连接到加密交易所 |

### 连接生命周期

```
启动
  ↓
Manager.Connect("binance", "wss://stream.binance.com:9443/ws")
  ↓
Conn.connect() → 建立 WS → 发送订阅消息
  ↓
Conn.readLoop() ──收到消息──→ parse → marketHub.Publish(topic, data)
  ↓
连接断开 → 指数退避重连 (1s, 2s, 4s, ... max 30s)
  ↓
重连成功 → 重新订阅所有 channels
```

### 协议映射

| 交易所 | WS 端点 | 订阅格式 | 消息格式 | 心跳 |
|--------|---------|---------|---------|------|
| Binance | `wss://stream.binance.com:9443/ws` | `{"method":"SUBSCRIBE","params":["btcusdt@ticker"],"id":1}` | `{"e":"24hrTicker","s":"BTCUSDT","c":"50000.00",...}` | 每 3 分钟 server 发送 ping |
| OKX | `wss://ws.okx.com:8443/ws/v5/public` | `{"op":"subscribe","args":[{"channel":"tickers","instId":"BTC-USDT"}]}` | `{"arg":{"channel":"tickers"},"data":[{"instId":"BTC-USDT","last":"50000",...}]}` | 每 15 秒 send ping |
| Gate.io | `wss://api.gateio.ws/ws/v4/` | `{"time":...,"channel":"spot.tickers","event":"subscribe","payload":["BTC_USDT"]}` | `{"channel":"spot.tickers","event":"update","result":[{"currency_pair":"BTC_USDT","last":"50000",...}]}` | 每 5 秒 server ping |

### 错误处理

- 连接断开 → 指数退避重连（独立 goroutine，不影响其他连接）
- 订阅失败 → log + 重试 3 次，失败后回退到 HTTP 轮询
- 消息解析错误 → log + 跳过，不影响后续消息
- 全部断开 → 自动回退到 `QuotePoller` 的 HTTP 轮询链（原有逻辑不变）

### 测试策略

- `wsconn/manager_test.go`: 连接、订阅、重连、断开
- 使用 `httptest` 或 `wsconn/testutil.go` mock WS server
- `binance_ws_test.go`: 模拟 Binance 消息格式验证解析逻辑

## Acceptance Criteria

- [ ] Binance WebSocket 连接建立成功，接收实时 ticker 数据
- [ ] OKX WebSocket 连接建立成功
- [ ] Gate.io WebSocket 连接建立成功
- [ ] WS 数据通过 `marketHub.Publish` → `wsHub.Broadcast` 到达前端
- [ ] 连接断开后 30s 内自动重连并恢复订阅
- [ ] WS 活跃时，QuotePoller 跳过该 symbol 的 HTTP 轮询
- [ ] 所有 WS 断开时，自动回退到 QuotePoller HTTP 轮询
- [ ] 加密面板（CryptoOverviewPanel）通过 WS 收到 <1s 延迟的行情更新
- [ ] 后端构建通过，所有已有测试通过
- [ ] CHANGELOG 更新

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 加密交易所 WS 频繁断连 | `wsconn.Manager` 内置指数退避重连（1s→30s），断连期间回退 HTTP 轮询 |
| 消息速率过高（Binance 1000msgs/s） | 每个连接用独立 goroutine 处理，`marketHub.Publish` 非阻塞 |
| 多个适配器需不同消息格式 | 每个交易所的 WS 解析器独立实现，共享 `wsconn.Manager` 的连接生命周期 |
| 资源泄漏（goroutine/连接） | Manager.Close() 关闭所有连接；每个 Conn 有独立 ctx + cancel |
| WS 和 HTTP 数据冲突 | 以 WS 为准（时效性更高），HTTP 回退作为保底 |

## 为什么不先做 A 股 WS？

A 股没有任何公开的 WebSocket 行情源。腾讯/新浪/东方财富/通达信全部只提供 HTTP REST。唯一的实时方案是付费 Level-2（通达信 TCP 私有协议），复杂度远高于加密 WS。**加密是唯一能验证全链路推模式的市场。**

此 spec 建立的 `wsconn.Manager` 框架在未来 A 股 Level-2 接入时可复用。
