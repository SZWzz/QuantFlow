# DupontAnalysis Implementation Plan

**Goal:** ROE decomposition tree + 3-year trend + peer radar chart. Shows exactly *why* ROE is what it is.

**Architecture:** `DupontService` computes the 3-factor breakdown from existing `FinancialData` (Revenue/NetIncome/TotalAssets/TotalEquity). `DupontPanel.vue` renders tree diagram + trend table + ECharts radar.

**Data sources (all existing):**
- `FinancialsService.GetFinancials()` → Revenue, NetIncome, TotalAssets, TotalEquity, TotalDebt, FCF
- `FinancialsService.ComputeRatios()` → PE, PB, ROE, ROA, NetMargin, DebtToEquity
- `PeerComparisonService.GetPeers()` → peer symbols for radar

## Global Constraints

- Dupont decomposition: ROE = (NetIncome/Revenue) × (Revenue/TotalAssets) × (TotalAssets/TotalEquity)
- Radar axes: ROE, 净利率, 周转率, 权益乘数, 毛利率, EPS
- 3-year trend from Sina multi-period financials (if available) or mootdx snapshots

---

### Task 1: DupontService + Go test

**Files:**
- Create: `internal/research/dupont.go`
- Test: `internal/research/dupont_test.go`

```go
// internal/research/dupont.go
package research

type DupontBreakdown struct {
    Symbol      string  `json:"symbol"`
    ROE         float64 `json:"roe"`
    NetMargin   float64 `json:"net_margin"`    // 净利率 = NetIncome / Revenue
    AssetTurnover float64 `json:"asset_turnover"` // 周转率 = Revenue / TotalAssets
    EquityMultiplier float64 `json:"equity_multiplier"` // 杠杆 = TotalAssets / TotalEquity
}

type DupontTrend struct {
    Period string          `json:"period"`
    ROE    float64         `json:"roe"`
    Breakdown DupontBreakdown `json:"breakdown"`
}

type PeerRadar struct {
    Symbol string             `json:"symbol"`
    Metrics map[string]float64 `json:"metrics"` // ROE, NetMargin, Turnover, Leverage, GrossMargin, EPS
}

func ComputeDupont(fd *FinancialData) *DupontBreakdown { ... }
func ComputeDupontTrend(symbol string, financials []*FinancialData) []DupontTrend { ... }
func ComputePeerRadar(symbol string, peers []string, getFD func(string)*FinancialData) []PeerRadar { ... }
```

**Test:** Verify ROE = NetMargin × Turnover × Leverage mathematically.

**Commit:** `feat(research): add Dupont decomposition and peer radar computation`

---

### Task 2: DupontPanel.vue + ECharts tree + radar

**Files:**
- Create: `frontend/src/terminal/panels/DupontPanel.vue`
- Test: `frontend/src/terminal/panels/__tests__/DupontPanel.test.ts`
- Modify: registry.ts → register `dupont-analysis`

**UI:** Three sections stacked vertically: (1) ROE tree diagram (CSS flexbox, no chart library needed), (2) 3-year trend table, (3) ECharts radar chart with peer comparison.

**Commit:** `feat(frontend): add DupontAnalysis panel with ROE tree and peer radar`

---

### Task 3: IPC + wire

**Files:**
- Create: `app_dupont.go` — `GetDupontAnalysis(symbol)` / `GetPeerRadar(symbol)`
- Modify: `frontend/src/lib/wails.ts` — bindings
- Modify: `app_startup.go` — init DupontService

**Commit:** `feat(backend+frontend): wire DupontAnalysis IPC`
