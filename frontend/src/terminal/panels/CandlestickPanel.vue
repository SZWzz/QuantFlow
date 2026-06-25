<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CandlestickChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import * as echarts from 'echarts'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'

use([CandlestickChart, BarChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const hasEcharts = computed(() => !!(echarts && VChart))

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const interval = ref(props.params?.interval || '1d')
const ohlcvData = ref<(string | number)[][]>([])
const loading = ref(false)

// Tab state
const activeTab = ref<'kline' | 'minute'>('kline')

// Minute chart data
interface MinuteTick {
  time: string
  price: number
  volume: number
  avg_price: number
}
const minuteTicks = ref<MinuteTick[]>([])
const prevClose = ref(0)
const minuteLoading = ref(false)
let minuteTimer: ReturnType<typeof setInterval> | null = null
let klineTimer: ReturnType<typeof setInterval> | null = null

async function loadOHLCV(sym: string) {
  loading.value = true
  try {
    const end = Math.floor(Date.now() / 1000)
    const start = end - 90 * 86400
    const result = await (window as any).go.main.App.FetchOHLCV(detectMarket(sym), sym, '1d', start, end)
    const bars = Array.isArray(result) ? result[0] : result
    ohlcvData.value = (bars as any[]).map((b: any) => {
      const date = typeof b.date === 'string' ? b.date : new Date(b.date || b.Date).toISOString().slice(0, 10)
      return [date, b.open ?? b.Open ?? 0, b.close ?? b.Close ?? 0, b.low ?? b.Low ?? 0, b.high ?? b.High ?? 0, b.volume ?? b.Volume ?? 0]
    })
  } catch {
    ohlcvData.value = []
  } finally {
    loading.value = false
  }
}

async function loadMinuteLine() {
  const app = (window as any).go?.main?.App
  if (!app) return
  minuteLoading.value = true
  try {
    const result = await app.GetMinuteLine(symbol.value)
    const ticks = Array.isArray(result) ? result[0] : result
    if (!Array.isArray(ticks) || ticks.length === 0) {
      minuteTicks.value = []
      return
    }
    const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
    for (const t of ticks) {
      existing.set(t.time, t)
    }
    minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
    if (prevClose.value === 0 && minuteTicks.value.length > 0) {
      prevClose.value = minuteTicks.value[0].price
    }
  } catch {
    // silent
  } finally {
    minuteLoading.value = false
  }
}

function startMinutePolling() {
  stopMinutePolling()
  loadMinuteLine()
  minuteTimer = setInterval(loadMinuteLine, 10000)
}

function stopMinutePolling() {
  if (minuteTimer) { clearInterval(minuteTimer); minuteTimer = null }
}

function startKlineRefresh() {
  if (klineTimer) clearInterval(klineTimer)
  klineTimer = setInterval(() => loadOHLCV(symbol.value), 30000)
}

function stopKlineRefresh() {
  if (klineTimer) { clearInterval(klineTimer); klineTimer = null }
}

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
    loadOHLCV(newSymbol)
  }
})

// Regenerate data on interval change
watch(interval, () => {
  loadOHLCV(symbol.value)
})

// Watch tab switch for minute polling
watch(activeTab, (tab) => {
  if (tab === 'minute') {
    startMinutePolling()
  } else {
    stopMinutePolling()
  }
})

// Watch symbol change — reload minute data
watch(() => symbol.value, () => {
  if (activeTab.value === 'minute') {
    minuteTicks.value = []
    prevClose.value = 0
    loadMinuteLine()
  }
})

// K-line auto-refresh for minute intervals
watch(interval, (iv) => {
  if (['1m', '5m', '15m', '30m', '1h'].includes(iv as string)) {
    if (activeTab.value === 'kline') {
      startKlineRefresh()
    }
  } else {
    stopKlineRefresh()
  }
})

const option = computed(() => {
  if (ohlcvData.value.length === 0) return {}
  const dates = ohlcvData.value.map((d: any) => d[0])
  const kdata = ohlcvData.value.map((d: any) => [d[1], d[2], d[3], d[4]]) // [open, close, low, high]
  const vdata = ohlcvData.value.map((d: any, i: number) => {
    const open = d[1] as number
    const close = d[2] as number
    return { value: d[5], itemStyle: { color: close >= open ? 'rgba(34,197,94,0.5)' : 'rgba(239,68,68,0.5)' } }
  })

  return {
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 10, top: 10, height: '62%' },
      { left: 60, right: 10, top: '78%', height: '15%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#334155' } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#334155' } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: '#64748b', fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { type: 'value', gridIndex: 1, axisLabel: { color: '#64748b', fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      {
        type: 'candlestick', name: 'K线',
        data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        itemStyle: { color: '#22c55e', color0: '#ef4444', borderColor: '#22c55e', borderColor0: '#ef4444' },
      },
      {
        type: 'bar', name: 'Volume',
        data: vdata, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      },
    ],
    tooltip: { trigger: 'axis' as const },
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 50, end: 100 },
    ],
  }
})

const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {}
  const times = minuteTicks.value.map(t => t.time)
  const prices = minuteTicks.value.map(t => t.price)
  const volumes = minuteTicks.value.map(t => t.volume)
  const isUp = prices.length > 0 && prices[prices.length - 1] >= prevClose.value
  const lineColor = isUp ? '#ef4444' : '#22c55e'

  return {
    backgroundColor: 'transparent',
    grid: { top: 20, right: 60, bottom: 40, left: 60 },
    xAxis: {
      type: 'category', data: times,
      axisLabel: { color: '#6b7280', fontSize: 10, interval: 30 },
      axisLine: { lineStyle: { color: '#374151' } },
    },
    yAxis: [
      {
        type: 'value', name: '价格',
        position: 'left',
        axisLabel: { color: '#6b7280', fontSize: 10 },
        splitLine: { lineStyle: { color: '#1f2937' } },
        min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
        max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
      },
      {
        type: 'value', name: '量',
        position: 'right',
        axisLabel: { color: '#6b7280', fontSize: 10, formatter: (v: number) => v >= 1e4 ? (v / 1e4).toFixed(1) + '万' : String(v) },
        splitLine: { show: false },
      }
    ],
    series: [
      {
        type: 'line', data: prices, yAxisIndex: 0,
        smooth: false, symbol: 'none',
        lineStyle: { color: lineColor, width: 1.5 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: lineColor === '#ef4444' ? 'rgba(239,68,68,0.3)' : 'rgba(34,197,94,0.3)' },
              { offset: 1, color: 'rgba(0,0,0,0)' }
            ]
          }
        },
      },
      {
        type: 'line', data: minuteTicks.value.map(t => t.avg_price), yAxisIndex: 0,
        smooth: true, symbol: 'none',
        lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' },
        name: '均价',
      },
      {
        type: 'bar', data: volumes, yAxisIndex: 1,
        itemStyle: { color: '#374151' },
        barWidth: 1,
      },
    ],
    tooltip: { trigger: 'axis' },
    markLine: prevClose.value > 0 ? {
      silent: true, symbol: 'none',
      lineStyle: { color: '#6b7280', type: 'dashed', width: 1 },
      data: [{ yAxis: prevClose.value, label: { formatter: `昨收 ${prevClose.value.toFixed(2)}`, color: '#6b7280', fontSize: 10 } }],
    } : undefined,
  }
})

onMounted(() => {
  const groupSym = ctx.getGroupSymbol(pg.groupId)
  if (groupSym && groupSym !== symbol.value) {
    symbol.value = groupSym
  }
  loadOHLCV(symbol.value)
})

onUnmounted(() => {
  stopMinutePolling()
  stopKlineRefresh()
})
</script>

<template>
  <div class="candlestick-panel">
    <div class="chart-header">
      <div class="header-left">
        <span class="symbol-display">{{ symbol }}</span>
        <div class="tab-btns">
          <button :class="{ active: activeTab === 'kline' }" class="tab-btn" @click="activeTab = 'kline'">K线</button>
          <button :class="{ active: activeTab === 'minute' }" class="tab-btn" @click="activeTab = 'minute'">分时</button>
        </div>
      </div>
      <div v-if="activeTab === 'kline'" class="interval-btns">
        <button v-for="i in ['1m','5m','15m','1h','1d','1w']" :key="i"
          :class="{ active: interval === i }" class="interval-btn"
          @click="interval = i">{{ i }}</button>
      </div>
    </div>
    <div class="chart-body">
      <div v-if="loading || minuteLoading" class="chart-fallback">加载中...</div>
      <VChart v-else-if="hasEcharts && activeTab === 'kline' && ohlcvData.length > 0" :key="symbol" :option="option" autoresize class="kline-chart" />
      <VChart v-else-if="hasEcharts && activeTab === 'minute'" :option="minuteChartOption" autoresize class="minute-chart" />
      <div v-else-if="activeTab === 'minute' && !minuteTicks.length" class="chart-fallback no-data">暂无分时数据</div>
      <div v-else class="chart-fallback">--</div>
    </div>
  </div>
</template>

<style scoped>
.candlestick-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--color-bg-panel);
}
.chart-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; border-bottom: 1px solid var(--color-border);
}
.header-left {
  display: flex; align-items: center; gap: 12px;
}
.symbol-display {
  font-size: var(--font-lg); font-weight: 700;
  color: var(--color-brand);
}
.tab-btns { display: flex; gap: 4px; }
.tab-btn {
  padding: 3px 12px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #9ca3af; font-size: 12px; cursor: pointer;
}
.tab-btn.active { background: #374151; color: #e5e7eb; border-color: #534ab7; }
.interval-btns { display: flex; gap: 2px; }
.interval-btn {
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-tertiary);
  border-radius: var(--radius-sm); cursor: pointer;
  font-size: var(--font-xs); font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}
.interval-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.interval-btn.active {
  background: var(--color-accent); color: #fff; border-color: var(--color-accent);
}
.chart-body { flex: 1; min-height: 0; padding: 8px; position: relative; }
.kline-chart { width: 100%; height: 100%; }
.minute-chart { width: 100%; height: 100%; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
.no-data { color: #6b7280; padding: 40px; text-align: center; }
</style>
