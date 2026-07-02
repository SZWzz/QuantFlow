<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, inject, nextTick, reactive } from 'vue'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import InfoBar from '@/terminal/components/panel/InfoBar.vue'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { DrawingController } from '@/lib/chart/DrawingController'
import { Crosshair } from '@/lib/chart/Crosshair'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'
import { buildKlineOption, buildMinuteOption, buildMultiDayOption, type KlineDataItem } from '@/lib/buildChartOption'
import { useWailsApp, type OHLCVBar, type MultiDayMinute, type QuoteData } from '@/lib/composables/useWailsApp'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { detectLimitUpDown } from '@/lib/chart/EventMarker'
import type { EventMarker } from '@/lib/chart/EventMarker'
import { useWebSocket, type KlineUpdate } from '@/lib/composables/useWebSocket'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Shared minute data cache from parent DockView
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())

const topOverlay = ref<'none' | 'ma' | 'bb' | 'sar' | 'ema'>('none')
const bottomMode = ref<'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv'>('volume')
const minuteBottomMode = ref<'volume' | 'macd' | 'kdj'>('volume')

const klineChartRef = ref<any>(null)
const drawingCanvasRef = ref<HTMLCanvasElement | null>(null)
const drawingMode = ref(false)
const drawingColor = ref('#58a6ff')
let dc: DrawingController | null = null
let crosshair: Crosshair | null = null
const crosshairCanvasRef = ref<HTMLCanvasElement | null>(null)

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
const intervals = ['1m', '5m', '15m', '30m', '1h', '1d', '1w'] as const
const ohlcvData = ref<KlineDataItem[]>([])
const loading = ref(false)
const indicatorCache = createIndicatorCache()
const errorMsg = ref('')
let loadSeq = 0
const theme = useChartTheme()

// Tab state
const activeTab = ref<'kline' | 'minute' | 'multiDay'>('kline')

// ── Depth sidebar (minute tab) ──
const showDepth = ref(false)
const depthData = ref<{ bids: {price:number;size:number}[]; asks: {price:number;size:number}[] } | null>(null)
const depthLoading = ref(false)
const depthSimulated = ref(false)
const depthPrice = ref(0)
const depthChange = ref(0)
const depthChangePct = ref(0)

const depthMaxSize = computed(() => {
  if (!depthData.value) return 1
  const all = [...depthData.value.bids, ...depthData.value.asks]
  return Math.max(...all.map(l => l.size), 1)
})

const quoteData = ref<QuoteData | null>(null)
const eventMarkers = ref<EventMarker[]>([])
const indexOverlaySymbol = ref('')
const indexOverlayData = ref<any>(null)

async function loadQuote() {
  const app = useWailsApp()
  if (!app) return
  try {
    const [snap] = await app.GetQuote(detectMarket(symbol.value), symbol.value)
    quoteData.value = {
      ...snap,
      price: snap.last,
      change_percent: snap.change_pct ?? snap.changePct,
      market_cap: snap.marketCap,
      pe_ratio: snap.pe_ratio ?? snap.pe,
    }
  } catch (e) {
    console.error('[Candlestick] loadQuote:', e)
  }
}

function formatSize(size: number): string {
  if (size >= 10000) return (size / 10000).toFixed(1) + '万'
  return size.toFixed(0)
}

function barWidth(size: number): string {
  return ((size / depthMaxSize.value) * 100).toFixed(0) + '%'
}

async function loadDepth() {
  const app = (window as any).go?.main?.App
  if (!app) return
  depthLoading.value = true
  try {
    const mkt = detectMarket(symbol.value)
    const [quoteResult, depthResult] = await Promise.all([
      app.GetQuote(mkt, symbol.value).catch(() => null),
      app.GetDepth(mkt, symbol.value).catch(() => null),
    ])
    const snapshot = Array.isArray(quoteResult) ? quoteResult[0] : quoteResult
    if (snapshot) {
      depthPrice.value = snapshot.last || 0
      depthChange.value = snapshot.change || 0
      depthChangePct.value = snapshot.change_pct || snapshot.changePct || 0
    }
    if (depthResult && depthResult.bids?.length > 0) {
      depthData.value = {
        bids: depthResult.bids.map((l: any) => ({ price: l.price, size: l.size })),
        asks: depthResult.asks.map((l: any) => ({ price: l.price, size: l.size })),
      }
      depthSimulated.value = false
    } else if (snapshot?.bid > 0 && snapshot?.ask > 0) {
      const bids: {price:number;size:number}[] = []
      const asks: {price:number;size:number}[] = []
      const step = (snapshot.ask - snapshot.bid) / 5 || 0.02
      for (let i = 0; i < 5; i++) {
        bids.push({ price: +(snapshot.bid - i * step).toFixed(2), size: Math.round(1000 / (i + 1)) })
        asks.push({ price: +(snapshot.ask + i * step).toFixed(2), size: Math.round(800 / (i + 1)) })
      }
      depthData.value = { bids, asks }
      depthSimulated.value = true
    } else {
      depthData.value = null
    }
  } catch(e) {
    console.error('[Candlestick] depth:', e)
    depthData.value = null
  } finally {
    depthLoading.value = false
  }
}

function toggleDepth() {
  showDepth.value = !showDepth.value
  if (showDepth.value && !depthData.value && !depthLoading.value) {
    loadDepth()
  }
}

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
let quoteTimer: ReturnType<typeof setInterval> | null = null
const { connect: wsConnect, onMessage: wsOnMessage, connected: wsConnected } = useWebSocket()
let symbolSubCleanup: (() => void) | null = null

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

async function loadOHLCV(sym: string, incremental = false) {
  const seq = ++loadSeq
  loading.value = true
  try {
    const end = Math.floor(Date.now() / 1000)
    const iv = interval.value
    let start: number
    if (incremental && ohlcvData.value.length > 0) {
      const lastDate = ohlcvData.value[ohlcvData.value.length - 1].date
      start = Math.floor(new Date(lastDate.replace(' ', 'T')).getTime() / 1000)
    } else {
      const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 450 : 365
      start = end - lookbackDays * 86400
    }
    const app = useWailsApp()
    if (!app) { loading.value = false; return }
    const [rawBars] = await app.FetchOHLCV(detectMarket(sym), sym, iv, 'qfq', start, end)
    if (seq !== loadSeq) return
    const isIntraday = ['1m','5m','15m','30m','1h'].includes(iv)
    if (!rawBars?.length && !incremental) { ohlcvData.value = []; loading.value = false; return }
    if (incremental && ohlcvData.value.length > 0) {
      if (rawBars?.length) {
        const mergeMap = new Map(ohlcvData.value.map(b => [b.date, b]))
        for (const b of rawBars) {
          const rawDate = b.date || ''
          const d = new Date(rawDate)
          const date = isIntraday
            ? d.toISOString().slice(0, 16).replace('T', ' ')
            : d.toISOString().slice(0, 10)
          mergeMap.set(date, { date, open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume })
        }
        ohlcvData.value = Array.from(mergeMap.values()).sort((a, b) => a.date.localeCompare(b.date))
      }
    } else if (rawBars) {
      ohlcvData.value = rawBars.map(b => {
        const rawDate = b.date || ''
        const d = new Date(rawDate)
        const date = isIntraday
          ? d.toISOString().slice(0, 16).replace('T', ' ')
          : d.toISOString().slice(0, 10)
        return { date, open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume }
      })
    }
  } catch(e: any) {
    if (seq !== loadSeq) return
    console.error('[Candlestick]', e)
    errorMsg.value = 'K线数据加载失败: ' + (e.message || '未知错误')
    setTimeout(() => { errorMsg.value = '' }, 8000)
    if (!ohlcvData.value.length) ohlcvData.value = []
  }
  if (seq === loadSeq) loading.value = false
}

async function loadMinuteLine() {
  const seq = ++loadSeq
  const app = useWailsApp()
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
    if (seq !== loadSeq) return
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
  const app = useWailsApp()
  if (!app) return
  multiDayLoading.value = true
  try {
    const result = await app.GetMultiDayMinute(symbol.value, 3)
    const d = (result as MultiDayMinute)?.days || []
    multiDayData.value = d.map((day: { date: string; ticks: MinuteTick[] }) => ({
      date: day.date || '',
      ticks: (day.ticks || []).map((t: MinuteTick) => ({
        time: t.time || '',
        price: t.price || 0,
        volume: t.volume || 0,
        avg_price: t.avg_price || 0,
      })),
      prevClose: (day.ticks?.[0]?.price || 0),
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
  stopKlineRefresh()
  loadOHLCV(symbol.value, true)
  if (wsConnected.value) return
  klineTimer = window.setInterval(() => loadOHLCV(symbol.value, true), 30000)
}

function stopKlineRefresh() {
  if (klineTimer) { clearInterval(klineTimer); klineTimer = null }
}

function initWebSocket() {
  if (symbolSubCleanup) symbolSubCleanup()
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`
  wsConnect(wsUrl, [`kline:${symbol.value}:${interval.value}`])
  symbolSubCleanup = wsOnMessage(`kline:${symbol.value}:${interval.value}`, (update: KlineUpdate) => {
    mergeOHLCVUpdate(ohlcvData.value, update)
  })
}

function mergeOHLCVUpdate(data: KlineDataItem[], update: KlineUpdate) {
  if (!data.length) return
  const d = new Date(update.time * 1000)
  const isIntraday = ['1m', '5m', '15m', '30m', '1h'].includes(update.interval)
  const updateDate = isIntraday
    ? d.toISOString().slice(0, 16).replace('T', ' ')
    : d.toISOString().slice(0, 10)
  const last = data[data.length - 1]
  if (last.date === updateDate) {
    data[data.length - 1] = { ...last, open: update.open, high: update.high, low: update.low, close: update.close, volume: update.volume }
    ohlcvData.value = [...data]
  } else if (update.is_closed) {
    data.push({ date: updateDate, open: update.open, high: update.high, low: update.low, close: update.close, volume: update.volume })
    ohlcvData.value = [...data]
  }
}

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
    loadOHLCV(newSymbol)
  }
})

watch(symbol, () => {
  loadQuote()
  initWebSocket()
})

watch(ohlcvData, (data) => {
  eventMarkers.value = data.length >= 2 ? detectLimitUpDown(data) : []
}, { immediate: true })

watch(indexOverlaySymbol, async (sym) => {
  if (!sym) { indexOverlayData.value = null; return }
  try {
    const app = useWailsApp()
    if (!app) return
    const now = Math.floor(Date.now() / 1000)
    const [bars] = await app.FetchOHLCV('CN', sym, '1d', 'qfq', now - 365 * 86400, now)
    if (!bars?.length) return
    const stockMin = Math.min(...ohlcvData.value.map(d => d.low))
    const stockMax = Math.max(...ohlcvData.value.map(d => d.high))
    const idxCloses = bars.map((b: any) => b.close)
    const idxMin = Math.min(...idxCloses)
    const idxMax = Math.max(...idxCloses)
    const idxRange = idxMax - idxMin
    if (idxRange === 0) return
    const data = idxCloses.map((v: number, i: number) => [
      i, stockMin + ((v - idxMin) / idxRange) * (stockMax - stockMin),
    ])
    indexOverlayData.value = {
      symbol: sym,
      name: sym === '000001' ? '上证' : sym === '399001' ? '深成' : '创业板',
      data,
    }
  } catch (e) {
    console.error('[Candlestick] loadIndex:', e)
  }
})

// Regenerate data on interval change
watch(interval, () => {
  loadOHLCV(symbol.value)
  initWebSocket()
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
    if (showDepth.value) loadDepth()
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
  if (!ohlcvData.value.length) return {} as ECBasicOption
  return buildKlineOption(ohlcvData.value, topOverlay.value, bottomMode.value, theme, indicatorCache, symbol.value, interval.value, eventMarkers.value, indexOverlayData.value)
})

const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {} as ECBasicOption
  return buildMinuteOption(minuteTicks.value, prevClose.value, minuteBottomMode.value, theme, indicatorCache, symbol.value)
})

const multiDayChartOption = computed(() => {
  const day = multiDayData.value[selectedDayIndex.value]
  if (!day || !day.ticks.length) return {} as ECBasicOption
  return buildMultiDayOption(day.ticks, day.prevClose, theme, symbol.value)
})

function onKeyDown(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return

  switch (e.key) {
    case 'ArrowLeft':
      e.preventDefault()
      const idx = intervals.indexOf(interval as any)
      if (idx > 0) interval.value = intervals[idx - 1]
      break
    case 'ArrowRight':
      e.preventDefault()
      const idx2 = intervals.indexOf(interval as any)
      if (idx2 < intervals.length - 1) interval.value = intervals[idx2 + 1]
      break
    case 'g':
    case 'G':
      e.preventDefault()
      jumpToDate()
      break
    case 'Escape':
      if (drawingMode.value) {
        drawingMode.value = false
        dc?.setMode('cursor')
      }
      break
    case 'Delete':
    case 'Backspace':
      if (drawingMode.value) {
        dc?.clearAll()
        e.preventDefault()
      }
      break
    case 'D':
      if (e.shiftKey) {
        e.preventDefault()
        toggleDrawingMode()
      }
      break
  }
}

function onDrawingMouseDown(e: MouseEvent) { dc?.onMouseDown(e) }
function onDrawingMouseMove(e: MouseEvent) { dc?.onMouseMove(e) }
function onDrawingMouseUp(e: MouseEvent) { dc?.onMouseUp(e) }

function toggleDrawingMode() {
  drawingMode.value = !drawingMode.value
  dc?.setMode(drawingMode.value ? 'trendline' : 'cursor')
}

function jumpToDate() {
  const dateStr = prompt('跳转到日期 (YYYY-MM-DD):')
  if (!dateStr) return
  const target = new Date(dateStr).getTime()
  if (isNaN(target)) return

  const echarts = klineChartRef.value?.getEchartsInstance?.()
  if (!echarts) return

  const timestamps = ohlcvData.value.map(d => d[0])
  let bestIdx = 0
  let bestDiff = Infinity
  for (let i = 0; i < timestamps.length; i++) {
    const diff = Math.abs(timestamps[i] - target / 1000)
    if (diff < bestDiff) {
      bestDiff = diff
      bestIdx = i
    }
  }

  const totalLen = ohlcvData.value.length
  if (totalLen === 0) return
  const center = bestIdx / totalLen
  const range = 0.15
  echarts.dispatchAction({
    type: 'dataZoom',
    start: Math.max(0, (center - range) * 100),
    end: Math.min(100, (center + range) * 100),
  })
}

const contextMenu = reactive({ visible: false, x: 0, y: 0 })

function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  contextMenu.x = e.clientX
  contextMenu.y = e.clientY
  contextMenu.visible = true
}

function closeContextMenu() {
  contextMenu.visible = false
}

function switchInterval(s: string) {
  interval.value = s
  closeContextMenu()
}

function switchOverlay(o: string) {
  topOverlay.value = o as any
  closeContextMenu()
}

watch(option, () => {
  nextTick(() => dc?.render())
})

watch(drawingMode, (mode) => {
  if (mode) crosshair?.hide()
})

onMounted(() => {
  const groupSym = ctx.getGroupSymbol(pg.groupId)
  if (groupSym && groupSym !== symbol.value) {
    symbol.value = groupSym
  }
  loadOHLCV(symbol.value)
  loadQuote()
  initWebSocket()
  quoteTimer = setInterval(loadQuote, 30000)

  // DrawingController initialization
  nextTick(() => {
    const echarts = klineChartRef.value?.getEchartsInstance?.()
    if (echarts && drawingCanvasRef.value) {
      dc = new DrawingController()
      dc.mount(echarts, drawingCanvasRef.value, symbol.value)
    }
    if (echarts && crosshairCanvasRef.value) {
      crosshair = new Crosshair()
      crosshair.mount(echarts, crosshairCanvasRef.value)
      echarts.on('mousemove', (params: any) => {
        if (!drawingMode.value && params?.event) {
          crosshair?.show(params.event.offsetX, params.event.offsetY)
        }
      })
      echarts.on('mouseout', () => {
        crosshair?.hide()
      })
    }
  })

  document.addEventListener('click', closeContextMenu)
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  stopMinutePolling()
  stopKlineRefresh()
  if (quoteTimer) { clearInterval(quoteTimer); quoteTimer = null }
  // Save current data to shared cache so data survives component destruction
  if (symbol.value && minuteTicks.value.length > 0) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }
  dc?.destroy()
  crosshair?.destroy()
  if (symbolSubCleanup) symbolSubCleanup()
  document.removeEventListener('click', closeContextMenu)
  window.removeEventListener('keydown', onKeyDown)
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
        <button class="drawing-btn" @click="toggleDrawingMode()" :class="{ active: drawingMode }" title="画线工具 (Shift+D)">
          ✏️
        </button>
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
        <button :class="{ active: topOverlay === 'sar' }" class="indicator-btn" @click="topOverlay = 'sar'">SAR</button>
        <button :class="{ active: topOverlay === 'ema' }" class="indicator-btn" @click="topOverlay = 'ema'">EMA</button>
      </div>
      <div class="indicator-group">
        <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
        <button :class="{ active: bottomMode === 'volume' }" class="indicator-btn" @click="bottomMode = 'volume'">{{ $t('kline.volume') }}</button>
        <button :class="{ active: bottomMode === 'macd' }" class="indicator-btn" @click="bottomMode = 'macd'">MACD</button>
        <button :class="{ active: bottomMode === 'kdj' }" class="indicator-btn" @click="bottomMode = 'kdj'">KDJ</button>
        <button :class="{ active: bottomMode === 'rsi' }" class="indicator-btn" @click="bottomMode = 'rsi'">RSI</button>
        <button :class="{ active: bottomMode === 'wr' }" class="indicator-btn" @click="bottomMode = 'wr'">WR</button>
        <button :class="{ active: bottomMode === 'cci' }" class="indicator-btn" @click="bottomMode = 'cci'">CCI</button>
        <button :class="{ active: bottomMode === 'obv' }" class="indicator-btn" @click="bottomMode = 'obv'">OBV</button>
      </div>
      <div class="indicator-group">
        <span class="indicator-label">叠加指数</span>
        <select v-model="indexOverlaySymbol" class="toolbar-select">
          <option value="">不叠加</option>
          <option value="000001">上证指数</option>
          <option value="399001">深证成指</option>
          <option value="399006">创业板指</option>
        </select>
      </div>
    </div>
    <div v-if="activeTab === 'minute' || activeTab === 'multiDay'" class="indicator-bar">
      <div class="indicator-group">
        <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
        <button :class="{ active: minuteBottomMode === 'volume' }" class="indicator-btn" @click="minuteBottomMode = 'volume'">{{ $t('kline.volume') }}</button>
        <button :class="{ active: minuteBottomMode === 'macd' }" class="indicator-btn" @click="minuteBottomMode = 'macd'">MACD</button>
        <button :class="{ active: minuteBottomMode === 'kdj' }" class="indicator-btn" @click="minuteBottomMode = 'kdj'">KDJ</button>
      </div>
      <div v-if="activeTab === 'minute'" class="indicator-group">
        <button class="indicator-btn depth-toggle" :class="{ active: showDepth }" @click="toggleDepth">📊 {{ $t('misc.depth') }}</button>
      </div>
    </div>
    <div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
    <InfoBar
      :quote="quoteData"
      :symbol="symbol"
      :name="name ?? ''"
    />
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
      <div v-else-if="activeTab === 'kline' && ohlcvData.length > 0" class="chart-area" @contextmenu="onContextMenu">
        <KlineChart
          ref="klineChartRef"
          :option="option"
          :symbol="symbol"
          :loading="loading && !ohlcvData.length"
        />
        <canvas
          ref="drawingCanvasRef"
          class="canvas-overlay"
          :class="{ 'drawing-mode': drawingMode }"
          @mousedown="onDrawingMouseDown"
          @mousemove="onDrawingMouseMove"
          @mouseup="onDrawingMouseUp"
        />
        <canvas
          ref="crosshairCanvasRef"
          class="crosshair-overlay"
        />
        <div class="drawing-toolbar" v-if="drawingMode">
          <button @click="dc?.setMode('cursor')" :class="{ active: dc?.mode === 'cursor' }" title="光标">↖</button>
          <button @click="dc?.setMode('trendline')" :class="{ active: dc?.mode === 'trendline' }" title="趋势线">╱</button>
          <button @click="dc?.setMode('horizontal')" :class="{ active: dc?.mode === 'horizontal' }" title="水平线">━</button>
          <button @click="dc?.setMode('fibonacci')" :class="{ active: dc?.mode === 'fibonacci' }" title="斐波那契">F</button>
          <button @click="dc?.setMode('text')" :class="{ active: dc?.mode === 'text' }" title="文字">T</button>
          <input type="color" v-model="drawingColor" @input="dc?.setColor(drawingColor)" />
          <button @click="dc?.clearAll()" title="全部清除">✕</button>
        </div>
      </div>
      <template v-else-if="activeTab === 'minute'">
        <div class="minute-layout">
          <div class="minute-chart-area">
            <KlineChart v-if="minuteTicks.length" :symbol="`${symbol}-minute`" :option="minuteChartOption" :loading="minuteLoading && !minuteTicks.length" />
            <div v-else class="chart-fallback no-data">{{ $t('kline.no_minute_data') }}</div>
          </div>
          <div v-if="showDepth" class="depth-sidebar">
            <div class="dp-last-price" :style="{ color: marketChangeColor(symbol, depthChangePct) }">
              <span class="dp-name">{{ name || symbol }}</span>
              <span class="dp-val">{{ depthPrice.toFixed(2) }}</span>
              <span class="dp-chg">{{ depthChange >= 0 ? '+' : '' }}{{ depthChange.toFixed(2) }} ({{ depthChangePct >= 0 ? '+' : '' }}{{ depthChangePct.toFixed(2) }}%)</span>
            </div>
            <div class="dp-ob-header">
              <span class="h-bid">{{ $t('quote.bid') }}</span><span class="h-bs">{{ $t('common.size') }}</span><span class="h-bar"></span>
              <span class="h-ask">{{ $t('quote.ask') }}</span><span class="h-as">{{ $t('common.size') }}</span><span class="h-bar"></span>
            </div>
            <div v-for="i in 5" :key="i" class="dp-ob-row">
              <span class="dp-bid-p">{{ depthData?.bids[5-i]?.price.toFixed(2) ?? '' }}</span>
              <span class="dp-bid-s">{{ depthData?.bids[5-i] ? formatSize(depthData.bids[5-i].size) : '' }}</span>
              <span class="dp-bar-w"><span class="dp-bar bid" :style="{width: depthData?.bids[5-i] ? barWidth(depthData.bids[5-i].size) : '0%'}"></span></span>
              <span class="dp-ask-p">{{ depthData?.asks[i-1]?.price.toFixed(2) ?? '' }}</span>
              <span class="dp-ask-s">{{ depthData?.asks[i-1] ? formatSize(depthData.asks[i-1].size) : '' }}</span>
              <span class="dp-bar-w"><span class="dp-bar ask" :style="{width: depthData?.asks[i-1] ? barWidth(depthData.asks[i-1].size) : '0%'}"></span></span>
            </div>
            <div v-if="depthSimulated" class="dp-sim">模拟</div>
          </div>
        </div>
      </template>
      <div v-else class="chart-fallback">--</div>
    </div>
    <div
      v-if="contextMenu.visible"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
    >
      <div class="menu-item" @click="switchInterval('1d')">日线</div>
      <div class="menu-item" @click="switchInterval('1w')">周线</div>
      <div class="menu-item" @click="switchInterval('1h')">60分钟</div>
      <div class="menu-item" @click="switchInterval('30m')">30分钟</div>
      <div class="menu-item" @click="switchInterval('15m')">15分钟</div>
      <div class="menu-item" @click="switchInterval('5m')">5分钟</div>
      <div class="menu-item" @click="switchInterval('1m')">1分钟</div>
      <div class="menu-separator"></div>
      <div class="menu-item" @click="switchOverlay('none')">清除叠加</div>
      <div class="menu-item" @click="switchOverlay('ma')">叠加MA</div>
      <div class="menu-item" @click="switchOverlay('bb')">叠加布林带</div>
      <div class="menu-separator"></div>
      <div class="menu-item danger" @click="dc?.clearAll(); closeContextMenu()">清除画线</div>
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
.chart-body { flex: 1; min-height: 0; padding: 8px; display: flex; flex-direction: column; }
.kline-chart { width: 100%; height: 100%; }
.chart-fallback { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); }
.no-data { color: var(--color-text-tertiary); padding: 40px; text-align: center; }

/* Multi-Day Minute */
.multi-day-chart-wrapper {
  flex: 1; min-height: 0; display: flex; flex-direction: column;
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
.err-toast { padding: 6px 12px; background: rgba(239,68,68,0.15); color: #ef4444; font-size: 12px; border-radius: 4px; margin-bottom: 8px; }

/* Depth sidebar */
.depth-toggle { }
.minute-layout { display: flex; flex: 1; min-height: 0; gap: 8px; }
.minute-chart-area { flex: 1; min-width: 0; min-height: 0; display: flex; }
.depth-sidebar { width: 220px; flex-shrink: 0; display: flex; flex-direction: column; gap: 3px; padding: 6px; background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); border-radius: 6px; overflow: hidden; }
.dp-last-price { display: flex; align-items: baseline; gap: 6px; padding-bottom: 4px; border-bottom: 1px solid var(--color-border-strong); font-size: 12px; }
.dp-name { font-size: 10px; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-val { font-size: 14px; font-weight: 700; }
.dp-chg { font-size: 10px; white-space: nowrap; }
.dp-ob-header { display: flex; font-size: 9px; color: var(--color-text-tertiary); text-transform: uppercase; border-bottom: 1px solid var(--color-border-strong); padding: 2px 0; }
.dp-ob-header .h-bid { width: 48px; text-align: left; }
.dp-ob-header .h-bs { width: 52px; text-align: right; }
.dp-ob-header .h-bar { flex: 1; }
.dp-ob-header .h-ask { width: 48px; text-align: left; }
.dp-ob-header .h-as { width: 52px; text-align: right; }
.dp-ob-row { display: flex; font-size: 10px; font-variant-numeric: tabular-nums; padding: 1px 0; align-items: center; }
.dp-bid-p { width: 48px; flex-shrink: 0; color: var(--color-down); text-align: left; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-bid-s { width: 52px; flex-shrink: 0; text-align: right; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-bar-w { flex: 1; min-width: 16px; position: relative; height: 12px; }
.dp-bar { position: absolute; top: 1px; height: 9px; border-radius: 2px; opacity: 0.35; }
.dp-bar.bid { background: var(--color-down); right: 0; }
.dp-bar.ask { background: var(--color-up); left: 0; }
.dp-ask-p { width: 48px; flex-shrink: 0; color: var(--color-up); text-align: left; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-ask-s { width: 52px; flex-shrink: 0; text-align: right; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-sim { font-size: 9px; color: var(--color-text-tertiary); text-align: center; padding: 1px; background: var(--color-border-strong); border-radius: 3px; }

.chart-area {
  position: relative;
  flex: 1;
  min-height: 0;
}

.canvas-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 10;
}

.canvas-overlay.drawing-mode {
  pointer-events: auto;
  cursor: crosshair;
}

.crosshair-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 11;
}

.drawing-toolbar {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  padding: 6px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: 6px;
  z-index: 11;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.drawing-toolbar button {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}

.drawing-toolbar button:hover {
  border-color: var(--color-text-secondary);
}

.drawing-toolbar button.active {
  border-color: var(--color-accent);
  background: rgba(88, 166, 255, 0.15);
  color: var(--color-accent);
}

.drawing-toolbar input[type="color"] {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  padding: 2px;
  cursor: pointer;
}

.toolbar-select {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: 'JetBrains Mono', monospace;
  outline: none;
}
.toolbar-select:hover { border-color: var(--color-accent); color: var(--color-accent); }
.toolbar-select option { background: var(--color-bg-elevated); color: var(--color-text-primary); }

.drawing-btn {
  padding: 3px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  line-height: 1;
}

.drawing-btn:hover {
  border-color: var(--color-text-secondary);
}

.drawing-btn.active {
  border-color: var(--color-accent);
  background: rgba(88, 166, 255, 0.15);
}

.context-menu {
  position: fixed;
  z-index: 1000;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  border-radius: 6px;
  padding: 4px 0;
  min-width: 140px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.2);
}

.menu-item {
  padding: 6px 16px;
  font-size: 13px;
  cursor: pointer;
  color: var(--color-text-primary);
  transition: background 0.1s;
}

.menu-item:hover {
  background: var(--color-accent);
  color: #fff;
}

.menu-item.danger:hover {
  background: var(--color-up);
}

.menu-separator {
  height: 1px;
  margin: 4px 8px;
  background: var(--color-border-subtle);
}
</style>
