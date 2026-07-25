<!-- frontend/src/terminal/panels/GovDataPanel.vue -->
<!-- Unified macro panel: FRED (US) + CN (akshare 中国宏观) + BIS (全球) + commodities.
     Signal semantics (看涨/看跌/中性) computed server-side for CN and by
     govdata_service for FRED; UI badge + summary counts shared across sources. -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import { detectMarket } from '@/lib/wails'
import 'echarts'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useDataStore } from '@/stores/data'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import {
  PanelHeader,
  PanelTabs,
  LoadingState,
  EmptyState,
  SignalBadge,
  TrendIndicator,
} from '@/terminal/components/panel'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'
import { useWailsApp } from '@/lib/composables/useWailsApp'

const { t } = useI18n()
const dataStore = useDataStore()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface MacroSignal {
  indicator_id: string
  name: string
  name_cn: string
  latest_value: number
  change: number
  direction: string   // up, down, flat
  signal: string      // bullish, bearish, neutral
  unit: string
  category: string
  updated_at: number
  // CN source: the summary payload embeds a short series per indicator so the
  // detail chart can render instantly without a second round-trip.
  series?: IndicatorPoint[]
  latest_date?: string
}

interface IndicatorPoint {
  date: string
  value: number
}

interface CommodityQuote {
  symbol: string
  name: string
  name_cn: string
  price: number
  open: number
  high: number
  low: number
  change_pct: number
  unit: string
  updated: string
}

const signals = ref<MacroSignal[]>([])
const loading = ref(true)
const loadError = ref('')
const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')
const selectedSignal = ref<MacroSignal | null>(null)
const indicatorData = ref<IndicatorPoint[]>([])
const chartLoading = ref(false)

const signalsCacheKey = computed(() => `gov:signals:${activeSource.value}`)

// Three-source switch: FRED (US gov data) / CN (akshare 中国宏观) / BIS (global)
const activeSource = ref<'fred' | 'cn' | 'bis'>('fred')

const sources = [
  { key: 'fred' as const, label: 'FRED 美国' },
  { key: 'cn' as const, label: '中国宏观' },
  { key: 'bis' as const, label: 'BIS 全球' },
]

const sourceCnLabels: Record<string, string> = {
  fred: 'FRED 美联储经济数据',
  cn: '中国宏观经济指标',
  bis: 'BIS 国际清算银行',
}

// BIS dataflow IDs → Chinese display names. BIS SDMX API returns English-only
// names; we translate the commonly-used ones here for a consistent CN experience.
const bisCnNames: Record<string, string> = {
  WS_EER: '有效汇率指数',
  WS_EER_R: '实际有效汇率指数',
  WS_CBPOL: '央行政策利率',
  WS_DT1: '债务证券',
  WS_LTINT: '长期利率',
  WS_STINT: '短期利率',
  WS_MON: '货币供应量',
  WS_XRU: '汇率',
  WS_CRD: '非金融部门信贷',
  WS_HP: '房价指数',
  WS_REER: '实际有效汇率',
  WS_CUST: '海关与外汇管制',
  WS_FDI: '外国直接投资',
  WS_CUR: '官方外汇储备币种构成',
  WS_LONG_CPI: '长期CPI预测',
  WS_CBTA: '跨境银行债权',
  WS_CREDIT_GAP: '信贷缺口',
  WS_TC: '信贷总额',
  WS_DSR: '偿债率',
  WS_GLI: '全球流动性指标',
  WS_SPP: '股票价格',
  WS_CPP: '企业债务价格',
  WS_DPP: '政府债务价格',
  WS_CBS_PUB: '跨境银行统计',
  WS_LBS_D_PUB: '地方银行业统计',
  WS_DEBT_SEC2_PUB: '债务证券统计',
  WS_NA_SEC_DSS: '国民账户/债务证券',
  WS_OTC_DERIV2: '场外衍生品',
  WS_DER_OTC_TOV: '场外衍生品交易量',
  WS_XTD_DERIV: '交易所衍生品',
  WS_CPMI_MACRO: '支付系统宏观统计',
  WS_CPMI_CASHLESS: '非现金支付',
  WS_CPMI_CT1: '支付系统1类',
  WS_CPMI_CT2: '支付系统2类',
  WS_CPMI_DEVICES: '支付终端设备',
  WS_CPMI_INSTITUT: '支付机构',
  WS_CPMI_PARTICIP: '支付系统参与方',
  WS_CPMI_SYSTEMS: '支付系统',
  DSI_CIVPART: '民间参与率',
}

const activeCategory = ref('all')

const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const controls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', label: t('common.refresh'), action: loadSignals })
  return list
})

// Request sequence counter: guards against stale responses when switching
// sources mid-flight (e.g., BIS SDMX is slow, user taps "中国宏观" before it
// resolves — without this guard the slow response would overwrite CN data).
let loadSeq = 0

async function loadSignals() {
  const seq = ++loadSeq
  const cached = dataStore.getCached<MacroSignal[]>(signalsCacheKey.value)
  if (cached) {
    signals.value = cached
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = ''
  state.value = 'loading'
  signals.value = []
  selectedSignal.value = null
  indicatorData.value = []
  try {
    const app = useWailsApp()

    // ── FRED source: economic indicators + real-time commodity quotes ──
    if (activeSource.value === 'fred') {
      if (app?.GetEconomicIndicators) {
        const result = await app.GetEconomicIndicators()
        if (seq !== loadSeq) return
        signals.value = result.signals || []
      }
      // Merge real-time commodity quotes (CL/NG) into the energy category
      if (app?.GetCommodityQuotes) {
        const result = await app.GetCommodityQuotes()
        if (seq !== loadSeq) return
        const commodities = (result.commodities || []) as CommodityQuote[]
        for (const c of commodities) {
          signals.value.push({
            indicator_id: c.symbol,
            name: c.name,
            name_cn: c.name_cn,
            latest_value: c.price,
            change: c.change_pct,
            direction: c.change_pct >= 0 ? 'up' : 'down',
            signal: c.change_pct >= 0 ? 'bullish' : 'bearish',
            unit: c.unit,
            category: 'energy',
            updated_at: Date.now() / 1000,
          })
        }
      }
    }

    // ── CN source: akshare get_summary returns categories + values, where
    //    each value carries {latest_value, latest_date, unit, name_cn, category,
    //    change, direction, signal, polarity, series}. The series is cached on
    //    the signal so the detail chart renders instantly. Signal semantics
    //    (polarity: positive/negative/inverse) are computed server-side. ──
    else if (activeSource.value === 'cn' && app?.FetchData) {
      const result = await app.FetchData('akshare', 'macro_cn_summary', ['CN'], '', '', {})
      if (seq !== loadSeq) return
      if (result?.data) {
        try {
          const raw = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
          const parsed = raw?.data || raw
          // Only render cards for endpoints that actually returned data
          // (in parsed.values). Previously this iterated parsed.categories
          // which lists all 85 akshare endpoints — most had no value, so
          // the panel was full of empty "点击查看数据" cards.
          const vals = parsed?.values || {}
          if (Object.keys(vals).length > 0) {
            const items: MacroSignal[] = []
            for (const [ep, v] of Object.entries(vals)) {
              const vv = v as any
              items.push({
                indicator_id: `cn_${ep}`,
                name: ep,
                name_cn: vv?.name_cn || ep,
                latest_value: vv?.latest_value ?? 0,
                change: vv?.change ?? 0,
                direction: vv?.direction || 'flat',
                signal: vv?.signal || 'neutral',
                unit: vv?.unit || '',
                category: vv?.category || 'Core Indicators',
                updated_at: 0,
                series: vv?.series,
                latest_date: vv?.latest_date,
              })
            }
            signals.value = items
          } else { loadError.value = '中国宏观数据加载失败（无可用端点）' }
        } catch(e: any) { loadError.value = 'Parse error: ' + e.message }
      } else if (result?.error) { loadError.value = result.error }
    }

    // ── BIS source: uses get_summary to fetch catalog + latest values in one
    // parallel call, so cards show the latest data point immediately.
    else if (activeSource.value === 'bis' && app?.FetchData) {
      const result = await app.FetchData('macro', 'bis', [], '', '', { cmd: 'get_summary' })
      if (seq !== loadSeq) return
      if (result?.data) {
        try {
          const raw = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
          const           flows = raw?.dataflows || raw?.data?.dataflows || []
          signals.value = flows
            .filter((f: any) => f.id) // skip entries without an id
            .map((f: any) => ({
              indicator_id: `bis_${f.id}`,
              name: f.name || f.id,
              name_cn: bisCnNames[f.id] || f.names?.en || f.name || f.id,
              latest_value: f.latest_value,
              change: 0,
              direction: 'flat',
              signal: 'neutral',
              unit: f.unit || '',
              category: 'BIS 全球',
              updated_at: 0,
              latest_date: f.latest_date || '',
            }))
        } catch(e: any) { loadError.value = 'BIS parse error: ' + e.message }
      } else if (result?.error) { loadError.value = result.error }
    }
  } catch(e: any) {
    loadError.value = e?.message || String(e)
    console.error('[GovData] loadSignals:', e)
  }
  if (!loadError.value && signals.value.length > 0) {
    dataStore.setCached(signalsCacheKey.value, signals.value, 300_000)
  }
  loading.value = false
  if (loadError.value) state.value = 'error'
  else if (signals.value.length === 0) state.value = 'empty'
  else state.value = 'loaded'
}

let detailSeq = 0

async function loadIndicatorDetail(signal: MacroSignal) {
  const seq = ++detailSeq
  selectedSignal.value = signal
  const detailKey = `gov:detail:${signal.indicator_id}`
  const cached = dataStore.getCached<IndicatorPoint[]>(detailKey)
  if (cached) {
    indicatorData.value = cached
    chartLoading.value = false
    return
  }
  chartLoading.value = true
  try {
    const app = useWailsApp()

    // ── FRED: commodity OHLCV or FRED indicator history ──
    if (activeSource.value === 'fred') {
      // Commodities use OHLCV API instead of FRED history
      if (signal.indicator_id.startsWith('hf_')) {
        const tradingSymbol = signal.indicator_id === 'hf_CL' ? 'CL=F' : 'NG=F'
        const end = Math.floor(Date.now() / 1000)
        const start = end - 90 * 86400
        try {
          if (app?.FetchOHLCV) {
            const result = await app.FetchOHLCV(detectMarket(tradingSymbol), tradingSymbol, '1D', start, end)
            if (seq !== detailSeq) return
            const bars = Array.isArray(result) ? result[0] : result
            indicatorData.value = (bars || []).map((b: any) => ({
              date: typeof b.date === 'string' ? b.date.slice(0, 10) : new Date(b.date || b.Date).toISOString().slice(0, 10),
              value: b.close ?? b.Close ?? 0,
            }))
          }
        } catch(e) { console.error('[GovData] commodity OHLCV:', e); indicatorData.value = [] }
        if (indicatorData.value.length > 0) {
          dataStore.setCached(detailKey, indicatorData.value, 600_000)
        }
        chartLoading.value = false
        return
      }
      if (app?.GetIndicatorData) {
        const result = await app.GetIndicatorData(signal.indicator_id, 12)
        if (seq !== detailSeq) return
        indicatorData.value = result.data || []
      }
    }

    // ── CN: prefer the series cached on the signal from get_summary (instant).
    //    For endpoints not in the FAST core set (no cached series), fall back to
    //    a single macro_cn_indicator request which returns a normalized series. ──
    else if (activeSource.value === 'cn' && app?.FetchData) {
      if (Array.isArray(signal.series) && signal.series.length > 0) {
        indicatorData.value = signal.series.map((p: any) => ({
          date: p.date, value: p.value,
        }))
      } else {
        const sid = signal.indicator_id.replace(/^cn_/, '')
        const result = await app.FetchData('akshare', 'macro_cn_indicator', [sid], '', '', {})
        if (seq !== detailSeq) return
        if (result?.data) {
          try {
            const d = JSON.parse(result.data)
            const series = d?.series || []
            indicatorData.value = series.map((p: any) => ({
              date: p.date, value: p.value,
            }))
          } catch(e: any) { console.error('[GovData] cn detail parse:', e) }
        }
      }
    }

    // ── BIS: fetch specific dataset time series via 'fetch' command ──
    // macro_bis.py's 'fetch' command flattens SDMX-JSON into [{date, value}]
    // for charting. 'get_data' returns raw SDMX which is harder to parse.
    else if (app?.FetchData) {
      const sid = signal.indicator_id.replace(/^bis_/, '')
      const params: Record<string, string> = { cmd: 'fetch' }
      params.dataset = sid
      // Pass country 'all' as second positional arg (macro_bis fetch uses it)
      const result = await app.FetchData('macro', 'bis', [sid, 'all'], '', '', params)
      if (seq !== detailSeq) return
      if (result?.data) {
        try {
          const d = JSON.parse(result.data)
          // fetch command returns {success, data: [{date, value}], metadata}
          const values = d?.data || []
          if (Array.isArray(values)) {
            indicatorData.value = values.map((v: any) => ({
              date: String(v.date || ''),
              value: typeof v.value === 'number' ? v.value : 0,
            }))
          }
        } catch(e: any) { console.error('[GovData] bis detail parse:', e) }
      }
    }
  } catch(e) { console.error('[GovData] loadDetail:', e) }
  if (indicatorData.value.length > 0) {
    dataStore.setCached(detailKey, indicatorData.value, 600_000)
  }
  chartLoading.value = false
}

onMounted(() => {
  loadSignals()
})

// Dynamic categories: derived from loaded signals so each source's own
// category taxonomy (FRED: gdp/inflation/..., CN: Core Indicators/Monetary,
// BIS: BIS) drives the filter tabs. Falls back to FRED defaults while loading
// so the tab framework is visible before data arrives.
const categories = computed(() => {
  const cats = new Set(signals.value.map(s => s.category).filter(Boolean))
  if (cats.size === 0) {
    return ['all', 'gdp', 'inflation', 'employment', 'rates', 'energy', 'housing']
  }
  return ['all', ...Array.from(cats)]
})

const categoryLabels: Record<string, string> = {
  all: '全部', gdp: 'GDP/增长', inflation: '通胀', employment: '就业',
  rates: '利率', energy: '能源', housing: '房地产',
}

function categoryLabel(cat: string): string {
  return categoryLabels[cat] || cat
}

const filteredSignals = computed(() => {
  if (activeCategory.value === 'all') return signals.value
  return signals.value.filter(s => s.category === activeCategory.value)
})

// Count signals by type for the header summary
const signalCounts = computed(() => {
  let bullish = 0, bearish = 0, neutral = 0
  for (const s of signals.value) {
    if (s.signal === 'bullish') bullish++
    else if (s.signal === 'bearish') bearish++
    else neutral++
  }
  return { bullish, bearish, neutral }
})

// Chart option for selected indicator's time series — colored by signal
const chartOption = computed(() => {
  if (!selectedSignal.value || indicatorData.value.length === 0) return {}
  const dates = indicatorData.value.map(p => p.date)
  const values = indicatorData.value.map(p => p.value)
  const theme = useChartTheme()

  // Resolve signal colors from CSS variables
  const upColor = 'var(--color-up)'
  const downColor = 'var(--color-down)'
  const axisColor = theme.axisColor

  const lineColor = selectedSignal.value?.signal === 'bullish'
    ? upColor : selectedSignal.value?.signal === 'bearish'
    ? downColor : axisColor

  return {
    tooltip: {
      trigger: 'axis' as const,
      formatter: (params: any) => `${params[0].axisValue}<br/>${selectedSignal.value!.name_cn}: ${params[0].value}`
    },
    grid: { left: 60, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: dates,
      axisLabel: { rotate: 45, fontSize: 10 }
    },
    yAxis: {
      type: 'value' as const,
      name: selectedSignal.value?.unit || '',
      axisLabel: { fontSize: 10 }
    },
    series: [{
      type: 'line',
      data: values,
      smooth: true,
      areaStyle: {
        color: selectedSignal.value?.signal === 'bullish'
          ? upColor + '1A' : selectedSignal.value?.signal === 'bearish'
          ? downColor + '1A' : axisColor + '1A'
      },
      lineStyle: { color: lineColor, width: 2 },
      itemStyle: { color: lineColor },
      markLine: {
        silent: true,
        data: [{ type: 'average', name: t('misc.mean') }],
        lineStyle: { color: 'var(--color-accent)', type: 'dashed' }
      },
      showSymbol: false
    }]
  }
})

function formatValue(v: number | null | undefined, unit: string): string {
  if (v == null) return 'N/A'
  if (unit === '%') return v.toFixed(2) + '%'
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  if (v >= 1) return v.toFixed(2)
  return v.toFixed(4)
}

function formatChange(c: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)}%`
}

function changeClass(c: number): string {
  if (c > 0) return 'up'
  if (c < 0) return 'down'
  return 'muted'
}
</script>

<template>
  <div class="govdata-panel" :data-panel-id="panelId">
    <!-- Header: source switch + signal summary -->
    <PanelHeader
      :title="sourceCnLabels[activeSource]"
      :controls="controls"
    >
      <template #extra>
        <div class="signal-summary">
          <span class="summary-badge" v-if="signalCounts.bullish > 0">
            <SignalBadge signal="bullish" size="sm" /> {{ signalCounts.bullish }} 看涨
          </span>
          <span class="summary-badge" v-if="signalCounts.bearish > 0">
            <SignalBadge signal="bearish" size="sm" /> {{ signalCounts.bearish }} 看跌
          </span>
          <span class="summary-badge" v-if="signalCounts.neutral > 0">
            <SignalBadge signal="neutral" size="sm" /> {{ signalCounts.neutral }} 中性
          </span>
        </div>
      </template>
    </PanelHeader>

    <!-- Source switch tabs -->
    <PanelTabs
      variant="pill"
      :tabs="sources.map(s => ({ key: s.key, label: s.label }))"
      :active="activeSource"
      @change="(k: string) => { activeSource = k as any; activeCategory = 'all'; loadSignals() }"
    />

    <!-- Category filter tabs (dynamic per source) -->
    <PanelTabs
      v-if="categories.length > 2"
      variant="button"
      :tabs="categories.map(cat => ({ key: cat, label: categoryLabel(cat) }))"
      :active="activeCategory"
      @change="(k: string) => { activeCategory = k; selectedSignal = null }"
    />

    <!-- Main content: indicator grid + detail -->
    <PanelShell :state="state" :error="loadError" @retry="loadSignals">
      <template #loaded>
        <div class="content-area">
          <!-- Indicator cards grid -->
          <div class="indicator-grid" :class="{ 'with-detail': selectedSignal }">
            <div
              v-for="signal in filteredSignals"
              :key="signal.indicator_id"
              :class="['indicator-card', { selected: selectedSignal?.indicator_id === signal.indicator_id }]"
              @click="loadIndicatorDetail(signal)"
            >
              <div class="card-header">
                <span class="card-name">{{ signal.name_cn }}</span>
                <SignalBadge :signal="(signal.signal as 'bullish'|'bearish'|'neutral')" />
              </div>
              <div class="card-value">
                <template v-if="signal.latest_value != null && signal.latest_value !== 0 || signal.latest_date">
                  <span class="value">{{ formatValue(signal.latest_value, signal.unit) }}</span>
                </template>
                <span v-else class="card-tap">点击查看数据</span>
              </div>
              <div class="card-change">
                <TrendIndicator :direction="(signal.direction as 'up'|'down'|'flat')" />
                <span :class="['change-text', changeClass(signal.change)]">
                  {{ formatChange(signal.change) }}
                </span>
                <span class="card-unit">{{ signal.unit }}</span>
              </div>
            </div>
          </div>

          <!-- Detail panel -->
          <div v-if="selectedSignal" class="detail-panel">
            <div class="detail-header">
              <div>
                <h4>{{ selectedSignal.name_cn }}</h4>
                <p class="detail-subtitle" v-if="selectedSignal.latest_date">{{ selectedSignal.latest_date }}</p>
              </div>
              <button class="btn-close" @click="selectedSignal = null">&times;</button>
            </div>

            <!-- Signal info -->
            <div class="detail-info">
              <div class="info-row">
                <span class="info-label">{{ $t('macro.latest_value') }}</span>
                <span class="info-value">{{ formatValue(selectedSignal.latest_value, selectedSignal.unit) }}</span>
              </div>
              <div class="info-row">
                <span class="info-label">{{ $t('common.change') }}</span>
                <span :class="['info-value', changeClass(selectedSignal.change)]">
                  <TrendIndicator :direction="(selectedSignal.direction as 'up'|'down'|'flat')" /> {{ formatChange(selectedSignal.change) }}
                </span>
              </div>
              <div class="info-row">
                <span class="info-label">{{ $t('macro.signal') }}</span>
                <span class="info-value">
                  <SignalBadge :signal="(selectedSignal.signal as 'bullish'|'bearish'|'neutral')" /> {{ selectedSignal.signal === 'bullish' ? '看涨' : selectedSignal.signal === 'bearish' ? '看跌' : '中性' }}
                </span>
              </div>
              <div class="info-row">
                <span class="info-label">{{ $t('macro.unit') }}</span>
                <span class="info-value">{{ selectedSignal.unit }}</span>
              </div>
            </div>

            <!-- Time series chart -->
            <div class="chart-container" v-if="indicatorData.length > 0 && !chartLoading">
              <VChart :option="chartOption" style="height: 250px" autoresize />
            </div>
            <LoadingState v-else-if="chartLoading" type="inline" />
            <EmptyState v-else icon="inbox" :title="$t('macro.no_history')" />

            <!-- Trend summary -->
            <div class="trend-summary" v-if="selectedSignal.direction !== 'flat'">
              <span :class="['trend-text', changeClass(selectedSignal.change)]">
                <TrendIndicator :direction="(selectedSignal.direction as 'up'|'down'|'flat')" /> {{ selectedSignal.direction === 'up' ? '上升趋势' : '下降趋势' }}
              </span>
              <span v-if="selectedSignal.signal === 'bullish'">{{ $t('macro.positive_signal') }}</span>
              <span v-else-if="selectedSignal.signal === 'bearish'">{{ $t('macro.negative_signal') }}</span>
            </div>
          </div>
        </div>
      </template>
    </PanelShell>
  </div>
</template>

<style scoped>
.govdata-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  color: var(--color-text-primary);
}

.signal-summary {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.summary-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-xs);
  padding: var(--space-xs) var(--space-sm);
  border-radius: var(--radius-sm);
  background: var(--color-bg-subtle);
}

.content-area { display: flex; flex: 1; overflow: hidden; }

/* Indicator grid with responsive breakpoints */
.indicator-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  padding: var(--panel-padding);
  overflow-y: auto;
  align-content: start;
}
.indicator-grid.with-detail {
  flex: 0 0 55%;
}

/* Responsive: 2 columns at 600px, 1 column at 400px */
@media (max-width: 600px) {
  .indicator-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .indicator-grid.with-detail {
    flex: 0 0 45%;
  }
}
@media (max-width: 400px) {
  .indicator-grid {
    grid-template-columns: 1fr;
  }
  .indicator-grid.with-detail {
    flex: 0 0 40%;
  }
  .detail-panel {
    min-width: 200px;
  }
}
@media (max-width: 280px) {
  .indicator-grid {
    grid-template-columns: 1fr;
    padding: var(--panel-padding-sm);
  }
  .detail-panel {
    min-width: 160px;
    padding: var(--panel-padding-sm);
  }
}

.indicator-card {
  display: flex;
  flex-direction: column;
  padding: var(--card-padding);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  min-width: 0;
}
.indicator-card:hover {
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}
.indicator-card.selected {
  border-color: var(--color-accent);
  background: var(--color-bg-selected);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  gap: 4px;
}
.card-name {
  font-size: var(--font-xs);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-value { margin-bottom: 4px; }
.value {
  font-size: var(--font-xl);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.card-tap {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

.card-change {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-xs);
}
.change-text { font-variant-numeric: tabular-nums; }
.card-unit {
  color: var(--color-text-tertiary);
  font-size: var(--font-xs);
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.up { color: var(--color-up); }
.down { color: var(--color-down); }
.muted { color: var(--color-text-tertiary); }

/* Detail panel */
.detail-panel {
  flex: 1;
  border-left: 1px solid var(--color-border);
  padding: var(--panel-padding);
  overflow-y: auto;
  min-width: 300px;
}
.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}
.detail-header h4 { margin: 0; font-size: var(--font-lg); }
.detail-subtitle {
  margin: var(--space-xs) 0 0 0;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}
.btn-close {
  background: none; border: none; font-size: var(--font-lg);
  color: var(--color-text-secondary); cursor: pointer;
}

.detail-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  margin-bottom: 12px;
  padding: var(--card-padding);
  background: var(--color-bg-subtle);
  border-radius: var(--radius-md);
}
.info-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.info-label {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.info-value {
  font-size: var(--font-sm);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  display: flex;
  align-items: center;
  gap: 4px;
}

.chart-container { margin-bottom: 12px; }

.trend-summary {
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
  font-size: var(--font-xs);
  line-height: 1.5;
  display: flex;
  align-items: center;
  gap: 8px;
}
.trend-text { font-weight: 600; display: flex; align-items: center; gap: 4px; }
</style>
