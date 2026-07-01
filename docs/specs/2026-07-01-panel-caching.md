# Panel 前端缓存全面覆盖

## Motivation

目前 57 个面板中，26 个已使用 `usePanelCache` + `fetchWithCache` 进行请求去重和 TTL 缓存，31 个面板每次挂载/刷新都直接调用 Go Wails 方法，无任何缓存层。导致：

1. **重复请求** — 切换 tab、切换标的、面板重建时反复拉取同一数据
2. **Python sidecar 压力** — 多个面板同时请求同一 AKShare/macro 端点，每个都触发 gRPC 子进程（60-90s）
3. **Go 侧 `FetchData` TTL 缓存仅覆盖 `FetchData` 路径**，直接调 Wails 方法的面板仍无缓存

## 设计

### 总体策略

统一使用已有的 `usePanelCache` composable（`fetchWithCache`），对所有面板按数据特性配置不同 TTL：

| 数据类型 | TTL | 说明 |
|---------|-----|------|
| 宏观/基本面 (macro_cn, financials, settlement) | 30min | 低频变化 |
| 衍生品/日历 (options, futures, IPO, earnings) | 15min | 盘中变化极慢 |
| 资金流向/板块 (fundflow, sector, margin) | 5min | 盘中缓慢变化 |
| 链上/鲸鱼 (whale, gas, defi) | 3min | 中等频率 |
| 深度/盘口 (depth, auction, funding) | 1min | 较高频率 |
| 行情报价 (quote, ticker, watchlist) | 不缓存 | 需近实时 |

### 修改模式

每个面板的修改模式一致：

```typescript
// Before
const { data } = await app.GetSomeData(symbol)

// After  
const { data } = await fetchWithCache(`cache_key:${symbol}`, () => app.GetSomeData(symbol), TTL)
```

例外：对使用 `useDataFetch` 的面板，保留其状态管理，在 fetch 逻辑内层包装 `fetchWithCache`。

### 数据流

```
Panel mounted
  → fetchWithCache(key, fn, ttl)
    → usePanelCache: check Map<key, {data, expiresAt}>
      → hit? return cached data (no backend call)
      → miss? call fn() → store in Map → return data
```

`usePanelCache` 是前端内存缓存，页面刷新后丢失。持久化由 Go 侧 `FetchData` TTL 缓存覆盖。

## 涉及的面板

分组按修改相似度，方便 plan 分 task：

### Group A — 简单 fetchWithCache 包装（15 面板）

单个 API 调用，直接 `const { data } = await fetchWithCache(...)` 替换现有 `await app.Xxx()` 即可：

| 面板 | 方法 | TTL |
|------|------|-----|
| ForecastPanel | `GetForecast` | 30min |
| HKSettlementPanel | `GetHKSettlementInfo`, `GetHKTradingCalendar` | 30min |
| DragonTigerPanel | `GetDailyDragonTiger`, `GetDragonTiger` | 5min |
| SurfaceChartPanel | `GetVolatilitySurface` | 15min |
| PredictionMarketPanel | `GetPredictionMarkets`, `GetPredictionEventDetail`, `GetPredictionSignals` | 15min |
| SchedulePanel | `ListScheduleTasks` | 5min |
| CBArbitragePanel | `GetCBArbitrageData` | 15min |
| HKDerivativesPanel | `GetHKDerivatives` | 15min |
| HKIPOPanel | `GetHKIPOCalendar` | 15min |
| GeopoliticsPanel | `GetGeopoliticsRisks`, `GetGeopoliticsDetail` | 30min |
| SatellitePanel | `GetSatelliteSnapshots`, `GetSatelliteDetail` | 30min |
| WhaleTrackingPanel | `GetWhaleTransactions` | 3min |
| DefiTVLPanel | `GetDeFiTVL` | 3min |
| CryptoOverviewPanel | `GetCryptoOverview` | 3min |
| FuturesPanel | `FetchData(akshare/...)` | 15min |

### Group B — 高频轮询 + cache（7 面板）

已有 `setInterval` 轮询，加上缓存避免重复请求（但轮询逻辑仍按原间隔触发）：

| 面板 | 间隔 | 方法 | TTL |
|------|------|------|-----|
| GasFeePanel | 12s | `GetGasFees` | 1min |
| DepthComparisonPanel | 15s | `GetCryptoDepth` | 1min |
| FundingRatePanel | 30s | `GetCryptoFundingRates` | 1min |
| LiquidationPanel | 30s | `GetCryptoLiquidations` | 1min |
| MarketDepthPanel | 手动 | `GetQuote`, `GetAuction`, `GetDepth` | 1min |
| MarketOverviewPanel | 60s | `GetBlockRank` | 5min |
| WatchlistPanel | 手动 | `GetQuote` | 不缓存（实时） |

### Group C — 交易/实时（7 面板）

数据敏感性高，不加缓存：

| 面板 | 说明 |
|------|------|
| TickerTapePanel | 行情滚动，需实时 |
| QuoteDetailPanel | 个股详情，需实时 |
| OrderEntryPanel | 下单面板，需实时 |
| BasketOrderPanel | 篮子下单，需实时 |
| PositionDetail | 持仓，需实时 |
| AIChatPanel | 对话无缓存语义 |
| StockScannerPanel | 扫描结果每次不同 |

### Group D — 已有自定义缓存（2 面板）

确认已有缓存机制，暂不修改：

| 面板 | 缓存机制 |
|------|---------|
| GovDataPanel | `dataStore.getCached/setCached` |
| MarketOverviewPanel | `dataStore.fetchMarketOverview` 自带缓存 |

## 修改列表

### 新增引入
- `import { usePanelCache } from '@/lib/composables/usePanelCache'`

### 新增调用
- `const { fetchWithCache } = usePanelCache()`

### 替换模式

```typescript
// 所有 Group A 面板：直接包装
const { data: result } = await fetchWithCache(
  `key:${symbol}`,
  () => app.GetXxx(symbol),
  TTL_ms
)

// 所有 Group B 面板：轮询函数内包装
async function fetchData() {
  const { data } = await fetchWithCache(`key`, () => app.GetXxx(), TTL_MS)
  if (data) panelData.value = data
}
```

## Acceptance Criteria

- [ ] Group A 15 个面板全部添加 `fetchWithCache` 包装，TTL 符合配置
- [ ] Group B 7 个面板轮询函数内添加 `fetchWithCache`，轮询间隔不变
- [ ] Group C 7 个面板确认不修改
- [ ] TypeScript 类型检查通过：`npx vue-tsc --noEmit` 无新增错误
- [ ] 构建通过：`make build-full`
- [ ] 宏观面板首次加载后 30min 内再次打开不再触发 gRPC 调用

## Risks / Trade-offs

- **内存占用** — `usePanelCache` 是 `Map<string, {data, expiresAt}>`，31 面板各 1-3 个 key，最多 ~100 条目，每条 ~10-100KB，总量 <10MB，可接受
- **数据陈旧** — 缓存数据可能在 TTL 内过时（如 DragonTiger 收盘后更新）。方案：TTL 设保守值；对盘中变化极快的品种（depth, funding rate）用 1min 短 TTL
- **开发成本** — 22 个面板需修改，每个修改量约 3-8 行，总计 ~150 行改动，风险可控
