<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  MarkLineComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

use([
  BarChart,
  LineChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  MarkLineComponent,
  CanvasRenderer,
])

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol ?? ctx.getGroupSymbol(pg.groupId) ?? '600519')
const { name } = useStockName(symbol)
const { t } = useI18n()
const lookback = ref(props.params?.lookback ?? 252)
const lookbackOptions = [30, 60, 90, 252]

const { fetchWithCache } = usePanelCache()

const binData = ref<{ x: number; y: number }[]>([])
const normalCurve = ref<{ x: number; y: number }[]>([])

const meanVal = ref(0)
const stdVal = ref(0)
const skewnessVal = ref(0)
const kurtosisVal = ref(0)
const jarqueBeraVal = ref(0)

const hasECharts = ref(false)
const loading = ref(false)
const loadError = ref('')
const dataReady = ref(false)

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

function normalPDF(x: number, mean: number, std: number): number {
  if (std === 0) return 0
  const coeff = 1 / (std * Math.sqrt(2 * Math.PI))
  const exp = -0.5 * ((x - mean) / std) ** 2
  return coeff * Math.exp(exp)
}

async function compute() {
  const app = (window as any).go?.main?.App
  if (!app) { dataReady.value = false; return }
  loading.value = true
  loadError.value = ''
  try {
    const { data: result } = await fetchWithCache<any>('return_dist:' + symbol.value, () => app.GetReturnDistribution(symbol.value, lookback.value, 30))
    if (!result?.bins || !result?.counts) { dataReady.value = false; return }
    const bins: number[] = result.bins
    const counts: number[] = result.counts
    binData.value = bins.map((x, i) => ({ x, y: counts[i] || 0 }))

    const total = counts.reduce((a: number, b: number) => a + b, 0)
    if (total > 0) {
      let weightedSum = 0, weightedSumSq = 0
      for (let i = 0; i < bins.length; i++) {
        weightedSum += bins[i] * counts[i]
        weightedSumSq += bins[i] * bins[i] * counts[i]
      }
      const mean = weightedSum / total
      const variance = weightedSumSq / total - mean * mean
      const std = Math.sqrt(Math.max(variance, 0))
      meanVal.value = mean
      stdVal.value = std
      normalCurve.value = bins.map(x => ({ x, y: normalPDF(x, mean, std) * total }))
    }
    dataReady.value = true
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    dataReady.value = false
  } finally {
    loading.value = false
  }
}

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    compute()
  }
})

const chartOption = computed(() => {
  if (!dataReady.value || binData.value.length === 0) return null

  const mean = meanVal.value
  const std = stdVal.value
  const theme = useChartTheme()

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis' as const,
      backgroundColor: theme.bgColor,
      borderColor: theme.splitColor,
      textStyle: { color: theme.tooltipText, fontSize: 11 },
    },
    legend: {
      data: ['Returns', t('misc.normal_fit')],
      textStyle: { color: theme.axisColor, fontSize: 10 },
      top: 0,
    },
    grid: { left: '8%', right: '4%', top: '12%', bottom: '8%' },
    xAxis: {
      type: 'value' as const,
      axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => (v * 100).toFixed(1) + '%' },
      axisLine: { lineStyle: { color: theme.splitColor } },
      splitLine: { lineStyle: { color: theme.bgColor } },
    },
    yAxis: {
      type: 'value' as const,
      axisLabel: { color: theme.axisColor, fontSize: 10 },
      axisLine: { lineStyle: { color: theme.splitColor } },
      splitLine: { lineStyle: { color: theme.bgColor } },
    },
    series: [
      {
        name: 'Returns',
        type: 'bar',
        data: binData.value.map((b) => [b.x, b.y]),
        barWidth: '90%',
        itemStyle: { color: theme.palette[0] + '73', borderColor: theme.palette[0] + 'B3', borderWidth: 1, borderRadius: [2, 2, 0, 0] },
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { type: 'dashed' as const, color: theme.axisColor, width: 1 },
          label: { color: theme.axisColor, fontSize: 9, formatter: (p: { value: number }) => ((p.value as number) * 100).toFixed(2) + '%' },
          data: [
            { xAxis: mean, name: 'μ' },
            { xAxis: mean + std, name: '+1σ' },
            { xAxis: mean - std, name: '-1σ' },
          ],
        },
      },
      {
        name: t('misc.normal_fit'),
        type: 'line',
        data: normalCurve.value.map((p) => [p.x, p.y]),
        smooth: true,
        symbol: 'none',
        lineStyle: { color: theme.palette[5], width: 2 },
      },
    ],
  }
})

function fmtNumber(v: number, decimals: number = 4): string { return v.toFixed(decimals) }

onMounted(() => { try { hasECharts.value = true } catch { hasECharts.value = false } })
</script>

<template>
  <div class="distribution-panel">
    <PanelHeader title="收益率分布">
      <template #controls>
        <button v-if="addToWfControl" class="btn btn-sm" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
      </template>
    </PanelHeader>

    <div class="controls-row">
      <label class="control-label">
        Symbol
        <input v-model="symbol" type="text" class="symbol-input" placeholder="e.g. 600519" />
      </label>
      <label class="control-label">
        回溯
        <select v-model="lookback" class="lookback-select">
          <option v-for="opt in lookbackOptions" :key="opt" :value="opt">{{ opt }}d</option>
        </select>
      </label>
      <button class="btn btn-primary" @click="compute">计算</button>
    </div>

    <div v-if="dataReady" class="stats-row">
      <div class="stat-card"><span class="stat-label">{{ t('misc.mean') }}</span><span class="stat-value">{{ fmtNumber(meanVal * 100, 4) }}%</span></div>
      <div class="stat-card"><span class="stat-label">{{ t('misc.stddev') }}</span><span class="stat-value">{{ fmtNumber(stdVal * 100, 4) }}%</span></div>
      <div class="stat-card"><span class="stat-label">{{ t('misc.skewness') }}</span><span class="stat-value">{{ fmtNumber(skewnessVal) }}</span></div>
      <div class="stat-card"><span class="stat-label">{{ t('misc.kurtosis') }}</span><span class="stat-value">{{ fmtNumber(kurtosisVal) }}</span></div>
      <div class="stat-card"><span class="stat-label">{{ t('misc.jarque_bera') }}</span><span class="stat-value">{{ fmtNumber(jarqueBeraVal, 2) }}</span></div>
    </div>

    <LoadingState v-if="loading" type="chart" />
    <EmptyState v-else-if="!dataReady && !loadError" title="输入标的并点击计算" description="输入股票代码后点击计算查看收益分布" />
    <div v-else-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-else-if="dataReady" class="chart-body">
      <template v-if="hasECharts">
        <VChart v-if="chartOption" :option="chartOption" autoresize class="echarts-container" />
      </template>
      <div v-else class="fallback-table-wrap">
        <table class="dist-table">
          <thead><tr><th>区间中心</th><th>频率</th><th>正态拟合</th></tr></thead>
          <tbody>
            <tr v-for="(bin, idx) in binData" :key="idx">
              <td>{{ (bin.x * 100).toFixed(3) }}%</td>
              <td>{{ bin.y }}</td>
              <td>{{ normalCurve[idx] ? normalCurve[idx].y.toFixed(1) : '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.distribution-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.controls-row { display: flex; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); align-items: flex-end; }
.control-label { font-size: var(--font-xs); color: var(--color-text-secondary); display: flex; flex-direction: column; gap: var(--space-xs); }
.symbol-input { background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); color: var(--color-text-primary); border-radius: var(--radius-sm); padding: var(--space-xs) var(--space-sm); font-size: var(--font-xs); width: 120px; font-family: var(--font-mono); }
.lookback-select { background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); color: var(--color-text-primary); border-radius: var(--radius-sm); padding: var(--space-xs); font-size: var(--font-xs); }
.stats-row { display: flex; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); flex-wrap: wrap; }
.stat-card { display: flex; flex-direction: column; gap: var(--space-xs); padding: var(--space-sm) var(--space-md); background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-sm); min-width: 80px; }
.stat-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.stat-value { font-size: var(--font-sm); font-weight: 600; font-variant-numeric: tabular-nums; color: var(--color-text-primary); }
.chart-body { flex: 1; min-height: 0; display: flex; align-items: center; justify-content: center; }
.echarts-container { width: 100%; height: 100%; }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
.fallback-table-wrap { width: 100%; height: 100%; overflow: auto; padding: var(--space-sm) var(--panel-padding); scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent; }
.dist-table { border-collapse: collapse; font-size: var(--font-xs); width: 100%; }
.dist-table th, .dist-table td { padding: var(--space-xs) var(--space-sm); text-align: center; border-bottom: 1px solid var(--color-border-subtle); }
.dist-table th { color: var(--color-text-secondary); font-weight: 500; }
.dist-table td { font-variant-numeric: tabular-nums; color: var(--color-text-primary); }
</style>
