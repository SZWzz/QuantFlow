<script setup lang="ts">
import { ref, shallowRef, computed, onMounted, onUnmounted, watch, inject, nextTick, reactive } from 'vue'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import InfoBar from '@/terminal/components/panel/InfoBar.vue'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { DrawingController } from '@/lib/chart/DrawingController'
import { Crosshair } from '@/lib/chart/Crosshair'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket, isTradingHours } from '@/lib/wails'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'
import { buildKlineOption, buildMinuteOption, type KlineDataItem } from '@/lib/buildChartOption'
import { useWailsApp, type OHLCVBar, type QuoteData, type MinuteTick } from '@/lib/composables/useWailsApp'
import { useMinuteChart } from '@/lib/composables/useMinuteChart'
import { useDataStore } from '@/stores/data'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { detectLimitUpDown } from '@/lib/chart/EventMarker'
import type { EventMarker } from '@/lib/chart/EventMarker'
import { useWebSocket, type KlineUpdate } from '@/lib/composables/useWebSocket'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { logger } from '@/lib/logger'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import ChartToolbar from './candlestick/ChartToolbar.vue'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Shared minute data cache from parent DockView
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())

const topOverlay = ref<'none' | 'ma' | 'bb' | 'sar' | 'ema'>('none')
const bottomMode = ref<'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv'>('volume')
const minuteBottomMode = ref<'volume' | 'macd' | 'kdj' | 'rsi' | 'obv'>('volume')

const klineChartRef = ref<any>(null)
const drawingCanvasRef = ref<HTMLCanvasElement | null>(null)
const drawingMode = ref(false)
let dc: DrawingController | null = null
let crosshair: Crosshair | null = null
const crosshairCanvasRef = ref<HTMLCanvasElement | null>(null)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { control: addToWfControl } = useAddToWorkflow(props.panelId, symbol)

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
const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')
const loadError = ref('')
let loadSeq = 0

// OHLCV progressive loading state
const ohlcvExpanding = ref(false)
const ohlcvExpandStart = ref(0) // earliest loaded unix timestamp

const theme = useChartTheme()
// 画线默认颜色取自 chart theme（P5：hex → useChartTheme 字段）
const drawingColor = ref(theme.palette[0])

// Tab state
const activeTab = ref<'kline' | 'minute'>('kline')

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
    // Feed the actual previous close to useMinuteChart so the baseline
    // is correct (indices can gap significantly from their prior close).
    if (snap.prevClose && snap.prevClose > 0) {
      prevClose.value = snap.prevClose
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

// Minute chart data (powered by useMinuteChart composable)
function computeDataKey(ticks: MinuteTick[]): string {
  if (ticks.length === 0) return '0|'
  const last = ticks[ticks.length - 1]
  return `${ticks.length}|${last.time}|${last.price}`
}
const prevClose = ref(0)
const { minuteTicks, minuteLoading, loadMinuteLine, startPolling: startMinutePoll, stopPolling: stopMinutePoll } =
  useMinuteChart(symbol, prevClose, { polling: true, pollingInterval: 5000 })
let klineTimer: ReturnType<typeof setInterval> | null = null
let quoteTimer: ReturnType<typeof setInterval> | null = null
const { connect: wsConnect, onMessage: wsOnMessage, connected: wsConnected } = useWebSocket()
let symbolSubCleanup: (() => void) | null = null

function getTodayDateString(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

async function loadOHLCV(sym: string, incremental = false) {
  const seq = ++loadSeq
  state.value = 'loading'
  loading.value = true
  try {
    const end = Math.floor(Date.now() / 1000)
    const iv = interval.value
    let start: number
    if (incremental && ohlcvData.value.length > 0) {
      const lastDate = ohlcvData.value[ohlcvData.value.length - 1].date
      start = Math.floor(new Date(lastDate.replace(' ', 'T')).getTime() / 1000)
    } else {
      const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : 365
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
    loadError.value = errorMsg.value
    state.value = 'error'
    setTimeout(() => { errorMsg.value = '' }, 8000)
    if (!ohlcvData.value.length) ohlcvData.value = []
  }
  if (seq === loadSeq) {
    loading.value = false
    if (state.value !== 'error') {
      state.value = ohlcvData.value.length > 0 ? 'loaded' : 'empty'
    }
  }
}

function startKlineRefresh() {
  stopKlineRefresh()
  // Skip polling if outside trading hours (non-CN markets have live WebSocket)
  if (detectMarket(symbol.value) !== 'CN' && !isTradingHours(detectMarket(symbol.value))) return
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

function loadData() {
  loadOHLCV(symbol.value)
  loadQuote()
  initWebSocket()
  if (['1m', '5m', '15m', '30m', '1h'].includes(interval.value)) {
    if (activeTab.value === 'kline') {
      startKlineRefresh()
    }
  } else {
    stopKlineRefresh()
  }
}

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
    loadOHLCV(newSymbol)
  }
})

watch([symbol, interval], () => loadData(), { immediate: true })

watch(ohlcvData, (data) => {
  if (data.length > 0) {
    const isIntraday = ['1m','5m','15m','30m','1h'].includes(interval.value)
    const dateStr = isIntraday ? data[0].date.replace(' ', 'T') : data[0].date + 'T00:00:00'
    ohlcvExpandStart.value = Math.floor(new Date(dateStr).getTime() / 1000)
  }
  eventMarkers.value = data.length >= 2 ? detectLimitUpDown(data) : []
}, { immediate: true })

// dataZoom expansion: when user scrolls near the start, load earlier data
let expandThrottle: ReturnType<typeof setTimeout> | null = null
function onDataZoom(params: any) {
  // ECharts 5.5+ may send { batch: [...] } with multiple zooms OR
  // { start, end } directly depending on interaction type. Handle both.
  const dz = (params.batch && params.batch[0]) ? params.batch[0] : params
  const dzStart = dz?.start
  // Debug: always log when dataZoom fires, so we know the event works.
  if (dzStart != null && dzStart < 20) {
    console.log('[Candlestick] datazoom event: start=%s, batch=%s', dzStart, params.batch ? 'present' : 'absent')
  }
  if (dzStart == null) return
  if (dzStart > 5) return  // only trigger when near the beginning (<5%)
  if (ohlcvExpanding.value) return // guard: already expanding
  if (['1m','5m','15m','30m','1h'].includes(interval.value)) return // skip intraday

  if (expandThrottle) clearTimeout(expandThrottle)
  expandThrottle = setTimeout(async () => {
    console.log('[Candlestick] expand trigger: start=%s expandStart=%s', dz.start, new Date(ohlcvExpandStart.value * 1000).toISOString().slice(0, 10))
    ohlcvExpanding.value = true
    const earlierStart = ohlcvExpandStart.value - 365 * 86400
    const app = useWailsApp()
    if (!app) { ohlcvExpanding.value = false; return }
    try {
      // Snapshot current zoom (visible date range) so we can restore it
      // after prepending older data — prevents the chart from jumping.
      const chart = klineChartRef.value?.getEchartsInstance()
      let anchorDate: string | null = null
      if (chart) {
        const opt = chart.getOption()
        const dz0 = (opt as any)?.dataZoom?.[0]
        if (dz0) {
          const oldDates = ohlcvData.value
          const visStartIdx = Math.floor((dz0.start / 100) * oldDates.length)
          anchorDate = visStartIdx < oldDates.length ? oldDates[visStartIdx]?.date ?? null : null
        }
      }

      const [rawBars] = await app.FetchOHLCV(detectMarket(symbol.value), symbol.value, interval.value, 'qfq', earlierStart, ohlcvExpandStart.value)
      if (rawBars?.length) {
        const mergeMap = new Map(ohlcvData.value.map(b => [b.date, b]))
        const isIntraday = ['1m','5m','15m','30m','1h'].includes(interval.value)
        for (const b of rawBars) {
          const d = new Date(b.date || '')
          const date = isIntraday
            ? d.toISOString().slice(0, 16).replace('T', ' ')
            : d.toISOString().slice(0, 10)
          mergeMap.set(date, { date, open: b.open, close: b.close, low: b.low, high: b.high, volume: b.volume })
        }
        ohlcvData.value = Array.from(mergeMap.values()).sort((a, b) => a.date.localeCompare(b.date))
        // Update to the actual earliest bar date, not the requested start.
        // Adapters may return more data than requested (e.g. Tencent 2000-bar cap),
        // so using earlierStart would skip chunks and create gaps on the next expansion.
        if (ohlcvData.value.length > 0) {
          const firstDate = ohlcvData.value[0].date
          const ds = isIntraday ? firstDate.replace(' ', 'T') : firstDate + 'T00:00:00'
          ohlcvExpandStart.value = Math.floor(new Date(ds).getTime() / 1000)
        }

        // Restore zoom to the same visible date range, now mapped to the
        // new (larger) dataset so the chart does not visibly jump.
        if (anchorDate && chart) {
          await nextTick()
          const newDates = ohlcvData.value
          const newIdx = newDates.findIndex(b => b.date === anchorDate)
          if (newIdx >= 0) {
            const newStart = (newIdx / newDates.length) * 100
            const newEnd = Math.min(100, newStart + ((chart.getOption() as any)?.dataZoom?.[0]?.end ?? 100) - ((chart.getOption() as any)?.dataZoom?.[0]?.start ?? 0))
            chart.dispatchAction({ type: 'dataZoom', start: newStart, end: newEnd })
          }
        }
      }
    } catch (e) {
      console.error('[Candlestick] OHLCV expand:', e)
    } finally {
      ohlcvExpanding.value = false
    }
  }, 500)
}

watch(indexOverlaySymbol, async (sym) => {
  if (!sym) { indexOverlayData.value = null; return }
  try {
    const app = useWailsApp()
    if (!app) return
    const now = Math.floor(Date.now() / 1000)
    const [bars] = await app.FetchOHLCV('CN', sym, '1d', 'qfq', now - 9125 * 86400, now)
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

// Watch tab switch for minute polling
watch(activeTab, (tab) => {
  if (tab === 'minute') {
    startMinutePoll()
  } else {
    stopMinutePoll()
  }
})

// Sync minute data to shared cache for other panels
watch(minuteTicks, (ticks) => {
  if (ticks.length) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, ticks)
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

const option = computed(() => {
  if (!ohlcvData.value.length) return {} as ECBasicOption
  return buildKlineOption(ohlcvData.value, topOverlay.value, bottomMode.value, theme, indicatorCache, symbol.value, interval.value, eventMarkers.value, indexOverlayData.value)
})

const minuteOptionCache = ref<{ key: string; option: ECBasicOption | null }>({ key: '', option: null })
const minuteChartOption = computed(() => {
  const ticks = minuteTicks.value
  if (!ticks.length) return {} as ECBasicOption
  const dataKey = computeDataKey(ticks)
  const fullKey = `${dataKey}|${minuteBottomMode.value}`
  if (fullKey === minuteOptionCache.value.key && minuteOptionCache.value.option) {
    return minuteOptionCache.value.option
  }
  const opt = buildMinuteOption(ticks, prevClose.value, minuteBottomMode.value, theme, indicatorCache, symbol.value)
  minuteOptionCache.value = { key: fullKey, option: opt }
  return opt
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
  if (drawingMode.value && !dc) {
    initChartControllers()
    if (!dc) {
      setTimeout(() => {
        initChartControllers()
        dc?.setMode('trendline')
      }, 120)
    }
  }
  dc?.setMode(drawingMode.value ? 'trendline' : 'cursor')
}

function jumpToDate() {
  const dateStr = prompt('跳转到日期 (YYYY-MM-DD):')
  if (!dateStr) return
  const target = new Date(dateStr).getTime()
  if (isNaN(target)) return

  const echarts = klineChartRef.value?.getEchartsInstance?.()
  if (!echarts) return

  const timestamps = ohlcvData.value.map(d => new Date(d.date).getTime() / 1000)
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

// Store echarts event handler refs for proper cleanup (prevents leak on re-init)
let echartsMoveHandler: ((params: any) => void) | null = null
let echartsOutHandler: (() => void) | null = null

function initChartControllers() {
  if (dc && crosshair) return

  const chart = klineChartRef.value
  const dCanvas = drawingCanvasRef.value
  const cCanvas = crosshairCanvasRef.value
  if (!chart || !dCanvas || !cCanvas) return
  if (!ohlcvData.value.length) return
  const echart = chart.getEchartsInstance?.()
  if (!echart) {
    setTimeout(() => initChartControllers(), 100)
    return
  }

  if (dc) {
    dc.saveDrawings()
    dc.destroy()
  }
  crosshair?.destroy()

  dc = new DrawingController()
  dc.mount(echart, dCanvas, symbol.value)
  // 与画线工具条的颜色选择器默认值保持同步（re-init 后亦保留用户已选颜色）
  dc.setColor(drawingColor.value)

  crosshair = new Crosshair()
  crosshair.mount(echart, cCanvas)

  if (echartsMoveHandler) echart.off('mousemove', echartsMoveHandler)
  if (echartsOutHandler) echart.off('mouseout', echartsOutHandler)
  echartsMoveHandler = (params: any) => {
    if (!drawingMode.value && params?.event) {
      crosshair?.show(params.event.offsetX, params.event.offsetY)
    }
  }
  echartsOutHandler = () => {
    crosshair?.hide()
  }
  echart.on('mousemove', echartsMoveHandler)
  echart.on('mouseout', echartsOutHandler)
}

watch(drawingMode, (mode) => {
  if (mode) crosshair?.hide()
})

watch(klineChartRef, (chart) => {
  if (!chart) return
  // Clean up old controllers before re-init (handles tab switch and initial mount)
  if (dc || crosshair) {
    dc?.saveDrawings()
    dc?.destroy()
    dc = null
    crosshair?.destroy()
    crosshair = null
  }
  nextTick(() => initChartControllers())
})

watch(symbol, () => {
  dc?.saveDrawings()
  dc?.destroy()
  dc = null
  crosshair?.destroy()
  crosshair = null
  if (klineChartRef.value) {
    nextTick(() => initChartControllers())
  }
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

  document.addEventListener('click', closeContextMenu)
  window.addEventListener('keydown', onKeyDown)
})

onUnmounted(() => {
  stopMinutePoll()
  stopKlineRefresh()
  if (quoteTimer) { clearInterval(quoteTimer); quoteTimer = null }
  // Save current data to shared cache so data survives component destruction
  if (symbol.value && minuteTicks.value.length > 0) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }
  if (dc) { dc.saveDrawings(); dc.destroy() }
  crosshair?.destroy()
  if (symbolSubCleanup) symbolSubCleanup()
  document.removeEventListener('click', closeContextMenu)
  window.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <div class="candlestick-panel">
    <ChartToolbar
      :symbol="symbol"
      :name="name ?? ''"
      :isInWatchlist="isInWatchlist"
      :activeTab="activeTab"
      :interval="interval"
      :drawingMode="drawingMode"
      :addToWfControl="addToWfControl"
      :topOverlay="topOverlay"
      :bottomMode="bottomMode"
      :minuteBottomMode="minuteBottomMode"
      :indexOverlaySymbol="indexOverlaySymbol"
      :showDepth="showDepth"
      @toggleWatchlist="toggleWatchlist"
      @toggleDrawingMode="toggleDrawingMode"
      @update:activeTab="activeTab = $event"
      @update:interval="interval = $event"
      @update:topOverlay="topOverlay = $event"
      @update:bottomMode="bottomMode = $event"
      @update:minuteBottomMode="minuteBottomMode = $event"
      @update:indexOverlaySymbol="indexOverlaySymbol = $event"
      @toggleDepth="toggleDepth"
    />
    <div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
    <InfoBar
      :quote="quoteData"
      :symbol="symbol"
      :name="name ?? ''"
    />
    <div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
    <PanelShell :state="state" :error="loadError" @retry="() => loadOHLCV(symbol.value)">
      <template #loaded>
        <div class="chart-body">
          <div v-if="activeTab === 'kline' && ohlcvData.length > 0" class="chart-area" @contextmenu="onContextMenu">
            <KlineChart
              ref="klineChartRef"
              :option="option"
              :symbol="symbol"
              :loading="loading && !ohlcvData.length"
              @dataZoom="onDataZoom"
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
                <template v-if="minuteTicks.length">
                  <KlineChart :symbol="`${symbol}-minute`" :option="minuteChartOption" :loading="minuteLoading && !minuteTicks.length" />
                </template>
                <SkeletonPanel v-else-if="minuteLoading" type="chart" />
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
      </template>
    </PanelShell>
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
  display: flex; flex-direction: column; height: 100%; overflow: hidden;
}
.chart-body { flex: 1; min-height: 0; padding: var(--space-sm); display: flex; flex-direction: column; }
.chart-fallback { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); }
.no-data { color: var(--color-text-tertiary); padding: var(--space-2xl); text-align: center; }
.err-toast { padding: var(--space-xs) var(--space-md); background: var(--color-up-soft); color: var(--color-up); font-size: var(--font-xs); border-radius: var(--radius-sm); margin-bottom: var(--space-sm); }

/* Depth sidebar */
.minute-layout { display: flex; flex: 1; min-height: 0; gap: var(--space-sm); }
.minute-chart-area { flex: 1; min-width: 0; min-height: 0; display: flex; }
.depth-sidebar { width: 220px; flex-shrink: 0; display: flex; flex-direction: column; gap: var(--space-xs); padding: var(--space-xs); background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); border-radius: var(--radius-md); overflow: hidden; }
.dp-last-price { display: flex; align-items: baseline; gap: var(--space-xs); padding-bottom: var(--space-xs); border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); }
.dp-name { font-size: var(--font-xs); color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-val { font-size: var(--font-base); font-weight: 700; }
.dp-chg { font-size: var(--font-xs); white-space: nowrap; }
.dp-ob-header { display: flex; font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; border-bottom: 1px solid var(--color-border-strong); padding: var(--space-xs) 0; }
.dp-ob-header .h-bid { width: 48px; text-align: left; }
.dp-ob-header .h-bs { width: 52px; text-align: right; }
.dp-ob-header .h-bar { flex: 1; }
.dp-ob-header .h-ask { width: 48px; text-align: left; }
.dp-ob-header .h-as { width: 52px; text-align: right; }
.dp-ob-row { display: flex; font-size: var(--font-xs); font-variant-numeric: tabular-nums; align-items: center; }
.dp-bid-p { width: 48px; flex-shrink: 0; color: var(--color-down); text-align: left; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-bid-s { width: 52px; flex-shrink: 0; text-align: right; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-bar-w { flex: 1; min-width: 16px; position: relative; height: 12px; }
.dp-bar { position: absolute; top: 1px; height: 9px; border-radius: var(--radius-sm); opacity: 0.35; }
.dp-bar.bid { background: var(--color-down); right: 0; }
.dp-bar.ask { background: var(--color-up); left: 0; }
.dp-ask-p { width: 48px; flex-shrink: 0; color: var(--color-up); text-align: left; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-ask-s { width: 52px; flex-shrink: 0; text-align: right; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dp-sim { font-size: var(--font-xs); color: var(--color-text-tertiary); text-align: center; padding: var(--space-xs); background: var(--color-border-strong); border-radius: var(--radius-sm); }

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
  top: var(--space-sm);
  right: var(--space-sm);
  display: flex;
  gap: var(--space-xs);
  padding: var(--space-xs);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  z-index: 11;
  box-shadow: var(--shadow-md);
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
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-sm);
}

.drawing-toolbar button:hover {
  border-color: var(--color-text-secondary);
}

.drawing-toolbar button.active {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.drawing-toolbar input[type="color"] {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  padding: var(--space-xs);
  cursor: pointer;
}

.context-menu {
  position: fixed;
  z-index: var(--z-tooltip);
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  padding: var(--space-xs) 0;
  min-width: 140px;
  box-shadow: var(--shadow-md);
}

.menu-item {
  padding: var(--space-xs) var(--space-lg);
  font-size: var(--font-sm);
  cursor: pointer;
  color: var(--color-text-primary);
  transition: background var(--transition-fast);
}

.menu-item:hover {
  background: var(--color-accent);
  color: var(--color-text-inverse);
}

.menu-item.danger:hover {
  background: var(--color-up);
}

.menu-separator {
  height: 1px;
  margin: var(--space-xs) var(--space-sm);
  background: var(--color-border-subtle);
}
</style>
