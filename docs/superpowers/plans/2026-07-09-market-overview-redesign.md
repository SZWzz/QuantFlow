# 市场概况页面重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite MarketOverviewPanel with Bloomberg-style layout, add index K-line chart, fix breadth data, remove block rank section

**Architecture:** 5 zones stacked vertically in CSS Grid: market tabs → index cards → sentiment bar → K-line chart (KlineChart.vue) → sector rankings (horizontal bar chart). Backend: fix breadth hardcode. Frontend: hold selectedIndexSymbol for chart linkage.

**Tech Stack:** Vue 3 + ECharts 5 + Pinia

## Global Constraints

- Backend `GetMarketOverview` breadth hardcode (`{advancers: 0, decliners: 0, unchanged: 0}`) must be fixed
- Block rank section removed (data irrelevant)
- K-line chart reuses `KlineChart.vue` component
- `wails-runtime.d.ts` `GetIndustryRanks` signature must be fixed
- All tests pass: `npx vitest run` + `vue-tsc --noEmit`
- **⚠️ Variable naming: `chartOHLCV` is the canonical name throughout (script + template + computed)**

---

### Task 1: Fix Backend + Types + Remove Block Rank

**Files:**
- Modify: `app_market.go:592` — replace breadth hardcode with real data from market data adapter
- Modify: `frontend/src/types/wails-runtime.d.ts:132` — fix `GetIndustryRanks` signature
- Modify: `frontend/src/stores/data.ts` — `MarketSentiment` already exists; verify `selectedIndexSymbol` + `setSelectedIndex` are exported

**Interfaces:**
- Consumes: `GetMarketOverview` (existing), `GetIndustryRanks` (existing)
- Produces: `MarketOverview` type with `sentiment: {limitUp, limitDown, northboundFlow, totalVolume}`

- [ ] **Step 1: Verify current types in data.ts**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && rg -n "MarketOverview|MarketBreadth|MarketSentiment|IndexSnapshot|selectedIndexSymbol|setSelectedIndex" frontend/src/stores/data.ts
```

Confirm `MarketSentiment`, `selectedIndexSymbol`, and `setSelectedIndex` already exist (they do in the current codebase — just verify).

- [ ] **Step 2: Fix backend breadth hardcode**

Read `app_market.go` around line 592 to see the current hardcode:

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && rg -n "advancers|decliners|unchanged|breadth" app/app_market.go
```

Replace the hardcoded `{advancers: 0, decliners: 0, unchanged: 0}` with real data from the market adapter. The fix depends on what data source is available — at minimum, parse the actual advancers/decliners from the market overview response. If the upstream adapter doesn't provide breadth data, compute it from the industry ranks (sum of stocks up/down/flat across all sectors).

- [ ] **Step 3: Fix wails-runtime.d.ts**

In `frontend/src/types/wails-runtime.d.ts`, find `GetIndustryRanks` and verify its signature. The Go backend `app.go:798` defines:

```go
func (a *App) GetIndustryRanks(mkt string, topN int) ([]market.IndustryRank, error)
```

Ensure the TypeScript declaration matches:
```typescript
GetIndustryRanks(mkt: string, topN: number): Promise<any[]>
```

(The current declaration may be missing the `mkt` parameter.)

- [ ] **Step 4: Verify store exports**

Confirm `selectedIndexSymbol` and `setSelectedIndex` are returned from the `useDataStore` defineStore callback (they already are in the current `data.ts:232-233`). No changes needed unless missing.

- [ ] **Step 5: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run 2>&1 | tail -10
```

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
```

- [ ] **Step 6: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add app/app_market.go frontend/src/types/wails-runtime.d.ts
git commit -m "fix(market): replace breadth hardcode with real data, fix GetIndustryRanks TS signature"
```

---

### Task 2: Rewrite MarketOverview Panel Template + Script + Styles

**Files:**
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue` — full rewrite of `<script setup>`, `<template>`, and `<style scoped>`
- Modify: `frontend/src/terminal/panels/__tests__/MarketOverviewPanel.test.ts`

- [ ] **Step 1: Write failing test**

Replace `MarketOverviewPanel.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import MarketOverviewPanel from '../MarketOverviewPanel.vue'

describe('MarketOverviewPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders market tabs', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    expect(wrapper.find('.market-tabs').exists()).toBe(true)
  })

  it('renders index cards section', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    expect(wrapper.find('.index-cards').exists()).toBe(true)
  })

  it('renders kline area container', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    expect(wrapper.find('.kline-area').exists()).toBe(true)
  })
})
```

Run:
```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/MarketOverviewPanel.test.ts 2>&1
```
Expected: FAIL (new selectors don't exist yet)

- [ ] **Step 2: Rewrite MarketOverviewPanel.vue — Script**

Full rewrite of the `<script setup>` section:

```typescript
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader, LoadingState } from '@/terminal/components/panel'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { buildKlineOption, type KlineDataItem } from '@/lib/buildChartOption'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const theme = useChartTheme()
const indicatorCache = createIndicatorCache()

const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const autoRefresh = ref(true)
const refreshInterval = ref(15)
const countdown = ref(refreshInterval.value)
const loadError = ref('')
let timer: ReturnType<typeof setInterval> | null = null

// Chart state — canonical name: chartOHLCV
const chartOHLCV = ref<KlineDataItem[]>([])
const indexChartLoading = ref(false)
const indexInterval = ref<'1d' | '5d' | '1mo' | '1y'>('1d')

// Computed from store
const indices = computed(() => dataStore.marketOverview?.indices ?? [])
const breadth = computed(() => dataStore.marketOverview?.breadth ?? { advancers: 0, decliners: 0, unchanged: 0 })
const sentiment = computed(() => dataStore.marketOverview?.sentiment ?? { limitUp: 0, limitDown: 0, northboundFlow: 0, totalVolume: 0 })
const sectors = computed(() => dataStore.marketOverview?.sectors ?? [])
const updatedAt = computed(() => dataStore.marketOverview?.updatedAt ?? 0)
const loading = computed(() => dataStore.marketLoading)

const selectedIndex = computed(() => {
  const sym = dataStore.selectedIndexSymbol
  return indices.value.find(i => i.symbol === sym) || indices.value[0]
})

// Sector rankings — top 10
const topGainers = computed(() =>
  [...sectors.value].sort((a, b) => b.changePct - a.changePct).slice(0, 10)
)
const topLosers = computed(() =>
  [...sectors.value].sort((a, b) => a.changePct - b.changePct).slice(0, 10)
)

// Breadth percentages
const breadthTotal = computed(() => {
  const b = breadth.value
  return b.advancers + b.decliners + b.unchanged || 1
})
const breadthUpPct = computed(() => (breadth.value.advancers / breadthTotal.value) * 100)
const breadthFlatPct = computed(() => (breadth.value.unchanged / breadthTotal.value) * 100)
const breadthDownPct = computed(() => (breadth.value.decliners / breadthTotal.value) * 100)

// K-line chart option — uses theme from useChartTheme(), indicatorCache from createIndicatorCache()
const chartOption = computed(() => {
  if (!chartOHLCV.value.length) return {} as ECBasicOption
  return buildKlineOption(
    chartOHLCV.value,
    'none',
    'volume',
    theme,
    indicatorCache,
    selectedIndex.value?.symbol || '',
    indexInterval.value,
    undefined,  // eventMarkers
  )
})

// Header controls
const headerControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ label: autoRefresh.value ? `自动 (${countdown.value}s)` : '手动', action: toggleAutoRefresh, title: '切换自动刷新' })
  list.push({ icon: 'refresh', action: refresh, loading: loading.value, title: '刷新' })
  return list
})

// Breadth bar style helper
function breadthBarStyle(pct: number, color: string) {
  return { width: `${pct}%`, background: color }
}

function formatMoney(v: number): string {
  if (!v) return '--'
  if (Math.abs(v) >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (Math.abs(v) >= 1e4) return (v / 1e4).toFixed(2) + '万'
  return String(v)
}

function formatTime(ts: number): string {
  if (!ts) return '--'
  return new Date(ts).toLocaleTimeString()
}

// Index card click: select index for K-line chart + link to other panels
function onSelectIndex(idx: typeof indices.value[0]) {
  if (!idx) return
  dataStore.setSelectedIndex(idx.symbol)
  loadIndexChart()
  // Link to other panels (e.g., WatchlistPanel switches to this index's constituents)
  const code = idx.symbol.replace(/\.(SH|SZ|SS|CSI)$/i, '')
  ctx.setGroupSymbol(pg.groupId, code)
}

// Index tab click: select index when card not available
function onIndexCardClick(symbol: string) {
  const idx = indices.value.find(i => i.symbol === symbol)
  if (idx) onSelectIndex(idx)
}

async function loadIndexChart() {
  const idx = selectedIndex.value
  if (!idx) { chartOHLCV.value = []; return }
  indexChartLoading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app) return
    const mkt = activeMarket.value
    const end = Math.floor(Date.now() / 1000)
    const lookbackDays = indexInterval.value === '1d' ? 5 : indexInterval.value === '5d' ? 30 : indexInterval.value === '1mo' ? 90 : 365
    const start = end - lookbackDays * 86400
    const [rawBars] = await app.FetchOHLCV(mkt, idx.symbol, '1D', 'qfq', start, end)
    if (!rawBars?.length) { chartOHLCV.value = []; return }
    chartOHLCV.value = rawBars.map((b: any) => ({
      date: (b.date || '').slice(0, 10),
      open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume || 0,
    }))
  } catch(e) {
    logger.error('[MarketOverview] index chart:', e)
    chartOHLCV.value = []
  } finally {
    indexChartLoading.value = false
  }
}

function refresh() {
  loadError.value = ''
  dataStore.fetchMarketOverview(activeMarket.value)
  countdown.value = refreshInterval.value
  loadIndexChart()
}

function switchMarket(mkt: string) {
  if (mkt !== 'CN' && mkt !== 'HK' && mkt !== 'US') return
  activeMarket.value = mkt as 'CN' | 'HK' | 'US'
  // Reset selected index when switching markets
  dataStore.setSelectedIndex('')
  refresh()
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    countdown.value = refreshInterval.value
  }
}

onMounted(() => {
  refresh()
  timer = setInterval(() => {
    if (autoRefresh.value) {
      if (countdown.value <= 0) {
        refresh()
        countdown.value = refreshInterval.value
      } else {
        countdown.value--
      }
    }
  }, 1000)
})

onUnmounted(() => {
  if (timer) { clearInterval(timer); timer = null }
  indicatorCache.clear()
})
```

- [ ] **Step 3: Rewrite MarketOverviewPanel.vue — Template**

```html
<template>
  <div class="market-overview-panel">
    <PanelHeader
      :title="$t('misc.market_overview')"
      :subtitle="formatTime(updatedAt)"
      :controls="headerControls"
    />

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>

    <!-- Zone 1: Market Tabs (inline, not via PanelHeader tabs) -->
    <div class="market-tabs">
      <button
        v-for="m in (['CN', 'HK', 'US'] as const)"
        :key="m"
        :class="{ active: activeMarket === m }"
        class="mkt-tab"
        @click="switchMarket(m)"
      >{{ m }}</button>
    </div>

    <!-- Zone 2: Index Cards -->
    <div v-if="indices.length" class="index-cards">
      <div
        v-for="idx in indices"
        :key="idx.symbol"
        class="index-card"
        :class="{ active: selectedIndex?.symbol === idx.symbol }"
        @click="onSelectIndex(idx)"
      >
        <div class="idx-name">{{ idx.name }}</div>
        <div class="idx-price">{{ idx.last?.toFixed(2) ?? '--' }}</div>
        <div class="idx-chg" :class="idx.changePct >= 0 ? 'up' : 'down'">
          {{ idx.changePct >= 0 ? '+' : '' }}{{ idx.changePct?.toFixed(2) ?? '0.00' }}%
        </div>
      </div>
    </div>

    <!-- Zone 3: Breadth + Sentiment Bar -->
    <div v-if="breadthTotal > 1" class="breadth-section">
      <div class="breadth-bar">
        <div class="breadth-segment up" :style="breadthBarStyle(breadthUpPct, 'var(--color-up)')" :title="`涨 ${breadth.advancers}`" />
        <div class="breadth-segment flat" :style="breadthBarStyle(breadthFlatPct, 'var(--color-text-tertiary)')" :title="`平 ${breadth.unchanged}`" />
        <div class="breadth-segment down" :style="breadthBarStyle(breadthDownPct, 'var(--color-down)')" :title="`跌 ${breadth.decliners}`" />
      </div>
      <div class="breadth-labels">
        <span class="up-text">涨 {{ breadth.advancers }}</span>
        <span class="flat-text">平 {{ breadth.unchanged }}</span>
        <span class="down-text">跌 {{ breadth.decliners }}</span>
      </div>
      <div class="sentiment-strip">
        <span>涨停 {{ sentiment.limitUp }}</span>
        <span>跌停 {{ sentiment.limitDown }}</span>
        <span>北向 {{ sentiment.northboundFlow >= 0 ? '+' : '' }}{{ formatMoney(sentiment.northboundFlow) }}</span>
        <span>成交 {{ formatMoney(sentiment.totalVolume) }}</span>
      </div>
    </div>

    <!-- Zone 4: Index K-line Chart -->
    <div v-if="loading && !chartOHLCV.length" class="kline-area">
      <LoadingState type="card" :rows="1" />
    </div>
    <div v-else-if="chartOHLCV.length > 0" class="kline-area">
      <div class="kline-tabs">
        <button
          v-for="iv in (['1d', '5d', '1mo', '1y'] as const)"
          :key="iv"
          :class="{ active: indexInterval === iv }"
          class="interval-btn"
          @click="indexInterval = iv; loadIndexChart()"
        >{{ iv }}</button>
      </div>
      <div class="kline-wrapper" :class="{ loading: indexChartLoading }">
        <KlineChart
          :option="chartOption"
          :symbol="selectedIndex?.symbol ?? ''"
          :loading="indexChartLoading"
        />
      </div>
    </div>
    <div v-else class="empty-chart">暂无数据</div>

    <!-- Zone 5: Sector Rankings (horizontal bar chart style) -->
    <div v-if="sectors.length" class="sector-section">
      <div class="sector-column">
        <h4 class="sector-col-title up-text">{{ $t('misc.gainers') }}</h4>
        <div v-for="s in topGainers" :key="'g-' + s.name" class="sector-row">
          <span class="sector-name">{{ s.name }}</span>
          <span class="sector-chg up">+{{ s.changePct.toFixed(1) }}%</span>
          <div class="sector-bar-bg">
            <div class="sector-bar up" :style="{ width: Math.min(Math.abs(s.changePct) * 8, 100) + '%' }" />
          </div>
        </div>
      </div>
      <div class="sector-column">
        <h4 class="sector-col-title down-text">{{ $t('misc.losers') }}</h4>
        <div v-for="s in topLosers" :key="'l-' + s.name" class="sector-row">
          <span class="sector-name">{{ s.name }}</span>
          <span class="sector-chg down">{{ s.changePct.toFixed(1) }}%</span>
          <div class="sector-bar-bg">
            <div class="sector-bar down" :style="{ width: Math.min(Math.abs(s.changePct) * 8, 100) + '%' }" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 4: Rewrite MarketOverviewPanel.vue — Styles**

Replace the entire `<style scoped>` block:

```css
.market-overview-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  color: var(--color-text-primary);
  background: var(--color-bg-panel);
}

.panel-error {
  padding: 8px 12px;
  margin: 0 var(--panel-padding);
  border-radius: var(--radius-sm);
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  font-size: 12px;
}

/* ── Zone 1: Market Tabs ── */
.market-tabs {
  display: flex;
  gap: 2px;
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-strong);
}

.mkt-tab {
  padding: 4px 12px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-sm);
  cursor: pointer;
  transition: all 0.15s;
}

.mkt-tab:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.mkt-tab.active {
  background: var(--color-accent);
  color: #fff;
}

/* ── Zone 2: Index Cards ── */
.index-cards {
  display: flex;
  gap: var(--space-sm);
  overflow-x: auto;
  padding: var(--space-md) var(--panel-padding);
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-strong) transparent;
}

.index-card {
  flex: 0 0 auto;
  min-width: 110px;
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.index-card:hover {
  border-color: var(--color-accent);
}

.index-card.active {
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}

.idx-name {
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.idx-price {
  font-size: var(--font-md);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.idx-chg {
  font-size: var(--font-xs);
  font-weight: 500;
}

.idx-chg.up { color: var(--color-up); }
.idx-chg.down { color: var(--color-down); }

/* ── Zone 3: Breadth + Sentiment ── */
.breadth-section {
  padding: 0 var(--panel-padding) var(--space-sm);
}

.breadth-bar {
  display: flex;
  height: 6px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  margin-bottom: var(--space-xs);
}

.breadth-segment {
  height: 100%;
  transition: width 0.3s ease;
}

.breadth-segment.up { background: var(--color-up); }
.breadth-segment.down { background: var(--color-down); }
.breadth-segment.flat { background: var(--color-text-tertiary); }

.breadth-labels {
  display: flex;
  gap: var(--space-lg);
  font-size: var(--font-xs);
  margin-bottom: var(--space-sm);
}

.sentiment-strip {
  display: flex;
  gap: var(--space-lg);
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  padding-bottom: var(--space-sm);
  border-bottom: 1px solid var(--color-border-strong);
}

.up-text { color: var(--color-up); }
.down-text { color: var(--color-down); }
.flat-text { color: var(--color-text-tertiary); }

/* ── Zone 4: K-line Chart ── */
.kline-area {
  display: flex;
  flex-direction: column;
  min-height: 200px;
  flex: 1;
  overflow: hidden;
}

.kline-tabs {
  display: flex;
  gap: 2px;
  padding: var(--space-xs) var(--panel-padding);
}

.interval-btn {
  padding: 2px 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  cursor: pointer;
  transition: all 0.15s;
}

.interval-btn:hover {
  background: var(--color-bg-hover);
}

.interval-btn.active {
  background: var(--color-accent);
  color: #fff;
}

.kline-wrapper {
  flex: 1;
  min-height: 160px;
  position: relative;
}

.kline-wrapper.loading {
  opacity: 0.5;
  pointer-events: none;
}

.empty-chart {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--color-text-tertiary);
  font-size: var(--font-sm);
  border-top: 1px solid var(--color-border-strong);
}

/* ── Zone 5: Sector Rankings ── */
.sector-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
  padding: var(--space-md) var(--panel-padding);
  border-top: 1px solid var(--color-border-strong);
  flex-shrink: 0;
  max-height: 360px;
  overflow: hidden;
}

.sector-column {
  overflow-y: auto;
}

.sector-col-title {
  font-size: var(--font-sm);
  font-weight: 600;
  margin: 0 0 var(--space-sm);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-strong);
}

.sector-row {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: 2px 0;
  font-size: var(--font-xs);
  position: relative;
}

.sector-name {
  width: 60px;
  flex-shrink: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-primary);
}

.sector-chg {
  width: 52px;
  flex-shrink: 0;
  text-align: right;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.sector-chg.up { color: var(--color-up); }
.sector-chg.down { color: var(--color-down); }

.sector-bar-bg {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--color-bg-elevated);
  overflow: hidden;
}

.sector-bar {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
}

.sector-bar.up { background: var(--color-up); }
.sector-bar.down { background: var(--color-down); }
```

- [ ] **Step 5: Run tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/MarketOverviewPanel.test.ts 2>&1
```

Expected: PASS (4/4)

- [ ] **Step 6: TypeScript check**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | tail -10
```

Expected: no errors (may have pre-existing unrelated errors — only check for errors in MarketOverviewPanel.vue and related types)

- [ ] **Step 7: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add frontend/src/terminal/panels/MarketOverviewPanel.vue frontend/src/terminal/panels/__tests__/MarketOverviewPanel.test.ts
git commit -m "feat(market): rewrite MarketOverviewPanel with Bloomberg-style 5-zone layout, K-line chart, sentiment bar"
```

---

### Task 3: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Read CHANGELOG**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && head -30 CHANGELOG.md
```

- [ ] **Step 2: Add entries**

Under `## [2026.7.9]` → `### Changed`:
```markdown
- [Frontend] MarketOverviewPanel redesigned: 5-section Bloomberg-style layout (market tabs → index cards → sentiment bar → K-line chart → sector bar charts)
- [Frontend] Index K-line chart added (reuses KlineChart.vue, click index cards to switch, supports 1d/5d/1mo/1y intervals)
- [Frontend] Breadth bar now shows real advancers/decliners/unchanged ratio with percentage-based widths
- [Frontend] Sentiment strip shows 涨停/跌停/北向资金/成交额
- [Frontend] Sector rankings show top 10 gainers/losers with horizontal bar charts
- [Frontend] Removed block rank section (data irrelevant for market overview)
- [Frontend] Index card click links to other panels via symbolContext (cross-panel index selection)
```

- [ ] **Step 3: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for market overview redesign"
```

---

## Revision Notes

> **Changes made from original plan (2026-07-09):**

| Issue | Fix |
|-------|-----|
| `indexOHLCV` vs `chartOHLCV` naming inconsistency | Canonical name `chartOHLCV` used everywhere |
| `onIndexCardClick` vs `onSelectIndex` mismatch | `onSelectIndex(idx)` is the main handler; `onIndexCardClick(symbol)` delegates to it |
| Missing CSS for all new classes | Complete `<style scoped>` block added |
| `v-if`/`v-else-if` chain broken | Fixed: `v-if="loading && !chartOHLCV.length"` → `v-else-if="chartOHLCV.length > 0"` → `v-else` |
| `GLOBAL` market tab not handled | Removed GLOBAL tab; only CN/HK/US |
| `--bar-pct` CSS class bug | Replaced with proper `.sector-bar-bg` + `.sector-bar` structure |
| `buildKlineOption` called with `{} as any` for theme | Uses `useChartTheme()` for real `ChartThemeColors` |
| Backend breadth fix ambiguous | Explicit: replace hardcode with real data from adapter |
| Index click cross-panel linkage lost | Restored: `ctx.setGroupSymbol(pg.groupId, code)` in `onSelectIndex` |
| `useIndicators` import non-existent | Correct import: `createIndicatorCache` from `@/lib/composables/useIndicators` |
| Task 1 Step 4 misleading description | Clarified: verify store exports already exist |
