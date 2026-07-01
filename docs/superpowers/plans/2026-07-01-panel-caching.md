# Panel 前端缓存 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `usePanelCache` + `fetchWithCache` to 22 frontend panels that currently call Go Wails methods without any caching.

**Architecture:** Each panel imports `fetchWithCache` from the existing `usePanelCache` composable and wraps its Go API call. TTL varies by data type (30min for macro, 1min for real-time crypto). No new infrastructure needed.

**Tech Stack:** Vue 3 + TypeScript, `usePanelCache` (already used by 26 other panels)

## Global Constraints

- Match exact existing import style (some use `import { usePanelCache }`, some need to add it)
- TTL in milliseconds (fetchWithCache's third param)
- Quote/real-time panels (Group C) stay untouched
- Every task must pass `npx vue-tsc --noEmit` (pre-existing errors excluded)
- Every task ends with a commit

---

### Task 1: ForecastPanel + HKSettlementPanel + DragonTigerPanel + SurfaceChartPanel

**Files:**
- Modify: `frontend/src/terminal/panels/ForecastPanel.vue`
- Modify: `frontend/src/terminal/panels/HKSettlementPanel.vue`
- Modify: `frontend/src/terminal/panels/DragonTigerPanel.vue`
- Modify: `frontend/src/terminal/panels/SurfaceChartPanel.vue`

- [ ] **Step 1: ForecastPanel.vue**

Add import after line 11:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 24 (`const chartTheme = useChartTheme()`), add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Replace lines 49-53 (within `onMounted` or fetch function) — find `loadSurface` or the onMounted call:
```typescript
// Before (in onMounted or watch callback):
const app = (window as any).go?.main?.App
if (!app?.GetForecast) return
const data = await app.GetForecast(symbol.value)
result.value = data

// After:
const { data } = await fetchWithCache(`forecast:${symbol.value}`, () => (window as any).go?.main?.App?.GetForecast(symbol.value), 30 * 60 * 1000)
result.value = data
```

- [ ] **Step 2: HKSettlementPanel.vue**

Add import at line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 8 (`const { t } = useI18n()`), add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchSettlementInfo()` (~line 46), replace:
```typescript
const result = await app.GetHKSettlementInfo()
settlementInfo.value = result as HKSettlementInfo
```
with:
```typescript
const { data: result } = await fetchWithCache('hk_settlement_info', () => app.GetHKSettlementInfo(), 30 * 60 * 1000)
settlementInfo.value = result as HKSettlementInfo
```

In `fetchCalendar()` (find it), replace the Go call similarly with key `hk_trade_calendar:${year}` and TTL 30min.

- [ ] **Step 3: DragonTigerPanel.vue**

Add import after line 5:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 35, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchDaily()` (~line 46), replace:
```typescript
const result = await app.GetDailyDragonTiger(date.value, minNetBuy.value)
```
with:
```typescript
// app already defined as (window as any).go?.main?.App
const { data: result } = await fetchWithCache(`dragon_tiger:${date.value}:${minNetBuy.value}`, () => app.GetDailyDragonTiger(date.value, minNetBuy.value), 5 * 60 * 1000)
```

In `fetchHistory()` (find it), wrap the `GetDragonTiger(symbol)` call with key `dragon_tiger_history:${symbol}` and TTL 5min.

- [ ] **Step 4: SurfaceChartPanel.vue**

Add import after line 9:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 18, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `loadSurface()` (~line 22), replace:
```typescript
const data = await app.GetVolatilitySurface(symbol.value)
surfaceData.value = data || []
```
with:
```typescript
const { data } = await fetchWithCache(`vol_surface:${symbol.value}`, () => app.GetVolatilitySurface(symbol.value), 15 * 60 * 1000)
surfaceData.value = data || []
```

- [ ] **Step 5: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "ForecastPanel|HKSettlementPanel|DragonTigerPanel|SurfaceChartPanel" || echo "No errors"`
Expected: No errors from these files

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/panels/ForecastPanel.vue frontend/src/terminal/panels/HKSettlementPanel.vue frontend/src/terminal/panels/DragonTigerPanel.vue frontend/src/terminal/panels/SurfaceChartPanel.vue
git commit -m "perf: add fetchWithCache to Forecast/HKSettlement/DragonTiger/SurfaceChart panels"
```

---

### Task 2: PredictionMarketPanel + SchedulePanel + CBArbitragePanel + HKDerivativesPanel

- [ ] **Step 1: PredictionMarketPanel.vue**

Add import after line 8:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 42, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `loadEvents()` (~line 49), replace:
```typescript
const result = await app.GetPredictionMarkets(cat, 30)
events.value = result.events || []
```
with:
```typescript
const { data: result } = await fetchWithCache(`prediction_markets:${activeCategory.value}`, () => app.GetPredictionMarkets(cat, 30), 15 * 60 * 1000)
events.value = result?.events || []
```

In `loadDetail()` (~line 64), replace the `GetPredictionEventDetail` call similarly with key `prediction_detail:${event.id}` and TTL 15min.

In `loadSignals()` (find it), wrap `GetPredictionSignals` with key `prediction_signals` and TTL 15min.

- [ ] **Step 2: SchedulePanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 12, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `loadTasks()` (~line 23), replace:
```typescript
const r = await (window as any).go.main.App.ListScheduleTasks()
tasks.value = Array.isArray(r) ? r : []
```
with:
```typescript
const { data: r } = await fetchWithCache('schedule_tasks', () => (window as any).go.main.App.ListScheduleTasks(), 5 * 60 * 1000)
tasks.value = Array.isArray(r) ? r : []
```

Note: After `saveTask()`, `toggleTask()`, `deleteTask()` — keep the existing `loadTasks()` call to refetch immediately (the cache will be stale-invalidated by the TTL, but the user action triggers a fresh load which will update the cache).

- [ ] **Step 3: CBArbitragePanel.vue**

Add import after line 5:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 12, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchData()` (~line 53), replace:
```typescript
const result = await app.GetCBArbitrageData()
data.value = result
```
with:
```typescript
const { data: result } = await fetchWithCache('cb_arbitrage', () => app.GetCBArbitrageData(), 15 * 60 * 1000)
data.value = result
```

Also update the `setInterval(fetchData, 120000)` to just call `fetchData` without interval (cache handles refresh).

- [ ] **Step 4: HKDerivativesPanel.vue**

Add import after line 5:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 10, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchData()` (~line 45), replace:
```typescript
const result = await app.GetHKDerivatives()
rawData.value = result as HKDerivativesResult
```
with:
```typescript
const { data: result } = await fetchWithCache('hk_derivatives', () => app.GetHKDerivatives(), 15 * 60 * 1000)
rawData.value = result as HKDerivativesResult
```

- [ ] **Step 5: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "PredictionMarketPanel|SchedulePanel|CBArbitragePanel|HKDerivativesPanel" || echo "No errors"`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/panels/PredictionMarketPanel.vue frontend/src/terminal/panels/SchedulePanel.vue frontend/src/terminal/panels/CBArbitragePanel.vue frontend/src/terminal/panels/HKDerivativesPanel.vue
git commit -m "perf: add fetchWithCache to PredictionMarket/Schedule/CBArbitrage/HKDerivatives panels"
```

---

### Task 3: HKIPOPanel + GeopoliticsPanel + SatellitePanel + WhaleTrackingPanel

- [ ] **Step 1: HKIPOPanel.vue**

Add import after line 6:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 12, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Find `fetchData()` or the onMounted fetch call. Look for `app.GetHKIPOCalendar`. Replace:
```typescript
const result = await app.GetHKIPOCalendar(year.value)
// ... data processing
```
with:
```typescript
const { data: result } = await fetchWithCache(`hk_ipo:${year.value}`, () => app.GetHKIPOCalendar(year.value), 15 * 60 * 1000)
// ... same data processing
```

- [ ] **Step 2: GeopoliticsPanel.vue**

Add import after line 6:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 49, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Find the `loadRisks()` / `fetchRisks()` function. Look for `app.GetGeopoliticsRisks`. Replace:
```typescript
const result = await app.GetGeopoliticsRisks()
risks.value = result.risks || []
```
with:
```typescript
const { data: result } = await fetchWithCache('geopolitics_risks', () => app.GetGeopoliticsRisks(), 30 * 60 * 1000)
risks.value = result?.risks || []
```

Find `loadDetail()` / `fetchDetail()`. Look for `app.GetGeopoliticsDetail`. Wrap with key `geopolitics_detail:${topicId}` and TTL 30min.

- [ ] **Step 3: SatellitePanel.vue**

Add import after line 7:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 36, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `loadRegions()` (~line 38), replace:
```typescript
const result = await app.GetSatelliteSnapshots()
regions.value = result.regions || []
```
with:
```typescript
const { data: result } = await fetchWithCache('satellite_snapshots', () => app.GetSatelliteSnapshots(), 30 * 60 * 1000)
regions.value = result?.regions || []
```

Find the detail fetch for `GetSatelliteDetail`, wrap with key `satellite_detail:${regionId}` and TTL 30min.

- [ ] **Step 4: WhaleTrackingPanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 23, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchData()` (~line 50), look for `app.GetWhaleTransactions`. Replace:
```typescript
const result = await app.GetWhaleTransactions(address.value, minUsd.value)
txs.value = result?.txs || []
```
with:
```typescript
const { data: result } = await fetchWithCache(`whale_txs:${address.value}:${minUsd.value}`, () => app.GetWhaleTransactions(address.value, minUsd.value), 3 * 60 * 1000)
txs.value = result?.txs || []
```

- [ ] **Step 5: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "HKIPOPanel|GeopoliticsPanel|SatellitePanel|WhaleTrackingPanel" || echo "No errors"`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/panels/HKIPOPanel.vue frontend/src/terminal/panels/GeopoliticsPanel.vue frontend/src/terminal/panels/SatellitePanel.vue frontend/src/terminal/panels/WhaleTrackingPanel.vue
git commit -m "perf: add fetchWithCache to HKIPO/Geopolitics/Satellite/WhaleTracking panels"
```

---

### Task 4: DefiTVLPanel + CryptoOverviewPanel + FuturesPanel

- [ ] **Step 1: DefiTVLPanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 18, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Find `fetchData()` / `loadData()`. Look for `app.GetDeFiTVL`. Replace:
```typescript
const result = await app.GetDeFiTVL()
protocols.value = result?.protocols || []
```
with:
```typescript
const { data: result } = await fetchWithCache('defi_tvl', () => app.GetDeFiTVL(), 3 * 60 * 1000)
protocols.value = result?.protocols || []
```

- [ ] **Step 2: CryptoOverviewPanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After the imports, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Change line 16-28 — wrap `useDataFetch`'s fn with cache:
```typescript
const { data: cryptos, loading, error, execute: refreshExec } = useDataFetch<CryptoRow[]>(async () => {
  const { data: result } = await fetchWithCache('crypto_overview', () => (window as any).go?.main?.App?.GetCryptoOverview([]), 3 * 60 * 1000)
  if (result?.cryptos) {
    return result.cryptos.map((c: any) => ({
      symbol: c.symbol?.replace('USDT', '') || c.symbol,
      price: c.price || 0,
      changePct24h: c.change_pct || 0,
    }))
  }
  return []
})
```

- [ ] **Step 3: FuturesPanel.vue**

Add import after line 2:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 5, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `loadData()` (~line 27), replace:
```typescript
const result = await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [], '', '', {})
if (result?.data) data.value = JSON.parse(result.data)
```
with:
```typescript
const { data: result } = await fetchWithCache('futures_data', () => w.go.main.App.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 15 * 60 * 1000)
if (result?.data) data.value = JSON.parse(result.data)
```

- [ ] **Step 4: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "DefiTVLPanel|CryptoOverviewPanel|FuturesPanel" || echo "No errors"`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/DefiTVLPanel.vue frontend/src/terminal/panels/CryptoOverviewPanel.vue frontend/src/terminal/panels/FuturesPanel.vue
git commit -m "perf: add fetchWithCache to DeFiTVL/CryptoOverview/Futures panels"
```

---

### Task 5: GasFeePanel + DepthComparisonPanel + FundingRatePanel + LiquidationPanel (Group B polling)

- [ ] **Step 1: GasFeePanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 6, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchData()` (~line 19), replace:
```typescript
const raw = await app.GetGasFees()
if (raw?.success === false) { gas.value = null; return }
gas.value = raw?.data as GasData || null
```
with:
```typescript
const { data: raw } = await fetchWithCache('gas_fees', () => app.GetGasFees(), 60 * 1000)
if (raw?.success === false) { gas.value = null; return }
gas.value = raw?.data as GasData || null
```

- [ ] **Step 2: DepthComparisonPanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 5, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchAll()` (~line 34), for each `GetCryptoDepth` call, wrap:
```typescript
// Before:
const raw = await app.GetCryptoDepth(ex, symbol.value, limit.value)
// After:
const { data: raw } = await fetchWithCache(`crypto_depth:${ex}:${symbol.value}:${limit.value}`, () => app.GetCryptoDepth(ex, symbol.value, limit.value), 60 * 1000)
```

- [ ] **Step 3: FundingRatePanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 19, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchRates()` (~line 45), replace:
```typescript
const result = await app.GetCryptoFundingRates([])
rates.value = (result || []).map(...)
```
with:
```typescript
const { data: result } = await fetchWithCache('funding_rates', () => app.GetCryptoFundingRates([]), 60 * 1000)
rates.value = (result || []).map(...)
```

- [ ] **Step 4: LiquidationPanel.vue**

Add import after line 3:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 22, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `fetchData()` (~line 38), replace:
```typescript
const result = await app.GetCryptoLiquidations(symbol.value, 100)
liquidations.value = (result || []).map(...)
```
with:
```typescript
const { data: result } = await fetchWithCache(`liquidations:${symbol.value}:100`, () => app.GetCryptoLiquidations(symbol.value, 100), 60 * 1000)
liquidations.value = (result || []).map(...)
```

- [ ] **Step 5: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "GasFeePanel|DepthComparisonPanel|FundingRatePanel|LiquidationPanel" || echo "No errors"`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/panels/GasFeePanel.vue frontend/src/terminal/panels/DepthComparisonPanel.vue frontend/src/terminal/panels/FundingRatePanel.vue frontend/src/terminal/panels/LiquidationPanel.vue
git commit -m "perf: add fetchWithCache to GasFee/DepthComparison/FundingRate/Liquidation panels"
```

---

### Task 6: MarketDepthPanel + MarketOverviewPanel + WatchlistPanel (Group B continued)

- [ ] **Step 1: MarketDepthPanel.vue**

Add import after line 5:
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After line 10, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

In `refresh()` (~line 59), wrap the `GetQuote` + `GetDepth` calls:
```typescript
// Replace:
const [quoteResult, depthResult] = await Promise.all([
  app.GetQuote(mkt, symbol.value),
  app.GetDepth(mkt, symbol.value).catch(() => null),
])
// With:
const [quoteResult, depthResult] = await Promise.all([
  fetchWithCache(`quote:${mkt}:${symbol.value}`, () => app.GetQuote(mkt, symbol.value), 60 * 1000).then(r => r.data),
  app.GetDepth(mkt, symbol.value).catch(() => null),  // depth not cached (real-time)
])
```

In `fetchAuction()` (~line 89), wrap:
```typescript
// Replace:
const result = await app.GetAuction(symbol.value)
// With:
const { data: result } = await fetchWithCache(`auction:${symbol.value}`, () => app.GetAuction(symbol.value), 60 * 1000)
```

- [ ] **Step 2: MarketOverviewPanel.vue**

Find the file. Add import (after existing imports):
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After the store imports, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Find the `GetBlockRank` call (~line 52). Wrap:
```typescript
// Before:
const result = await app.GetBlockRank(1, 0, 10)
// After:
const { data: result } = await fetchWithCache('block_rank', () => app.GetBlockRank(1, 0, 10), 5 * 60 * 1000)
```

- [ ] **Step 3: WatchlistPanel.vue**

Add import (check existing imports first):
```typescript
import { usePanelCache } from '@/lib/composables/usePanelCache'
```

After existing composable setups, add:
```typescript
const { fetchWithCache } = usePanelCache()
```

Find the `GetQuote` call (~line 35). Wrap with key `quote:${market}:${sym}` and TTL 0 (no cache for real-time quotes — only cache the SearchSymbols call if it exists).

For `SearchSymbols` (if called), add cache with key `search:${query}` and TTL 5min.

- [ ] **Step 4: Type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "MarketDepthPanel|MarketOverviewPanel|WatchlistPanel" || echo "No errors"`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/MarketDepthPanel.vue frontend/src/terminal/panels/MarketOverviewPanel.vue frontend/src/terminal/panels/WatchlistPanel.vue
git commit -m "perf: add fetchWithCache to MarketDepth/MarketOverview/Watchlist panels"
```

---

### Task 7: Final verification + CHANGELOG + build

- [ ] **Step 1: Full type check**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -v "node_modules" | grep -v "pre-existing"
```
Expected: No errors from modified files

- [ ] **Step 2: Full build**

```bash
make build-full 2>&1 | tail -5
```
Expected: Build OK

- [ ] **Step 3: Update CHANGELOG.md**

Add under `[2026.7.1]`:

```markdown
### Changed
- [Frontend] **22 面板接入 usePanelCache** — 统一使用 fetchWithCache 前端缓存，按数据类型配置不同 TTL（宏观/地缘 30min、衍生品/日历 15min、资金流向 5min、链上 3min、盘口 1min），减少重复请求和 Python sidecar 压力
```

- [ ] **Step 4: Commit CHANGELOG + push**

```bash
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for panel caching"
git push origin main
```
