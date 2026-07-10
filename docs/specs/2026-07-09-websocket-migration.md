# WebSocket 实时数据迁移 Spec

## Motivation

当前 QuantFlow 中存在大量 `setInterval` 轮询，每秒/每 5 秒/每 30 秒向 Go 后端发起 Wails IPC 调用，造成：

1. **无意义 IPC 开销**：闭市时段、面板隐藏时仍在轮询，浪费 CPU 和电池
2. **数据延迟**：轮询间隔内发生的价格变化要等下次轮询才可见
3. **缓存击穿**：多个面板同时轮询同一标的，Go 后端重复查询
4. **代码重复**：每个面板各自实现定时器、倒计时、自动/手动切换逻辑

而 WebSocket 基础设施（`ws.Hub` + `QuotePoller` + `MarketWSService`）已在 WatchlistPanel 验证可行——实时报价通过 WS 推送到前端，延迟低、开销小。

## 当前状态

### WS 基础设施（已有）

```
Go 后端                          前端
┌──────────────┐              ┌──────────────────┐
│ QuotePoller  │──Publish──→ │ ws.Hub           │──WebSocket─→│ useWebSocket   │
│ (后台轮询)    │              │ (topic→client)   │              │ (连接/订阅)     │
└──────────────┘              └──────────────────┘              └──────────────────┘
                                      │
                                      │ GET /ws/market
                                      │ (MarketWSService)
```

**已有 WS Topic**：`market:quote:{market}:{symbol}` — 单一股票/指数实时报价

**已有数据**：`QuoteSnapshot { last, change, changePct, volume, bid, ask, timestamp }`

### 面板轮询现状

| 面板 | 轮询方式 | 间隔 | 数据 | 改 WS 优先级 |
|------|---------|------|------|------------|
| **CandlestickPanel** 分时 | `setInterval` | 5s | `GetMinuteLine` | **P0** |
| **TickerBar** 全局 | `setInterval` | ~10s | `GetQuote` 批量 | **P0** |
| **CandlestickPanel** K线 | `setInterval` | 30s | `FetchOHLCV` | P1 |
| **LimitUpDownPanel** | `setInterval` | 30s | `GetAbnormalStocks` | P1 |
| **HeatmapPanel** | 手动+缓存 | 30s | `GetIndustryRanks` | P1 |
| **FuturesPanel** | 轮询 | ~10s | 期货行情 | P2 |
| **CryptoOverviewPanel** | 轮询 | 1min | 加密行情 | P2 |
| **FundingRatePanel** | 轮询 | 30s | 资金费率 | P2 |

## Design

### 新增 WS Topic

```
market:quote:{market}:{symbol}     # 已有 — 实时报价
market:minute:{symbol}             # 新增 — 分时 tick 增量推送
market:index:{market}              # 新增 — 指数快照（MarketOverview）
market:abnormal:{market}           # 新增 — 涨跌停/异动
market:industry:{market}           # 新增 — 行业排行
market:ticker:{market}             # 新增 — TickerBar 滚动报价
```

### Go 后端改动

#### 1. QuotePoller 扩展 — 支持多种数据类型

```go
// QuotePoller 当前只拉 QuoteSnapshot，扩展为多类型拉取
type PollTask struct {
    Topic    string           // "market:minute:000001.SH"
    Interval time.Duration    // 5s / 30s / 60s
    Fetcher  func() (any, error)
}

// 新增 Poller 类型
type MinutePoller struct { ... }   // 分时增量
type IndexPoller struct { ... }    // 指数快照
type TickerPoller struct { ... }   // 滚动报价
```

#### 2. ws.Hub 扩展 — 支持 topic 通配符

```go
// 已有: Hub.Broadcast(topic, data)
// 新增: 客户端订阅时支持通配符
//   subscribe: "market:quote:CN:*"    → 所有 A 股报价
//   subscribe: "market:ticker:CN"     → TickerBar 批量
```

#### 3. 发布时机优化

```go
// QuotePoller 根据 market.IsTradingHours() 自动启停
// 闭市时停止拉取，节省 CPU 和网络
if !market.IsTradingHours(mkt) {
    poller.Pause()
} else {
    poller.Resume()
}
```

### 前端改动

#### 1. useRealtimeData — 通用 WS hook

```typescript
// 新建: frontend/src/lib/composables/useRealtimeData.ts
export function useRealtimeData<T>(
  topics: string[] | (() => string[]),
  handler: (topic: string, data: T) => void,
) {
  const ws = useWebSocket()
  const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws/market`

  onMounted(() => {
    const t = typeof topics === 'function' ? topics() : topics
    if (t.length) ws.connect(wsUrl, t)
  })

  ws.onMessage('*', (msg) => {
    handler(msg.topic, msg.data)
  })

  onUnmounted(() => ws.disconnect())

  return { resubscribe: (newTopics: string[]) => ws.connect(wsUrl, newTopics) }
}
```

#### 2. 面板改造模式

每个面板改造遵循统一模式：
```
之前: setInterval(() => fetch(), INTERVAL)
之后: useRealtimeData(topics, (topic, data) => store.update(data))
```

### 分阶段迁移

#### Phase 1 — P0（CandlestickPanel 分时 + TickerBar）

**CandlestickPanel 分时图**：
- Go：新增 `MinutePoller`，按订阅列表 5s 拉取 `GetMinuteLine(symbol, sinceTimestamp)`，增量推送到 `market:minute:{symbol}`
- 前端：`useRealtimeData(['market:minute:600519'], (t, d) => mergeMinuteTicks(d))`
- 去掉：5s `setInterval` + `loadSeq` 竞态守卫（WS 天然有序）

**TickerBar**：
- Go：新增 `TickerPoller`，批量拉取活跃标的报价，推送到 `market:ticker:{market}`
- 前端：`useRealtimeData(['market:ticker:CN', 'market:ticker:HK', 'market:ticker:US'], handler)`
- 去掉：~10s 轮询

#### Phase 2 — P1（涨跌停 + 板块热力图）

**LimitUpDownPanel**：
- Go：`AbnormalPoller` → `market:abnormal:CN`
- 前端：WS 订阅替代 30s 轮询

**HeatmapPanel**：
- Go：`IndustryPoller` → `market:industry:{market}`
- 前端：WS 订阅替代手动刷新

#### Phase 3 — P2（剩余面板）

- FuturesPanel、CryptoOverviewPanel、FundingRatePanel 等
- 逐个改造，遵循统一模式

## Acceptance Criteria

- [ ] CandlestickPanel 分时图通过 WS 实时更新，无 5s `setInterval`
- [ ] TickerBar 通过 WS 实时更新，无轮询定时器
- [ ] 闭市时 QuotePoller 自动暂停，前端 WS 连接保持但不推送
- [ ] 切换股票/市场时 WS 自动重订阅
- [ ] 多个面板同时打开同一标的不产生重复 WS 消息
- [ ] 所有改面板的现有测试通过
- [ ] CHANGELOG 更新

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| WS 连接断开时无数据 | `useWebSocket` 已有指数退避重连（1s→30s），断连时回退到 IPC 轮询 |
| 大量订阅导致 Hub 广播风暴 | Hub 按 topic 分组，只推送给订阅客户端；批量 topic 合并 |
| Go routine 泄漏 | 每个 Poller 有独立 context + cancel；面板卸载时退订 |
| 分时增量推送顺序 | WS 基于 TCP，消息有序到达；loadSeq 守卫改为 WS seq |
| 向后兼容 | 新旧面板共存；未迁移面板继续轮询不受影响 |

## 不在此 Spec 范围

- K 线图 WS 推送（K 线数据量大、变化频率低，30s 轮询可接受）
- 工作流模式实时数据（工作流是异步执行的，不需要实时推送）
- Python sidecar 直连 WS（仍然走 Go→Python→Go→WS 链路）
