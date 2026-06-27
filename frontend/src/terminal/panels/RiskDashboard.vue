<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { graphic } from 'echarts/core'
import VChart from 'vue-echarts'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const metrics = ref({
  var_95: 2100, cvar_95: 2850, max_drawdown: -8.7,
  sharpe_ratio: 1.42, sortino_ratio: 1.89, annual_volatility: 12.5,
  total_exposure: 115300, max_dd_start: '2024-03', max_dd_end: '2024-04'
})

const fmt = (n: number, dec = 2) => n.toFixed(dec)

const ddChartOption = computed(() => {
  const theme = useChartTheme()
  return {
  backgroundColor: 'transparent',
  grid: { top: 10, right: 20, bottom: 30, left: 50 },
  xAxis: { type: 'category', data: ['Jan','Feb','Mar','Apr','May','Jun'], axisLabel: { color: theme.axisColor, fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { color: theme.axisColor, fontSize: 10, formatter: '{value}%' } },
  series: [{
    type: 'line', data: [0, -2.1, -8.7, -5.3, -2.0, 0],
    smooth: true, lineStyle: { color: '#f85149', width: 2 },
    areaStyle: { color: new graphic.LinearGradient(0,0,0,1,[
      {offset:0, color:'rgba(248,81,73,0.3)'}, {offset:1, color:'rgba(248,81,73,0.02)'}
    ]) },
    symbol: 'none'
  }]
}
})

// Phase 10.4: GARCH volatility section
const garchModel = ref<'garch' | 'gjr_garch' | 'egarch'>('garch')
const garchP = ref(1)
const garchQ = ref(1)
const garchAIC = ref(1234.5)
const garchBIC = ref(1250.0)
const garchVolatility = ref<number[]>([0.012, 0.015, 0.013, 0.011, 0.014, 0.018, 0.016, 0.013, 0.012, 0.015,
  0.017, 0.019, 0.018, 0.015, 0.014, 0.016, 0.02, 0.022, 0.019, 0.017,
  0.015, 0.013, 0.014, 0.016, 0.018, 0.021, 0.019, 0.017, 0.015, 0.014])

const volChartOption = computed(() => {
  const theme = useChartTheme()
  return {
  backgroundColor: 'transparent',
  title: { text: `GARCH(${garchP.value},${garchQ.value}) Volatility`, textStyle: { color: theme.textColor, fontSize: 12 }, left: 'center' },
  grid: { top: 35, right: 20, bottom: 25, left: 50 },
  xAxis: { type: 'category', data: garchVolatility.value.map((_, i) => i + 1), axisLabel: { color: theme.axisColor, fontSize: 9 } },
  yAxis: { type: 'value', axisLabel: { color: theme.axisColor, fontSize: 9, formatter: '{value}%' } },
  series: [{
    type: 'line',
    data: garchVolatility.value.map(v => +(v * 100).toFixed(2)),
    smooth: true,
    lineStyle: { color: '#f0883e', width: 2 },
    areaStyle: {
      color: new graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: 'rgba(240,136,62,0.25)' },
        { offset: 1, color: 'rgba(240,136,62,0.02)' },
      ]),
    },
    symbol: 'none',
  }],
}
})

const garchModels = ['garch', 'gjr_garch', 'egarch'] as const

const kpiCards = [
  { label: t('risk.var_95'), value: `$${fmt(metrics.value.var_95).replace(/\B(?=(\d{3})+(?!\d))/g, ',')}`, color: '#f0883e' },
  { label: t('risk.cvar_95'), value: `$${fmt(metrics.value.cvar_95).replace(/\B(?=(\d{3})+(?!\d))/g, ',')}`, color: '#f0883e' },
  { label: t('risk.max_drawdown'), value: `${fmt(metrics.value.max_drawdown)}%`, color: metrics.value.max_drawdown < -10 ? '#f85149' : '#f0883e' },
  { label: t('risk.sharpe'), value: fmt(metrics.value.sharpe_ratio), color: metrics.value.sharpe_ratio > 1 ? '#3fb950' : '#f0883e' },
  { label: t('risk.sortino'), value: fmt(metrics.value.sortino_ratio), color: metrics.value.sortino_ratio > 1 ? '#3fb950' : '#f0883e' },
  { label: t('risk.annual_vol'), value: `${fmt(metrics.value.annual_volatility)}%`, color: 'var(--color-text-tertiary)' },
]
</script>

<template>
    <div class="placeholder-banner">⚠ 占位面板 — 数据未连接后端</div>
  <div class="risk-dashboard-panel">
    <div class="kpi-grid">
      <div v-for="card in kpiCards" :key="card.label" class="kpi-card" :style="{ borderLeft: `3px solid ${card.color}` }">
        <span class="kpi-label">{{ card.label }}</span>
        <span class="kpi-value" :style="{ color: card.color }">{{ card.value }}</span>
      </div>
    </div>
    <div class="chart-section">
      <div class="chart-title">{{ t('risk.drawdown_chart') }}</div>
      <VChart :option="ddChartOption" autoresize style="height:200px" />
    </div>
    <div v-if="metrics.max_drawdown < 0" class="dd-info">
      <span>Peak-to-Trough: {{ metrics.max_dd_start }} → {{ metrics.max_dd_end }}</span>
    </div>

    <!-- Phase 10.4: GARCH Volatility Section -->
    <div class="section-header">
      <span class="section-label">{{ t('risk.garch_model') }}</span>
      <select v-model="garchModel" class="garch-select">
        <option v-for="m in garchModels" :key="m" :value="m">{{ m === 'gjr_garch' ? 'GJR-GARCH' : m.toUpperCase() }}</option>
      </select>
    </div>
    <div class="chart-section">
      <VChart :option="volChartOption" autoresize style="height:200px" />
    </div>
    <div class="garch-metrics">
      <span>AIC: {{ fmt(garchAIC, 2) }}</span>
      <span>BIC: {{ fmt(garchBIC, 2) }}</span>
    </div>
  </div>
</template>

<style scoped>
.placeholder-banner { background: #f59e0b; color: #000; padding: 6px 10px; text-align: center; font-size: 12px; font-weight: 600; }
.risk-dashboard-panel { padding: 12px; background: var(--bg); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--spacing); margin-bottom: 12px; }
.kpi-card { padding: 12px; background: var(--card); border-radius: 4px; }
.kpi-label { display: block; font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 4px; }
.kpi-value { font-size: 18px; font-weight: 700; }
.chart-section { background: var(--card); border-radius: 4px; padding: 8px; margin-bottom: 8px; }
.chart-title { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 4px; }
.dd-info { text-align: center; font-size: 11px; color: var(--muted); margin-bottom: 8px; }
.section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.section-label { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; }
.garch-select { background: var(--card); border: 1px solid var(--border); color: var(--text); padding: 3px 6px; border-radius: 4px; font-size: 10px; }
.garch-metrics { display: flex; justify-content: center; gap: 16px; font-size: 11px; color: var(--muted); margin-top: 4px; }
</style>
