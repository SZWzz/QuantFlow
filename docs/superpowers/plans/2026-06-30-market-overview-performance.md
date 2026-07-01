# 市场概括性能优化 — 实施计划

Phase 对应 spec `docs/specs/2026-06-30-market-overview-performance.md`。

## Task 1: Go 后端 — GetMarketOverview 缓存 TTL 30s → 60s

**文件**: `app_market.go:42`

```go
c.expires = time.Now().Add(60 * time.Second)
```

## Task 2: Go 后端 — FetchIndustryRanks 减少重试 + 移除全局限流

**文件**: `internal/market/adapters/eastmoney_signals.go:217-290`

- 删除 retry 循环，只查一次
- 失败时返回空切片 + error，不阻塞
- 注意：不修改 limiter.Wait()（仍需要避免触发反爬）

## Task 3: Go 后端 — MAC adapter 连接池

**文件**: `internal/market/adapters/mac.go`

- MacAdapter struct 增加 `conn net.Conn` + `mu sync.Mutex`
- `getConn()` 方法：返回缓存连接，断线时自动重连
- `sendMACRequest` 改造为复用连接：加锁 → 用缓存 conn 发送 → 读响应 → 解锁
- 新方法 `Close()` 关闭连接

## Task 4: 前端 — data.ts 并行化 + IndustryRanks 缓存

**文件**: `frontend/src/stores/data.ts:111-163`

- `fetchMarketOverview` 中 `GetMarketOverview` 和 `GetIndustryRanks` 用 `Promise.all` 并行
- IndustryRanks 结果存入 cache (5min TTL)，下次直接使用

## Task 5: 前端 — MarketOverviewPanel.vue 骨架屏

**文件**: `frontend/src/terminal/panels/MarketOverviewPanel.vue`

- loading 时用 CSS 骨架屏（灰度渐变脉冲动画），保持布局占位
- 缓存命中时立即显示旧数据，后台静默刷新
