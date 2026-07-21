<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import { graphic } from 'echarts/core'
import VChart from 'vue-echarts'
import { usePortfolioStore } from '@/stores/portfolio'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import type { PositionDetail } from '@/stores/portfolio'
import { logger } from '@/lib/logger'
import { computeDrawdowns, sharpeRatio } from '@/lib/stats'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const store = usePortfolioStore()
const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

// --- Tab state ---
const activeTab = ref<'overview' | 'risk' | 'chart'>('overview')

// --- Data (reactive via store) ---
const positions = computed<PositionDetail[]>(() => store.positions ?? [])

const kpi = computed(() => {
  const s = store.summary
  if (!s) return null
  return {
    total_value: s.total_value,
    cash_balance: s.cash_balance,
    market_value: s.market_value,
    total_pnl: s.total_pnl,
    total_pnl_pct: s.total_pnl_pct,
  }
})

const equityData = computed(() => {
  const curve = store.equityCurve
  if (!curve || curve.length === 0) return []
  return curve.map((p) => ({ date: p.date, equity: p.nav }))
})

const marketColors: Record<string, string> = {
  US: '#388e3c', CN: '#d32f2f', CRYPTO: '#f57c00', HK: '#1976d2',
}
let paletteIdx = 0
const fallbackPalette = ['#388e3c', '#d32f2f', '#f57c00', '#1976d2', '#8e24aa', '#00838f']

function colorForMarket(market: string): string {
  if (marketColors[market]) return marketColors[market]
  return fallbackPalette[paletteIdx++ % fallbackPalette.length]
}

const allocationData = computed(() => {
  paletteIdx = 0
  const byMarket = store.allocation?.by_market
  if (!byMarket) return []
  return Object.entries(byMarket).map(([market, value]) => ({
    market, value, color: colorForMarket(market),
  }))
})

// --- Shared: data for Risk + Chart tabs ---
const navSeries = computed<number[]>(() => (store.equityCurve || []).map(p => p.nav))
const datesArr = computed<string[]>(() => (store.equityCurve || []).map(p => p.date))
const benchmarkValues = computed<number[]>(() => (store.equityCurve || []).map(p => p.benchmark))

const dailyReturnsArr = computed<number[]>(() => {
  const navs = navSeries.value
  if (navs.length < 2) return []
  const rets: number[] = []
  for (let i = 1; i < navs.length; i++) if (navs[i - 1] > 0) rets.push(navs[i] / navs[i - 1] - 1)
  return rets
})

const drawdownsArr = computed(() => computeDrawdowns(navSeries.value))
const sharpeVal = computed(() => sharpeRatio(dailyReturnsArr.value))

// --- Risk tab ---
const totalExposure = computed(() =>
  (store.positions || []).reduce((s, p) => s + (p.market_price || 0) * (p.quantity || 0), 0),
)

const maxDDInfo = computed(() => {
  const navs = navSeries.value
  if (navs.length < 2) return { dd: 0, start: '', end: '' }
  let peak = navs[0], dd = 0, ddStart = 0, ddEnd = 0, peakIdx = 0
  for (let i = 1; i < navs.length; i++) {
    if (navs[i] > peak) { peak = navs[i]; peakIdx = i; continue }
    const d = (navs[i] / peak - 1) * 100
    if (d < dd) { dd = d; ddStart = peakIdx; ddEnd = i }
  }
  return { dd, start: datesArr.value[ddStart] || '', end: datesArr.value[ddEnd] || '' }
})

const annualVol = computed(() => {
  const rets = dailyReturnsArr.value
  if (rets.length < 5) return 0
  const mean = rets.reduce((a, b) => a + b, 0) / rets.length
  return Math.sqrt(rets.reduce((a, b) => a + (b - mean) ** 2, 0) / (rets.length - 1) * 252) * 100
})

const var95 = computed(() => {
  const rets = dailyReturnsArr.value
  if (rets.length < 10) return 0
  return [...rets].sort((a, b) => a - b)[Math.floor(rets.length * 0.05)] * totalExposure.value
})

const cvar95 = computed(() => {
  const rets = dailyReturnsArr.value
  if (rets.length < 10) return 0
  const sorted = [...rets].sort((a, b) => a - b)
  const tail = sorted.slice(0, Math.floor(sorted.length * 0.05) + 1)
  return tail.length > 0 ? (tail.reduce((a, b) => a + b, 0) / tail.length) * totalExposure.value : 0
})

const sortinoVal = computed(() => {
  const rets = dailyReturnsArr.value
  if (rets.length < 5) return 0
  const mean = rets.reduce((a, b) => a + b, 0) / rets.length
  const down = rets.filter(r => r < 0)
  if (down.length < 2) return 0
  const dv = down.reduce((a, b) => a + b ** 2, 0) / (down.length - 1)
  return Math.sqrt(dv) > 0 ? (mean * 252) / (Math.sqrt(dv) * Math.sqrt(252)) : 0
})

const riskKpiCards = computed(() => [
  { label: t('risk.var_95'), value: var95.value !== 0 ? '$' + fmtMoney(Math.abs(var95.value)) : '--', color: '#f0883e' },
  { label: t('risk.cvar_95'), value: cvar95.value !== 0 ? '$' + fmtMoney(Math.abs(cvar95.value)) : '--', color: '#f0883e' },
  { label: t('risk.max_drawdown'), value: maxDDInfo.value.dd !== 0 ? fmt(maxDDInfo.value.dd) + '%' : '--', color: maxDDInfo.value.dd < -10 ? '#f85149' : '#f0883e' },
  { label: t('risk.sharpe'), value: sharpeVal.value !== 0 ? fmt(sharpeVal.value, 2) : '--', color: sharpeVal.value > 1 ? '#3fb950' : sharpeVal.value > 0 ? '#f0883e' : '#f85149' },
  { label: t('risk.sortino'), value: sortinoVal.value !== 0 ? fmt(sortinoVal.value, 2) : '--', color: sortinoVal.value > 1 ? '#3fb950' : sortinoVal.value > 0 ? '#f0883e' : '#f85149' },
  { label: t('risk.annual_vol'), value: annualVol.value !== 0 ? fmt(annualVol.value) + '%' : '--', color: 'var(--color-text-tertiary)' },
])

const riskDrawdownChartOption = computed(() => {
  const theme = useChartTheme()
  const navs = navSeries.value
  const ddSeries: number[] = []
  let peak = navs[0] || 1
  for (const n of navs) { if (n > peak) peak = n; ddSeries.push(peak > 0 ? (n / peak - 1) * 100 : 0) }
  const labels = datesArr.value.map((d, i) => i % Math.max(1, Math.floor(datesArr.value.length / 6)) === 0 ? d.slice(5) : '')
  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 20, bottom: 30, left: 55 },
    xAxis: { type: 'category' as const, data: labels, axisLabel: { color: theme.axisColor, fontSize: 9 } },
    yAxis: { type: 'value' as const, axisLabel: { color: theme.axisColor, fontSize: 9, formatter: '{value}%' } },
    series: [{
      type: 'line', data: ddSeries, smooth: true,
      lineStyle: { color: '#f85149', width: 2 },
      areaStyle: { color: new graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(248,81,73,0.3)' }, { offset: 1, color: 'rgba(248,81,73,0.02)' }]) },
      symbol: 'none',
    }],
  }
})

// --- Chart tab ---
const chartLoading = ref(false)
const chartError = ref('')

const cumulativeReturn = computed(() => {
  const vals = navSeries.value
  if (vals.length < 2) return 0
  const first = vals[0], last = vals[vals.length - 1]
  if (!first || first === 0) return 0
  return ((last - first) / first) * 100
})

const annualizedReturn = computed(() => {
  const vals = navSeries.value
  if (vals.length < 2) return 0
  const years = vals.length / 252
  if (years === 0) return 0
  const first = vals[0], last = vals[vals.length - 1]
  if (!first || first === 0) return 0
  return (Math.pow(last / first, 1 / years) - 1) * 100
})

const maxDD = computed(() => {
  const dd = drawdownsArr.value.map(d => d.drawdown)
  if (dd.length === 0) return 0
  return Math.min(...dd) * 100
})

const calmarVal = computed(() => {
  if (Math.abs(maxDD.value) === 0) return 0
  return annualizedReturn.value / Math.abs(maxDD.value)
})

const chartMetricCards = computed(() => [
  { label: t('portfolio.total_pnl'), value: cumulativeReturn.value.toFixed(2) + '%', color: cumulativeReturn.value >= 0 ? '#22c55e' : '#ef4444' },
  { label: t('monteCarlo.annual_return'), value: annualizedReturn.value.toFixed(2) + '%', color: annualizedReturn.value >= 0 ? '#22c55e' : '#ef4444' },
  { label: t('risk.max_drawdown'), value: maxDD.value.toFixed(2) + '%', color: maxDD.value < -20 ? '#ef4444' : '#f59e0b' },
  { label: t('risk.sharpe'), value: sharpeVal.value.toFixed(2), color: sharpeVal.value >= 1 ? '#22c55e' : sharpeVal.value >= 0.5 ? '#f59e0b' : '#ef4444' },
  { label: 'Calmar Ratio', value: calmarVal.value.toFixed(2), color: calmarVal.value >= 1 ? '#22c55e' : calmarVal.value >= 0.3 ? '#f59e0b' : '#ef4444' },
])

const chartEquityOption = computed(() => {
  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 20, bottom: 30, left: 60 },
    xAxis: {
      type: 'category' as const, data: datesArr.value,
      axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: string) => v.slice(5) },
    },
    yAxis: {
      type: 'value' as const,
      axisLabel: { color: theme.axisColor, fontSize: 10 },
      splitLine: { lineStyle: { color: theme.bgColor } },
    },
    tooltip: { trigger: 'axis' as const },
    legend: { textStyle: { color: theme.axisColor, fontSize: 11 }, top: 0 },
    series: [
      {
        name: 'NAV', type: 'line' as const, data: navSeries.value, smooth: true,
        lineStyle: { color: '#58a6ff', width: 2 },
        areaStyle: { color: new graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: 'rgba(88,166,255,0.25)' }, { offset: 1, color: 'rgba(88,166,255,0.02)' }]) },
        symbol: 'none',
      },
      {
        name: 'Benchmark', type: 'line' as const, data: benchmarkValues.value, smooth: true,
        lineStyle: { color: theme.axisColor, width: 1.5, type: 'dashed' as const },
        symbol: 'none',
      },
    ],
  }
})

const chartDrawdownOption = computed(() => {
  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 20, bottom: 30, left: 60 },
    xAxis: {
      type: 'category' as const, data: datesArr.value,
      axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: string) => v.slice(5) },
    },
    yAxis: {
      type: 'value' as const,
      axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => (v * 100).toFixed(0) + '%' },
      splitLine: { lineStyle: { color: theme.bgColor } },
    },
    tooltip: {
      trigger: 'axis' as const,
      valueFormatter: (v: unknown) => {
        const num = typeof v === 'number' ? v : Number(v)
        return (num * 100).toFixed(2) + '%'
      },
    },
    series: [{
      name: 'Drawdown', type: 'line' as const, data: drawdownsArr.value.map(d => d.drawdown),
      smooth: true, lineStyle: { color: '#ef4444', width: 1.5 },
      areaStyle: { color: 'rgba(239,68,68,0.25)' }, symbol: 'none',
    }],
  }
})

const hasChartData = computed(() => store.equityCurve && store.equityCurve.length > 0)

async function refreshChart() {
  chartLoading.value = true
  chartError.value = ''
  try { await store.fetchEquityCurve() } catch (e: any) { chartError.value = e?.message || String(e) } finally { chartLoading.value = false }
}

// --- Lifecycle ---
onMounted(async () => {
  store.startAutoRefresh()
  store.fetchEquityCurve()
})

onUnmounted(() => {
  store.stopAutoRefresh()
})

// --- Helpers ---
function fmt(n: number, dec = 2): string { return n.toFixed(dec) }

function currencyForSymbol(symbol: string): string {
  if (symbol.endsWith('.SZ') || symbol.endsWith('.SH') || symbol.endsWith('.BJ')) return '¥'
  if (symbol.endsWith('.HK')) return 'HK$'
  if (symbol.includes('USDT') || symbol === 'BTC' || symbol === 'ETH') return 'USDT'
  return '$'
}

function fmtMoney(n: number, symbol?: string): string {
  const c = symbol ? currencyForSymbol(symbol) : '$'
  if (Math.abs(n) >= 1e8) return c + (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return c + (n / 1e4).toFixed(1) + '万'
  return c + n.toFixed(2)
}

function pnlClass(v: number): string { return v >= 0 ? 'up' : 'down' }
function pnlSign(v: number): string { return v >= 0 ? '+' : '' }

function marketClass(market: string): string {
  return 'market-badge market-' + market.toLowerCase()
}

function onPositionClick(pos: PositionDetail) {
  logger.info('[PortfolioSummary] navigate to PositionDetail:', pos.symbol)
}

// --- 持仓 table helpers ---
const totalMarketValue = computed(() =>
  positions.value.reduce((s, p) => s + p.market_price * p.quantity, 0),
)

function positionAllocPct(pos: PositionDetail): string {
  const total = totalMarketValue.value
  if (!total) return '0.0'
  return ((pos.market_price * pos.quantity / total) * 100).toFixed(1)
}

// --- Overview tab: equity curve chart ---
const overviewEquityChartOption = computed(() => {
  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 12, bottom: 30, left: 55 },
    xAxis: {
      type: 'category' as const,
      data: equityData.value.map((p) => p.date),
      axisLine: { lineStyle: { color: 'var(--color-border)' } },
      axisLabel: { color: theme.axisColor, fontSize: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value' as const,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: theme.axisColor, fontSize: 10,
        formatter: (v: number) => (v / 1000).toFixed(0) + 'k',
      },
      splitLine: { lineStyle: { color: 'var(--color-bg-input)' } },
    },
    series: [{
      type: 'line',
      data: equityData.value.map((p) => p.equity),
      smooth: true, symbol: 'none',
      lineStyle: { color: '#58a6ff', width: 2 },
      areaStyle: {
        color: new graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(88, 166, 255, 0.28)' },
          { offset: 1, color: 'rgba(88, 166, 255, 0.02)' },
        ]),
      },
    }],
    tooltip: {
      trigger: 'axis' as const,
      backgroundColor: 'var(--color-bg-subtle)',
      borderColor: 'var(--color-border)',
      textStyle: { color: theme.textColor, fontSize: 12 },
      formatter: (params: any) => {
        const p = params[0]
        return `${p.name}<br/>Equity: <b>$${(p.value as number).toLocaleString()}</b>`
      },
    },
  }
})

// --- Overview tab: allocation pie chart ---
const pieChartOption = computed(() => {
  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item' as const,
      backgroundColor: 'var(--color-bg-subtle)',
      borderColor: 'var(--color-border)',
      textStyle: { color: theme.textColor, fontSize: 12 },
      formatter: (params: any) =>
        `<b>${params.name}</b><br/>${t('portfolio.allocation')}: ${params.value}%`,
    },
    series: [{
      type: 'pie',
      radius: ['45%', '75%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 2, borderColor: 'var(--color-bg-panel)', borderWidth: 2 },
      label: {
        show: true, position: 'outside' as const,
        color: theme.axisColor, fontSize: 10,
        formatter: '{b}\n{d}%',
      },
      labelLine: { lineStyle: { color: 'var(--color-border)' } },
      data: allocationData.value.map((a) => ({
        name: a.market, value: a.value,
        itemStyle: { color: a.color },
      })),
    }],
  }
})
</script>

<template>
  <div class="portfolio-panel">
    <!-- Tab bar -->
    <div class="tab-bar">
      <button :class="['tab-btn', { active: activeTab === 'overview' }]" @click="activeTab = 'overview'">
        {{ t('research.overview') }}
      </button>
      <button :class="['tab-btn', { active: activeTab === 'risk' }]" @click="activeTab = 'risk'">
        {{ t('risk.dashboard') }}
      </button>
      <button :class="['tab-btn', { active: activeTab === 'chart' }]" @click="activeTab = 'chart'">
        {{ t('portfolio.equity_curve') }}
      </button>
      <span class="tab-actions">
        <button v-if="addToWfControl" class="wf-btn" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
      </span>
    </div>

    <!-- Tab: Overview -->
    <div v-if="activeTab === 'overview'" class="tab-content">
      <!-- KPI Cards -->
      <div class="kpi-row">
        <div class="kpi-card">
          <span class="kpi-label">{{ t('portfolio.total_value') }}</span>
          <span class="kpi-value">{{ kpi ? fmtMoney(kpi.total_value) : '--' }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">{{ t('portfolio.cash') }}</span>
          <span class="kpi-value">{{ kpi ? fmtMoney(kpi.cash_balance) : '--' }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">{{ t('portfolio.market_value') }}</span>
          <span class="kpi-value">{{ kpi ? fmtMoney(kpi.market_value) : '--' }}</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">{{ t('portfolio.total_pnl') }}</span>
          <span v-if="kpi" class="kpi-value" :class="pnlClass(kpi.total_pnl)">
            {{ pnlSign(kpi.total_pnl) }}{{ fmtMoney(kpi.total_pnl) }}
          </span>
          <span v-else class="kpi-value">--</span>
        </div>
        <div class="kpi-card">
          <span class="kpi-label">{{ t('portfolio.pnl_pct') }}</span>
          <span v-if="kpi" class="kpi-value" :class="pnlClass(kpi.total_pnl_pct)">
            {{ pnlSign(kpi.total_pnl_pct) }}{{ fmt(kpi.total_pnl_pct) }}%
          </span>
          <span v-else class="kpi-value">--</span>
        </div>
      </div>

      <!-- Charts Row -->
      <div class="charts-row">
        <div class="chart-box chart-equity">
          <h3 class="section-title">{{ t('portfolio.equity_curve') }}</h3>
          <VChart v-if="equityData.length > 0" class="chart-body" :option="overviewEquityChartOption" autoresize />
          <div v-else class="chart-empty">--</div>
        </div>
        <div class="chart-box chart-pie">
          <h3 class="section-title">{{ t('portfolio.allocation') }}</h3>
          <VChart v-if="allocationData.length > 0" class="chart-body" :option="pieChartOption" autoresize />
          <div v-else class="chart-empty">--</div>
        </div>
      </div>

      <!-- 持仓 Table -->
      <div class="positions-section">
        <h3 class="section-title">{{ t('portfolio.positions') }}</h3>
        <div class="table-wrap">
          <table class="pos-table">
            <thead>
              <tr>
                <th>{{ t('portfolio.symbol') }}</th>
                <th>{{ t('common.name') }}</th>
                <th>{{ t('portfolio.market') }}</th>
                <th class="num">{{ t('portfolio.quantity') }}</th>
                <th class="num">{{ t('portfolio.avg_price') }}</th>
                <th class="num">{{ t('portfolio.market_price') }}</th>
                <th class="num">{{ t('portfolio.pnl') }}</th>
                <th class="num">%</th>
                <th class="num">{{ t('portfolio.alloc') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="pos in positions"
                :key="pos.symbol"
                class="pos-row"
                @click="onPositionClick(pos)"
              >
                <td class="pos-symbol">{{ pos.symbol }}</td>
                <td class="pos-name">{{ pos.name || '' }}</td>
                <td><span :class="marketClass(pos.market)">{{ pos.market }}</span></td>
                <td class="num">{{ pos.quantity }}</td>
                <td class="num">{{ pos.avg_price.toFixed(2) }}</td>
                <td class="num">{{ fmtMoney(pos.market_price * pos.quantity, pos.symbol) }}</td>
                <td class="num" :class="pnlClass(pos.pnl)">
                  {{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl, pos.symbol) }}
                </td>
                <td class="num" :class="pnlClass(pos.pnl_pct)">
                  {{ pnlSign(pos.pnl_pct) }}{{ fmt(pos.pnl_pct) }}%
                </td>
                <td class="num">{{ positionAllocPct(pos) }}%</td>
              </tr>
              <tr v-if="positions.length === 0">
                <td colspan="9" class="empty-state-cell">--</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Tab: Risk -->
    <div v-if="activeTab === 'risk'" class="tab-content">
      <div v-if="!hasChartData && store.positions.length === 0" class="risk-empty">
        {{ t('common.no_data') }} — 需要持仓和净值历史数据
      </div>
      <template v-else>
        <div v-if="totalExposure > 0" class="risk-header">
          <span class="exposure-badge">{{ t('risk.total_exposure') }} ${{ fmtMoney(totalExposure) }}</span>
        </div>
        <div class="risk-kpi-grid">
          <div v-for="card in riskKpiCards" :key="card.label" class="risk-kpi-card" :style="{ borderLeft: `3px solid ${card.color}` }">
            <span class="risk-kpi-label">{{ card.label }}</span>
            <span class="risk-kpi-value" :style="{ color: card.color }">{{ card.value }}</span>
          </div>
        </div>
        <div v-if="navSeries.length > 1" class="risk-chart-section">
          <VChart :option="riskDrawdownChartOption" autoresize style="height:220px" />
        </div>
        <div v-if="maxDDInfo.dd < 0" class="risk-dd-info">
          Peak → Trough: {{ maxDDInfo.start }} → {{ maxDDInfo.end }} ({{ fmt(maxDDInfo.dd) }}%)
        </div>
      </template>
    </div>

    <!-- Tab: Chart -->
    <div v-if="activeTab === 'chart'" class="tab-content chart-tab">
      <div class="chart-panel-header">
        <h3>{{ t('portfolio.equity_curve') }}</h3>
        <button class="refresh-btn" @click="refreshChart">&#x21bb;</button>
      </div>

      <div v-if="chartError" class="chart-error">{{ chartError }}</div>
      <div v-if="chartLoading" class="chart-fallback">{{ $t('common.loading') }}</div>
      <div v-else-if="hasChartData" class="chart-curve-content">
        <div class="chart-section-top">
          <VChart :option="chartEquityOption" autoresize class="chart-equity-chart" />
        </div>
        <div class="chart-section-bottom">
          <VChart :option="chartDrawdownOption" autoresize class="chart-drawdown-chart" />
        </div>
        <div class="chart-stats-row">
          <div v-for="card in chartMetricCards" :key="card.label" class="chart-stat-card">
            <span class="chart-stat-label">{{ card.label }}</span>
            <span class="chart-stat-value" :style="{ color: card.color }">{{ card.value }}</span>
          </div>
        </div>
      </div>

      <div v-else class="chart-empty-state">
        <p>{{ t('common.loading') }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.portfolio-panel {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  overflow-y: auto;
  font-variant-numeric: tabular-nums;
  color: var(--text);
  font-size: var(--font-sm);
}

/* --- Tab Bar --- */
.tab-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 10px;
  border-bottom: 1px solid var(--input);
  padding-bottom: 4px;
}

.tab-btn {
  background: none;
  border: none;
  color: var(--muted);
  font-size: var(--font-sm);
  padding: 4px 12px;
  cursor: pointer;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  transition: color 0.15s, background 0.15s;
}

.tab-btn:hover { color: var(--text); }
.tab-btn.active {
  color: var(--text);
  background: var(--card);
  border: 1px solid var(--input);
  border-bottom-color: var(--card);
  margin-bottom: -5px;
}

.tab-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.wf-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: var(--font-lg);
  font-weight: 600;
  cursor: pointer;
  line-height: 1;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.wf-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(88, 166, 255, 0.1);
}

/* --- KPI Cards --- */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: var(--spacing);
  margin-bottom: 10px;
}

.kpi-card {
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 10px 8px;
  text-align: center;
}

.kpi-label {
  display: block;
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  margin-bottom: 4px;
}

.kpi-value {
  font-size: var(--font-lg);
  font-weight: 700;
  color: var(--text);
}

.kpi-value.up { color: var(--up); }
.kpi-value.down { color: var(--down); }

/* --- Charts Row --- */
.charts-row {
  display: flex;
  gap: var(--spacing);
  margin-bottom: 10px;
  min-height: 0;
}

.chart-box {
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 8px;
  display: flex;
  flex-direction: column;
}

.chart-box .section-title { margin-bottom: 4px; }
.chart-equity { flex: 0 0 60%; min-width: 0; }
.chart-pie { flex: 0 0 40%; min-width: 0; }

.chart-body { flex: 1; min-height: 150px; }

.chart-empty {
  flex: 1; min-height: 150px;
  display: flex; align-items: center; justify-content: center;
  color: var(--muted); font-size: var(--font-sm);
}

/* --- Section Title --- */
.section-title {
  font-size: var(--font-xs);
  text-transform: uppercase;
  color: var(--muted);
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--input);
}

/* --- 持仓 Table --- */
.positions-section {
  background: var(--card);
  border-radius: var(--radius-sm);
  padding: 8px;
}

.table-wrap { overflow-x: auto; }

.pos-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-xs);
}

.pos-table th {
  text-align: left; padding: 5px 8px;
  font-size: var(--font-xs); color: var(--muted); font-weight: 500;
  text-transform: uppercase; border-bottom: 1px solid var(--input); white-space: nowrap;
}

.pos-table th.num { text-align: right; }

.pos-table td {
  padding: 5px 8px; border-bottom: 1px solid var(--input); white-space: nowrap;
}

.pos-table td.num { text-align: right; font-variant-numeric: tabular-nums; }

.pos-row { cursor: pointer; transition: background 0.15s; }
.pos-row:hover { background: var(--input); }

.pos-symbol { font-weight: 600; font-size: var(--font-sm); color: var(--text); }
.pos-name { font-size: var(--font-xs); color: var(--muted); }

.empty-state-cell { text-align: center; color: var(--muted); padding: 24px; }

.up { color: var(--up); }
.down { color: var(--down); }

/* --- Market Badges --- */
.market-badge {
  display: inline-block; padding: 1px 6px; border-radius: var(--radius-sm);
  font-size: var(--font-xs); font-weight: 600;
}

.market-us {
  color: var(--color-down); background: rgba(56, 142, 60, 0.15);
  border: 1px solid rgba(56, 142, 60, 0.3);
}

.market-cn {
  color: var(--color-up); background: rgba(211, 47, 47, 0.15);
  border: 1px solid rgba(211, 47, 47, 0.3);
}

.market-hk {
  color: var(--color-accent); background: rgba(25, 118, 210, 0.15);
  border: 1px solid rgba(25, 118, 210, 0.3);
}

.market-crypto {
  color: var(--color-accent); background: rgba(245, 124, 0, 0.15);
  border: 1px solid rgba(245, 124, 0, 0.3);
}

/* --- Risk Tab --- */
.tab-content { min-height: 0; }

.risk-empty {
  display: flex; align-items: center; justify-content: center;
  padding: 40px 0; color: var(--color-text-tertiary); font-size: var(--font-sm);
}

.risk-header { margin-bottom: 10px; }

.exposure-badge {
  font-size: var(--font-xs); padding: 2px 8px; border-radius: var(--radius-sm);
  background: rgba(240, 136, 62, 0.15); color: var(--color-accent);
  font-family: 'JetBrains Mono', monospace;
}

.risk-kpi-grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin-bottom: 12px;
}

.risk-kpi-card {
  padding: 12px; background: var(--color-bg-elevated, var(--color-bg-panel));
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-subtle);
}

.risk-kpi-label {
  display: block; font-size: var(--font-xs); color: var(--color-text-tertiary);
  text-transform: uppercase; margin-bottom: 4px;
}

.risk-kpi-value { font-size: var(--font-xl); font-weight: 700; }

.risk-chart-section {
  background: var(--color-bg-elevated, var(--color-bg-panel));
  border-radius: var(--radius-lg); padding: 8px; margin-bottom: 8px;
  border: 1px solid var(--color-border-subtle);
}

.risk-dd-info {
  text-align: center; font-size: var(--font-xs); color: var(--color-text-tertiary);
  font-variant-numeric: tabular-nums;
}

/* --- Chart Tab --- */
.chart-tab {
  display: flex; flex-direction: column;
  overflow: hidden;
}

.chart-panel-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 8px; flex-shrink: 0;
}

.chart-panel-header h3 { margin: 0; font-size: var(--font-sm); font-weight: 600; }

.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm); background: var(--color-bg-elevated);
  color: var(--color-text-primary); cursor: pointer; font-size: var(--font-sm);
}

.refresh-btn:hover { background: var(--color-border-strong); }

.chart-error {
  color: var(--color-danger); font-size: var(--font-sm); margin-bottom: 8px;
}

.chart-fallback {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: var(--font-sm);
}

.chart-curve-content {
  flex: 1; display: flex; flex-direction: column; gap: 8px; overflow: hidden;
}

.chart-section-top { flex: 7; min-height: 0; }
.chart-section-bottom { flex: 3; min-height: 0; }

.chart-equity-chart, .chart-drawdown-chart { width: 100%; height: 100%; }

.chart-stats-row {
  flex-shrink: 0; display: flex; gap: 8px; overflow-x: auto; padding-top: 4px;
}

.chart-stat-card {
  flex: 1; min-width: 100px; padding: 8px 10px;
  border-radius: var(--radius-md); background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  display: flex; flex-direction: column; gap: 2px;
}

.chart-stat-label {
  font-size: var(--font-xs); color: var(--color-text-secondary); white-space: nowrap;
}

.chart-stat-value {
  font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums;
}

.chart-empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: var(--font-sm);
}
</style>
