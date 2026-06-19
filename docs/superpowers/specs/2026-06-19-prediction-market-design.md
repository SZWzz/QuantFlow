# PredictionMarket 预测市场模块 — 设计文档

> **Status**: Design — 等待实施计划
> **Part of**: 另类数据模块，子项目 1/4
> **Priority**: 🔴 高

## Motivation

Polymarket 是全球最大的去中心化预测市场平台，提供数百个事件（美联储决策、CPI、选举、加密价格等）的实时概率定价。将预测市场数据接入 QuantFlow 可让用户：

1. 浏览全球宏观+加密事件的市场隐含概率
2. 从概率突变中提取交易信号
3. 将预测市场信号接入工作流管道

Polymarket 有完整的公开 REST API，免费无需 API Key。

## Design

### Architecture

```
Polymarket REST API (clob.polymarket.com)
    │  GET /markets — 事件列表（分页/过滤）
    │  GET /markets/{id} — 事件详情 + outcomes 价格
    │  GET /markets/{id}/prices?interval=1d — 价格历史
    ▼
PolymarketAdapter (internal/market/adapters/polymarket.go)
    │  接口: PolymarketAdapter interface
    │  FetchEvents(category, limit) → []PredictionEvent
    │  FetchEvent(id) → PredictionEvent
    │  FetchPriceHistory(id, interval) → []PricePoint
    │
    ▼
PredictionMarketService (internal/research/prediction_market_service.go)
    │  业务逻辑 + 内存缓存(5min TTL) + 信号提取
    │  GetEvents(category) — 带缓存的事件列表
    │  GetEventDetail(id) — 单个事件详情
    │  ExtractSignals(minProbChange) — 概率突破检测
    │
    ├──► PredictionMarketPanel (Vue 前端)
    │     ├── 事件筛选栏：按类别过滤（全部/politics/economics/crypto/sports/tech/entertainment）
    │     ├── 事件表格：标题 | Yes概率 | 24h变化 | 成交量 | 到期时间
    │     ├── 事件详情卡：Yes/No 实时价 + 概率走势 ECharts 折线图
    │     └── 信号高亮：概率突变行高亮（变化 >5%）
    │
    └──► prediction_market 工作流节点
          输入端口: category (string) / min_volume (number) / min_prob_change (number)
          输出端口: top_events (JSON) / signal (Signal) / probability (number)
          参数: lookback_hours (默认 24)
```

### Data Model

```go
// PolymarketAdapter — 预测市场专用接口，不同于 market.Adapter
type PolymarketAdapter interface {
    Name() string
    IsAvailable(ctx context.Context) bool

    // FetchEvents returns prediction market events, optionally filtered by category.
    // Valid categories: politics, economics, crypto, sports, tech, entertainment, science.
    FetchEvents(ctx context.Context, category string, limit int) ([]PredictionEvent, error)

    // FetchEvent returns a single event with full outcome details.
    FetchEvent(ctx context.Context, id string) (*PredictionEvent, error)

    // FetchPriceHistory returns historical prices for an outcome.
    // interval: "1h", "6h", "1d".
    FetchPriceHistory(ctx context.Context, outcomeID string, interval string, limit int) ([]PricePoint, error)
}

type PredictionEvent struct {
    ID          string           `json:"id"`          // Polymarket slug (e.g., "fed-rate-cut-july")
    Title       string           `json:"title"`       // "Federal Reserve rate cut by July 2026?"
    Category    string           `json:"category"`    // politics / economics / crypto / sports / tech / entertainment
    Volume      float64          `json:"volume"`      // Total volume (USD)
    Liquidity   float64          `json:"liquidity"`   // Open interest
    EndDate     string           `json:"end_date"`    // Market close date (ISO 8601)
    Status      string           `json:"status"`      // open / closed / resolved
    Outcomes    []PredictionOutcome `json:"outcomes"`
    Description string           `json:"description"` // Market description text
    UpdatedAt   int64            `json:"updated_at"`  // Last update unix ms
}

type PredictionOutcome struct {
    ID         string  `json:"id"`         // Outcome ID
    Label      string  `json:"label"`      // "Yes" / "No" / specific option
    Price      float64 `json:"price"`      // Current price (0-1, = probability)
    Change24h  float64 `json:"change_24h"` // 24h price change (pct points)
}

type PricePoint struct {
    Timestamp int64   `json:"timestamp"` // Unix ms
    Price     float64 `json:"price"`     // 0.0 - 1.0
    Volume    float64 `json:"volume"`    // Cumulative volume at this point
}
```

### Polymarket API Mapping

Polymarket's CLOB API returns different field names. Adapter maps:
- `title` → `Title`
- `volume` / `liquidity` → `Volume` / `Liquidity`
- `end_date_iso` → `EndDate`
- `closed` → `Status == "closed"`
- `outcomes` (JSON array) → `[]PredictionOutcome`
- Category inferred from `tags` field

### Frontend Panel Design

**PredictionMarketPanel** (`prediction-market`):

| 区域 | 内容 |
|------|------|
| 工具栏 | 类别下拉过滤器 + 排序选择 (vol/change/endDate) + 刷新按钮 |
| 主表格 | 事件行：标题 | Yes价格 | 24h变化 | 成交量 | 到期倒计时 |
| 详情展开 | 点击行展开：概率走势图(ECharts 折线) + 全部 outcomes 价格 |
| 信号徽标 | 概率变化 >5% 的事件行显示 🔴 突破徽标 |
| 空状态 | "加载中..." / "API 不可用 — 显示模拟数据" |

Mock 数据：提供 5 个合理的中文模拟事件（美联储降息/BTC突破10万/CPI等），确保离线可渲染。

### Workflow Node Design

**prediction_market** 节点，category: "alternative_data"

- 输入: `category` (string), `min_prob_change` (number)
- 输出: `top_events` (JSON string of top-N events), `signal` (Signal: buy/sell/hold + confidence)
- 参数: `lookback_hours` (int, 默认 24), `signal_threshold` (float, 默认 0.05)
- 信号逻辑: 任一事件的 Yes 概率突破阈值 → 生成对应方向的 signal

### Files

#### New (6)
- `internal/market/adapters/polymarket.go` — Polymarket HTTP adapter
- `internal/market/adapters/polymarket_test.go` — Adapter tests
- `internal/research/prediction_market_service.go` — Service + cache + signal extraction
- `internal/workflow/nodes/prediction_market.go` — Workflow node
- `frontend/src/terminal/panels/PredictionMarketPanel.vue` — Frontend panel
- `frontend/src/terminal/panels/__tests__/PredictionMarketPanel.test.ts` — Panel test

#### Modified (4)
- `frontend/src/terminal/panels/registry.ts` — Register `prediction-market` panel
- `internal/workflow/nodes/register.go` — Register `prediction_market` node
- `internal/workflow/nodes/research_deps.go` — Add `predictionMarketService` var + setter
- `app.go` — Create adapter + service in startup(), export `GetPredictionMarkets()` method
- `CHANGELOG.md` — Record changes

### Graceful Degradation

```
Panel loads
  → Try Wails IPC: app.GetPredictionMarkets("all")
  → Go service:
      → Polymarket API available? → FetchEvents → return real data
      → API timeout/error? → return mock events (5 items)
  → Wails not available (vue-tsc/dev)?
      → Panel fallback: use inline mockCryptos-style mock data
```

## Acceptance Criteria

- [ ] `PolymarketAdapter.FetchEvents("all", 20)` returns real events from Polymarket API
- [ ] `PolymarketAdapter.FetchEvent("fed-rate-cut-july")` returns event with outcomes
- [ ] `PolymarketAdapter.FetchPriceHistory(outcomeID, "1d", 30)` returns price points
- [ ] Panel renders event table with category filter + sort + expand/collapse
- [ ] Panel shows ECharts probability trend line chart on event detail
- [ ] Panel degrades to mock data when API unavailable
- [ ] Workflow node `prediction_market` registered and executable
- [ ] Node outputs `top_events` JSON and `signal` Signal
- [ ] `app.go` exports `GetPredictionMarkets(category string, limit int)` 
- [ ] All existing tests pass: Go tests + frontend tests
- [ ] `go vet ./...` clean
- [ ] `npx vue-tsc --noEmit` clean

## Risks / Trade-offs

- **Polymarket CLOB API 字段可能有变化** — adapter 做 defensive parsing，缺失字段用 zero value
- **国内访问 Polymarket** — 可能需要代理；API 不可用时走 mock fallback
- **预测市场数据量小** — 无需复杂缓存，5 分钟 TTL 内存 map 足够
- **类别推断** — Polymarket 无显式 category 字段，从 tags/keywords 推断，可能存在误分类
