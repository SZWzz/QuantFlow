<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useDataStore } from '@/stores/data'
import { PanelHeader, LoadingState } from '@/terminal/components/panel'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { useWebSocket } from '@/lib/composables/useWebSocket'
import { useMinuteChart } from '@/lib/composables/useMinuteChart'
import { buildKlineOption } from '@/lib/buildChartOption'
import type { KlineDataItem } from '@/lib/buildChartOption'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'
import { marketUpColor, marketDownColor } from '@/lib/composables/useMarketColors'
import { logger } from '@/lib/logger'
import { isTradingHours } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ws = useWebSocket()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const theme = useChartTheme()
const indicatorCache = createIndicatorCache()

const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const loadError = ref('')

// Chart mode: 分时 | K线
const chartMode = ref<'minute' | 'kline'>('minute')

// K-line state
const chartOHLCV = ref<KlineDataItem[]>([])
const indexChartLoading = ref(false)
const indexInterval = ref<'1d' | '5d' | '1mo' | '1y'>('1d')

// Minute chart state (via composable)
const minuteSymbol = computed(() => selectedIndex.value?.symbol || '')
const prevClose = computed(() => selectedIndex.value?.prevClose || 0)
const { minuteTicks, minuteLoading, loadMinuteLine } = useMinuteChart(minuteSymbol, prevClose)

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

// ── K-line option (reuses buildKlineOption with volume sub-chart) ──
const klineOption = computed(() => {
  if (!chartOHLCV.value.length) return {} as ECBasicOption
  return buildKlineOption(
    chartOHLCV.value,
    'none',
    'volume',
    theme,
    indicatorCache,
    selectedIndex.value?.symbol || '',
    indexInterval.value,
    undefined,
  )
})

// ── Minute chart option (lightweight, compact) ──
const minuteOption = computed(() => {
  const ticks = minuteTicks.value
  if (!ticks.length) return {} as ECBasicOption

  const symbol = selectedIndex.value?.symbol || ''
  const upCol = marketUpColor(symbol)
  const downCol = marketDownColor(symbol)
  const times = ticks.map(t => t.time)
  const prices = ticks.map(t => t.price)
  const isUp = prevClose.value > 0 && prices[prices.length - 1] >= prevClose.value
  const lineColor = isUp ? upCol : downCol

  return {
    animation: false,
    backgroundColor: 'transparent',
    grid: [{ left: 54, right: 12, top: 8, bottom: 20, height: 'auto' }],
    xAxis: {
      type: 'category', data: times,
      axisLabel: { fontSize: 10, color: theme.axisColor, interval: 30 },
      axisLine: { lineStyle: { color: theme.splitColor } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value', scale: true,
      axisLabel: { fontSize: 10, color: theme.axisColor },
      splitLine: { lineStyle: { color: theme.gridColor } },
      splitNumber: 4,
    },
    series: [
      {
        type: 'line', name: '价格',
        data: prices, smooth: false, symbol: 'none',
        lineStyle: { color: lineColor, width: 1.5 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: isUp ? upCol + '40' : downCol + '40' },
              { offset: 1, color: 'rgba(0,0,0,0)' },
            ],
          },
        },
        markLine: prevClose.value > 0 ? {
          silent: true, symbol: 'none',
          lineStyle: { color: theme.axisColor, type: 'dashed', width: 1 },
          data: [{ yAxis: prevClose.value, label: { formatter: `昨收 ${prevClose.value.toFixed(2)}`, color: theme.axisColor, fontSize: 10, position: 'start' } }],
        } : undefined,
      },
    ],
    tooltip: {
      trigger: 'axis',
      formatter: (ps: any[]) => {
        if (!ps?.length) return ''
        const idx2 = ps[0].dataIndex
        const t = ticks[idx2]
        if (!t) return ''
        const chg = prevClose.value > 0 ? t.price - prevClose.value : 0
        const chgPct = prevClose.value > 0 ? (chg / prevClose.value) * 100 : 0
        const chgColor = chg >= 0 ? upCol : downCol
        return `<div style="font-size:12px">${t.time}</div>
<div>价格: <b>${t.price.toFixed(2)}</b></div>
<div>涨跌: <span style="color:${chgColor}">${chg >= 0 ? '+' : ''}${chg.toFixed(2)} (${chgPct >= 0 ? '+' : ''}${chgPct.toFixed(2)}%)</span></div>
<div>均价: ${t.avg_price.toFixed(2)}</div>`
      },
    },
  } as ECBasicOption
})

// ── Active chart option ──
const chartOption = computed(() => {
  return chartMode.value === 'minute' ? minuteOption.value : klineOption.value
})

// Header controls
const headerControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
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

// Index card click
function onSelectIndex(idx: typeof indices.value[0]) {
  if (!idx) return
  dataStore.setSelectedIndex(idx.symbol)
  loadChart()
}

// ── Data loading ──
async function loadKlineChart() {
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
    logger.error('[MarketOverview] kline chart:', e)
    chartOHLCV.value = []
  } finally {
    indexChartLoading.value = false
  }
}

function loadChart() {
  if (chartMode.value === 'minute') {
    loadMinuteLine()
  } else {
    loadKlineChart()
  }
}

function switchChartMode(mode: 'minute' | 'kline') {
  chartMode.value = mode
  loadChart()
}

function refresh() {
  loadError.value = ''
  dataStore.fetchMarketOverview(activeMarket.value)
  loadChart()
}

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

function switchMarket(mkt: string) {
  if (mkt !== 'CN' && mkt !== 'HK' && mkt !== 'US') return
  activeMarket.value = mkt as 'CN' | 'HK' | 'US'
  dataStore.setSelectedIndex('')
  chartMode.value = 'minute'
  refresh()
  startAutoRefresh()
}

// Detect market from symbol for WebSocket topic
function detectMarket(symbol: string): string {
  if (symbol.endsWith('.SH') || symbol.endsWith('.SZ') || symbol.endsWith('.BJ')) return 'CN'
  if (symbol.endsWith('.HK')) return 'HK'
  if (symbol.startsWith('^')) return 'US'
  return 'CN'
}

// WebSocket setup
let wsConnected = false
const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/market`

function connectWS() {
  const topics = indices.value.map(idx => `market:quote:${detectMarket(idx.symbol)}:${idx.symbol}`)
  if (!topics.length) return
  ws.disconnect()
  ws.connect(wsUrl, topics)
  wsConnected = true
}

ws.onMessage('*', (msg: any) => {
  if (msg.topic?.startsWith('market:quote:') && dataStore.marketOverview) {
    const parts = msg.topic.split(':')
    const symbol = parts[parts.length - 1]
    const idx = dataStore.marketOverview.indices.find(i => i.symbol === symbol)
    if (idx && msg.data) {
      if (msg.data.last !== undefined) idx.last = msg.data.last
      if (msg.data.changePct !== undefined) idx.changePct = msg.data.changePct
    }
  }
})

// Auto-select first index when data loads, connect WebSocket
watch(indices, (val) => {
  if (val.length) {
    if (!wsConnected) connectWS()
    if (!dataStore.selectedIndexSymbol) {
      dataStore.setSelectedIndex(val[0].symbol)
      loadChart()
    }
  }
})

onMounted(() => {
  refresh()
  startAutoRefresh()
})

onUnmounted(() => {
  ws.disconnect()
  indicatorCache.clear()
  stopAutoRefresh()
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

    <!-- Zone 4: Chart (分时 / K线) -->
    <div v-if="loading && !chartOHLCV.length && !minuteTicks.length" class="chart-area">
      <LoadingState type="card" :rows="1" />
    </div>
    <div v-else-if="selectedIndex" class="chart-area">
      <!-- Chart mode tabs: 分时 | 日K -->
      <div class="chart-tabs">
        <button
          :class="{ active: chartMode === 'minute' }"
          class="chart-tab"
          @click="switchChartMode('minute')"
        >分时</button>
        <button
          :class="{ active: chartMode === 'kline' }"
          class="chart-tab"
          @click="switchChartMode('kline')"
        >日K</button>
        <template v-if="chartMode === 'kline'">
          <span class="chart-tab-sep" />
          <button
            v-for="iv in (['1d', '5d', '1mo', '1y'] as const)"
            :key="iv"
            :class="{ active: indexInterval === iv }"
            class="chart-tab interval"
            @click="indexInterval = iv; loadKlineChart()"
          >{{ iv }}</button>
        </template>
      </div>
      <div class="chart-wrapper" :class="{ loading: indexChartLoading || minuteLoading }">
        <KlineChart
          :option="chartOption"
          :symbol="selectedIndex?.symbol ?? ''"
          :loading="indexChartLoading || minuteLoading"
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
  gap: 6px;
  padding: var(--space-sm) var(--panel-padding);
  flex-wrap: wrap;
}

.index-card {
  flex: 1 1 0;
  min-width: 0;
  padding: 6px 8px;
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  overflow: hidden;
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

/* ── Zone 4: Chart Area ── */
.chart-area {
  display: flex;
  flex-direction: column;
  min-height: 300px;
  flex: 0 0 auto;
  height: 320px;
  overflow: hidden;
}

.chart-wrapper {
  flex: 1;
  min-height: 260px;
  position: relative;
}

.chart-tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: var(--space-xs) var(--panel-padding);
}

.chart-tab {
  padding: 3px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.chart-tab:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.chart-tab.active {
  background: var(--color-accent);
  color: #fff;
}

.chart-tab.interval {
  padding: 3px 8px;
  font-size: 11px;
}

.chart-tab-sep {
  width: 1px;
  height: 14px;
  background: var(--color-border-strong);
  margin: 0 4px;
}

.chart-wrapper.loading {
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
  flex: 1 1 0;
  min-height: 0;
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
