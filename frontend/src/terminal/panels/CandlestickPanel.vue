<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CandlestickChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import * as echarts from 'echarts'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { marketUpColor, marketDownColor, marketChangeColor } from '@/lib/composables/useMarketColors'

use([CandlestickChart, BarChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Shared minute data cache from parent DockView
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())

const hasEcharts = computed(() => !!(echarts && VChart))

// Color scheme: per-market (CN 红涨绿跌, others 绿涨红跌)
function upColor() { return marketUpColor(symbol.value) }
function downColor() { return marketDownColor(symbol.value) }

// Market-aware trading hours check (polling guard).
function isTradingHours(): boolean {
  const now = new Date()
  const day = now.getDay()
  if (day === 0 || day === 6) return false
  const market = detectMarket(symbol.value)
  if (market === 'CRYPTO') return true
  if (market === 'HK') {
    // HKEX: 09:30-12:00, 13:00-16:00 (Mon-Fri)
    const h = now.getHours()
    const m = now.getMinutes()
    const t = h * 60 + m
    return (t >= 9 * 60 + 30 && t <= 12 * 60) || (t >= 13 * 60 && t <= 16 * 60)
  }
  if (market === 'US') {
    // NYSE/Nasdaq: 09:30-16:00 ET ≈ 13:30-21:00 UTC (DST + standard range)
    const ut = now.getUTCHours() * 60 + now.getUTCMinutes()
    return ut >= 13 * 60 + 30 && ut <= 21 * 60
  }
  // CN default: 09:30-11:30, 13:00-15:00 (Mon-Fri)
  const h = now.getHours()
  const m = now.getMinutes()
  const t = h * 60 + m
  return (t >= 9 * 60 + 30 && t <= 11 * 60 + 30) || (t >= 13 * 60 && t <= 15 * 60)
}

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

function getTodayDateString(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function parseMinuteTimeToUnix(timeStr: string): number {
  // timeStr like "09:30", combine with today's date
  const today = getTodayDateString()
  const d = new Date(`${today}T${timeStr}:00+08:00`)
  return Math.floor(d.getTime() / 1000)
}

async function loadOHLCV(sym: string) {
  // TODO: move to store
  loading.value = true
  try {
    const end = Math.floor(Date.now() / 1000)
    // Lookback: minute intervals → 5 days, daily → 90 days, weekly → 180 days
    const iv = interval.value
    const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 180 : 90
    const start = end - lookbackDays * 86400
    const result = await (window as any).go.main.App.FetchOHLCV(detectMarket(sym), sym, iv, start, end)
    const bars = Array.isArray(result) ? result[0] : result
    const isIntraday = ['1m','5m','15m','30m','1h'].includes(iv)
    ohlcvData.value = (bars as any[]).map((b: any) => {
      const rawDate = b.date || b.Date || ''
      const d = new Date(rawDate)
      const date = isIntraday
        ? d.toISOString().slice(0, 16).replace('T', ' ')  // "2026-06-25 09:35"
        : d.toISOString().slice(0, 10)                     // "2026-06-25"
      return [date, b.open ?? b.Open ?? 0, b.close ?? b.Close ?? 0, b.low ?? b.Low ?? 0, b.high ?? b.High ?? 0, b.volume ?? b.Volume ?? 0]
    })
  } catch(e) {
    console.error('[Candlestick]', e)
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
    // Calculate sinceTimestamp from last tick
    const lastTick = minuteTicks.value.length > 0
      ? minuteTicks.value[minuteTicks.value.length - 1]
      : null
    const sinceTimestamp = lastTick
      ? parseMinuteTimeToUnix(lastTick.time)
      : 0

    const result = await app.GetMinuteLine(symbol.value, sinceTimestamp)  // TODO: move to store
    const ticks: MinuteTick[] = Array.isArray(result) ? result[0] : result
    if (!Array.isArray(ticks) || ticks.length === 0) {
      return
    }

    if (sinceTimestamp === 0) {
      // First load: full replacement
      minuteTicks.value = ticks
    } else {
      // Incremental update: deduplicate and merge
      const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
      for (const t of ticks) {
        existing.set(t.time, t)
      }
      minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
    }

    if (prevClose.value === 0 && minuteTicks.value.length > 0) {
      prevClose.value = minuteTicks.value[0].price
    }

    // Update shared cache
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  } catch(e) {
    console.error('[Candlestick] minute fetch:', e)
  } finally {
    minuteLoading.value = false
  }
}

function startMinutePolling() {
  stopMinutePolling()
  // Always load once so the user can see today's chart, even after close.
  loadMinuteLine()
  // Only auto-refresh during trading hours.
  if (!isTradingHours()) return
  minuteTimer = setInterval(() => {
    if (!isTradingHours()) {
      stopMinutePolling()
      return
    }
    loadMinuteLine()
  }, 10000)
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
watch(() => symbol.value, (newSymbol, oldSymbol) => {
  // Save old symbol data to shared cache
  if (oldSymbol) {
    const cacheKey = `${oldSymbol}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }

  // Try to restore new symbol data from shared cache
  const cacheKey = `${newSymbol}:${getTodayDateString()}`
  const cached = minuteDataCache.get(cacheKey)
  if (cached && cached.length > 0) {
    minuteTicks.value = cached
    prevClose.value = cached[0].price
  } else {
    minuteTicks.value = []
    prevClose.value = 0
  }

  if (activeTab.value === 'minute') {
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
    return { value: d[5], itemStyle: { color: close >= open ? upColor() : downColor() } }
  })

  return {
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 10, top: 10, height: '62%' },
      { left: 60, right: 10, top: '78%', height: '15%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: 'var(--color-border-strong)' } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: 'var(--color-border-strong)' } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { type: 'value', gridIndex: 1, axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      {
        type: 'candlestick', name: 'K线',
        data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        itemStyle: { color: upColor(), color0: downColor(), borderColor: upColor(), borderColor0: downColor() },
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
  const lineColor = isUp ? upColor() : downColor()

  return {
    animation: false,
    animationDurationUpdate: 0,
    animationEasingUpdate: 'linear',
    backgroundColor: 'transparent',
    // Two separate grids: price chart on top (62%), volume bars below (15%)
    grid: [
      { left: 60, right: 20, top: 10, height: '62%' },
      { left: 60, right: 20, top: '78%', height: '15%' },
    ],
    xAxis: [
      {
        type: 'category', data: times, gridIndex: 0,
        axisLabel: { show: false },
        axisLine: { lineStyle: { color: 'var(--color-border-strong)' } },
        axisTick: { show: false },
      },
      {
        type: 'category', data: times, gridIndex: 1,
        axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10, interval: 30 },
        axisLine: { lineStyle: { color: 'var(--color-border-strong)' } },
      },
    ],
    yAxis: [
      {
        type: 'value', gridIndex: 0, position: 'left',
        axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10 },
        splitLine: { lineStyle: { color: 'var(--color-bg-elevated)' } },
        min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
        max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
      },
      {
        type: 'value', gridIndex: 1, position: 'left',
        axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10, formatter: (v: number) => v >= 1e4 ? (v / 1e4).toFixed(1) + '万' : String(v) },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        type: 'line', name: '价格', data: prices,
        xAxisIndex: 0, yAxisIndex: 0,
        smooth: false, symbol: 'none',
        lineStyle: { color: lineColor, width: 1.5 },
        areaStyle: {
          color: {
            type: 'linear', x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: isUp ? upColor() + '40' : downColor() + '40' },
              { offset: 1, color: 'rgba(0,0,0,0)' }
            ]
          }
        },
        markLine: prevClose.value > 0 ? {
          silent: true, symbol: 'none',
          lineStyle: { color: 'var(--color-text-tertiary)', type: 'dashed', width: 1 },
          data: [{ yAxis: prevClose.value, label: { formatter: `昨收 ${prevClose.value.toFixed(2)}`, color: 'var(--color-text-tertiary)', fontSize: 10 } }],
        } : undefined,
      },
      {
        type: 'line', name: '均价', data: minuteTicks.value.map(t => t.avg_price),
        xAxisIndex: 0, yAxisIndex: 0,
        smooth: true, symbol: 'none',
        lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' },
      },
      {
        type: 'bar', name: '成交量', data: volumes,
        xAxisIndex: 1, yAxisIndex: 1,
        itemStyle: { color: 'var(--color-border-strong)' },
        barWidth: 1,
      },
    ],
    tooltip: { trigger: 'axis' },
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
  // Save current data to shared cache so data survives component destruction
  if (symbol.value && minuteTicks.value.length > 0) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }
})
</script>

<template>
  <div class="candlestick-panel">
    <div class="chart-header">
      <div class="header-left">
        <span class="symbol-display">{{ symbol }}</span>
        <div class="tab-btns">
          <button :class="{ active: activeTab === 'kline' }" class="tab-btn" @click="activeTab = 'kline'">{{ $t('kline.kline') }}</button>
          <button :class="{ active: activeTab === 'minute' }" class="tab-btn" @click="activeTab = 'minute'">{{ $t('kline.minute') }}</button>
        </div>
      </div>
      <div v-if="activeTab === 'kline'" class="interval-btns">
        <button v-for="i in ['1m','5m','15m','30m','1h','1d','1w']" :key="i"
          :class="{ active: interval === i }" class="interval-btn"
          @click="interval = i">{{ i }}</button>
      </div>
    </div>
    <div class="chart-body">
      <div v-if="loading || minuteLoading" class="chart-fallback">{{ $t('common.loading') }}</div>
      <VChart v-else-if="hasEcharts && activeTab === 'kline' && ohlcvData.length > 0" :key="symbol" :option="option" autoresize class="kline-chart" />
      <VChart v-else-if="hasEcharts && activeTab === 'minute'" :option="minuteChartOption" :update-options="{ notMerge: false }" autoresize class="minute-chart" />
      <div v-else-if="activeTab === 'minute' && !minuteTicks.length" class="chart-fallback no-data">{{ $t('kline.no_minute_data') }}</div>
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
  padding: 3px 12px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-secondary); font-size: 12px; cursor: pointer;
}
.tab-btn.active { background: var(--color-border-strong); color: var(--color-text-primary); border-color: #534ab7; }
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
.no-data { color: var(--color-text-tertiary); padding: 40px; text-align: center; }
</style>
