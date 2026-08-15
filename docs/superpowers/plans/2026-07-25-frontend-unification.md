# Frontend Unification — (window as any) Migration + i18n Completeness

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** (1) Eliminate all `(window as any).go?.main?.App` type escapes by extending `useWailsApp` composable to cover all exposed Go App methods, and migrate all 49+ call sites. (2) Add 54 missing English i18n keys.

**Architecture:** Extend the existing `useWailsApp()` composable's `WailsApp` interface with all methods panels currently call via `(window as any)`. Then do a mechanical find-and-replace migration across all `.vue` and `.ts` files in `frontend/src/terminal/`. The i18n fix is a straightforward key-by-key translation addition to `en.ts`.

**Tech Stack:** TypeScript, Vue 3 composables

---

### Task 1: Extend useWailsApp interface to cover all App methods

**Files:**
- Modify: `frontend/src/lib/composables/useWailsApp.ts`

- [ ] **Step 1: Read existing useWailsApp.ts and document all missing methods**

Collect all method names used via `(window as any).go?.main?.App` from the grep output above:
```
GetAuditFindings, GetFinancialAnalysis, GetDelistingRisk (already exist)
Missing: GetFinancialStatements, GetTradingMode, GetIndustryRanks, GetFundFlow,
GetGeopoliticsRisks, GetHKConnectFlow, GetHKSettlementInfo, GetIPOData,
GetPredictionMarkets, GetSatelliteSnapshots, GetSECFilings,
GetSystemStats, GetValuationDCF, GetVolatilitySurface, ListBacktestHistory,
GetQuote (already), GetCryptoOverview, GetCongressTrades, etc.
```

- [ ] **Step 2: Extend WailsApp interface**

Add to `frontend/src/lib/composables/useWailsApp.ts`:

```typescript
export interface WailsApp {
  // --- existing ---
  FetchOHLCV(market: string, symbol: string, interval: string, fq: string, start: number, end: number): Promise<[OHLCVBar[], string]>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<[MinuteTick[], string]>
  GetQuote(market: string, symbol: string): Promise<[QuoteData, string]>
  GetAuditFindings(symbol: string): Promise<Record<string, any>>
  GetFinancialAnalysis(symbol: string): Promise<Record<string, any>>
  GetDelistingRisk(symbol: string): Promise<Record<string, any>>

  // --- market data ---
  GetMarketOverview(mkt: string): Promise<Record<string, any>>
  GetCryptoOverview(symbols: string[]): Promise<Record<string, any>>
  GetFundFlow(symbol: string, flowType?: string): Promise<Record<string, any>>
  GetNorthboundFlow(): Promise<Record<string, any>>
  GetFinancialStatements(symbol: string): Promise<Record<string, any>>
  GetIndustryRanks(market: string, lookback?: number): Promise<Record<string, any>[]>
  GetPredictionMarkets(category: string, limit: number): Promise<Record<string, any>>
  GetSatelliteSnapshots(): Promise<Record<string, any>>
  GetSECFilings(symbol: string): Promise<Record<string, any>[]>
  GetVolatilitySurface(symbol: string): Promise<Record<string, any>>
  GetValuationDCF(symbol: string): Promise<Record<string, any>>
  GetGeopoliticsRisks(): Promise<Record<string, any>>
  GetSystemStats(): Promise<Record<string, any>>
  GetIPOData(market: string): Promise<Record<string, any>[]>

  // --- trading ---
  GetTradingMode(): Promise<string>
  ListBacktestHistory(limit: number, offset: number): Promise<Record<string, any>>

  // --- HK Connect ---
  GetHKConnectFlow(): Promise<Record<string, any>>
  GetHKSettlementInfo(): Promise<Record<string, any>>

  // --- Congress trading ---
  GetCongressTrades(): Promise<Record<string, any>[]>
  // ... add any other methods that panels call
}
```

Also add a `setup()` return type hint so components can destructure:
```typescript
export function useWailsApp(): WailsApp | null { ... }
```

- [ ] **Step 3: Run type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/composables/useWailsApp.ts
git commit -m "refactor: extend WailsApp interface with all panel-called methods"
```

---

### Task 2: Migrate TerminalMode.vue, TickerBar.vue, SymbolSearch.vue, TearOffPanel.vue, LiveModeBanner.vue

**Files:**
- Modify: `frontend/src/terminal/TerminalMode.vue`
- Modify: `frontend/src/terminal/TickerBar.vue`
- Modify: `frontend/src/terminal/SymbolSearch.vue`
- Modify: `frontend/src/terminal/TearOffPanel.vue`
- Modify: `frontend/src/terminal/components/LiveModeBanner.vue`

**Pattern for each component:**

Before:
```typescript
const mode = await (window as any).go?.main?.App?.GetTradingMode()
```

After:
```typescript
import { useWailsApp } from '@/lib/composables/useWailsApp'

const app = useWailsApp()
// ...
const mode = await app?.GetTradingMode()
```

- [ ] **Step 1: Migrate TerminalMode.vue** — replace `(window as any).go?.main?.App` with `useWailsApp()`
- [ ] **Step 2: Migrate TickerBar.vue** — same pattern
- [ ] **Step 3: Migrate SymbolSearch.vue** — same pattern
- [ ] **Step 4: Migrate TearOffPanel.vue** — same pattern
- [ ] **Step 5: Migrate LiveModeBanner.vue** — same pattern
- [ ] **Step 6: Run tests and type check**

```bash
cd frontend && npx vitest run && npx vue-tsc --noEmit
```
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/
git commit -m "refactor: migrate TerminalMode, TickerBar, SymbolSearch, TearOffPanel, LiveModeBanner from (window as any) to useWailsApp"
```

---

### Task 3: Migrate first batch of panels (15 files)

**Files (panel directory):**
Modify all 15 panels where `(window as any).go?.main?.App` appears. Same pattern as Task 2.

Batch A, alphabetical A–F:
- `AIChatPanel.vue`, `AuditPanel.vue`, `BacktestPanel.vue`, `BasketOrderPanel.vue`, `BondsPanel.vue`, `BrokerConfig.vue`
- `CBArbitragePanel.vue`, `CandlestickPanel.vue`, `ChanlunPanel.vue`, `CorrelationPanel.vue`, `CryptoOverviewPanel.vue`
- `DarkPoolPanel.vue`, `DefiTVLPanel.vue`, `DepthComparisonPanel.vue`, `DistributionPanel.vue`

- [ ] **Step 1: Process each panel** — `useWailsApp()` import → replace `(window as any)` calls → verify with `vue-tsc`
- [ ] **Step 2: Run tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "refactor: migrate Batch A panels (A–F) to useWailsApp"
```

---

### Task 4: Migrate second batch of panels (18 files)

Batch B, alphabetical E–W:
- `EarningsCalendarPanel.vue`, `EconomicCalendarPanel.vue`, `ExDividendPanel.vue`, `FactorAnalysisPanel.vue`, `FinancialsPanel.vue`
- `FundFlowPanel.vue`, `FundingRatePanel.vue`, `FundsPanel.vue`, `FuturesPanel.vue`
- `GasFeePanel.vue`, `GeopoliticsPanel.vue`, `GovDataPanel.vue`
- `HKConnectPanel.vue`, `HKDerivativesPanel.vue`, `HKSettlementPanel.vue`

- [ ] **Step 1: Process each panel** → verify → commit

```bash
git commit -m "refactor: migrate Batch B panels (E–H) to useWailsApp"
```

---

### Task 5: Migrate third batch of panels (16 files)

Batch C, alphabetical I–W:
- `IndicatorPanel.vue`, `IPOCalendarPanel.vue`, `LiquidationPanel.vue`, `MarginPanel.vue`, `MarketOverviewPanel.vue`, `MarketScannerPanel.vue`
- `NewsPanel.vue`, `OptionsPanel.vue`, `OrderEntryPanel.vue`, `PredictionMarketPanel.vue`
- `SEC13FPanel.vue`, `SatellitePanel.vue`, `SectorRotationPanel.vue`, `ShortInterestPanel.vue`
- `StockScannerPanel.vue`, `SurfaceChartPanel.vue`

- [ ] **Step 1: Process each panel** → verify → commit

```bash
git commit -m "refactor: migrate Batch C panels (I–W) to useWailsApp"
```

---

### Task 6: Migrate final batch of panels (13 files)

Batch D, alphabetical S–W:
- `SystemMonitorPanel.vue`, `TickerTapePanel.vue`, `ValuationPanel.vue`, `WashSalePanel.vue`, `WatchlistPanel.vue`, `WhaleTrackingPanel.vue`

Also migrate the DockTab.vue:
- `frontend/src/terminal/DockView/DockTab.vue`

- [ ] **Step 1: Process each panel** → verify → commit

```bash
git commit -m "refactor: migrate final batch and DockTab to useWailsApp"
```

---

### Task 7: Verify full test suite

- [ ] **Step 1: Run all frontend tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 2: Type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/
git commit -m "chore: verify useWailsApp migration — all tests pass"
```

---

### Task 8: Add 54 missing English i18n keys

**Files:**
- Modify: `frontend/src/lib/i18n/en.ts`

**Missing keys (54 total), grouped by section:**

- [ ] **Step 1: Add missing `common` keys**

```typescript
    more: 'More',
    price: 'Price',
    retry: 'Retry',
    symbol: 'Symbol',
    value: 'Value',
    yes: 'Yes',
```

Add in the `common` section of `en.ts`.

- [ ] **Step 2: Add missing `kline` keys**

```typescript
  kline: {
    no_minute_data: 'No minute data available',
  },
```

- [ ] **Step 3: Add missing `misc` keys**

```typescript
    asset_market: 'Asset / Market',
    benchmark: 'Benchmark',
    gainers: 'Gainers',
    heatmap: 'Heatmap',
    pinned: 'Pinned',
    volatility_surface: 'Volatility Surface',
    welcome_subtitle: 'Your Quant Terminal',
```

- [ ] **Step 4: Add missing `ml` keys**

```typescript
    ic: 'Information Coefficient',
    pred_distribution: 'Prediction Distribution',
    sharpe: 'Sharpe Ratio',
```

- [ ] **Step 5: Add missing `monitor` keys**

```typescript
    cvar_label: 'CVaR',
    median_terminal: 'Median Terminal Value',
```

- [ ] **Step 6: Add missing `news` key**

```typescript
    no_news: 'No news available',
```

- [ ] **Step 7: Add missing `portfolio` keys**

```typescript
    avg_price: 'Avg Price',
    position_count: 'Positions',
    symbol: 'Symbol',
```

- [ ] **Step 8: Add missing `research` keys**

```typescript
    congress_loading: 'Loading congress trading data…',
    financial_ratios: 'Financial Ratios',
    income_stmt: 'Income Statement',
    insider_name: 'Name',
    insider_net: 'Net (Sell-Buy)',
    keywords: 'Keywords',
    no_overview: 'No overview available',
    peer: 'Peers',
    revenue_growth: 'Revenue Growth',
    senator_name: 'Senator',
```

- [ ] **Step 9: Add missing `schedule` keys**

```typescript
    every_hour: 'Every Hour',
    name: 'Name',
    timeout: 'Timeout',
```

- [ ] **Step 10: Add missing `settings` keys**

```typescript
    appearance: 'Appearance',
    density: 'Density',
    language: 'Language',
```

- [ ] **Step 11: Add missing `trade` keys**

```typescript
    all_status: 'All',
    broker: 'Broker',
    cancelled: 'Cancelled',
    filled_pct: 'Filled %',
    order_type: 'Order Type',
    stop_price: 'Stop Price',
    today_orders: 'Today\'s Orders',
```

- [ ] **Step 12: Add missing `workflow` keys**

```typescript
    drawing_tools: 'Drawing Tools',
    execution: 'Execution',
    fee: 'Fees',
    properties: 'Properties',
    rename: 'Rename',
    run: 'Run',
```

- [ ] **Step 13: Verify i18n key parity**

```bash
# Run the key comparison again — should show 0 missing
python3 -c "
import re
for lang in ['zh','en']:
    with open(f'src/lib/i18n/{lang}.ts') as f:
        content = f.read()
    lines = content.split('\n')
    keys = set()
    section = ''
    for line in lines:
        m = re.match(r'^\s{2}([a-z_]+):\s*\{', line)
        if m: section = m.group(1); continue
        m = re.match(r'^\s{4}([a-z_]+):', line)
        if m and section: keys.add(f'{section}.{m.group(1)}')
        m = re.match(r'^\s{6}([a-z_]+):', line)
        if m and section: keys.add(f'{section}.{m.group(1)}')
    print(f'{lang}: {len(keys)} keys')
"
```

Expected: zh and en both show the same key count

- [ ] **Step 14: Commit**

```bash
git add frontend/src/lib/i18n/en.ts
git commit -m "fix(i18n): add 54 missing English translation keys for parity with zh"
```

---

### Task 9: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entries**

```markdown
### Changed
- [Frontend] Unified all (window as any).go?.main?.App access through useWailsApp composable across 49+ call sites

### Fixed
- [Frontend] Added 54 missing English i18n keys (common, kline, misc, ml, monitor, news, portfolio, research, schedule, settings, trade, workflow)
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for frontend unification"
```
