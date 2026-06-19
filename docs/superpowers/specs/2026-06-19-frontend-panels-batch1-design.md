# Frontend Panels Batch 1 — 17 Panel Design

## Motivation

QuantFlow currently has 29 registered frontend panels. The proposal targets 50+ panels. This batch adds 17 high-priority panels across three categories: market data (5), charts & portfolio (7), and trading execution (5). All Go backend APIs are already implemented and wired — zero backend changes needed.

## Architecture

All new panels follow the existing pattern:

```
Panel.vue → Pinia Store → Wails IPC → Go App method → Service → Adapter
                                                    ↘ mock fallback
```

### Store Strategy

| Store | Existing | Extended With |
|-------|----------|---------------|
| `data` | quotes, OHLCV cache, sourceStatus | `marketOverview` state + `fetchMarketOverview()` |
| `portfolio` | summary, allocation, positions (10s auto-refresh) | `orders`, `trades`, `equityCurve` + fetch methods |
| `research` | sentiment, stockResearch, congressTrades | no changes needed |

### New Shared Library

`frontend/src/lib/stats.ts` — pure frontend statistical functions:
- `pearsonMatrix(returns: number[][]): number[][]` — Pearson correlation
- `histogramBins(data: number[], binCount: number): {x: number, y: number}[]`
- `simulateGBM(params): number[][]` — Geometric Brownian Motion Monte Carlo
- `computeDrawdowns(equity: number[]): {peak: number, trough: number, drawdown: number}[]`
- `sharpeRatio(returns: number[]): number`

### Panel Registration

Each panel registers via `defineAsyncComponent` in `registry.ts`:
```ts
register('panel-id', () => import('./PanelName.vue'))
```

---

## Batch 1: Market Data Panels (5)

### 1. MarketOverviewPanel (`market-overview`)

**Layout**: Three sections stacked vertically.

**Section A — Index Cards Row**: Horizontal scrolling row of index cards. Each card shows:
- Index name (上证/深证/创业板/科创50/恒生/S&P500/Nasdaq)
- Last price (large)
- Change % (colored: red↑ / green↓ for CN convention)
- Mini sparkline (last 20 bars thumbnail)

**Section B — Market Breadth Bar**: Single horizontal bar showing:
- Advancing count (green) / Declining count (red) / Unchanged (gray)
- Advance/Decline ratio numeric display

**Section C — Sector Rankings**: Two-column list:
- Left: Top 10 sectors by gain
- Right: Bottom 5 sectors by loss

**Data flow**:
```
Panel onMounted →
  dataStore.fetchMarketOverview()
    → App.GetQuoteBatch(indices[])      // 7 index quotes
    → App.GetIndustryRanks(30)           // sector rankings
    → assemble into MarketOverview { indices, breadth, sectors }
    → mock fallback with realistic static data
Auto-refresh: 30s polling via setInterval
```

**Store additions** (`data.ts`):
```ts
interface MarketOverview {
  indices: { symbol: string; name: string; last: number; changePct: number; sparkline: number[] }[]
  breadth: { advancers: number; decliners: number; unchanged: number }
  sectors: { name: string; changePct: number }[]
}
// new state: marketOverview ref<MarketOverview | null>
// new method: fetchMarketOverview()
```

### 2. MarketDepthPanel (`market-depth`)

**Layout**: Symbol input at top, two sections below.

**Section A — Order Book** (left): 5-level bid/ask ladder
- Columns: Level | Bid Price | Bid Size | Ask Price | Ask Size
- Price colors: bid=green, ask=red
- Visual: bar chart background proportional to size

**Section B — Tick Timeline** (right): Last 20 tick-by-tick trades
- Each row: Time | Price | Volume | Direction (B/S colored)
- Auto-scrolling, newest at top

**Data flow**:
```
Panel →
  App.GetQuote(market, symbol) → extract bids[], asks[], ticks[]
  → mock with realistic order book generation
Auto-refresh: 2s polling
```

**No new store needed** — uses existing `dataStore.updateQuote`.

### 3. HeatmapPanel (`heatmap`)

**Layout**: Full-panel ECharts treemap.

**Visual encoding**:
- Each rectangle = one sector/industry
- Area = total market cap of sector
- Color = average % change (red=up, green=down, intensity = magnitude)
- Label: sector name + avg change%

**Data flow**:
```
Panel →
  App.GetIndustryRanks(60) → returns sector name + changePct + marketCap
  → ECharts treemap series with visualMap
  → drill-down: click sector → show constituent stocks (Phase 2)
```

**No new store needed** — uses inline fetch.

### 4. TickerTapePanel (`ticker-tape`)

**Layout**: Single horizontal scrolling bar at panel height ~40px.

**Content**: Continuous loop of stock quotes:
- `[Code] Name Price Change%`
- Each item separator: `|`
- Real-time color changes on price update

**Data flow**:
```
Panel → subscribe to dataStore.quotes (reactive Map)
      → extract watched symbols → render scrolling list
      → CSS animation: translateX infinite scroll
Speed controls: pause on hover, speed slider
Watched symbols: configurable list, default top 20 by market cap
```

**No new store needed** — uses existing `dataStore.quotes`.

### 5. CryptoOverviewPanel (`crypto-overview`)

**Layout**: Two sections.

**Section A — Dominance**: BTC dominance % + ETH dominance % (progress bars)

**Section B — Top 20 Table**:
- Columns: Rank | Symbol | Price | 24h Change% | 24h Volume | Market Cap
- Sortable by any column

**Data flow**:
```
Panel →
  App.GetQuote("CRYPTO", "BTCUSDT,ETHUSDT,...") // batch 20 symbols
  → fallback: fetch CoinGecko public API directly from frontend
Auto-refresh: 15s
```

**No new store needed** — uses inline fetch.

---

## Batch 2: Charts & Portfolio Panels (7)

### 6. EquityCurvePanel (`equity-curve`)

**Layout**: ECharts dual-axis chart.

**Main chart** (top 70%): Equity curve line(s)
- Primary: portfolio NAV line (blue)
- Benchmark: overlay index line (gray dashed, e.g., 000300.SH)
- Optional: drawdown shading below x-axis

**Drawdown chart** (bottom 30%): Area chart
- Red/green gradient fill below zero
- Peak markers

**Stats row** (below charts): 5 cards
- Cumulative Return | Annualized Return | Max Drawdown | Sharpe Ratio | Calmar Ratio

**Data flow**:
```
Panel →
  portfolioStore.fetchEquityCurve()
    → App.GetPortfolioSummary() // current valuation
    → OHLCV data for benchmark
    → assemble equity curve from daily P&L snapshots
    → frontend compute drawdowns + ratios via stats.ts
```

**Store additions** (`portfolio.ts`):
```ts
interface EquityCurvePoint { date: string; nav: number; benchmark: number }
// new state: equityCurve ref<EquityCurvePoint[] | null>
// new method: fetchEquityCurve()
```

### 7. SurfaceChartPanel (`surface-chart`)

**Layout**: ECharts GL 3D surface chart, full panel.

**Axes**: X=maturity, Y=strike, Z=implied volatility (color heat).

**Controls**: Symbol selector, date slider.

**Data flow**: Mock volatility surface generation (parametric SVI/SABR model stub). Designed for future option chain adapter integration.

**No new store needed** — standalone component with inline mock.

### 8. CorrelationPanel (`correlation`)

**Layout**: ECharts heatmap (lower triangular matrix).

**Input**: Symbol list (editable textarea or from watchlist), lookback period selector.

**Output**: N×N heatmap with numeric values in cells.
- Color: blue=positive, red=negative, white=0
- Tooltip: symbol pair + exact value
- Click cell → scatter plot popup of the two symbols' returns

**Data flow**:
```
Panel →
  App.GetOHLCV(market, symbol, start, end) // one per symbol, parallel
  → compute daily log returns for each
  → stats.ts pearsonMatrix(returns[])
  → ECharts heatmap
```

**No new store needed** — standalone with inline fetch.

### 9. DistributionPanel (`distribution`)

**Layout**: ECharts combined chart.

**Main**: Histogram bars (daily returns) + normal fit curve overlay.

**Annotations**: Vertical dashed lines at mean, ±1σ, ±2σ, skewness indicator.

**Stats cards**: Mean | Std Dev | Skewness | Kurtosis | Jarque-Bera

**Data flow**: Same returns data as CorrelationPanel → `stats.ts histogramBins()` → ECharts bar + line series.

**No new store needed**.

### 10. DrawingPanel (`drawing`)

**Layout**: Tool palette (left) + chart canvas (right).

**Tools**: Cursor | Trendline | Horizontal line | Fibonacci retracement | Parallel channel | Text annotation | Eraser

**Interaction**: Click tool → click-drag on canvas → annotation renders. Each annotation stored as object:
```ts
interface Drawing {
  id: string; type: string; points: {x: number; y: number}[];
  style: { color: string; width: number; dash?: number[] };
  label?: string;
}
```

**Persistence**: `localStorage` per symbol + layout.

**No new store needed** — self-contained. Canvas rendered via HTML5 Canvas overlay on existing CandlestickPanel pattern.

### 11. MonteCarloPanel (`monte-carlo`)

**Layout**: Two charts + inputs sidebar.

**Inputs** (sidebar): Initial capital | Annual return % | Annual volatility % | Years | Simulations (100-5000) | Confidence level %

**Chart A** (top 60%): N simulation paths (semi-transparent lines) + 50th percentile line (bold) + confidence interval band.

**Chart B** (bottom 40%): Terminal value distribution histogram + VaR/CVaR markers.

**Stats**: Median terminal value | 95% VaR | 95% CVaR | Prob(loss) | Prob(double)

**Data flow**: Pure frontend — `stats.ts simulateGBM()` → ECharts. No backend dependency.

**No new store needed**.

### 12. RebalancePanel (`rebalance`)

**Layout**: Three sections.

**Section A — Current vs Target** (stacked bar chart): Two bars per market/sector — current allocation (filled) vs target (outlined).

**Section B — Trade List**: Table showing needed adjustments
- Symbol | Current Weight | Target Weight | Delta $ | Action (Buy/Sell) | Quantity

**Section C — Execute button**: Generates target allocation and optionally creates basket orders.

**Data flow**:
```
Panel →
  portfolioStore.allocation (already subscribed, 10s refresh)
  → user inputs target weights
  → frontend calculates deltas
  → confirm → App.PlaceOrder() per symbol (optional)
```

**No new store needed** — uses existing `portfolioStore`.

---

## Batch 3: Trading Panels (5)

### 13. OrderBlotterPanel (`order-blotter`) 🔴

**Layout**: Filter bar + data table + stats footer.

**Filter bar**: Status dropdown (All/Filled/Partial/Cancelled/Pending/Rejected), symbol search input, date range picker.

**Table columns**: Time | OrderID | Symbol | Side (Buy/Sell colored) | Type (Market/Limit/Stop) | Qty | Price | Filled% | Status (colored badge) | Cancel button

**Stats footer**: Today's orders count | Fill rate % | Total traded value

**Data flow**:
```
Panel onMounted →
  portfolioStore.fetchOrders()
    → App.GetOrders() // already implemented in Go
    → mock: generate 15-20 realistic orders with varied statuses
Auto-refresh: 5s polling
Cancel button → App.CancelOrder(orderId)
```

**Store additions** (`portfolio.ts`):
```ts
// new state: orders ref<Order[]>
// new method: fetchOrders(), cancelOrder(id)
```

### 14. ExecutionPanel (`execution`)

**Layout**: Data table with pagination.

**Table columns**: Time | Symbol | Side | Price | Qty | Value | OrderID | Fee

**Pagination**: 50 rows per page, load more button.

**Data flow**:
```
Panel →
  portfolioStore.fetchTrades()
    → App.GetTrades() // already implemented in Go
Auto-refresh: 5s
```

**Store additions** (`portfolio.ts`):
```ts
// new state: trades ref<Trade[]>
// new method: fetchTrades()
```

### 15. BasketOrderPanel (`basket-order`)

**Layout**: Three-column layout.

**Left — Basket Builder**: Input rows (symbol | weight% | qty | price), Add/Remove/Import CSV buttons.

**Center — Summary**: Total estimated cost, symbol count, execution mode selector (All Market / All Limit / Weighted).

**Right — Execution Log**: Real-time progress as each leg executes, success/failure per symbol.

**Data flow**: Frontend constructs basket → `App.PlaceOrder()` per symbol sequentially → show progress. Mock: simulate step-by-step fills with 200ms delay between each.

**No new store needed** — standalone component.

### 16. BrokerStatusPanel (`broker-status`)

**Layout**: Card grid (2 columns).

**Each card**: Broker name | Connection dot (green/yellow/red) | Latency ms | Last heartbeat | Account balance | Today's trades count | Test Connection button

**Cards**: Paper Trading | Futu | Binance | (future: Alpaca/IBKR/OKX dimmed as "not configured")

**Data flow**:
```
Panel →
  App.GetQuote() probe per market → measure latency
  App.GetPortfolioSummary() → account value
  broker status from dataStore.sourceStatus (if available)
Auto-refresh: 30s
```

**No new store needed** — uses existing `dataStore.sourceStatus` + inline calls.

### 17. ActionCenterPanel (`action-center`)

**Layout**: Feed-style list.

**Event types** (each with distinct icon + color):
- 🔴 Stop-loss triggered: symbol + price + loss%
- 🟡 Take-profit triggered: symbol + price + gain%
- 🔵 Dividend announcement: symbol + ex-date + amount
- ⚪ Split announcement: symbol + ratio + date
- 🟠 Large order pending approval: symbol + qty + value

**Actions**: Dismiss | Confirm (for approval events) | View details

**Pagination**: Infinite scroll, newest first.

**Data flow**: Mock event generation seeded from portfolio positions + random triggers. Designed for future integration with `App.GetNotifications()` + risk engine.

**No new store needed** — standalone with mock data.

---

## Files Changed

### New Files (18)
```
frontend/src/lib/stats.ts                          # Statistical utility functions
frontend/src/terminal/panels/MarketOverviewPanel.vue
frontend/src/terminal/panels/MarketDepthPanel.vue
frontend/src/terminal/panels/HeatmapPanel.vue
frontend/src/terminal/panels/TickerTapePanel.vue
frontend/src/terminal/panels/CryptoOverviewPanel.vue
frontend/src/terminal/panels/EquityCurvePanel.vue
frontend/src/terminal/panels/SurfaceChartPanel.vue
frontend/src/terminal/panels/CorrelationPanel.vue
frontend/src/terminal/panels/DistributionPanel.vue
frontend/src/terminal/panels/DrawingPanel.vue
frontend/src/terminal/panels/MonteCarloPanel.vue
frontend/src/terminal/panels/RebalancePanel.vue
frontend/src/terminal/panels/OrderBlotterPanel.vue
frontend/src/terminal/panels/ExecutionPanel.vue
frontend/src/terminal/panels/BasketOrderPanel.vue
frontend/src/terminal/panels/BrokerStatusPanel.vue
frontend/src/terminal/panels/ActionCenterPanel.vue
```

### Modified Files (4)
```
frontend/src/stores/data.ts                        # +marketOverview state + fetchMarketOverview
frontend/src/stores/portfolio.ts                   # +orders +trades +equityCurve + fetch methods
frontend/src/stores/data.test.ts                   # +marketOverview tests
frontend/src/terminal/panels/registry.ts            # +17 panel registrations
```

### No Changes
- Go backend: zero changes (all APIs already implemented)
- Python sidecar: zero changes

---

## Data Flow Summary

```
┌─────────────────────────────────────────────────────────┐
│ Frontend Panels (17 new)                                 │
├─────────────────────┬───────────────────────────────────┤
│ Market (5)           │ dataStore (marketOverview, quotes)│
│                      │ → App.GetQuote / GetIndustryRanks │
├─────────────────────┼───────────────────────────────────┤
│ Charts+Portfolio (7) │ portfolioStore + stats.ts        │
│                      │ → App.GetPortfolio* / GetOHLCV   │
│                      │ → pure frontend (MonteCarlo,      │
│                      │   Correlation, Distribution)      │
├─────────────────────┼───────────────────────────────────┤
│ Trading (5)          │ portfolioStore (orders, trades)   │
│                      │ → App.GetOrders / GetTrades       │
│                      │   / PlaceOrder / CancelOrder      │
├─────────────────────┴───────────────────────────────────┤
│ Go Backend (zero changes)                               │
│ Python Sidecar (zero changes)                           │
└─────────────────────────────────────────────────────────┘
```

## Acceptance Criteria

### Market Panels
- [ ] MarketOverviewPanel shows 7 index cards + breadth bar + sector rankings
- [ ] MarketDepthPanel shows 5-level bid/ask + tick timeline
- [ ] HeatmapPanel renders ECharts treemap colored by sector change%
- [ ] TickerTapePanel scrolls continuously with real quote colors
- [ ] CryptoOverviewPanel shows Top 20 crypto with dominance bars

### Chart & Portfolio Panels
- [ ] EquityCurvePanel shows NAV + benchmark + drawdown with stats cards
- [ ] SurfaceChartPanel renders 3D surface with mock volatility data
- [ ] CorrelationPanel shows N×N heatmap with Pearson values
- [ ] DistributionPanel shows histogram + normal fit + stats
- [ ] DrawingPanel supports trendline/Fibonacci/text annotation tools
- [ ] MonteCarloPanel shows simulation paths + terminal value distribution
- [ ] RebalancePanel shows current vs target allocation + trade list

### Trading Panels
- [ ] OrderBlotterPanel shows order table with filters + cancel button
- [ ] ExecutionPanel shows trade history with pagination
- [ ] BasketOrderPanel supports multi-symbol order entry + execution progress
- [ ] BrokerStatusPanel shows broker cards with connection status
- [ ] ActionCenterPanel shows event feed with dismiss/confirm actions

### Global
- [ ] All 17 panels registered in registry.ts, searchable via CommandBar
- [ ] All panels gracefully degrade to mock when Go backend unavailable
- [ ] Existing 76+ frontend tests still pass (`npx vitest run`)
- [ ] No Go test regressions (`go test ./...`)
- [ ] Theme-aware (CSS variables, dark/light support)
- [ ] Updated CHANGELOG.md with all 17 panels

## Risks / Trade-offs

- **Large batch size**: 17 panels + stats.ts + store extensions. Risk of inconsistent quality. Mitigation: follow strict panel template, reuse patterns from existing panels.
- **ECharts dependency**: HeatmapPanel, EquityCurvePanel, CorrelationPanel, DistributionPanel, MonteCarloPanel all use ECharts. Ensure `vue-echarts` is in dependencies (already used by CandlestickPanel).
- **No backend changes**: Some panels (SurfaceChart, ActionCenter) are mock-heavy. Acceptable — panels render and are ready for future backend integration.
- **localStorage for DrawingPanel**: Drawings persist per symbol but aren't synced to SQLite. Acceptable for v1 — can add SQLite persistence later.
- **OrderBlotter cancel button**: App.CancelOrder() exists but may not be fully implemented for all brokers. Wrap in try/catch with user feedback.
