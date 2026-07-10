# Performance Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize four performance pain points: MarketOverview auto-refresh, minute chart composable dedup, OHLCV progressive loading, and bundle lazy loading.

**Architecture:** Extract shared logic into a `useMinuteChart` composable and a public `isTradingHours` utility; add smart 30s polling to MarketOverview; reduce OHLCV initial fetch from 25y to 1y with dataZoom-triggered expansion. Bundle lazy loading (#3) is already implemented via the panel registry's dynamic `import()` — no code changes needed.

**Tech Stack:** Vue 3 + TypeScript + ECharts (vue-echarts) + Wails IPC

## Global Constraints

- Follow existing project patterns (Composition API `<script setup>`, Pinia stores)
- No new dependencies
- Each task independently testable (manual smoke test)
- TDD where applicable (composable unit tests)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `frontend/src/lib/wails.ts` | Modify | Add `isTradingHours(market)` utility (extracted from CandlestickPanel) |
| `frontend/src/lib/composables/useMinuteChart.ts` | Create | Shared minute chart data composable |
| `frontend/src/terminal/panels/MarketOverviewPanel.vue` | Modify | Auto-refresh polling + useMinuteChart |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | Modify | useMinuteChart + OHLCV progressive loading |
| `frontend/src/terminal/TerminalMode.vue` | Modify | (Task 4 only if needed) Verify lazy loading |

---

### Task 1: Extract `isTradingHours` to shared utility

**Files:**
- Modify: `frontend/src/lib/wails.ts`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue:44-68`

**Interfaces:**
- Produces: `export function isTradingHours(market: string): boolean` — accepts 'CN' | 'HK' | 'US' | 'CRYPTO', returns true if currently within trading hours

- [ ] **Step 1: Add `isTradingHours` to `wails.ts`**

Open `frontend/src/lib/wails.ts` and add after the `detectMarket` function:

```typescript
/** Check if the given market is currently in trading hours (local time). */
export function isTradingHours(market: string): boolean {
  const now = new Date()
  const day = now.getDay()
  if (day === 0 || day === 6) return false
  if (market === 'CRYPTO') return true
  if (market === 'HK') {
    // HKEX: 09:30-12:00, 13:00-16:00 (Mon-Fri)
    const h = now.getHours()
    const m = now.getMinutes()
    const t = h * 60 + m
    return (t >= 9 * 60 + 30 && t <= 12 * 60) || (t >= 13 * 60 && t <= 16 * 60)
  }
  if (market === 'US') {
    // NYSE/Nasdaq: 09:30-16:00 ET ≈ 13:30-21:00 UTC
    const ut = now.getUTCHours() * 60 + now.getUTCMinutes()
    return ut >= 13 * 60 + 30 && ut <= 21 * 60
  }
  // CN default: 09:30-11:30, 13:00-15:00 (Mon-Fri)
  const h = now.getHours()
  const m = now.getMinutes()
  const t = h * 60 + m
  return (t >= 9 * 60 + 30 && t <= 11 * 60 + 30) || (t >= 13 * 60 && t <= 15 * 60)
}
```

- [ ] **Step 2: Replace CandlestickPanel's inline `isTradingHours` with import**

In `frontend/src/terminal/panels/CandlestickPanel.vue`:

Remove the inline `isTradingHours` function (lines 45-68).

Add import (line 9 area, next to existing `detectMarket` import):
```typescript
import { detectMarket, isTradingHours } from '@/lib/wails'
```

Update the one call site that currently reads `symbol.value` from closure — it's used in `startKlineRefresh`:
```typescript
// Before (line ~373):
if (detectMarket(symbol.value) !== 'CN' && !isTradingHours()) return

// After:
if (detectMarket(symbol.value) !== 'CN' && !isTradingHours(detectMarket(symbol.value))) return
```

Also the minute polling guard:
```typescript
// Before (line ~348):
if (detectMarket(symbol.value) !== 'CN') { ... return }

// No change needed — minute guard doesn't call isTradingHours currently
```

- [ ] **Step 3: Verify with Go vet + build**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

Expected: no new type errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/wails.ts frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "refactor: extract isTradingHours to shared utility"
```

---

### Task 2: MarketOverview auto-refresh polling

**Files:**
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`

**Interfaces:**
- Consumes: `isTradingHours(market: string): boolean` from `@/lib/wails`
- Produces: 30s polling timer during trading hours

- [ ] **Step 1: Add auto-refresh timer with trading-hours guard**

In `frontend/src/terminal/panels/MarketOverviewPanel.vue`, add import:
```typescript
import { isTradingHours } from '@/lib/wails'
```

Add timer state and control functions after the existing `refresh()` function (around line 268):

```typescript
// ── Auto-refresh polling (trading hours: 30s interval) ──
let autoRefreshTimer: ReturnType<typeof setInterval> | null = null

function startAutoRefresh() {
  stopAutoRefresh()
  if (!isTradingHours(activeMarket.value)) return
  autoRefreshTimer = setInterval(() => {
    if (!isTradingHours(activeMarket.value)) {
      stopAutoRefresh()
      return
    }
    dataStore.fetchMarketOverview(activeMarket.value)
  }, 30000)
}

function stopAutoRefresh() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
}
```

- [ ] **Step 2: Wire into lifecycle and market switching**

Modify `switchMarket` to restart polling:
```typescript
function switchMarket(mkt: string) {
  if (mkt !== 'CN' && mkt !== 'HK' && mkt !== 'US') return
  activeMarket.value = mkt as 'CN' | 'HK' | 'US'
  dataStore.setSelectedIndex('')
  chartMode.value = 'minute'
  refresh()
  startAutoRefresh()  // restart polling for new market
}
```

Modify `onMounted`:
```typescript
onMounted(() => {
  refresh()
  startAutoRefresh()
})
```

Modify `onUnmounted`:
```typescript
onUnmounted(() => {
  ws.disconnect()
  indicatorCache.clear()
  stopAutoRefresh()
})
```

- [ ] **Step 3: Verify build**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/MarketOverviewPanel.vue
git commit -m "feat: MarketOverview auto-refresh every 30s during trading hours"
```

---

### Task 3: Create `useMinuteChart` composable

**Files:**
- Create: `frontend/src/lib/composables/useMinuteChart.ts`

**Interfaces:**
- Produces:
  ```typescript
  function useMinuteChart(symbol: Ref<string>, prevClose: Ref<number>, opts?: {
    polling?: boolean
    pollingInterval?: number
  }): {
    minuteTicks: ShallowRef<MinuteTick[]>
    minuteLoading: Ref<boolean>
    loadMinuteLine: () => Promise<void>
    startPolling: () => void
    stopPolling: () => void
  }
  ```

- [ ] **Step 1: Write the composable**

Create `frontend/src/lib/composables/useMinuteChart.ts`:

```typescript
import { ref, shallowRef } from 'vue'
import type { Ref, ShallowRef } from 'vue'
import { useDataStore } from '@/stores/data'
import { useWailsApp, type MinuteTick } from '@/lib/composables/useWailsApp'

function getTodayDateString(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function parseMinuteTimeToUnix(timeStr: string): number {
  const today = getTodayDateString()
  const d = new Date(`${today}T${timeStr}:00+08:00`)
  return Math.floor(d.getTime() / 1000)
}

export function useMinuteChart(
  symbol: Ref<string>,
  prevClose: Ref<number>,
  opts?: { polling?: boolean; pollingInterval?: number },
) {
  const minuteTicks = shallowRef<MinuteTick[]>([])
  const minuteLoading = ref(false)
  let loadSeq = 0
  let minuteTimer: ReturnType<typeof setInterval> | null = null

  async function loadMinuteLine() {
    const seq = ++loadSeq
    const app = useWailsApp()
    if (!app) return
    minuteLoading.value = true
    try {
      const lastTick = minuteTicks.value.length > 0
        ? minuteTicks.value[minuteTicks.value.length - 1]
        : null
      const sinceTimestamp = lastTick
        ? parseMinuteTimeToUnix(lastTick.time)
        : 0

      const dataStore = useDataStore()
      const result = await dataStore.fetchMinuteLine(symbol.value, sinceTimestamp)
      if (seq !== loadSeq) return
      const ticks: MinuteTick[] = Array.isArray(result) ? result[0] : result
      if (!Array.isArray(ticks) || ticks.length === 0) return

      if (sinceTimestamp === 0) {
        minuteTicks.value = ticks
      } else {
        const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
        for (const t of ticks) {
          existing.set(t.time, t)
        }
        minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
      }
    } catch (e) {
      console.error('[useMinuteChart] fetch:', e)
    } finally {
      minuteLoading.value = false
    }
  }

  function startPolling() {
    stopPolling()
    if (!opts?.polling) return
    const interval = opts.pollingInterval ?? 5000
    loadMinuteLine()
    minuteTimer = window.setInterval(() => loadMinuteLine(), interval)
  }

  function stopPolling() {
    if (minuteTimer) {
      clearInterval(minuteTimer)
      minuteTimer = null
    }
  }

  return { minuteTicks, minuteLoading, loadMinuteLine, startPolling, stopPolling }
}
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/composables/useMinuteChart.ts
git commit -m "feat: add useMinuteChart composable for shared minute chart data"
```

---

### Task 4: Wire `useMinuteChart` into MarketOverviewPanel

**Files:**
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`

**Interfaces:**
- Consumes: `useMinuteChart` from `@/lib/composables/useMinuteChart`

- [ ] **Step 1: Replace inline minute state with composable**

In `frontend/src/terminal/panels/MarketOverviewPanel.vue`, add import:
```typescript
import { useMinuteChart } from '@/lib/composables/useMinuteChart'
```

Remove these lines (37-40):
```typescript
// Minute chart state
const minuteTicks = ref<MinuteTick[]>([])
const minuteLoading = ref(false)
const prevClose = ref(0)
```

Replace with composable call. The `symbol` comes from `computed(() => selectedIndex.value?.symbol || '')` and `prevClose` from `computed(() => selectedIndex.value?.prevClose || 0)`. But since `useMinuteChart` needs `Ref<string>`, we need a computed ref:

```typescript
const minuteSymbol = computed(() => selectedIndex.value?.symbol || '')
const minutePrevClose = computed(() => selectedIndex.value?.prevClose || 0)
const { minuteTicks, minuteLoading, loadMinuteLine } = useMinuteChart(minuteSymbol, minutePrevClose)
```

- [ ] **Step 2: Replace `loadMinuteChart`**

Delete the inline `loadMinuteChart` function (lines 198-229). Replace calls:

In `loadChart()` (line 256):
```typescript
function loadChart() {
  if (chartMode.value === 'minute') {
    loadMinuteLine()  // from composable
  } else {
    loadKlineChart()
  }
}
```

In the `indices` watcher (line 316):
```typescript
if (!dataStore.selectedIndexSymbol) {
  dataStore.setSelectedIndex(val[0].symbol)
  loadChart()
}
```
(No change needed here)

- [ ] **Step 3: Keep the compact minute chart option**

The MarketOverviewPanel's `minuteOption` computed stays as-is — it's the compact version specific to this panel. It reads from `minuteTicks.value` and `minutePrevClose.value` from the composable.

- [ ] **Step 4: Verify build**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/MarketOverviewPanel.vue
git commit -m "refactor: MarketOverviewPanel uses useMinuteChart composable"
```

---

### Task 5: Wire `useMinuteChart` into CandlestickPanel

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: `useMinuteChart` from `@/lib/composables/useMinuteChart`

- [ ] **Step 1: Replace inline minute state with composable**

In `frontend/src/terminal/panels/CandlestickPanel.vue`, add import:
```typescript
import { useMinuteChart } from '@/lib/composables/useMinuteChart'
```

Remove the inline `MinuteTick` interface (lines 210-216), `computeDataKey` (lines 217-221), `minuteTicks` (line 222), `minuteLoading` (line 224), `getTodayDateString` (lines 232-235), `parseMinuteTimeToUnix` (lines 237-242), `loadMinuteLine` (lines 297-343), `startMinutePolling` (lines 345-355), `stopMinutePolling` (lines 358-360).

Replace with:
```typescript
const { minuteTicks, minuteLoading, loadMinuteLine, startPolling: startMinutePoll, stopPolling: stopMinutePoll } =
  useMinuteChart(symbol, prevClose, { polling: true, pollingInterval: 5000 })
```

- [ ] **Step 2: Update minute cache injection**

Remove the `minuteDataCache` usage in `loadMinuteLine` — the composable doesn't manage shared cache. Instead, add a watcher to sync:

```typescript
// Sync minute data to shared cache for other panels
watch(minuteTicks, (ticks) => {
  if (ticks.length) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, ticks)
  }
})
```

Keep `getTodayDateString` as a local function since it's still used by the cache key.

- [ ] **Step 3: Update template references**

`startMinutePolling` → `startMinutePoll`
`stopMinutePolling` → `stopMinutePoll`

In `watch(activeTab, ...)` around line 457:
```typescript
watch(activeTab, (tab) => {
  if (tab === 'minute') {
    startMinutePoll()
  } else {
    stopMinutePoll()
  }
})
```

In the symbol change watcher (line 471): update cache sync to use the new name.

- [ ] **Step 4: Verify build**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "refactor: CandlestickPanel uses useMinuteChart composable"
```

---

### Task 6: OHLCV progressive loading (365d initial + dataZoom expansion)

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: existing `loadOHLCV(sym, incremental)` function
- Produces: dataZoom event handler that triggers OHLCV expansion

- [ ] **Step 1: Reduce initial lookback to 365 days**

In `loadOHLCV`, change line 255:
```typescript
// Before:
const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : 9125
// After:
const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : 365
```

- [ ] **Step 2: Add dataZoom expansion handler**

Add state to track expansion:
```typescript
// OHLCV progressive loading state
const ohlcvExpanding = ref(false)
const ohlcvExpandStart = ref(0) // earliest loaded unix timestamp
```

After the `ohlcvData` watcher (around line 424), update `ohlcvExpandStart`:
```typescript
watch(ohlcvData, (data) => {
  if (data.length > 0) {
    ohlcvExpandStart.value = Math.floor(new Date(data[0].date.replace(' ', 'T')).getTime() / 1000)
  }
  // ... existing logic
})
```

Add dataZoom event setup. In `initChartControllers` (or wherever echarts instance is accessible), add after the existing echarts init:

```typescript
// dataZoom expansion: when user scrolls near the start, load earlier data
let expandThrottle: ReturnType<typeof setTimeout> | null = null
function onDataZoom(params: any) {
  if (!params.batch) return
  const dz = params.batch[0]
  if (!dz || dz.start > 5) return  // only trigger when near the beginning (<5%)
  if (ohlcvExpanding.value) return // guard: already expanding
  if (['1m','5m','15m','30m','1h'].includes(interval.value)) return // skip intraday

  if (expandThrottle) clearTimeout(expandThrottle)
  expandThrottle = setTimeout(async () => {
    ohlcvExpanding.value = true
    const earlierStart = ohlcvExpandStart.value - 365 * 86400
    const app = useWailsApp()
    if (!app) { ohlcvExpanding.value = false; return }
    try {
      const [rawBars] = await app.FetchOHLCV(detectMarket(symbol.value), symbol.value, interval.value, 'qfq', earlierStart, ohlcvExpandStart.value)
      if (rawBars?.length) {
        const mergeMap = new Map(ohlcvData.value.map(b => [b.date, b]))
        for (const b of rawBars) {
          const d = new Date(b.date || '')
          const date = d.toISOString().slice(0, 10)
          mergeMap.set(date, { date, open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume })
        }
        ohlcvData.value = Array.from(mergeMap.values()).sort((a, b) => a.date.localeCompare(b.date))
        ohlcvExpandStart.value = earlierStart
      }
    } catch (e) {
      console.error('[Candlestick] OHLCV expand:', e)
    } finally {
      ohlcvExpanding.value = false
    }
  }, 500)
}
```

Wire the event. Update KlineChart to expose dataZoom events — modify `KlineChart.vue` to emit:

```vue
<script setup lang="ts">
// Add emit
const emit = defineEmits<{ dataZoom: [params: any] }>()

// In VChart, add @datazoom handler via the chart instance
const chartRef = shallowRef<InstanceType<typeof VChart>>()
// ... existing code

function getEchartsInstance() {
  const inst = (chartRef.value as any)?.chart ?? null
  return inst
}

// Watch for chart ready and bind datazoom event
import { watch } from 'vue'
watch(chartRef, (ref) => {
  const inst = (ref as any)?.chart
  if (inst) {
    inst.on('datazoom', (params: any) => emit('dataZoom', params))
  }
})
</script>
```

In `CandlestickPanel.vue`, add the handler to KlineChart:
```vue
<KlineChart
  ref="klineChartRef"
  :option="option"
  :symbol="symbol"
  :loading="loading && !ohlcvData.length"
  @dataZoom="onDataZoom"
/>
```

- [ ] **Step 3: Verify build**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/components/panel/KlineChart.vue
git commit -m "feat: OHLCV progressive loading — 365d initial + dataZoom expansion"
```

---

### Task 7: Bundle lazy loading verification (Task #3)

**Files:**
- Verify: `frontend/src/terminal/panels/registry.ts`

**Status:** Already implemented. The panel registry already uses `() => import(...)` for all heavy panels:
- `'audit' → () => import('./AuditPanel.vue')`
- `'backtest' → () => import('./BacktestPanel.vue')`
- `'model-registry' → () => import('./ModelRegistryPanel.vue')`
- `'ai-chat' → () => import('./AIChatPanel.vue')`

Vite automatically code-splits these into separate chunks (confirmed in build output). The `index.html` only eagerly loads `index.js`, `vendor-vue.js`, and `vendor-wails.js` (~250 KB total).

**`vendor-chart` (1MB)** is loaded eagerly because MarketOverviewPanel includes `KlineChart` which imports ECharts. This is acceptable — the landing page shows charts. Deferring would require removing the chart from the default view, which is a UX regression, not a performance win.

- [ ] **Step 1: Verify current state**

```bash
cat frontend/dist/index.html
# Confirm only index, vendor-vue, vendor-wails are in modulepreload
```

- [ ] **Step 2: Document and commit**

Update CHANGELOG with the finding. No code changes needed.

```bash
git add CHANGELOG.md
git commit -m "docs: note bundle lazy loading already in place via registry dynamic imports"
```

---

## Verification Checklist

After all tasks complete, run full verification:

```bash
# Type check
cd frontend && npx vue-tsc --noEmit

# Build
cd /Volumes/shenzy/vibe_coding/QuantFlow && wails3 build

# Manual smoke test (launch app):
# 1. Open MarketOverview — confirm auto-refresh in trading hours
# 2. Open CandlestickPanel — confirm minute chart loads, switch to minute tab
# 3. Switch stock — confirm OHLCV loads quickly (365 bars)
# 4. Scroll K-line to far left — confirm expansion triggers
# 5. Open AI Chat panel — confirm async chunk loads
```
