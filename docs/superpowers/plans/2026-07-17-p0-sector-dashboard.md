# SectorDashboard Implementation Plan

**Goal:** Build an industry dashboard with heatmap + valuation table, bridging macro → sector → stock analysis chain.

**Architecture:** New `SectorService` aggregates industry ranks (existing `GetIndustryRanks`) with PE/PB percentile computed from OHLCV + EPS history. Frontend `SectorDashboard.vue` renders heatmap + valuation table with drill-down to `StockResearch`.

**Tech Stack:** Go 1.25+ (slog), Vue 3 Composition API + ECharts, existing EastMoney adapter chain

**Data sources (all existing, no new adapters):**
- `GetIndustryRanks(market, topN)` → industry list + change%
- `QuoteSnapshot.Pe` / `.MarketCap` → per-stock PE/市值 for aggregation
- `FetchOHLCV` + EPS → PE historical percentile
- `EastMoneyConceptAdapter` → concept/industry classification

## Global Constraints

- No new Python dependencies — all data from existing adapters
- SQLite for caching computed percentiles (avoid recalc on every open)
- Vue 3 `<script setup lang="ts">`, ECharts for heatmap
- Tests: vitest for Vue, table-driven for Go

---

### Task 1: SectorService + PE percentile computation + Go test

**Files:**
- Create: `internal/market/sector_service.go`
- Test: `internal/market/sector_service_test.go`

**Interfaces:**
- Consumes: `*market.AdapterRegistry` (existing), `*sql.DB` (for cache)
- Produces: `GetSectorHeatmap(market) → []SectorHeat`, `GetSectorValuation(market) → []SectorValuation`

```go
// internal/market/sector_service.go
package market

type SectorHeat struct {
    Name      string  `json:"name"`
    ChangePct float64 `json:"change_pct"`
    Volume    float64 `json:"volume"`     // 成交额(亿)
    PE        float64 `json:"pe"`         // 行业中位数 PE
    PEPct     float64 `json:"pe_pct"`     // PE 历史分位 0-100
}

type SectorValuation struct {
    Name   string  `json:"name"`
    PE     float64 `json:"pe"`
    PEPct  float64 `json:"pe_pct"`
    PB     float64 `json:"pb"`
    PBPct  float64 `json:"pb_pct"`
    ROE    float64 `json:"roe"`
}

type SectorService struct {
    reg *AdapterRegistry
    db  *sql.DB
}

func NewSectorService(reg *AdapterRegistry, db *sql.DB) *SectorService {
    return &SectorService{reg: reg, db: db}
}

func (s *SectorService) GetSectorHeatmap(ctx context.Context, market string) ([]SectorHeat, error) {
    // 1. Get industry ranks from existing adapter
    ranks, err := s.reg.FetchIndustryRanksWithFallback(ctx, market, 60)
    if err != nil { return nil, err }

    // 2. For each industry, get representative stocks and aggregate PE
    // 3. Load PE percentiles from cache or compute from OHLCV
    // ...
}

func (s *SectorService) GetSectorValuation(ctx context.Context, market string) ([]SectorValuation, error) {
    // Return PE/PB/ROE with historical percentiles
    // ...
}

// computePEPercentile calculates where current PE sits in 5-year history
func computePEPercentile(currentPE float64, historicalPEs []float64) float64 { ... }
```

**Test:** Verify heatmap returns 31 sectors for CN market, PE percentile in 0-100 range.

**Commit:** `feat(market): add SectorService with industry heatmap and PE percentile computation`

---

### Task 2: SectorDashboard.vue + ECharts heatmap

**Files:**
- Create: `frontend/src/terminal/panels/SectorDashboard.vue`
- Test: `frontend/src/terminal/panels/__tests__/SectorDashboard.test.ts`
- Modify: `frontend/src/terminal/panels/registry.ts` (register `sector-dashboard`)

```vue
<script setup lang="ts">
// ECharts treemap for industry heatmap + valuation table
// Click industry → terminal.openPanel('stock-research', { industry })
// Market tabs: A股 | 港股 | 美股
</script>

<template>
  <div class="sector-dashboard">
    <div class="market-tabs">
      <button v-for="m in ['CN','HK','US']" :class="{active: market===m}" @click="switchMarket(m)">
        {{ marketLabel(m) }}
      </button>
    </div>
    <div class="dashboard-grid">
      <div class="heatmap-panel">
        <div ref="heatmapChart" class="chart-container" />
      </div>
      <div class="valuation-table">
        <table>
          <thead><tr><th>行业</th><th>涨跌</th><th>PE</th><th>分位</th><th>PB</th><th>分位</th></tr></thead>
          <tbody>
            <tr v-for="s in sectors" @click="drillDown(s.name)">
              <td>{{ s.name }}</td>
              <td :class="s.changePct>=0?'up':'down'">{{ s.changePct }}%</td>
              <td>{{ s.pe }}</td><td>{{ s.pePct }}%</td>
              <td>{{ s.pb }}</td><td>{{ s.pbPct }}%</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
```

**Commit:** `feat(frontend): add SectorDashboard with industry heatmap and valuation table`

---

### Task 3: IPC binding + panel registration

**Files:**
- Create: `app_sector.go` — `GetSectorHeatmap(market)` / `GetSectorValuation(market)`
- Modify: `frontend/src/lib/wails.ts` — add `GetSectorHeatmap` / `GetSectorValuation`
- Modify: `frontend/src/terminal/panels/registry.ts` — register

```go
// app_sector.go
func (a *App) GetSectorHeatmap(ctx context.Context, market string) ([]market.SectorHeat, error) {
    if a.sectorSvc == nil { return nil, fmt.Errorf("sector service not initialized") }
    return a.sectorSvc.GetSectorHeatmap(ctx, market)
}
```

**Commit:** `feat(backend+frontend): wire SectorDashboard IPC and panel registration`

---

### Task 4: Wire SectorService in startup + integration test

**Files:**
- Modify: `app_startup.go` — init `a.sectorSvc = market.NewSectorService(a.marketReg, a.db)`
- Modify: `app.go` — add `sectorSvc *market.SectorService` field
- Test: `internal/market/sector_service_test.go` — integration test

**Commit:** `feat(market): wire SectorService at startup with integration test`
