<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { marketUpColor, marketDownColor, marketChangeColor } from '@/lib/composables/useMarketColors'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { sma, ema, bb, macd, kdj, rsi, wr, createIndicatorCache } from '@/lib/composables/useIndicators'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Shared minute data cache from parent DockView
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())

const topOverlay = ref<'none' | 'ma' | 'bb'>('none')
const bottomMode = ref<'volume' | 'macd' | 'kdj' | 'rsi' | 'wr'>('volume')
const minuteBottomMode = ref<'volume' | 'macd' | 'kdj'>('volume')

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
const { name } = useStockName(symbol)

const WS_KEY = 'quantflow-watchlist'
function getWatchlist(): string[] {
  try {
    const saved = localStorage.getItem(WS_KEY)
    if (saved) { const arr = JSON.parse(saved); if (Array.isArray(arr)) return arr }
  } catch {}
  return []
}
function saveWatchlist(syms: string[]) {
  localStorage.setItem(WS_KEY, JSON.stringify(syms))
  window.dispatchEvent(new CustomEvent('watchlist-changed'))
}
const isInWatchlist = ref(false)
watch(symbol, () => {
  isInWatchlist.value = getWatchlist().includes(symbol.value)
}, { immediate: true })
function toggleWatchlist() {
  const list = getWatchlist()
  if (isInWatchlist.value) {
    saveWatchlist(list.filter(s => s !== symbol.value))
    isInWatchlist.value = false
  } else {
    list.push(symbol.value)
    saveWatchlist(list)
    isInWatchlist.value = true
  }
}
const interval = ref(props.params?.interval || '1d')
const ohlcvData = ref<(string | number)[][]>([])
const loading = ref(false)
const indicatorCache = createIndicatorCache()

// Tab state
const activeTab = ref<'kline' | 'minute' | 'multiDay'>('kline')

// Multi-day minute data
interface DayMinute {
  date: string
  ticks: MinuteTick[]
  prevClose: number
}
const multiDayData = ref<DayMinute[]>([])
const multiDayLoading = ref(false)
const selectedDayIndex = ref(0)

const selectedDayDate = computed(() => {
  if (multiDayData.value.length === 0) return ''
  return multiDayData.value[selectedDayIndex.value]?.date || ''
})

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
    // Lookback: minute intervals → 5 days, weekly → 450 days, daily → 365 days
    // 365d → ~250 trading days for daily MA60 (needs 60); 450d → ~64 weeks for weekly MA60.
    const iv = interval.value
    const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 450 : 365
    const start = end - lookbackDays * 86400
    const fqfactor = 'qfq'  // default to 前复权 for A-shares
    const result = await (window as any).go.main.App.FetchOHLCV(detectMarket(sym), sym, iv, fqfactor, start, end)
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

async function fetchMultiDayMinute() {
  const app = (window as any).go?.main?.App
  if (!app) return
  multiDayLoading.value = true
  try {
    const result = await app.GetMultiDayMinute(symbol.value, 3)
    const days: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    multiDayData.value = days.map((d: any) => ({
      date: d.date || '',
      ticks: (d.ticks || []).map((t: any) => ({
        time: t.time || '',
        price: t.price || 0,
        volume: t.volume || 0,
        avg_price: t.avg_price || 0,
      })),
      prevClose: d.prev_close || d.prevClose || (d.ticks?.[0]?.price || 0),
    }))
    selectedDayIndex.value = 0
  } catch(e) {
    console.error('[Candlestick] multi-day minute:', e)
    multiDayData.value = []
  } finally {
    multiDayLoading.value = false
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
  }, 5000)
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
  } else if (tab === 'multiDay') {
    fetchMultiDayMinute()
  } else {
    stopMinutePolling()
  }
})

// Watch symbol change — reload minute data
watch(() => symbol.value, (newSymbol, oldSymbol) => {
  if (oldSymbol && newSymbol !== oldSymbol) {
    indicatorCache.clear()
  }
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
  if (ohlcvData.value.length === 0) return {} as ECBasicOption
  const dates = ohlcvData.value.map((d: any) => d[0])
  const kdata = ohlcvData.value.map((d: any) => [d[1], d[2], d[3], d[4]])
  const close = ohlcvData.value.map((d: any) => d[2] as number)
  const high = ohlcvData.value.map((d: any) => d[4] as number)
  const low = ohlcvData.value.map((d: any) => d[3] as number)
  const vdata = ohlcvData.value.map((d: any, i: number) => {
    const open = d[1] as number
    const cl = d[2] as number
    return { value: (d[5] as number) / 10000, itemStyle: { color: cl >= open ? upColor() : downColor() } }
  })

  const cacheKey = `${symbol.value}-${interval.value}-${ohlcvData.value.length}-${topOverlay.value}-${bottomMode.value}`
  const theme = useChartTheme()
  const gridH = '52%'
  const bottomTop = '68%'
  const bottomH = '26%'

  const series: any[] = [
    {
      type: 'candlestick', name: 'K线',
      data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
      itemStyle: { color: upColor(), color0: downColor(), borderColor: upColor(), borderColor0: downColor() },
    },
  ]

  if (topOverlay.value === 'ma') {
    ;[5, 10, 20, 60].forEach(p => {
      series.push({
        type: 'line', name: `MA${p}`, data: indicatorCache.getCached(`sma-${cacheKey}-${p}`, () => sma(close, p)),
        gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        symbol: 'none', lineStyle: { width: 1 },
      })
    })
  } else if (topOverlay.value === 'bb') {
    const b = indicatorCache.getCached(`bb-${cacheKey}-20-2`, () => bb(close, 20, 2))
    series.push({ type: 'line', name: 'BB上轨', data: b.upper, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#4caf50' } })
    series.push({ type: 'line', name: 'BB中轨', data: b.middle, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } })
    series.push({ type: 'line', name: 'BB下轨', data: b.lower, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#4caf50' } })
  }

  if (bottomMode.value === 'volume') {
    series.push({ type: 'bar', name: 'Volume', data: vdata, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1 })
  } else if (bottomMode.value === 'macd') {
    const m = indicatorCache.getCached(`macd-${cacheKey}`, () => macd(close))
    series.push(
      { type: 'line', name: 'DIF', data: m.dif, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'DEA', data: m.dea, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'bar', name: 'MACD', data: m.hist.map((v: number | null) => {
        if (v === null) return null
        return { value: v, itemStyle: { color: v >= 0 ? '#ef5350' : '#66bb6a' } }
      }), gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1 },
    )
  } else if (bottomMode.value === 'kdj') {
    const kd = indicatorCache.getCached(`kdj-${cacheKey}`, () => kdj(close, high, low))
    series.push(
      { type: 'line', name: 'K', data: kd.k, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'D', data: kd.d, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'line', name: 'J', data: kd.j, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ab47bc' } },
    )
  } else if (bottomMode.value === 'rsi') {
    const r = indicatorCache.getCached(`rsi-${cacheKey}-14`, () => rsi(close, 14))
    series.push({
      type: 'line', name: 'RSI', data: r, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#ec407a' },
      markLine: { silent: true, symbol: 'none', data: [
        { yAxis: 70, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
        { yAxis: 30, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
      ]},
    })
  } else if (bottomMode.value === 'wr') {
    const w = indicatorCache.getCached(`wr-${cacheKey}-14`, () => wr(close, high, low, 14))
    series.push({
      type: 'line', name: 'WR', data: w, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#42a5f5' },
      markLine: { silent: true, symbol: 'none', data: [
        { yAxis: -20, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
        { yAxis: -80, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
      ]},
    })
  }

  let bottomYAxis: any = { type: 'value', gridIndex: 1, axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false } }
  if (bottomMode.value === 'volume') {
    bottomYAxis = { ...bottomYAxis, axisLabel: { ...bottomYAxis.axisLabel, formatter: (v: number) => v >= 1 ? v.toFixed(1) + '万' : String(v) } }
  } else if (bottomMode.value === 'kdj' || bottomMode.value === 'rsi') {
    bottomYAxis = { ...bottomYAxis, min: 0, max: 100 }
  } else if (bottomMode.value === 'wr') {
    bottomYAxis = { ...bottomYAxis, min: -100, max: 0 }
  }

  return {
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 10, top: 10, height: gridH },
      { left: 60, right: 10, top: bottomTop, height: bottomH },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { ...bottomYAxis, scale: true },
    ],
    series,
    tooltip: { trigger: 'axis' as const },
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 0, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], bottom: 0, height: 20 },
    ],
  } as ECBasicOption
})

const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {} as ECBasicOption
  const times = minuteTicks.value.map(t => t.time)
  const prices = minuteTicks.value.map(t => t.price)
  const volumes = minuteTicks.value.map(t => t.volume / 10000)
  const isUp = prices.length > 0 && prices[prices.length - 1] >= prevClose.value
  const lineColor = isUp ? upColor() : downColor()
  const theme = useChartTheme()

  const grid: any[] = []
  const xAxis: any[] = []
  const yAxis: any[] = []
  const series: any[] = []

  // Price grid (shared across all modes)
  const priceBot = minuteBottomMode.value === 'volume' ? '78%' : '55%'
  grid.push({ left: 60, right: 20, top: 10, height: minuteBottomMode.value === 'volume' ? '62%' : '40%' })
  xAxis.push({ type: 'category', data: times, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } }, axisTick: { show: false } })
  yAxis.push({ type: 'value', gridIndex: 0, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { lineStyle: { color: theme.bgColor } },
    min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
    max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
  })
  series.push(
    { type: 'line', name: '价格', data: prices, xAxisIndex: 0, yAxisIndex: 0, smooth: false, symbol: 'none', lineStyle: { color: lineColor, width: 1.5 },
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: isUp ? upColor() + '40' : downColor() + '40' },
        { offset: 1, color: 'rgba(0,0,0,0)' },
      ]}},
      markLine: prevClose.value > 0 ? { silent: true, symbol: 'none', lineStyle: { color: theme.axisColor, type: 'dashed', width: 1 }, data: [{ yAxis: prevClose.value, label: { formatter: `昨收 ${prevClose.value.toFixed(2)}`, color: theme.axisColor, fontSize: 10 } }] } : undefined,
    },
    { type: 'line', name: '均价', data: minuteTicks.value.map(t => t.avg_price), xAxisIndex: 0, yAxisIndex: 0, smooth: true, symbol: 'none', lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' } },
  )

  // Bottom grid: volume or MACD/KDJ
  const botGridIdx = 1
  const botAxisIdx = 1
  grid.push({ left: 60, right: 20, top: priceBot, height: minuteBottomMode.value === 'volume' ? '15%' : '35%' })
  xAxis.push({ type: 'category', data: times, gridIndex: botGridIdx, axisLabel: { color: theme.axisColor, fontSize: 10, interval: 30 }, axisLine: { lineStyle: { color: theme.splitColor } } })

  if (minuteBottomMode.value === 'volume') {
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => v >= 1 ? v.toFixed(1) + '万' : String(v) }, splitLine: { show: false } })
    series.push({ type: 'bar', name: '成交量', data: volumes, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, itemStyle: { color: theme.splitColor }, barWidth: 1 })
  } else if (minuteBottomMode.value === 'macd') {
    const m = macd(prices)
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true })
    series.push(
      { type: 'line', name: 'DIF', data: m.dif, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'DEA', data: m.dea, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'bar', name: 'MACD', data: m.hist.map((v: number | null) => v === null ? null : { value: v, itemStyle: { color: v >= 0 ? '#ef5350' : '#66bb6a' } }), xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx },
    )
  } else if (minuteBottomMode.value === 'kdj') {
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
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true })
    series.push(
      { type: 'line', name: 'K', data: kd.k, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'D', data: kd.d, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'line', name: 'J', data: kd.j, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ab47bc' } },
    )
  }

  return {
    animation: false, animationDurationUpdate: 0, animationEasingUpdate: 'linear',
    backgroundColor: 'transparent', grid, xAxis, yAxis, series,
    tooltip: { trigger: 'axis' },
  } as ECBasicOption
})

const multiDayChartOption = computed(() => {
  const day = multiDayData.value[selectedDayIndex.value]
  if (!day || !day.ticks.length) return {} as ECBasicOption
  const ticks = day.ticks
  const times = ticks.map(t => t.time)
  const prices = ticks.map(t => t.price)
  const volumes = ticks.map(t => t.volume / 10000)
  const isUp = prices.length > 0 && prices[prices.length - 1] >= day.prevClose
  const lineColor = isUp ? upColor() : downColor()
  const theme = useChartTheme()

  return {
    animation: false,
    animationDurationUpdate: 0,
    animationEasingUpdate: 'linear',
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 20, top: 10, height: '62%' },
      { left: 60, right: 20, top: '78%', height: '15%' },
    ],
    xAxis: [
      {
        type: 'category', data: times, gridIndex: 0,
        axisLabel: { show: false },
        axisLine: { lineStyle: { color: theme.splitColor } },
        axisTick: { show: false },
      },
      {
        type: 'category', data: times, gridIndex: 1,
        axisLabel: { color: theme.axisColor, fontSize: 10, interval: 30 },
        axisLine: { lineStyle: { color: theme.splitColor } },
      },
    ],
    yAxis: [
      {
        type: 'value', gridIndex: 0, position: 'left',
        axisLabel: { color: theme.axisColor, fontSize: 10 },
        splitLine: { lineStyle: { color: theme.bgColor } },
        min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
        max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
      },
      {
        type: 'value', gridIndex: 1, position: 'left',
        axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => v >= 1 ? v.toFixed(1) + '万' : String(v) },
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
        markLine: day.prevClose > 0 ? {
          silent: true, symbol: 'none',
          lineStyle: { color: theme.axisColor, type: 'dashed', width: 1 },
          data: [{ yAxis: day.prevClose, label: { formatter: `昨收 ${day.prevClose.toFixed(2)}`, color: theme.axisColor, fontSize: 10 } }],
        } : undefined,
      },
      {
        type: 'line', name: '均价', data: ticks.map(t => t.avg_price),
        xAxisIndex: 0, yAxisIndex: 0,
        smooth: true, symbol: 'none',
        lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' },
      },
      {
        type: 'bar', name: '成交量', data: volumes,
        xAxisIndex: 1, yAxisIndex: 1,
        itemStyle: { color: theme.splitColor },
        barWidth: 1,
      },
    ],
    tooltip: { trigger: 'axis' },
  } as ECBasicOption
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
        <span class="symbol-display">{{ symbol }} {{ name }}</span>
        <button
          class="watchlist-btn"
          :class="{ inList: isInWatchlist }"
          @click="toggleWatchlist"
        >{{ isInWatchlist ? '取消自选' : '加入自选' }}</button>
        <div class="tab-btns">
          <button :class="{ active: activeTab === 'kline' }" class="tab-btn" @click="activeTab = 'kline'">{{ $t('kline.kline') }}</button>
          <button :class="{ active: activeTab === 'minute' }" class="tab-btn" @click="activeTab = 'minute'">{{ $t('kline.minute') }}</button>
          <button :class="{ active: activeTab === 'multiDay' }" class="tab-btn" @click="activeTab = 'multiDay'">{{ $t('kline.multi_day_minute') }}</button>
        </div>
      </div>
      <div v-if="activeTab === 'kline'" class="interval-btns">
        <button v-for="i in ['1m','5m','15m','30m','1h','1d','1w']" :key="i"
          :class="{ active: interval === i }" class="interval-btn"
          @click="interval = i">{{ i }}</button>
      </div>
    </div>
    <div v-if="activeTab === 'kline'" class="indicator-bar">
      <div class="indicator-group">
        <span class="indicator-label">{{ $t('kline.overlay') }}</span>
        <button :class="{ active: topOverlay === 'none' }" class="indicator-btn" @click="topOverlay = 'none'">无</button>
        <button :class="{ active: topOverlay === 'ma' }" class="indicator-btn" @click="topOverlay = 'ma'">MA</button>
        <button :class="{ active: topOverlay === 'bb' }" class="indicator-btn" @click="topOverlay = 'bb'">{{ $t('kline.bb') }}</button>
      </div>
      <div class="indicator-group">
        <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
        <button :class="{ active: bottomMode === 'volume' }" class="indicator-btn" @click="bottomMode = 'volume'">{{ $t('kline.volume') }}</button>
        <button :class="{ active: bottomMode === 'macd' }" class="indicator-btn" @click="bottomMode = 'macd'">MACD</button>
        <button :class="{ active: bottomMode === 'kdj' }" class="indicator-btn" @click="bottomMode = 'kdj'">KDJ</button>
        <button :class="{ active: bottomMode === 'rsi' }" class="indicator-btn" @click="bottomMode = 'rsi'">RSI</button>
        <button :class="{ active: bottomMode === 'wr' }" class="indicator-btn" @click="bottomMode = 'wr'">WR</button>
      </div>
    </div>
    <div v-if="activeTab === 'minute' || activeTab === 'multiDay'" class="indicator-bar">
      <div class="indicator-group">
        <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
        <button :class="{ active: minuteBottomMode === 'volume' }" class="indicator-btn" @click="minuteBottomMode = 'volume'">{{ $t('kline.volume') }}</button>
        <button :class="{ active: minuteBottomMode === 'macd' }" class="indicator-btn" @click="minuteBottomMode = 'macd'">MACD</button>
        <button :class="{ active: minuteBottomMode === 'kdj' }" class="indicator-btn" @click="minuteBottomMode = 'kdj'">KDJ</button>
      </div>
    </div>
    <div class="chart-body">
      <!-- Only show loading overlay on initial load (no data yet); skip during polling to avoid flashing the chart -->
      <div v-if="(activeTab === 'kline' && loading && !ohlcvData.length) || (activeTab === 'minute' && minuteLoading && !minuteTicks.length) || (activeTab === 'multiDay' && multiDayLoading && !multiDayData.length)" class="chart-fallback">{{ $t('common.loading') }}</div>
      <template v-else-if="activeTab === 'multiDay'">
        <div v-if="multiDayData.length === 0" class="chart-fallback no-data">{{ $t('kline.no_minute_data') }}</div>
        <div v-else class="multi-day-chart-wrapper">
          <div class="day-selector">
            <select v-model="selectedDayIndex" class="day-select">
              <option v-for="(d, i) in multiDayData" :key="i" :value="i">{{ d.date }}</option>
            </select>
          </div>
          <KlineChart :symbol="`${symbol}-multi`" :option="multiDayChartOption" :loading="multiDayLoading && !multiDayData.length" />
        </div>
      </template>
      <KlineChart v-else-if="activeTab === 'kline' && ohlcvData.length > 0" :symbol="symbol" :option="option" :loading="loading && !ohlcvData.length" />
      <template v-else-if="activeTab === 'minute'">
        <KlineChart v-if="minuteTicks.length" :symbol="`${symbol}-minute`" :option="minuteChartOption" :loading="minuteLoading && !minuteTicks.length" />
        <div v-else class="chart-fallback no-data">{{ $t('kline.no_minute_data') }}</div>
      </template>
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
.watchlist-btn {
  padding: 2px 10px; border: 1px solid var(--color-accent); border-radius: 4px;
  background: transparent; color: var(--color-accent); cursor: pointer;
  font-size: 11px; white-space: nowrap; transition: all var(--transition-fast);
}
.watchlist-btn:hover { background: var(--color-accent); color: #fff; }
.watchlist-btn.inList { border-color: var(--color-down); color: var(--color-down); }
.watchlist-btn.inList:hover { background: var(--color-down); color: #fff; }
.tab-btns { display: flex; gap: 4px; }
.tab-btn {
  padding: 3px 12px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-secondary); font-size: 12px; cursor: pointer;
}
.tab-btn.active { background: var(--color-border-strong); color: var(--color-text-primary); border-color: var(--color-accent); }
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
  background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent);
}
.chart-body { flex: 1; min-height: 0; padding: 8px; position: relative; }
.kline-chart { width: 100%; height: 100%; }
.minute-chart { width: 100%; height: 100%; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
.no-data { color: var(--color-text-tertiary); padding: 40px; text-align: center; }

/* Multi-Day Minute */
.multi-day-chart-wrapper {
  width: 100%; height: 100%; display: flex; flex-direction: column;
}
.day-selector {
  display: flex; justify-content: flex-end; padding: 4px 0; margin-bottom: 4px;
}
.day-select {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 12px;
  cursor: pointer;
}
.minute-chart { width: 100%; flex: 1; }
.indicator-bar {
  display: flex; gap: 16px; align-items: center;
  padding: 4px 10px; border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
}
.indicator-group { display: flex; align-items: center; gap: 4px; }
.indicator-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-right: 4px; }
.indicator-btn {
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-tertiary);
  border-radius: var(--radius-sm); cursor: pointer;
  font-size: var(--font-xs); font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}
.indicator-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.indicator-btn.active {
  background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent);
}
</style>
