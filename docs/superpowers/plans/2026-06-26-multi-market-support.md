# Multi-Market Full Support — Implementation Plan (15 tasks)

> **Spec:** [docs/specs/2026-06-26-multi-market-support.md](../../specs/2026-06-26-multi-market-support.md)

**Phases:** Phase A (P0 后端正确性, 7 tasks) → Phase B (P1 前端补齐, 5 tasks) → Phase C (P2 增强, 3 tasks)

---

## Phase A: P0 — 后端正确性 (7 tasks)

### A-1: Fix Go API `GetMarketSnapshot` hardcoded market

**Files:**
- `app.go` — `GetMarketSnapshot`

**Changes:**
```go
func (a *App) GetMarketSnapshot(ctx context.Context, symbols []string) ([]map[string]interface{}, error) {
    reg := a.getMarketReg()
    if reg == nil {
        return nil, fmt.Errorf("market registry not initialized")
    }
    result := make([]map[string]interface{}, 0, len(symbols))
    for _, sym := range symbols {
        mkt := market.MarketForSymbol(sym)
        snap, _, err := reg.FetchQuoteWithFallback(ctx, mkt, sym)
        if err != nil {
            continue
        }
        result = append(result, map[string]interface{}{
            "symbol":     sym,
            "price":      snap.Last,
            "change":     snap.Change,
            "change_pct": snap.ChangePct,
            "volume":     snap.Volume,
        })
    }
    return result, nil
}
```

**Test:** Write a test that calls `GetMarketSnapshot` with mixed symbols.
**Commit:** `[Fix] market: GetMarketSnapshot use MarketForSymbol instead of hardcoded CN`

---

### A-2: Fix Go API `GetCorrelationMatrix` hardcoded market

**Files:**
- `app.go` — `GetCorrelationMatrix`

**Changes:** Each symbol uses its own market:
```go
func (a *App) GetCorrelationMatrix(ctx context.Context, symbols []string, lookback int) (map[string]map[string]float64, error) {
    reg := a.getMarketReg()
    if reg == nil {
        return nil, fmt.Errorf("market registry not initialized")
    }
    returns := make(map[string][]float64)
    end := time.Now().Unix()
    start := end - int64(lookback*86400)
    for _, sym := range symbols {
        mkt := market.MarketForSymbol(sym)
        bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, sym, "1d", start, end)
        if err != nil || len(bars) < 2 { continue }
        rets := make([]float64, 0, len(bars)-1)
        for i := 1; i < len(bars); i++ {
            if bars[i-1].Close > 0 {
                rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
            }
        }
        returns[sym] = rets
    }
    return portfolio.CorrelationMatrix(returns), nil
}
```

**Commit:** `[Fix] market: GetCorrelationMatrix use MarketForSymbol per symbol`

---

### A-3: Fix Go API `GetReturnDistribution` and `GetVolatilitySurface`

**Files:**
- `app.go` — `GetReturnDistribution`, `GetVolatilitySurface`

**Changes:** Replace `"CN"` with `market.MarketForSymbol(symbol)` in both functions.

**Commit:** `[Fix] market: GetReturnDistribution/GetVolatilitySurface use MarketForSymbol`

---

### A-4: Add `gateio` to CRYPTO fallback chain

**Files:**
- `internal/market/registry.go` — `FallbackChains["CRYPTO"]`

**Changes:**
```go
var FallbackChains = map[string][]string{
    "CRYPTO": {"binance", "okx", "coingecko", "gateio"},
}
```

**Test:** Update `registry_test.go` if it references CRYPTO chain.
**Commit:** `[Fix] market: add gateio to CRYPTO fallback chain (accessible from CN)`

---

### A-5: Fix Polygon adapter or remove from US chain

**Files:**
- `internal/market/adapters/polygon.go` — will remove stub
- `internal/market/registry.go` — remove "polygon" from US chain

**Changes:**
```go
// registry.go
"US": {"yahoo", "sina", "finnhub"},  // polygon removed (was stub)
```

Mark polygon.go as deprecated with a comment.

**Commit:** `[Fix] market: remove polygon stub from US fallback chain`

---

### A-6: Fix `OrderEntryPanel` hardcoded market

**Files:**
- `frontend/src/terminal/panels/OrderEntryPanel.vue`

**Changes:**
```typescript
// Find the GetQuote call that hardcodes 'CN'
const market = detectMarket(symbol.value)
const snap = await GetQuote(market, symbol.value)
```

**Test:** `vue-tsc --noEmit` passes.
**Commit:** `[Fix] frontend: OrderEntryPanel use detectMarket instead of hardcoded CN`

---

### A-7: Fix `GetMinuteLine` multi-market routing

**Files:**
- `app.go` — `GetMinuteLine`

**Changes:**
```go
func (a *App) GetMinuteLine(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
    mkt := market.MarketForSymbol(symbol)
    switch mkt {
    case "CN":
        return a.getMinuteLineCN(ctx, symbol)
    case "US":
        return a.getMinuteLineUS(ctx, symbol)
    case "HK":
        return a.getMinuteLineHK(ctx, symbol)
    default:
        return nil, fmt.Errorf("minute data not available for market %s", mkt)
    }
}
```

Helper methods delegate to the appropriate adapter (yahoo for US, tencent for HK).

**Commit:** `[Fix] market: GetMinuteLine multi-market routing`

---

## Phase B: P1 — 前端面板补齐 (5 tasks)

### B-1: Multi-market TickerTape

**Files:**
- `frontend/src/terminal/panels/TickerTapePanel.vue`

**Changes:** Replace hardcoded CN-only list with market-grouped defaults. Add a market toggle (CN/HK/US/CRYPTO) as a prop or store setting. The ticker scroll shows symbols from the selected market group.

**Commit:** `[Feat] frontend: TickerTapePanel multi-market support`

---

### B-2: Multi-market MarketOverview

**Files:**
- `app.go` — modify `GetMarketOverview` to accept a market parameter
- `frontend/src/terminal/panels/MarketOverviewPanel.vue` — add market tabs
- `frontend/src/stores/data.ts` — market-aware caching

**Changes:** Add `market` param (default "CN") to `GetMarketOverview`. Frontend adds HK index (恒指/国企指数) and US index (SPX/IXIC/DJI) data sources.

**Commit:** `[Feat] frontend+backend: MarketOverviewPanel multi-market`

---

### B-3: Multi-market Heatmap

**Files:**
- `frontend/src/terminal/panels/HeatmapPanel.vue`

**Changes:** Add market toggle similar to MarketOverview. For non-CN markets, show US sectors or HK industry data.

**Commit:** `[Feat] frontend: HeatmapPanel multi-market`

---

### B-4: PositionDetail real data

**Files:**
- `frontend/src/terminal/panels/PositionDetail.vue`

**Changes:** Replace all hardcoded mock data with real API calls to `GetPortfolioSummary()`, `GetPositions()`. The panel should load position data from the portfolio store, not inline mock values.

**Commit:** `[Fix] frontend: PositionDetail use real portfolio data instead of mock`

---

### B-5: Dynamic trading hours per market

**Files:**
- `frontend/src/terminal/panels/CandlestickPanel.vue`

**Changes:**
```typescript
function isTradingHours(market: string): boolean {
    const now = new Date()
    const utcMin = now.getUTCHours() * 60 + now.getUTCMinutes()
    switch (market) {
        case 'CN':
            const cst = utcMin + 480; return (cst >= 570 && cst < 690) || (cst >= 780 && cst < 900)
        case 'HK':
            const hkt = utcMin + 480; return (hkt >= 570 && hkt < 960)
        case 'US':
            const et = utcMin - 240; return (et >= 570 && et < 960) // EDT
        default: return false
    }
}
```

Also pass `market` to `isTradingHours` on each poll cycle.

**Commit:** `[Fix] frontend: CandlestickPanel dynamic trading hours per market`

---

## Phase C: P2 — 体验增强 (3 tasks)

### C-1: Market selector UI

**Files:**
- `frontend/src/stores/session.ts` — add `activeMarket` field
- `frontend/src/terminal/CommandBar.vue` or settings — market dropdown

Add a market selector in the command bar or settings panel. On market change, relevant panels refresh.

**Commit:** `[Feat] frontend: market selector UI`

---

### C-2: Symbol search market filter

**Files:**
- `frontend/src/terminal/SymbolSearch.vue`
- `frontend/src/lib/symbolSearch.ts`

Add filter tabs/badges to limit search results by market. The backend `SearchSymbols` already supports market filtering.

**Commit:** `[Feat] frontend: SymbolSearch market filter`

---

### C-3: Auto color scheme per market

**Files:**
- `frontend/src/terminal/panels/QuoteDetailPanel.vue`
- `frontend/src/terminal/panels/CandlestickPanel.vue`
- (other price-sensitive panels)

Auto-switch CN color convention (红涨绿跌) and US convention (绿涨红跌) based on the symbol's market, regardless of global setting.

**Commit:** `[Feat] frontend: per-market color scheme auto-switch`

---

## Verify

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go test ./internal/market/... -count=1
go test ./internal/workflow/... -count=1
go vet ./internal/...
go build ./...
cd frontend
npx vue-tsc --noEmit
npx vitest run src/stores/ src/terminal/panels/ --reporter=verbose
```

### Update CHANGELOG.md

Add entries for all changes.

**Commit:** `[Chore] CHANGELOG: multi-market full support`
