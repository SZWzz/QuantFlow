<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { graphic } from 'echarts/core'
import VChart from 'vue-echarts'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = usePortfolioStore()
const loading = ref(false)

const fmt = (n: number, dec = 2) => n.toFixed(dec)
const fmtMoney = (n: number) => n >= 1e8 ? (n / 1e8).toFixed(2) + '亿' : n >= 1e4 ? (n / 1e4).toFixed(1) + '万' : n.toFixed(0)

const navSeries = computed(() => (store.equityCurve || []).map(p => p.nav))

const dailyReturns = computed(() => {
  const navs = navSeries.value
  if (navs.length < 2) return [] as number[]
  const rets: number[] = []
  for (let i = 1; i < navs.length; i++) if (navs[i - 1] > 0) rets.push(navs[i] / navs[i - 1] - 1)
  return rets
})

const totalExposure = computed(() =>
  (store.positions || []).reduce((s, p) => s + (p.market_price || 0) * (p.quantity || 0), 0)
)

const maxDrawdown = computed(() => {
  const navs = navSeries.value
  if (navs.length < 2) return { dd: 0, start: '', end: '' }
  let peak = navs[0], dd = 0, ddStart = 0, ddEnd = 0, peakIdx = 0
  for (let i = 1; i < navs.length; i++) {
    if (navs[i] > peak) { peak = navs[i]; peakIdx = i; continue }
    const d = (navs[i] / peak - 1) * 100
    if (d < dd) { dd = d; ddStart = peakIdx; ddEnd = i }
  }
  const dates = (store.equityCurve || []).map(p => p.date)
  return { dd, start: dates[ddStart] || '', end: dates[ddEnd] || '' }
})

const annualVol = computed(() => {
  const rets = dailyReturns.value
  if (rets.length < 5) return 0
  const mean = rets.reduce((a, b) => a + b, 0) / rets.length
  return Math.sqrt(rets.reduce((a, b) => a + (b - mean) ** 2, 0) / (rets.length - 1) * 252) * 100
})

const sharpeRatio = computed(() => {
  const rets = dailyReturns.value
  if (rets.length < 5) return 0
  const mean = rets.reduce((a, b) => a + b, 0) / rets.length
  const std = Math.sqrt(rets.reduce((a, b) => a + (b - mean) ** 2, 0) / (rets.length - 1))
  return std > 0 ? (mean * 252) / (std * Math.sqrt(252)) : 0
})

const sortinoRatio = computed(() => {
  const rets = dailyReturns.value
  if (rets.length < 5) return 0
  const mean = rets.reduce((a, b) => a + b, 0) / rets.length
  const down = rets.filter(r => r < 0)
  if (down.length < 2) return 0
  const dv = down.reduce((a, b) => a + b ** 2, 0) / (down.length - 1)
  return Math.sqrt(dv) > 0 ? (mean * 252) / (Math.sqrt(dv) * Math.sqrt(252)) : 0
})

const var95 = computed(() => {
  const rets = dailyReturns.value
  if (rets.length < 10) return 0
  return [...rets].sort((a, b) => a - b)[Math.floor(rets.length * 0.05)] * totalExposure.value
})

const cvar95 = computed(() => {
  const rets = dailyReturns.value
  if (rets.length < 10) return 0
  const sorted = [...rets].sort((a, b) => a - b)
  const tail = sorted.slice(0, Math.floor(sorted.length * 0.05) + 1)
  return tail.length > 0 ? (tail.reduce((a, b) => a + b, 0) / tail.length) * totalExposure.value : 0
})

const ddChartOption = computed(() => {
  const theme = useChartTheme()
  const navs = navSeries.value
  const dates = (store.equityCurve || []).map(p => p.date)
  const ddSeries: number[] = []
  let peak = navs[0] || 1
  for (const n of navs) { if (n > peak) peak = n; ddSeries.push(peak > 0 ? (n / peak - 1) * 100 : 0) }
  const labels = dates.map((d, i) => i % Math.max(1, Math.floor(dates.length / 6)) === 0 ? d.slice(5) : '')
  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 20, bottom: 30, left: 55 },
    xAxis: { type: 'category', data: labels, axisLabel: { color: theme.axisColor, fontSize: 9 } },
    yAxis: { type: 'value', axisLabel: { color: theme.axisColor, fontSize: 9, formatter: '{value}%' } },
    series: [{
      type: 'line', data: ddSeries, smooth: true,
      lineStyle: { color: '#f85149', width: 2 },
      areaStyle: { color: new graphic.LinearGradient(0,0,0,1,[{offset:0, color:'rgba(248,81,73,0.3)'},{offset:1, color:'rgba(248,81,73,0.02)'}]) },
      symbol: 'none',
    }],
  }
})

const hasData = computed(() => (store.equityCurve || []).length > 0 || store.positions.length > 0)

const kpiCards = computed(() => [
  { label: 'VaR 95%', value: var95.value !== 0 ? '$' + fmtMoney(Math.abs(var95.value)) : '--', color: '#f0883e' },
  { label: 'CVaR 95%', value: cvar95.value !== 0 ? '$' + fmtMoney(Math.abs(cvar95.value)) : '--', color: '#f0883e' },
  { label: '最大回撤', value: maxDrawdown.value.dd !== 0 ? fmt(maxDrawdown.value.dd) + '%' : '--', color: maxDrawdown.value.dd < -10 ? '#f85149' : '#f0883e' },
  { label: 'Sharpe', value: sharpeRatio.value !== 0 ? fmt(sharpeRatio.value, 2) : '--', color: sharpeRatio.value > 1 ? '#3fb950' : sharpeRatio.value > 0 ? '#f0883e' : '#f85149' },
  { label: 'Sortino', value: sortinoRatio.value !== 0 ? fmt(sortinoRatio.value, 2) : '--', color: sortinoRatio.value > 1 ? '#3fb950' : sortinoRatio.value > 0 ? '#f0883e' : '#f85149' },
  { label: '年化波动', value: annualVol.value !== 0 ? fmt(annualVol.value) + '%' : '--', color: 'var(--color-text-tertiary)' },
])

onMounted(async () => {
  loading.value = true
  try { await store.fetchEquityCurve(); await store.fetchPositions() } finally { loading.value = false }
})
</script>

<template>
  <div class="risk-dashboard-panel">
    <div class="panel-header">
      <h3>风险仪表盘</h3>
      <span v-if="totalExposure > 0" class="exposure-badge">敞口 ${{ fmtMoney(totalExposure) }}</span>
    </div>
    <SkeletonPanel v-if="loading && !hasData" type="card" :rows="2" />
    <div v-else-if="!hasData" class="status">暂无数据 — 需要持仓和净值历史数据</div>
    <template v-else>
      <div class="kpi-grid">
        <div v-for="card in kpiCards" :key="card.label" class="kpi-card" :style="{ borderLeft: `3px solid ${card.color}` }">
          <span class="kpi-label">{{ card.label }}</span>
          <span class="kpi-value" :style="{ color: card.color }">{{ card.value }}</span>
        </div>
      </div>
      <div v-if="navSeries.length > 1" class="chart-section">
        <VChart :option="ddChartOption" autoresize style="height:220px" />
      </div>
      <div v-if="maxDrawdown.dd < 0" class="dd-info">
        Peak → Trough: {{ maxDrawdown.start }} → {{ maxDrawdown.end }} ({{ fmt(maxDrawdown.dd) }}%)
      </div>
    </template>
  </div>
</template>

<style scoped>
.risk-dashboard-panel { padding: 12px; background: var(--color-bg-panel, var(--color-bg-panel)); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; color: var(--color-text, var(--color-border)); }
.panel-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.exposure-badge { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(240,136,62,0.15); color: var(--color-accent); font-family: 'JetBrains Mono', monospace; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; padding: 40px 0; }
.kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin-bottom: 12px; }
.kpi-card { padding: 12px; background: var(--color-bg-elevated, var(--color-bg-panel)); border-radius: var(--radius-lg); border: 1px solid var(--color-border-subtle); }
.kpi-label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 4px; }
.kpi-value { font-size: 18px; font-weight: 700; }
.chart-section { background: var(--color-bg-elevated, var(--color-bg-panel)); border-radius: var(--radius-lg); padding: 8px; margin-bottom: 8px; border: 1px solid var(--color-border-subtle); }
.dd-info { text-align: center; font-size: 11px; color: var(--color-text-tertiary); font-variant-numeric: tabular-nums; }
</style>
