# 市场概括面板性能优化

## Motivation

用户切换到市场概括面板时，每次触发 3 次后端调用（GetMarketOverview → GetIndustryRanks → GetBlockRank），串联执行导致总等待时间可达 5-15 秒。具体瓶颈：

1. `GetIndustryRanks` 有 3 次重试 + 500ms 全局限流器，EastMoney push2 API 不可靠时最坏 10s+
2. 前端的 `GetMarketOverview` 和 `GetIndustryRanks` 串行 await，浪费并行机会
3. IndustryRanks 前后端均无缓存
4. MAC adapter 每次创建 TCP 新连接（到 `119.147.212.81:7709`），无连接复用

## Design

### 数据流（优化后）

```
before (旧):
  GetMarketOverview (30s cache, 1-3s) —→ GetIndustryRanks (3 retries, 2-10s) —→ GetBlockRank (TCP 新连接, 0.5-2s)
                                                                                总共: 3.5-15s

after (新):
  ┌─ GetMarketOverview (60s cache, 1-3s) ─┐
  ├─ GetIndustryRanks (1 retry, 0.5-3s)   ├── Promise.all 并行
  └─ GetBlockRank (连接池复用, 0.1-0.5s)   ┘  总共: 0.5-3s
                                          ↓
                                    marketOverview.value + 5min cache
```

### 修改文件

| 文件 | 改动 |
|------|------|
| `app_market.go` | 缓存 TTL 30s → 60s |
| `internal/market/adapters/eastmoney_signals.go` | FetchIndustryRanks 重试次数 2 → 0（不重试），失败时立即返回空切片 |
| `frontend/src/stores/data.ts` | 1. `GetMarketOverview` 和 `GetIndustryRanks` 并行执行 2. IndustryRanks 前端缓存 5 分钟 |
| `internal/market/adapters/mac.go` | MacAdapter 增加 TCP 连接池复用（`sync.Mutex` + 持久连接，断线自动重连） |
| `frontend/src/terminal/panels/MarketOverviewPanel.vue` | 1. loading 状态改为骨架屏 2. 缓存未命中时显示旧数据 + 后台刷新 |

### API 变更

无。所有变更均为内部实现优化，不修改暴露的 Wails 方法签名。

### 缓存策略

| 数据 | 缓存位置 | TTL | 说明 |
|------|----------|-----|------|
| MarketOverview indices | Go `marketOverviewCache` | 60s | 指数报价 |
| IndustryRanks | 前端 Pinia dataStore | 5min | 行业涨跌排名，变化较慢 |
| BlockRank | 无变化 | — | 实时数据，不缓存 |

## Acceptance Criteria

- [ ] 面板加载时间从 5-15s 降低到 ≤3s
- [ ] `GetIndustryRanks` 在 EastMoney 不可用时立即返回空（≤2s），不阻塞
- [ ] MAC adapter 连接复用：第二次开始不新建 TCP
- [ ] IndustryRanks 5 分钟内不重复请求
- [ ] 面板显示骨架屏/旧数据，不白屏
- [ ] Go build 通过

## Risks / Trade-offs

- **MAC 连接池引入并发复杂度**：Mutex 序列化所有请求，不支持并发调用同一 adapter。但当前 GetBlockRank 调用频率很低（15s 一次），序列化影响可以忽略。
- **IndustryRanks 不重试**：极端情况下 EastMoney 偶发失败会显示空行业排名，而不是等 6s 重试获取。权衡后认为空数据比卡住更好。
- **前端缓存增加内存**：~10KB 级别，可忽略。
