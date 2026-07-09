<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { PanelHeader, LoadingState } from '@/terminal/components/panel'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import type { KlineDataItem } from '@/lib/buildChartOption'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { marketUpColor, marketDownColor } from '@/lib/composables/useMarketColors'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)

const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const autoRefresh = ref(true)
const refreshInterval = ref(15)
const countdown = ref(refreshInterval.value)
const loadError = ref('')
let timer: ReturnType<typeof setInterval> | null = null

// Chart state
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
  if (sym) {
    const found = indices.value.find(i => i.symbol === sym)
    if (found) return found
  }
  return indices.value[0] || null
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

// Lightweight K-line option for compact market overview display.
// Uses fixed pixel grid heights (not percentage) to avoid volume/K-line overlap
// in the constrained panel space. No dataZoom slider to save vertical room.
const chartOption = computed(() => {
  const data = chartOHLCV.value
  if (!data.length) return {} as ECBasicOption

  const symbol = selectedIndex.value?.symbol || ''
  const upCol = marketUpColor(symbol)
  const downCol = marketDownColor(symbol)

  const dates = data.map(d => d.date)
  const kdata = data.map(d => [d.open, d.close, d.low, d.high])
  const vdata = data.map(d => ({
    value: d.volume / 10000,
    itemStyle: { color: d.close >= d.open ? upCol : downCol },
  }))

  return {
    animation: false,
    backgroundColor: 'transparent',
    grid: [
      { left: 54, right: 12, top: 8, height: '58%' },
      { left: 54, right: 12, top: '72%', height: '22%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { show: false }, axisTick: { show: false }, splitLine: { show: false } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { show: false }, axisTick: { show: false }, splitLine: { show: false } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { fontSize: 10, color: '#888' }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } }, splitNumber: 3 },
      { type: 'value', gridIndex: 1, scale: true, axisLabel: { show: false }, splitLine: { show: false }, splitNumber: 2 },
    ],
    series: [
      {
        type: 'candlestick', name: 'K线',
        data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        itemStyle: { color: upCol, color0: downCol, borderColor: upCol, borderColor0: downCol },
      },
      {
        type: 'bar', name: '成交量',
        data: vdata, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      },
    ],
    tooltip: {
      trigger: 'axis',
      formatter: (ps: any[]) => {
        if (!ps?.length) return ''
        const item = data[ps[0].dataIndex]
        if (!item) return ''
        return `<div style="font-size:12px">${item.date}</div>
<div>开: ${item.open.toFixed(2)} 收: ${item.close.toFixed(2)}</div>
<div>高: ${item.high.toFixed(2)} 低: ${item.low.toFixed(2)}</div>
<div>量: ${(item.volume / 10000).toFixed(0)}万</div>`
      },
    },
  } as ECBasicOption
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

// Index card click: select index for K-line chart only (no cross-panel linkage)
function onSelectIndex(idx: typeof indices.value[0]) {
  if (!idx) return
  dataStore.setSelectedIndex(idx.symbol)
  loadIndexChart()
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
    const lookbackDays = indexInterval.value === '1d' ? 60 : indexInterval.value === '5d' ? 180 : indexInterval.value === '1mo' ? 365 : 730
    const start = end - lookbackDays * 86400
    const [rawBars] = await app.FetchOHLCV(mkt, idx.symbol, '1D', 'qfq', start, end)
    if (!rawBars?.length) { chartOHLCV.value = []; return }
    chartOHLCV.value = rawBars.map((b: any) => ({
      date: (b.date || '').slice(0, 10),
      open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume || 0,
    }))
  } catch (e) {
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
})
</script>

<template>
  <div class="market-overview-panel">
    <PanelHeader
      :title="$t('misc.market_overview')"
      :subtitle="formatTime(updatedAt)"
      :controls="headerControls"
    />

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>

    <!-- Zone 1: Market Tabs -->
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

<style scoped>
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
  min-height: 240px;
  flex: 1 0 auto;
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
  max-height: 200px;
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
</style>
