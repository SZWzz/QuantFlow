# MarketStyle Implementation Plan

**Goal:** Market style quadrant (large/small × value/growth) + sentiment thermometer + northbound/margin flows.

**Architecture:** `StyleService` computes style rotation from index comparisons (SSE50 vs CSI1000, CSI300 growth vs value). `MarketStylePanel.vue` renders quadrant scatter + sentiment gauges + flow charts.

**Data sources (all existing):**
- `MarketOverview` indices → SSE50, CSI300, CSI500, CSI1000, Chinext prices
- `MarketSentiment` → limitUp/limitDown, northboundFlow, totalVolume
- `THSNorthboundAdapter` → northbound daily flow
- `macro_cn.market_margin_sh/sz` → margin balance via Python sidecar

---

### Task 1: StyleService + Go test

**Files:**
- Create: `internal/market/style_service.go`
- Test: `internal/market/style_service_test.go`

```go
// internal/market/style_service.go
type StyleQuadrant struct {
    Index     string  `json:"index"`
    Size      float64 `json:"size"`      // normalized market cap score
    Style     float64 `json:"style"`     // value (low PE/PB) vs growth score
    Return1M  float64 `json:"return_1m"`
    Return3M  float64 `json:"return_3m"`
}

type MarketSentimentGauge struct {
    LimitUp     int     `json:"limit_up"`
    LimitDown   int     `json:"limit_down"`
    Turnover    float64 `json:"turnover"`     // 亿
    TurnoverRate float64 `json:"turnover_rate"` // %
    NorthboundCum float64 `json:"northbound_cum"` // 30日累计 亿
    MarginBalance float64 `json:"margin_balance"` // 万亿
}

type StyleService struct {
    reg          *AdapterRegistry
    northbound   *adapters.THSNorthboundAdapter
    bridge       *python.PythonBridge
}

func (s *StyleService) GetStyleQuadrant(ctx context.Context) ([]StyleQuadrant, error) { ... }
func (s *StyleService) GetSentiment(ctx context.Context) (*MarketSentimentGauge, error) { ... }
```

**Commit:** `feat(market): add StyleService with quadrant and sentiment computation`

---

### Task 2: MarketStylePanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/MarketStylePanel.vue`
- Modify: registry.ts → register `market-style`

**UI:** Top row: ECharts scatter quadrant (size × style axes, labeled indices). Middle row: sentiment thermometer (limitUp/Down bars, turnover gauge). Bottom: northbound 30-day cumulative + margin balance chart.

**Commit:** `feat(frontend): add MarketStyle panel with quadrant and sentiment`

---

### Task 3: IPC + wire

**Files:**
- Create: `app_style.go`
- Modify: `frontend/src/lib/wails.ts`
- Modify: `app_startup.go` / `app.go`

**Commit:** `feat(backend+frontend): wire MarketStyle IPC`
