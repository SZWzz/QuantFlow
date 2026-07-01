<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePortfolioStore } from '@/stores/portfolio'
import { computeDrawdowns, sharpeRatio } from '@/lib/stats'
import VChart from 'vue-echarts'
import { use, graphic } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

const loading = ref(false)
const loadError = ref('')

const hasEcharts = computed(() => {
  try {
    return !!VChart
  } catch {
    return false
  }
})

const navs = computed<number[]>(() => {
  return (store.equityCurve ?? []).map((p) => p.nav)
})

const benchmarkValues = computed<number[]>(() => {
  return (store.equityCurve ?? []).map((p) => p.benchmark)
})

const dates = computed<string[]>(() => {
  return (store.equityCurve ?? []).map((p) => p.date)
})

const dailyReturns = computed<number[]>(() => {
  const vals = navs.value
  const rets: number[] = []
  for (let i = 1; i < vals.length; i++) {
    if (vals[i - 1] && vals[i - 1] > 0) {
      rets.push((vals[i] - vals[i - 1]) / vals[i - 1])
    }
  }
  return rets
})

const drawdowns = computed(() => {
  return computeDrawdowns(navs.value)
})

const cumulativeReturn = computed(() => {
  const vals = navs.value
  if (vals.length < 2) return 0
  const first = vals[0]
  const last = vals[vals.length - 1]
  if (!first || first === 0) return 0
  return ((last - first) / first) * 100
})

const annualizedReturn = computed(() => {
  const vals = navs.value
  if (vals.length < 2) return 0
  const years = vals.length / 252
  if (years === 0) return 0
  const first = vals[0]
  const last = vals[vals.length - 1]
  if (!first || first === 0) return 0
  const totalReturn = last / first
  return (Math.pow(totalReturn, 1 / years) - 1) * 100
})

const maxDrawdown = computed(() => {
  const dd = drawdowns.value.map((d) => d.drawdown)
  if (dd.length === 0) return 0
  return Math.min(...dd) * 100
})

const sharpe = computed(() => {
  return sharpeRatio(dailyReturns.value)
})

const calmarRatio = computed(() => {
  const annRet = annualizedReturn.value
  const maxDd = Math.abs(maxDrawdown.value)
  if (maxDd === 0) return 0
  return annRet / maxDd
})

const equityChartOption = computed(() => {
  const theme = useChartTheme()
  return {
  backgroundColor: 'transparent',
  grid: { top: 10, right: 20, bottom: 30, left: 60 },
  xAxis: {
    type: 'category',
    data: dates.value,
    axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: string) => v.slice(5) },
  },
  yAxis: {
    type: 'value',
    axisLabel: { color: theme.axisColor, fontSize: 10 },
    splitLine: { lineStyle: { color: theme.bgColor } },
  },
  tooltip: { trigger: 'axis' },
  legend: { textStyle: { color: theme.axisColor, fontSize: 11 }, top: 0 },
  series: [
    {
      name: 'NAV',
      type: 'line',
      data: navs.value,
      smooth: true,
      lineStyle: { color: '#58a6ff', width: 2 },
      areaStyle: {
        color: new graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(88,166,255,0.25)' },
          { offset: 1, color: 'rgba(88,166,255,0.02)' },
        ]),
      },
      symbol: 'none',
    },
    {
      name: 'Benchmark',
      type: 'line',
      data: benchmarkValues.value,
      smooth: true,
      lineStyle: { color: theme.axisColor, width: 1.5, type: 'dashed' },
      symbol: 'none',
    },
  ],
}
})

const ddChartOption = computed(() => {
  const theme = useChartTheme()
  return {
  backgroundColor: 'transparent',
  grid: { top: 10, right: 20, bottom: 30, left: 60 },
  xAxis: {
    type: 'category',
    data: dates.value,
    axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: string) => v.slice(5) },
  },
  yAxis: {
    type: 'value',
    axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => (v * 100).toFixed(0) + '%' },
    splitLine: { lineStyle: { color: theme.bgColor } },
  },
  tooltip: {
    trigger: 'axis',
    valueFormatter: (v: unknown) => {
      const num = typeof v === 'number' ? v : Number(v)
      return (num * 100).toFixed(2) + '%'
    },
  },
  series: [
    {
      name: 'Drawdown',
      type: 'line',
      data: drawdowns.value.map((d) => d.drawdown),
      smooth: true,
      lineStyle: { color: '#ef4444', width: 1.5 },
      areaStyle: {
        color: 'rgba(239,68,68,0.25)',
      },
      symbol: 'none',
    },
  ],
}
})

const metricCards = computed(() => [
  { label: t('portfolio.total_pnl'), value: cumulativeReturn.value.toFixed(2) + '%', color: cumulativeReturn.value >= 0 ? '#22c55e' : '#ef4444' },
  { label: t('monteCarlo.annual_return'), value: annualizedReturn.value.toFixed(2) + '%', color: annualizedReturn.value >= 0 ? '#22c55e' : '#ef4444' },
  { label: t('risk.max_drawdown'), value: maxDrawdown.value.toFixed(2) + '%', color: maxDrawdown.value < -20 ? '#ef4444' : '#f59e0b' },
  { label: t('risk.sharpe'), value: sharpe.value.toFixed(2), color: sharpe.value >= 1 ? '#22c55e' : sharpe.value >= 0.5 ? '#f59e0b' : '#ef4444' },
  { label: 'Calmar Ratio', value: calmarRatio.value.toFixed(2), color: calmarRatio.value >= 1 ? '#22c55e' : calmarRatio.value >= 0.3 ? '#f59e0b' : '#ef4444' },
])

async function refresh() {
  loading.value = true
  loadError.value = ''
  try {
    await store.fetchEquityCurve()
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="equity-curve-panel">
    <div class="panel-header">
      <h3>{{ t('portfolio.equity_curve') }}</h3>
      <div class="header-controls">
        <button class="refresh-btn" @click="refresh">&#x21bb;</button>
      </div>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-if="loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <div v-else-if="store.equityCurve && store.equityCurve.length > 0" class="curve-content">
      <!-- Top section: Equity curve chart (70% height) -->
      <div class="chart-section-top">
        <VChart v-if="hasEcharts" :option="equityChartOption" autoresize class="equity-chart" />
        <div v-else class="fallback-table-wrap">
          <table class="fallback-table">
            <thead>
              <tr><th>{{ t('common.date') }}</th><th>{{ $t('misc.nav') }}</th><th>{{ $t('misc.benchmark') }}</th></tr>
            </thead>
            <tbody>
              <tr v-for="(pt, idx) in store.equityCurve" :key="idx">
                <td>{{ pt.date }}</td>
                <td>{{ pt.nav.toLocaleString() }}</td>
                <td>{{ pt.benchmark.toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Bottom section: Drawdown chart (30% height) -->
      <div class="chart-section-bottom">
        <VChart v-if="hasEcharts" :option="ddChartOption" autoresize class="drawdown-chart" />
        <div v-else class="fallback-msg">{{ t('misc.echarts_missing') }}</div>
      </div>

      <!-- Stats row: metric cards -->
      <div class="stats-row">
        <div v-for="card in metricCards" :key="card.label" class="stat-card">
          <span class="stat-label">{{ card.label }}</span>
          <span class="stat-value" :style="{ color: card.color }">{{ card.value }}</span>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>{{ store.equityCurve ? t('common.loading') : t('common.no_data') }}</p>
    </div>
  </div>
</template>

<style scoped>
.equity-curve-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; align-items: center; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:hover { background: var(--color-border-strong); }

.curve-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow: hidden;
}

.chart-section-top {
  flex: 7;
  min-height: 0;
}
.chart-section-bottom {
  flex: 3;
  min-height: 0;
}

.equity-chart, .drawdown-chart {
  width: 100%;
  height: 100%;
}

.fallback-table-wrap {
  max-height: 100%;
  overflow-y: auto;
}
.fallback-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.fallback-table th {
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  padding: 4px 8px;
  text-align: right;
  border-bottom: 1px solid var(--color-border-strong);
}
.fallback-table th:first-child { text-align: left; }
.fallback-table td {
  padding: 3px 8px;
  text-align: right;
  border-bottom: 1px solid var(--color-bg-elevated);
  font-variant-numeric: tabular-nums;
}
.fallback-table td:first-child { text-align: left; color: var(--color-text-secondary); }
.fallback-msg {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-tertiary);
  font-size: 12px;
}

.stats-row {
  flex-shrink: 0;
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-top: 4px;
}
.stat-card {
  flex: 1;
  min-width: 100px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.stat-label {
  font-size: 10px;
  color: var(--color-text-secondary);
  white-space: nowrap;
}
.stat-value {
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 13px;
}
</style>
