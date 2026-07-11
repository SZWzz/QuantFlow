# 自适应轮询器优化

## Motivation

当前 `QuotePoller` 和 `MinutePoller` 使用固定 5s 轮询间隔，无论市场状态如何：

```go
// poller.go:34
interval: 5 * time.Second  // 交易时段 5s，闭市时也 5s
```

问题：
1. **闭市时空转**：A 股 15:00-次日 9:30 + 周末 + 节假日，QuotePoller 仍在每 5s 轮询，产生大量无用 HTTP 请求
2. **开盘时段延迟浪费**：加密市场 24/7 交易，5s 间隔对 <100ms 波动的加密行情太慢
3. **多个 Poller 独立 ticker**：QuotePoller（5s）+ MinutePoller（5s）= 每秒最多 4 次 HTTP 请求（2 pollers × 2 markets × 1s 内交错），同市场 symbol 本可合并

## Design

### 1. 自适应间隔

```go
// internal/market/poller.go
type PollerConfig struct {
    TradingInterval   time.Duration  // 交易时段间隔 (默认 3s)
    OffHoursInterval  time.Duration  // 闭市间隔 (默认 60s)
    OffHoursEnabled   bool           // 闭市是否完全停轮
}

var DefaultPollerConfig = PollerConfig{
    TradingInterval:  3 * time.Second,
    OffHoursInterval: 60 * time.Second,
    OffHoursEnabled:  true,  // 闭市完全停轮
}
```

Poller `pollOnce` 入口增加市场状态感知：

```go
func (p *QuotePoller) pollOnce(ctx context.Context) {
    // 检查订阅列表中所有市场的交易状态
    p.mu.RLock()
    hasActive := false
    for key := range p.subs {
        market, _ := splitSubscriberKey(key)
        if IsTradingHours(market) {
            hasActive = true
            break
        }
    }
    p.mu.RUnlock()

    if !hasActive {
        if p.config.OffHoursEnabled {
            return  // 闭市完全停轮
        }
        // 非交易时段用长间隔 (但 pollOnce 由 ticker 驱动，间隔由 adjustInterval 控制)
    }
    // ... 正常轮询逻辑
}
```

增加动态间隔调整：

```go
// 在 Run() 的 ticker 循环中动态调整间隔
func (p *QuotePoller) Run(ctx context.Context) {
    ticker := time.NewTicker(p.config.TradingInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            p.pollOnce(ctx)
            // 根据市场状态动态调整下一个 tick 间隔
            newInterval := p.adjustInterval()
            ticker.Reset(newInterval)
        }
    }
}

func (p *QuotePoller) adjustInterval() time.Duration {
    p.mu.RLock()
    defer p.mu.RUnlock()

    hasTrading := false
    for key := range p.subs {
        market, _ := splitSubscriberKey(key)
        if IsTradingHours(market) {
            hasTrading = true
            break
        }
    }

    if hasTrading {
        return p.config.TradingInterval  // 3s
    }
    if p.config.OffHoursEnabled {
        return p.config.OffHoursInterval // 60s
    }
    return p.config.TradingInterval
}
```

### 2. 批量请求聚合

同市场多个 symbol 合并为一次 HTTP 请求（新浪/腾讯支持逗号分隔）：

```go
// internal/market/poller.go

// groupByMarket 按市场分组 symbol
func (p *QuotePoller) groupByMarket() map[string][]string {
    p.mu.RLock()
    defer p.mu.RUnlock()

    groups := make(map[string][]string)
    for key := range p.subs {
        market, symbol := splitSubscriberKey(key)
        groups[market] = append(groups[market], symbol)
    }
    return groups
}

// batchFetch 批量获取报价（适配器可选实现 BatchQuoteProvider 接口）
func (p *QuotePoller) batchFetch(ctx context.Context, groups map[string][]string) {
    for market, symbols := range groups {
        // 检查适配器是否支持批量查询
        if provider, ok := p.reg.GetBatchProvider(market); ok {
            quotes, err := provider.FetchBatchQuotes(ctx, symbols)
            if err != nil {
                // 回退到逐个查询
                p.fetchIndividual(ctx, market, symbols)
                continue
            }
            for _, q := range quotes {
                topic := "market:quote:" + q.Symbol
                p.marketHub.Publish(topic, q)
                p.wsHub.Broadcast(topic, q)
            }
        } else {
            p.fetchIndividual(ctx, market, symbols)
        }
    }
}
```

新增 `BatchQuoteProvider` 可选接口：

```go
// internal/market/adapter.go
type BatchQuoteProvider interface {
    // FetchBatchQuotes fetches quotes for multiple symbols in one request.
    // Market-specific: CN symbols can be comma-separated for sina/tencent.
    FetchBatchQuotes(ctx context.Context, symbols []string) ([]*QuoteSnapshot, error)
}
```

### 3. 修改文件

| 文件 | 改动 |
|------|------|
| `internal/market/poller.go` | 新增 `PollerConfig`、自适应间隔、市场状态感知、批量聚合 |
| `internal/market/minute_poller.go` | 同步自适应间隔逻辑 |
| `internal/market/adapter.go` | 新增 `BatchQuoteProvider` 可选接口 |
| `internal/market/registry.go` | 新增 `GetBatchProvider(market) (BatchQuoteProvider, bool)` |
| `internal/market/adapters/sina.go` | 实现 `BatchQuoteProvider` (新浪已支持逗号分隔) |
| `internal/market/adapters/tencent.go` | 实现 `BatchQuoteProvider` (腾讯已支持逗号分隔) |
| `app_startup.go` | 传 `PollerConfig` 给 QuotePoller/MinutePoller |

## Acceptance Criteria

- [ ] 交易时段 QuotePoller 间隔 3s，闭市完全停轮
- [ ] 跨市场（如 CN + CRYPTO）只要有任一市场交易就保持 3s 轮询
- [ ] 新浪/腾讯适配器实现批量查询，10 个 CN symbol 合并为 1 次 HTTP 请求
- [ ] 批量查询失败时自动回退到逐个查询
- [ ] MinutePoller 同步自适应间隔
- [ ] 无批量支持的市场（如 Yahoo）维持逐个查询，不受影响
- [ ] 非交易时段 Zero HTTP 请求（QuotePoller 完全停轮）
- [ ] 后端构建通过，所有已有测试通过
- [ ] CHANGELOG 更新

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 批量查询部分失败时难以精确回退 | 逐个查询作为保底，不影响正确性 |
| 自适应间隔 ticker.Reset 可能有竞态 | `select` 中 `ticker.C` 和 `adjustInterval` 串行执行，安全 |
| 闭市完全停轮后开盘第一笔数据延迟 | 开盘时 `IsTradingHours` 恢复 true，下一个 tick 立即轮询（最多 3s 延迟） |
| 不同市场交易时段不同（CN 9:30-15:00 vs US 9:30-16:00 ET） | `IsTradingHours(market)` 已按市场区分，Poller 以此判断 |
