# Implementation Plan: Fix Frontend Mock Data — All Panels

> Spec: [docs/specs/2026-06-24-fix-frontend-mock-data.md](../../specs/2026-06-24-fix-frontend-mock-data.md)
> Date: 2026-06-24
> Scope: 13 panels + 4 stores + 8 Go APIs

## Overview

5 tasks, 4 are independent (can parallel dispatch). Task B3 (new Go APIs) must complete before B5 (wire computed panels). Execute B1+B2+B3 in parallel, then B4, then B5.

---

## Task B1: Wire core quote panels (Watchlist + QuoteDetail + Candlestick + TickerTape)

**Go APIs already exist**: `GetQuote("CN", symbol)`, `FetchOHLCV("CN", symbol, "1d", ...)`

### B1-1: WatchlistPanel — replace mockQuotes with GetQuote

**File:** `frontend/src/terminal/panels/WatchlistPanel.vue`

Remove `mockQuotes` object (lines 35-44). In `addSymbol()` or a watcher, call:

```ts
import { GetQuote } from '@/lib/wails'

const quotes = ref<Record<string, any>>({})

async function refreshQuote(symbol: string) {
  try {
    const [snapshot, source] = await GetQuote('CN', symbol)
    quotes.value[symbol] = snapshot
  } catch {
    quotes.value[symbol] = { price: 0, changePct: 0, error: true }
  }
}
```

Template: replace `mockQuotes[sym]?.price` → `quotes[sym]?.price`.

### B1-2: QuoteDetailPanel — replace mock snapshot

**File:** `frontend/src/terminal/panels/QuoteDetailPanel.vue`

Remove hardcoded `mockQuote` (lines 21-33). Watch symbol change → call `GetQuote('CN', symbol)`:

```ts
const quote = ref<any>(null)

watch(() => props.symbol, async (sym) => {
  if (!sym) return
  try {
    const [snapshot] = await GetQuote('CN', sym)
    quote.value = snapshot
  } catch { quote.value = null }
}, { immediate: true })
```

### B1-3: CandlestickPanel — replace random walk with FetchOHLCV

**File:** `frontend/src/terminal/panels/CandlestickPanel.vue`

Remove `generateMockOHLCV` (lines 36-54). Call:

```ts
import { FetchOHLCV } from '@/lib/wails'

async function loadOHLCV(symbol: string) {
  const end = Math.floor(Date.now() / 1000)
  const start = end - 90 * 86400
  try {
    const [bars, source] = await FetchOHLCV('CN', symbol, '1d', start, end)
    ohlcvData.value = bars.map(b => [b.date, b.open, b.close, b.low, b.high, b.volume])
  } catch { /* keep empty */ }
}
```

### B1-4: TickerTapePanel — replace hardcoded symbols with real quotes

**File:** `frontend/src/terminal/panels/TickerTapePanel.vue`

Remove `mockTickers` (lines 15-38). Use a rotating set of real symbols:

```ts
const SYMBOLS = ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276']
const tickers = ref<any[]>([])

onMounted(async () => {
  for (const sym of SYMBOLS) {
    try {
      const [snapshot] = await GetQuote('CN', sym)
      tickers.value.push(snapshot)
    } catch {}
  }
})
```

### B1-5: Test & commit

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
git add frontend/src/terminal/panels/WatchlistPanel.vue QuoteDetailPanel.vue CandlestickPanel.vue TickerTapePanel.vue
git commit -m "[Frontend] wire quote panels to real GetQuote/FetchOHLCV API"
```

---

## Task B2: Wire portfolio/trade panels + action center

**Go APIs already exist**: `GetPortfolioSummary()`, `GetTrades()`, `GetOrders()`, `GetPositions()`

### B2-1: PortfolioSummary — wire to existing store API

**File:** `frontend/src/terminal/panels/PortfolioSummary.vue`

Remove hardcoded mock data + `setInterval` timer (lines 25-68). Import existing `usePortfolioStore` which already calls `GetPortfolioSummary`:

```ts
import { usePortfolioStore } from '@/stores/portfolio'
const portfolio = usePortfolioStore()

watchEffect(() => {
  summary.value = portfolio.summary
})
portfolio.fetchSummary() // already defined in store
```

### B2-2: TradeHistory — wire to GetTrades/GetOrders

**File:** `frontend/src/terminal/panels/TradeHistory.vue`

Remove hardcoded trade/order arrays (lines 30-40):

```ts
import { GetTrades, GetOrders } from '@/lib/wails'

async function load() {
  try {
    trades.value = await GetTrades()
    orders.value = await GetOrders()
  } catch {}
}
onMounted(load)
```

### B2-3: ActionCenterPanel — add simple risk event collector

**File:** `frontend/src/terminal/panels/ActionCenterPanel.vue`

Replace hardcoded events (lines 17-31). Since there's no dedicated risk event API yet, collect from existing trade/order data:

```ts
import { GetTrades } from '@/lib/wails'

async function load() {
  const trades = await GetTrades()
  events.value = trades.slice(-12).map(t => ({
    title: `${t.side === 'buy' ? '买入' : '卖出'} ${t.symbol}`,
    status: 'info',
    time: t.timestamp,
    detail: `${t.quantity}股 @ ${t.price}`
  }))
}
```

### B2-4: Test & commit

```bash
git add frontend/src/terminal/panels/PortfolioSummary.vue TradeHistory.vue ActionCenterPanel.vue
git commit -m "[Frontend] wire portfolio/trade panels to real OMS API"
```

---

## Task B3: Add new Go backend APIs (market overview + analytics)

### B3-1: GetMarketOverview (index data + market breadth)

**File:** `app.go` — add new export:

```go
// GetMarketOverview returns index snapshots and market breadth.
func (a *App) GetMarketOverview(ctx context.Context) (map[string]interface{}, error) {
	indices := []string{"000001", "399001", "399006", "000688", "000300"}
	result := make([]map[string]interface{}, 0, len(indices))
	for _, code := range indices {
		snap, _, err := a.GetQuote(ctx, "CN", code)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"code":   code,
			"name":   snap.Symbol,
			"price":  snap.Price,
			"change": snap.Change,
			"changePct": snap.ChangePct,
		})
	}
	return map[string]interface{}{
		"indices": result,
		"breadth": map[string]int{"advancers": 0, "decliners": 0, "unchanged": 0},
	}, nil
}
```

### B3-2: GetCryptoOverview

**File:** `app.go`:

```go
func (a *App) GetCryptoOverview(ctx context.Context, symbols []string) (map[string]interface{}, error) {
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "DOTUSDT"}
	}
	reg := a.getMarketReg()
	results := make([]map[string]interface{}, 0)
	for _, sym := range symbols {
		snap, _, err := reg.FetchQuoteWithFallback(ctx, "CRYPTO", sym)
		if err != nil { continue }
		results = append(results, map[string]interface{}{
			"symbol": sym, "price": snap.Price, "changePct": snap.ChangePct,
		})
	}
	return map[string]interface{}{"cryptos": results}, nil
}
```

### B3-3: GetCorrelationMatrix + GetReturnDistribution + GetVolatilitySurface + GetRebalanceSuggestions

**File:** `internal/portfolio/analytics.go` (new):

```go
package portfolio

import "math"

// CorrelationMatrix computes Pearson correlation between asset returns.
func CorrelationMatrix(returns map[string][]float64) map[string]map[string]float64 { ... }

// ReturnDistribution computes daily return histogram bins.
func ReturnDistribution(returns []float64, bins int) ([]float64, []float64) { ... }

// VolatilitySurface computes a constant surface from historical vols at different windows.
func VolatilitySurface(returns []float64, windows []int) [][]float64 { ... }
```

**File:** `app.go` — wire these:

```go
func (a *App) GetCorrelationMatrix(ctx context.Context, symbols []string, lookback int) (map[string]map[string]float64, error)
func (a *App) GetReturnDistribution(ctx context.Context, symbol string, lookback int, bins int) ([]float64, []float64, error)
func (a *App) GetVolatilitySurface(ctx context.Context, symbol string) ([][]float64, error)
func (a *App) GetRebalanceSuggestions(ctx context.Context) ([]map[string]interface{}, error)
```

### B3-4: Test & commit

```bash
go test ./internal/portfolio/ -v && go vet ./internal/portfolio/
go build -o quantflow .  # verify builds
git add -A && git commit -m "[Engine] add market overview + analytics Go APIs"
```

---

## Task B4: Wire remaining panels (MarketOverview + Crypto + Correlation + Distribution + Surface + Rebalance + MarketDepth)

After Task B3 completes, wire the remaining 7 panels to the new APIs.

Each panel follows the same pattern: remove mock data → add API call → update template bindings. Similar to Task B1/B2 patterns.

### B4-1: data.ts — wire MarketOverview to GetMarketOverview

Replace `generateMockIndices()` call with `GetMarketOverview()`.

### B4-2: MarketDepthPanel — simulated depth from OHLCV

Use High/Low/Close from GetQuote to display simulated 5-level depth.

### B4-3: CryptoOverviewPanel — wire to GetCryptoOverview

### B4-4: CorrelationPanel — wire to GetCorrelationMatrix

### B4-5: DistributionPanel — wire to GetReturnDistribution

### B4-6: SurfaceChartPanel — wire to GetVolatilitySurface

### B4-7: RebalancePanel — wire to GetRebalanceSuggestions

---

## Task B5: Fix store fallback logic

### B5-1: data.ts — remove generateMockIndices/Sectors
### B5-2: portfolio.ts — remove generateMockOrders/Trades/EquityCurve
### B5-3: research.ts — only mock on API error, not silently
### B5-4: symbolSearch.ts — remove hardcoded symbol list

Each store change: try API first → if error, show "数据加载失败" not mock data.
