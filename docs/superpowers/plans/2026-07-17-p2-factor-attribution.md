# FactorAttribution Implementation Plan

**Goal:** Portfolio return decomposition into market β + style factors + industry factors + stock-specific α.

**Architecture:** Leverages existing Python `factor_engine.py` for Barra-style factor computation. `FactorAttributionPanel.vue` renders waterfall chart + factor exposure heatmap.

**Data sources (existing):**
- Python `factor_engine.py` → Barra CNE5 factors (size, value, momentum, volatility, quality, growth, leverage, liquidity)
- Portfolio positions → `OMS.GetAllPositions()`
- OHLCV → stock returns

---

### Task 1: FactorAttributionService + Go test

**Files:**
- Create: `internal/portfolio/attribution.go`
- Test: `internal/portfolio/attribution_test.go`

```go
// internal/portfolio/attribution.go
type FactorAttribution struct {
    TotalReturn   float64              `json:"total_return"`
    MarketBeta    float64              `json:"market_beta"`    // β × market return
    StyleFactors  map[string]float64   `json:"style_factors"`  // size, value, momentum...
    IndustryFactors map[string]float64 `json:"industry_factors"`
    Alpha         float64              `json:"alpha"`          // stock-specific
}

type AttributionService struct {
    bridge *python.PythonBridge  // for factor engine
    oms    *trading.OMS
}

func (s *AttributionService) ComputeAttribution(ctx context.Context) (*FactorAttribution, error) {
    // 1. Get portfolio positions + weights from OMS
    // 2. Call Python factor_engine for Barra factor exposures
    // 3. Compute factor returns × exposures = contributions
    // 4. α = total return - sum(factor contributions)
}
```

**Commit:** `feat(portfolio): add FactorAttribution service with Python bridge`

---

### Task 2: FactorAttributionPanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/FactorAttributionPanel.vue`
- Modify: registry.ts → register `factor-attribution`

**UI:** Waterfall chart: total return → −market β → −size → −value → ... → = α. Below: factor exposure heatmap (positions × factors). Period selector (1D/1W/1M/3M/YTD).

**Commit:** `feat(frontend): add FactorAttribution panel with waterfall chart`

---

### Task 3: IPC + wire

**Commit:** `feat(backend+frontend): wire FactorAttribution IPC`
