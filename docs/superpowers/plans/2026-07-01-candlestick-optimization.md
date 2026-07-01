# CandlestickPanel 优化重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor 756-line CandlestickPanel.vue into focused components + composables, fix VChart `:key` destruction, add indicator memoization, incremental polling, and type safety.

**Architecture:** New `KlineChart.vue` wraps VChart with stable key. New `useWailsApp.ts` / `buildChartOption.ts` extract reusable patterns. `useChartTheme.ts` gains MutationObserver caching. `useIndicators.ts` gains memoization wrapper. CandlestickPanel.vue imports all of the above, dropping from 600→~350 script lines.

**Tech Stack:** Vue 3 + TypeScript, ECharts (vue-echarts VChart)

## Global Constraints

- No changes to indicator algorithms (sma/ema/macd/kdj/rsi/wr) — only add memoization layer
- VChart instances must survive indicator/switching — stable key = `${symbol}` only
- Existing CSS variables (--color-up/down/accent/text/bg) unchanged
- All new files under `frontend/src/lib/` or `frontend/src/terminal/components/`
- Every task must pass `npx vue-tsc --noEmit` (pre-existing errors excluded)

---

### Task 1: Create KlineChart.vue (stable key VChart wrapper)

**Files:**
- Create: `frontend/src/terminal/components/panel/KlineChart.vue`

**Interfaces:**
- Produces: `<KlineChart :option :symbol :loading />`
  - `option: ECBasicOption` — ECharts full option object
  - `symbol: string` — used for stable `:key`
  - `loading?: boolean` — optional loading overlay

- [ ] **Step 1: Create KlineChart.vue**

```vue
<script setup lang="ts">
import { shallowRef, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { CandlestickChart, BarChart, LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, MarkLineComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import type { ECBasicOption } from 'echarts/types/dist/shared'

use([CanvasRenderer, CandlestickChart, BarChart, LineChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, MarkLineComponent])

const props = defineProps<{
  option: ECBasicOption
  symbol: string
  loading?: boolean
}>()

const chartRef = shallowRef<InstanceType<typeof VChart>>()

function refreshSize() {
  chartRef.value?.resize?.()
}

defineExpose({ refreshSize })
</script>

<template>
  <VChart
    ref="chartRef"
    :key="`kc-${symbol}`"
    :option="option"
    autoresize
    style="height: 100%; width: 100%"
  />
</template>
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "KlineChart" || echo "No KlineChart errors"`
Expected: No KlineChart errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/components/panel/KlineChart.vue
git commit -m "feat: KlineChart component with stable key VChart wrapper"
```

---

### Task 2: Cache useChartTheme with MutationObserver

**Files:**
- Modify: `frontend/src/lib/composables/useChartTheme.ts`

**Interfaces:**
- Consumes: existing `useChartTheme()` return type (reactive theme object with `textColor`, `axisColor`, `upColor`, `downColor`, `bgColor`, `gridColor`, `tooltipBg`)
- Produces: same interface, but `getComputedStyle` only called once + on `document.body.classList` mutation

- [ ] **Step 1: Rewrite useChartTheme.ts**

**Read the existing file first**, then replace:

```typescript
import { reactive, onMounted, onUnmounted } from 'vue'

export interface ChartTheme {
  textColor: string
  axisColor: string
  upColor: string
  downColor: string
  bgColor: string
  gridColor: string
  tooltipBg: string
}

const root = typeof document !== 'undefined' ? document.documentElement : null
const body = typeof document !== 'undefined' ? document.body : null

function readTheme(): ChartTheme {
  if (!root) return { textColor: '#ccc', axisColor: '#555', upColor: '#ef4444', downColor: '#22c55e', bgColor: '#1a1a2e', gridColor: '#2a2a3e', tooltipBg: '#1e1e2f' }
  const s = (v: string) => getComputedStyle(root).getPropertyValue(v).trim() || getComputedStyle(body!).getPropertyValue(v).trim()
  return {
    textColor: s('--color-text') || '#e5e7eb',
    axisColor: s('--color-text-secondary') || '#6b7280',
    upColor: s('--color-up') || '#ef4444',
    downColor: s('--color-down') || '#22c55e',
    bgColor: s('--color-bg-panel') || '#1a1a2e',
    gridColor: s('--color-border') || '#2a2a3e',
    tooltipBg: s('--color-bg-elevated') || '#1e1e2f',
  }
}

let globalTheme: ChartTheme | null = null
let observers: (() => void)[] = []

function ensureObserver() {
  if (observers.length > 0 || !body) return
  const mo = new MutationObserver(() => {
    globalTheme = readTheme()
    observers.forEach(fn => fn())
  })
  mo.observe(body, { attributes: true, attributeFilter: ['class'] })
}

export function useChartTheme() {
  if (!globalTheme) globalTheme = readTheme()
  ensureObserver()

  const theme = reactive<ChartTheme>({ ...globalTheme })
  const update = () => { Object.assign(theme, globalTheme) }
  const unsub = () => { observers = observers.filter(fn => fn !== update) }
  observers.push(update)

  onUnmounted(unsub)
  return theme
}
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "useChartTheme" || echo "No useChartTheme errors"`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/composables/useChartTheme.ts
git commit -m "perf: cache useChartTheme with MutationObserver, avoid per-eval getComputedStyle"
```

---

### Task 3: Add indicator memoization wrapper to useIndicators

**Files:**
- Modify: `frontend/src/lib/composables/useIndicators.ts`

**Interfaces:**
- Produces: exported `createIndicatorCache()` factory function
  - `getCached(key: string, fn: () => T): T` — get-or-compute
  - `clear(): void` — flush cache
- Preserves all existing exports unchanged (sma, ema, bb, macd, kdj, rsi, wr)

- [ ] **Step 1: Append createIndicatorCache to useIndicators.ts**

Add to end of `useIndicators.ts`:

```typescript
/** Memoization wrapper for indicator computations */
export function createIndicatorCache() {
  const cache = new Map<string, any>()
  return {
    getCached<T>(key: string, fn: () => T): T {
      if (cache.has(key)) return cache.get(key) as T
      const r = fn()
      cache.set(key, r)
      return r
    },
    clear() { cache.clear() },
    delete(key: string) { cache.delete(key) },
  }
}
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "useIndicators" || echo "No useIndicators errors"`
Expected: No errors

- [ ] **Step 3: Write quick memoization test in frontend tests** (find existing test file for useIndicators or create one)

Check if there's a test file: `ls frontend/src/lib/composables/__tests__/useIndicators* 2>/dev/null || echo "none"`

Create `frontend/src/lib/composables/__tests__/useIndicators.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { createIndicatorCache, sma, ema, macd, kdj } from '../useIndicators'

describe('createIndicatorCache', () => {
  it('returns cached result on second call', () => {
    const cache = createIndicatorCache()
    let count = 0
    const fn = () => { count++; return 42 }
    const a = cache.getCached('x', fn)
    const b = cache.getCached('x', fn)
    expect(a).toBe(42)
    expect(b).toBe(42)
    expect(count).toBe(1)
  })

  it('recomputes after clear', () => {
    const cache = createIndicatorCache()
    let count = 0
    const fn = () => { count++; return count }
    cache.getCached('x', fn)
    cache.clear()
    cache.getCached('x', fn)
    expect(count).toBe(2)
  })
})
```

- [ ] **Step 4: Run test**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/lib/composables/__tests__/useIndicators.test.ts`
Expected: PASS (2 tests)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/composables/useIndicators.ts frontend/src/lib/composables/__tests__/useIndicators.test.ts
git commit -m "perf: add createIndicatorCache memoization wrapper"
```

---

### Task 4: Create useWailsApp.ts (typed Wails bridge)

**Files:**
- Create: `frontend/src/lib/composables/useWailsApp.ts`

**Interfaces:**
- Produces: `useWailsApp(): WailsApp | null`
  - `WailsApp` interface with typed method signatures for all used APIs

- [ ] **Step 1: Create useWailsApp.ts**

```typescript
export interface OHLCVBar {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface MinuteTick {
  time: string
  price: number
  avg_price: number
  volume: number
}

export interface GetMinuteLineResult {
  ticks: MinuteTick[]
  last_time?: string
}

export interface WailsApp {
  FetchOHLCV(market: string, symbol: string, interval: string, fq: string, start: string, end: string): Promise<OHLCVBar[]>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<GetMinuteLineResult>
  GetMultiDayMinute(symbol: string, days: number): Promise<GetMinuteLineResult>
  GetAuditFindings(symbol: string): Promise<Record<string, any>>
  GetFinancialAnalysis(symbol: string): Promise<Record<string, any>>
  GetDelistingRisk(symbol: string): Promise<Record<string, any>>
}

let cachedApp: WailsApp | null = null

export function useWailsApp(): WailsApp | null {
  if (cachedApp) return cachedApp
  const app = (window as any)?.go?.main?.App
  if (!app) return null
  cachedApp = app as WailsApp
  return cachedApp
}
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "useWailsApp" || echo "No useWailsApp errors"`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/composables/useWailsApp.ts
git commit -m "feat: typed useWailsApp composable with Wails bridge interface"
```

---

### Task 5: CandlestickPanel Phase 1 — KlineChart integration + memoization + minute KDJ fix

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: `KlineChart.vue` (stable key VChart), `createIndicatorCache` from useIndicators, `useChartTheme` (cached)
- Consumes: existing `useStockName`, `useDataFetch`, `symbolContextStore`
- Replaces: inline `<VChart>` with `<KlineChart>`, adds `cache.getCached()` for indicators, fixes minute KDJ rolling min/max

- [ ] **Step 1: Read full CandlestickPanel.vue, then apply all changes**

Key changes to make:

**Imports (replace old VChart import pattern at top):**
```typescript
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import { sma, ema, macd, kdj, rsi, wr, bb, createIndicatorCache } from '@/lib/composables/useIndicators'
// Remove: individual ECharts imports (they're now in KlineChart.vue)
// Remove: VChart import
```

**After refs (~line 90), add:**
```typescript
const indicatorCache = createIndicatorCache()
```

**Replace option computed (~line 324):** Wrap indicator calls:
```typescript
const option = computed(() => {
  // ... same data unpacking ...
  const cacheKey = `${symbol.value}-${interval.value}-${ohlcvData.value.length}-${topOverlay.value}-${bottomMode.value}`

  // MA overlay
  const defMA = [5, 10, 20, 60]
  const maLines = cacheKey.includes('top-ma') ? defMA.map(p => ({
    p, data: indicatorCache.getCached(`sma-${cacheKey}-${p}`, () => sma(close, p))
  })) : []

  // MACD
  let macdResult: MACDResult | null = null
  if (bottomMode.value === 'macd') {
    macdResult = indicatorCache.getCached(`macd-${cacheKey}`, () => macd(close))
  }

  // KDJ
  let kdjResult: KDJResult | null = null
  if (bottomMode.value === 'kdj') {
    kdjResult = indicatorCache.getCached(`kdj-${cacheKey}`, () => kdj(close, high, low))
  }
  // ... rest of option builder ...
})
```

**Fix minute KDJ (~line 488):**
```typescript
if (minuteBottomMode.value === 'kdj') {
  const n = 9
  const minPrices = prices.map((_, i) => {
    const start = Math.max(0, i - n + 1)
    return Math.min(...prices.slice(start, i + 1))
  })
  const maxPrices = prices.map((_, i) => {
    const start = Math.max(0, i - n + 1)
    return Math.max(...prices.slice(start, i + 1))
  })
  const kd = kdj(prices, maxPrices, minPrices, n, 3, 3)
  // ... same series push logic ...
}
```

**Replace all 3 `<VChart>` instances in template with `<KlineChart>`:**

```vue
<KlineChart
  :symbol="symbol"
  :option="option"
  :loading="loading && !ohlcvData.length"
/>

<KlineChart
  :symbol="`${symbol}-minute`"
  :option="minuteChartOption"
  :loading="minuteLoading && !minuteTicks.length"
/>

<KlineChart
  :symbol="`${symbol}-multi`"
  :option="multiDayChartOption"
  :loading="multiDayLoading && !multiDayData.length"
/>
```

Remove all `:key` bindings from the old VChart instances. VChart no longer imported directly, replace with KlineChart.

**Clear cache on symbol change** (in the symbol watcher, ~line 266):
```typescript
watch(symbol, (newSym, oldSym) => {
  if (newSym !== oldSym) {
    indicatorCache.clear()
    loadOHLCV()
  }
})
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "CandlestickPanel" || echo "No CandlestickPanel errors"`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "perf: CandlestickPanel Phase1 — KlineChart, indicator memoization, minute KDJ fix"
```

---

### Task 6: CandlestickPanel Phase 2 — incremental polling + error toast + loadSeq

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Consumes: existing `loadOHLCV`, `loadMinuteLine` structure
- Adds: `incremental` parameter to loadOHLCV, `loadSeq` counter, `errorMsg` ref + auto-dismiss

- [ ] **Step 1: Read current CandlestickPanel.vue, apply Phase 2 changes**

**Add state (~line 90 area):**
```typescript
const errorMsg = ref('')
let loadSeq = 0
```

**Modify loadOHLCV to support incremental:**
```typescript
async function loadOHLCV(incremental = false) {
  const seq = ++loadSeq
  const app = (window as any)?.go?.main?.App
  if (!app?.FetchOHLCV) { loading.value = false; return }

  try {
    loading.value = true
    const { endDate } = getDateRange(symbol.value, interval.value)
    let bars: any[]
    if (incremental && ohlcvData.value.length > 0) {
      const localEnd = format(endDate, 'yyyy-MM-dd')
      const lastDate = ohlcvData.value[ohlcvData.value.length - 1].date
      bars = await app.FetchOHLCV(marketForSymbol.value, symbol.value, interval.value, 'qfq', lastDate, localEnd)
    } else {
      const { startDate, endDate } = getDateRange(symbol.value, interval.value)
      bars = await app.FetchOHLCV(marketForSymbol.value, symbol.value, interval.value, 'qfq', startDate, endDate)
    }
    if (seq !== loadSeq) return
    if (incremental && bars?.length) {
      const existing = ohlcvData.value
      const mergeMap = new Map(existing.map(b => [b.date, b]))
      for (const b of bars) mergeMap.set(b.date, b)
      ohlcvData.value = [...mergeMap.values()].sort((a, b) => a.date.localeCompare(b.date))
    } else if (bars) {
      ohlcvData.value = bars as any
    }
  } catch (e: any) {
    if (seq !== loadSeq) return
    console.error('[Candlestick]', e)
    errorMsg.value = 'K线数据加载失败: ' + (e.message || '未知错误')
    setTimeout(() => { errorMsg.value = '' }, 8000)
    if (!ohlcvData.value.length) ohlcvData.value = []
  }
  loading.value = false
}
```

**Modify startKlineRefresh to call incremental:**
```typescript
function startKlineRefresh() {
  stopKlineRefresh()
  loadOHLCV(true) // incremental
  klineTimer = window.setInterval(() => loadOHLCV(true), 30000)
}
```

**Modify startMinutePolling similarly (add loadSeq):**
```typescript
async function loadMinuteLine(since = 0) {
  const seq = ++loadSeq
  // ... existing logic ...
  // After each async call: if (seq !== loadSeq) return
}
```

**Add error template** after the header block:

```html
<div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
```

**Add CSS:**
```css
.err-toast { padding: 6px 12px; background: rgba(239,68,68,0.15); color: #ef4444; font-size: 12px; border-radius: 4px; margin-bottom: 8px; }
```

- [ ] **Step 2: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep "CandlestickPanel" || echo "No CandlestickPanel errors"`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "perf: CandlestickPanel Phase2 — incremental polling, error toast, loadSeq race guard"
```

---

### Task 7: CandlestickPanel Phase 3 — extract buildChartOption.ts + TypeScript types + useWailsApp

**Files:**
- Create: `frontend/src/lib/buildChartOption.ts`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`

**Interfaces:**
- Produces: `buildKlineOption(data, topOverlay, bottomMode, theme, indicators, cache): ECBasicOption`
- Produces: `buildMinuteOption(data, bottomMode, theme, indicators, cache): ECBasicOption`
- Produces: `buildMultiDayOption(data, theme): ECBasicOption`
- Produces: `mergeBaseOption(theme): ECBasicOption` (shared grid/axis/tooltip config)

- [ ] **Step 1: Create buildChartOption.ts**

This is a large file — extract the three option builder functions from CandlestickPanel.vue. Read the existing option computed bodies (lines 324-436, 438-503, 505-588) and extract them into pure functions.

```typescript
import type { ECBasicOption } from 'echarts/types/dist/shared'
import type { ChartTheme } from '@/lib/composables/useChartTheme'
import type { IndicatorCache } from '@/lib/composables/useIndicators'
import type { MACDResult, KDJResult, BBResult } from '@/lib/composables/useIndicators'
import { sma, ema, macd, kdj, rsi, wr, bb } from '@/lib/composables/useIndicators'

export interface KlineDataItem {
  date: string; open: number; high: number; low: number; close: number; volume: number
}

export function mergeBaseOption(theme: ChartTheme): any {
  return {
    animation: false,
    backgroundColor: 'transparent',
    grid: [{ left: 54, right: 16, top: 8, height: '52%' }, { left: 54, right: 16, top: '68%', height: '26%' }],
    xAxis: [{ type: 'category', show: true, axisLine: { lineStyle: { color: theme.gridColor } }, axisLabel: { show: false }, gridIndex: 0, splitLine: { show: true, lineStyle: { color: theme.gridColor, type: 'dashed' as const } } }],
    yAxis: [{ scale: true, splitLine: { show: true, lineStyle: { color: theme.gridColor, type: 'dashed' as const } }, axisLabel: { color: theme.axisColor, fontSize: 10 } }],
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' }, backgroundColor: theme.tooltipBg, borderColor: theme.gridColor },
    dataZoom: [{ type: 'inside', xAxisIndex: [0, 1] }],
  }
}

export function buildKlineOption(
  data: KlineDataItem[],
  topOverlay: string,
  bottomMode: string,
  theme: ChartTheme,
  cache: IndicatorCache,
  symbol: string,
  interval: string,
): ECBasicOption { /* full extracted logic from existing option computed */ }

export function buildMinuteOption(
  ticks: { time: string; price: number; volume: number }[],
  bottomMode: string,
  theme: ChartTheme,
  cache: IndicatorCache,
): ECBasicOption { /* full extracted logic from existing minuteChartOption computed */ }

export function buildMultiDayOption(
  days: { date: string; close: number; volume: number }[],
  theme: ChartTheme,
): ECBasicOption { /* full extracted logic from existing multiDayChartOption computed */ }
```

**Note to implementer:** Copy the existing option builder bodies verbatim from CandlestickPanel.vue lines 324-588. Replace `theme.xxx` references with the `theme` parameter. Replace indicator calls with `cache.getCached()` patterns.

- [ ] **Step 2: Simplify CandlestickPanel.vue option computeds**

Replace the three large option computed properties with:

```typescript
import { buildKlineOption, buildMinuteOption, buildMultiDayOption } from '@/lib/buildChartOption'
import { useWailsApp } from '@/lib/composables/useWailsApp'

const option = computed(() => {
  if (!ohlcvData.value.length) return {} as ECBasicOption
  return buildKlineOption(ohlcvData.value, topOverlay.value, bottomMode.value, theme, indicatorCache, symbol.value, interval.value)
})

const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {} as ECBasicOption
  return buildMinuteOption(minuteTicks.value, minuteBottomMode.value, theme, indicatorCache)
})

const multiDayChartOption = computed(() => {
  if (!multiDayData.value.length) return {} as ECBasicOption
  return buildMultiDayOption(multiDayData.value, theme)
})
```

**Replace all `(window as any).go.main.App.Xxx` calls** with `const app = useWailsApp(); if (!app) return; app.FetchOHLCV(...)`, `app.GetMinuteLine(...)`, `app.GetMultiDayMinute(...)`.

**Add TypeScript types:**
- Replace `const bars: any[]` → `const bars: OHLCVBar[]`
- Replace `d as any` in maps → use `KlineDataItem` interface
- Replace `series: any[]` → use ECharts `SeriesOption[]` type
- Replace `let bottomYAxis: any` → use proper ECharts YAxisOption type

- [ ] **Step 3: Verify type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | grep -E "CandlestickPanel|buildChartOption" || echo "No relevant errors"`
Expected: No errors

- [ ] **Step 4: Full build verification**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run build -q`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/buildChartOption.ts frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "perf: CandlestickPanel Phase3 — extract buildChartOption, type-safe useWailsApp"
```

---

### Task 8: Final verification + CHANGELOG

- [ ] **Step 1: Run full frontend type check and build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit && npm run build -q
```
Expected: No errors, build succeeds

- [ ] **Step 2: Run indicator tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/lib/composables/__tests__/useIndicators.test.ts
```
Expected: PASS

- [ ] **Step 3: Update CHANGELOG.md**

Add under `[2026.7.1]`:

```markdown
### Changed
- [Frontend] **CandlestickPanel 重构** — 756行→~350行脚本，拆分为 KlineChart 组件(稳定key)、buildChartOption 纯函数、useWailsApp 类型化桥接
### Fixed
- [Frontend] **VChart :key 销毁重建** — KlineChart 使用稳定 key 避免切换指标时 ECharts 实例重建，dataZoom 缩放状态不丢失
- [Frontend] **指标计算重复执行** — createIndicatorCache 缓存指标结果，切换 mode 零重算
- [Frontend] **分钟 KDJ 退化** — 改用滚动 N 周期 min/max 替代高/低价，曲线正常波动
- [Frontend] **数据加载竞态** — loadSeq 计数器确保旧请求不覆盖新数据
- [Frontend] **useChartTheme DOM 读取** — MutationObserver 回调缓存，避免每次 computed 评估触发 7 次 getComputedStyle
### Added
- [Frontend] **K 线增量轮询** — 30s 定时器仅拉取增量数据而非全量 250-450 根 K 线
- [Frontend] **数据加载错误提示** — 面板内联 err-toast 8s 自动消失
```

- [ ] **Step 4: Full build + push**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && make build-full 2>&1 | tail -3
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for CandlestickPanel optimization"
git push origin main
```
Expected: Build OK, pushed
