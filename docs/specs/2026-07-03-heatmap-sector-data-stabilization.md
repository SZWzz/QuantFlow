# Heatmap Sector Data Stabilization

## Motivation

板块热力图 (`HeatmapPanel`) 长期无数据或数据错误，原因有三：
1. **CN**: `GetIndustryRanks` 只依赖 `push2.eastmoney.com` 单一数据源，失败时静默吞错误，面板显示"暂无板块数据"。
2. **HK/US**: `GetIndustryRanks` 无市场参数，永远只拉 A 股行业数据；且 HK/US 无对应的板块数据源。
3. **缓存 bug**: `marketOverviewCache` 忽略 `mkt` 参数，单 key 存所有市场数据，切换市场 60s 内返回脏数据。

## Design

### 数据流

```
HeatmapPanel.vue
  → dataStore.fetchMarketOverview(market)
    → app.GetMarketOverview(market)         // 指数 + 宽度数据
    → app.GetIndustryRanks(market, topN)    // 板块排名 (新增 market 参数)
```

### P1: CN 板块 — 东财 push2 + 重试 + 错误传播

`EastMoneySignalsAdapter.FetchIndustryRanks` 是唯一可用的 A 股板块排名数据源（`push2.eastmoney.com/api/qt/clist/get?fs=m:90+t:2`）。
该接口实际很稳定（同花顺/东财自身都在用），问题在于：
1. `GlobalEMLimiter` 限流（500ms）与接口返回慢叠加导致超时
2. 错误被静默吞掉（`slog.Warn` + 返回空）

**改进**:
- 增加 `RetryWithBudget` 包装（最多 2 次重试，500ms/1s 间隔）
- 将限流超时从 5s 放宽到 10s
- **不再静默吞错误**：三次重试均失败后，将 error 传播到前端
- 前端显示"板块数据获取失败，请稍后重试"而非空面板

### P2: HK 板块 — Tencent 新增 sector endpoint

**新增**: `TencentAdapter.FetchIndustryRanks(ctx, market, topN)`
- 请求 `http://web.ifzq.gtimg.cn/appstock/app/HK/hkranking?type=industry`
- 解析返回的行业排名 JSON
- 仅对 `market=="HK"` 生效

**Fallback**: 无（不降级，返回空 + warn log）

### P3: US 板块 — Finnhub sector API

**新增**: `FinnhubAdapter.FetchIndustryRanks(ctx, market, topN)`
- 请求 `GET https://finnhub.io/api/v1/sector?token={key}`
- 返回 11 大行业（Technology, Healthcare 等）涨跌百分比
- 仅对 `market=="US"` 生效

**Fallback**: 无（不降级，返回空 + warn log）

### P4: 修复缓存 bug

`marketOverviewCache` 改为 `map[string]cacheEntry` keyed by market:

```go
type marketOverviewCache struct {
    mu      sync.Mutex
    entries map[string]*cacheEntry
}
type cacheEntry struct {
    data    map[string]interface{}
    expires time.Time
}
func (c *marketOverviewCache) get(mkt string) (map[string]interface{}, bool)
func (c *marketOverviewCache) set(mkt string, data map[string]interface{})
```

### P5: 修复前端

- 为 HK/US 显示对应的 sector 标签
- 对 HK/US 市场无 sector data 时显示对应提示而非 CN 数据
- 前端请求 `GetIndustryRanks(market, topN)` 传递当前市场参数

### 接口变更

**Go 导出函数**:
```go
// Before: func (a *App) GetIndustryRanks(topN int) ([]IndustryRank, error)
// After:
func (a *App) GetIndustryRanks(mkt string, topN int) ([]IndustryRank, error)
```

**Pinia store**: `fetchMarketOverview` 传递 market 给 `GetIndustryRanks`

**新增 adapter 接口**:
```go
type IndustryRankProvider interface {
    FetchIndustryRanks(ctx context.Context, market string, topN int) ([]IndustryRank, error)
}
```

### 新增/修改文件

| 文件 | 变更 |
|------|------|
| `app.go` | `GetIndustryRanks` 加 `mkt` 参数，调用 `fetchIndustryRanksWithFallback` |
| `app_market.go` | `marketOverviewCache` 改为 per-market；`GetMarketOverview` 传递 market |
| (mac.go not needed — protocol is for block trades, not sector ranking) |
| `internal/market/adapters/tencent.go` | 新增 `FetchIndustryRanks` (HK sector) |
| `internal/market/adapters/finnhub.go` | 新增 `FetchIndustryRanks` (US sector) |
| `internal/market/adapters/adapter.go` | 新增 `IndustryRankProvider` 接口 |
| `internal/market/registry.go` | 新增 `FetchIndustryRanksWithFallback` 按 market 选择适配器 |
| `frontend/src/stores/data.ts` | `fetchMarketOverview` 传递 market 给 GetIndustryRanks |
| `frontend/src/terminal/panels/HeatmapPanel.vue` | HK/US 无数据时显示友好提示 |
| `docs` | CHANGELOG 更新 |

## Acceptance Criteria

- [ ] CN 板块热力图稳定显示：东财 push2 重试 3 次，失败时前端显示明确错误信息
- [ ] HK 板块显示腾讯财经港股行业排名（或无数据时提示"港股板块数据暂不可用"）
- [ ] US 板块显示 Finnhub 11 大行业涨跌
- [ ] 切换市场不显示脏缓存数据
- [ ] `GetIndustryRanks(mkt, topN)` 透传 market 参数，不再混用 CN 数据
- [ ] 测试覆盖：三个市场的 `FetchIndustryRanks` 单元测试

## Risks / Trade-offs

- **CN 单数据源风险**：东财 push2 是 CN 板块数据的唯一来源（经调研 TDX MAC 协议、同花顺、新浪均无合适的行业排名 API）。如果东财 push2 长期不可用，CN 板块热力图将不可用。这是一个**已知的外部依赖风险**，无法完全消除。
- **Finnhub 免费限制**：60 req/min，US sector 调用会占用额度，需控制调用频率（每 30s 一次）。
- **腾讯 HK sector API**：`web.ifzq.gtimg.cn/appstock/app/HK/hkranking?type=industry` 未经验证，需先爬取确认接口可用性和响应格式。
