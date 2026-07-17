# ValuationBand Implementation Plan

**Goal:** PE/PB Band chart — stock price overlaid with 5 valuation channels (μ-2σ, μ-1σ, μ, μ+1σ, μ+2σ) so users see where current price sits historically.

**Architecture:** `ValuationCalculator` computes historical PE from OHLCV close × shares / historical EPS, then derives μ and σ over a rolling window. `ValuationPanel.vue` renders ECharts line-overlay chart.

**Data sources (all existing):**
- `FetchOHLCV` → historical close prices
- `QuoteSnapshot.Pe` → current PE
- `FinancialsService.ComputeRatios()` → PE/PB/ROE
- mootdx `FetchFinance` → historical EPS snapshots

## Global Constraints

- Reuse existing `ValuationPanel.vue` (rename/refactor), or enhance with Band tab
- PE Band = 5 years of data, PB Band same
- Cache computed bands in SQLite `valuation_bands` table (migration 021)

---

### Task 1: ValuationCalculator + Go test

**Files:**
- Create: `internal/trading/valuation.go`
- Test: `internal/trading/valuation_test.go`

```go
// internal/trading/valuation.go
package trading

type BandPoint struct {
    Date   string  `json:"date"`
    Close  float64 `json:"close"`
    Band1  float64 `json:"band_1"`  // μ - 2σ
    Band2  float64 `json:"band_2"`  // μ - 1σ
    Band3  float64 `json:"band_3"`  // μ
    Band4  float64 `json:"band_4"`  // μ + 1σ
    Band5  float64 `json:"band_5"`  // μ + 2σ
}

type BandResult struct {
    Symbol     string      `json:"symbol"`
    Metric     string      `json:"metric"` // "pe" | "pb"
    Current    float64     `json:"current"`
    Mean       float64     `json:"mean"`
    StdDev     float64     `json:"stddev"`
    Percentile float64     `json:"percentile"`
    Points     []BandPoint `json:"points"`
}

// ComputePEBand calculates PE Band for a symbol over a given year range.
// Uses OHLCV close + historical EPS to derive historical PE.
func ComputePEBand(bars []OHLCVBar, epsHistory []EPSPoint, years int) (*BandResult, error) { ... }
func ComputePBBand(bars []OHLCVBar, bookValues []BookValuePoint, years int) (*BandResult, error) { ... }
```

**Test:** Verify band computation with mock OHLCV + EPS data.

**Commit:** `feat(trading): add PE/PB Band computation`

---

### Task 2: Enhanced ValuationPanel.vue + ECharts

**Files:**
- Modify: `frontend/src/terminal/panels/ValuationPanel.vue` — add Band tab
- Test: update existing test

**UI:** Tabs `[估值指标] [PE Band] [PB Band]`, symbol input, Band chart with 5 horizontal line channels + price line overlay. Below: current PE, mean, σ, percentile.

**Commit:** `feat(frontend): add PE/PB Band chart to ValuationPanel`

---

### Task 3: IPC + cache + wire

**Files:**
- Create: `app_valuation.go` — `GetPEBand(symbol, years)` / `GetPBBand(symbol, years)`
- Modify: `frontend/src/lib/wails.ts` — bindings
- Modify: `app_startup.go` — init ValuationCalculator
- Create: `internal/storage/migrations/021_valuation_cache.sql` — cache table

**Commit:** `feat(backend+frontend): wire PE/PB Band IPC with SQLite cache`
