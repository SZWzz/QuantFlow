# MacroDashboard Implementation Plan

**Goal:** China/US macro dashboard with GDP/CPI/PMI/M2 quadrants, policy event timeline, and trend charts.

**Architecture:** New `MacroService` routes macro requests to Python sidecar (`macro_cn.py` for CN, `GovDataAdapter`/FRED for US). Results cached in SQLite `macro_cache` table. `MacroDashboard.vue` renders 4-up cards + ECharts trend charts.

**Data sources (all existing):**
- `macro_cn.py` → 85 CN macro indicators via AKShare (GDP/CPI/PMI/M2/社融/LPR/外汇…)
- `GovDataAdapter` → FRED US macro (利率/就业/通胀)
- Python sidecar `FetchData(source="macro", dataType="macro_cn_summary")` → all indicators

## Global Constraints

- CN macro via Python sidecar → gRPC → subprocess → AKShare (may take 10-60s for slow endpoints)
- Cache to SQLite with 1-day TTL (macro data changes daily at most)
- US macro via FRED HTTP (already available via GovDataAdapter)

---

### Task 1: MacroService + cache + Go test

**Files:**
- Create: `internal/market/macro_service.go`
- Test: `internal/market/macro_service_test.go`
- Create: `internal/storage/migrations/022_macro_cache.sql`

```go
// internal/market/macro_service.go
type MacroIndicator struct {
    Country string  `json:"country"`
    Name    string  `json:"name"`
    Value   float64 `json:"value"`
    Unit    string  `json:"unit"`
    Date    string  `json:"date"`
    Change  float64 `json:"change"` // MoM or YoY
}

type MacroService struct {
    bridge *python.PythonBridge  // for CN macro via Python sidecar
    gov    *research.GovDataService // for US macro via FRED
    db     *sql.DB
    cache  map[string]*MacroCacheEntry
}

func (s *MacroService) GetMacroSnapshot(ctx context.Context, country string) (map[string][]MacroIndicator, error) {
    // Returns grouped indicators: growth, inflation, monetary, policy
}

func (s *MacroService) GetIndicatorHistory(ctx context.Context, country, indicator string, years int) ([]MacroIndicator, error) {
    // Historical time series for chart
}
```

**Test:** Verify CN/US snapshot returns 4 categories, cache works.

**Commit:** `feat(market): add MacroService with CN/US macro indicator routing`

---

### Task 2: MacroDashboard.vue + ECharts

**Files:**
- Create: `frontend/src/terminal/panels/MacroDashboard.vue`
- Test: `frontend/src/terminal/panels/__tests__/MacroDashboard.test.ts`
- Modify: registry.ts → register `macro-dashboard`

**UI:** Country tabs (中国 | 美国 | 对比). 4 card groups: 增长(GDP/PMI/IP), 通胀(CPI/PPI), 货币(M2/社融/SHIBOR), 政策(LPR/准备金/事件标注). Each card shows latest value + sparkline. Below: 2 ECharts trend charts (GDP 10Y, CPI vs PPI 5Y). Click indicator → full history chart.

**Commit:** `feat(frontend): add MacroDashboard with 4-quadrant cards and trend charts`

---

### Task 3: IPC + wire Python bridge

**Files:**
- Create: `app_macro.go` — `GetMacroSnapshot(country)` / `GetMacroHistory(country, indicator, years)`
- Modify: `frontend/src/lib/wails.ts` — bindings
- Modify: `app_startup.go` — init `a.macroSvc = market.NewMacroService(a.bridge, a.govDataSvc, a.db)`
- Modify: `app.go` — add `macroSvc *market.MacroService`

**Commit:** `feat(backend+frontend): wire MacroDashboard IPC with Python bridge`
