<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'
import { buildKlineOption, buildMinuteOption, buildMultiDayOption, type KlineDataItem } from '@/lib/buildChartOption'
import { useWailsApp, type OHLCVBar, type MultiDayMinute } from '@/lib/composables/useWailsApp'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Shared minute data cache from parent DockView
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())

const topOverlay = ref<'none' | 'ma' | 'bb'>('none')
const bottomMode = ref<'volume' | 'macd' | 'kdj' | 'rsi' | 'wr'>('volume')
const minuteBottomMode = ref<'volume' | 'macd' | 'kdj'>('volume')


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
const ohlcvData = ref<KlineDataItem[]>([])
const loading = ref(false)
const indicatorCache = createIndicatorCache()
const errorMsg = ref('')
let loadSeq = 0
const theme = useChartTheme()

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
  klineTimer = window.setInterval(() => loadOHLCV(symbol.value, true), 30000)
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
  if (!ohlcvData.value.length) return {} as ECBasicOption
  return buildKlineOption(ohlcvData.value, topOverlay.value, bottomMode.value, theme, indicatorCache, symbol.value, interval.value)
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
    <div v-if="errorMsg" class="err-toast">{{ errorMsg }}</div>
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
.err-toast { padding: 6px 12px; background: rgba(239,68,68,0.15); color: #ef4444; font-size: 12px; border-radius: 4px; margin-bottom: 8px; }
</style>
