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
import { histogramBins } from '@/lib/stats'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'

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

    // 计算 stats from histogram data
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
      textStyle: { color: '#e5e7eb', fontSize: 11 },
    },
    legend: {
      data: ['Returns', t('misc.normal_fit')],
      textStyle: { color: theme.axisColor, fontSize: 10 },
      top: 0,
    },
    grid: {
      left: '8%',
      right: '4%',
      top: '12%',
      bottom: '8%',
    },
    xAxis: {
      type: 'value' as const,
      axisLabel: {
        color: theme.axisColor,
        fontSize: 10,
        formatter: (v: number) => (v * 100).toFixed(1) + '%',
      },
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
        itemStyle: {
          color: 'rgba(59,130,246,0.45)',
          borderColor: 'rgba(59,130,246,0.7)',
          borderWidth: 1,
          borderRadius: [2, 2, 0, 0],
        },
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { type: 'dashed' as const, color: theme.axisColor, width: 1 },
          label: {
            color: theme.axisColor,
            fontSize: 9,
            formatter: (p: { value: number }) =>
              ((p.value as number) * 100).toFixed(2) + '%',
          },
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
        lineStyle: { color: '#ef4444', width: 2 },
      },
    ],
  }
})

function fmtNumber(v: number, decimals: number = 4): string {
  return v.toFixed(decimals)
}

onMounted(() => {
  try {
    hasECharts.value = true
  } catch {
    hasECharts.value = false
  }
})
</script>

<template>
  <div class="distribution-panel">
    <div class="panel-header">
      <h3>{{ t('misc.distribution') }}</h3>
      <span class="symbol-badge">{{ symbol }} {{ name }}</span>
    </div>

    <div class="controls-row">
      <label class="control-label">
        Symbol
        <input
          v-model="symbol"
          type="text"
          class="symbol-input"
          placeholder="e.g. 600519"
        />
      </label>

      <label class="control-label">
        回溯
        <select v-model="lookback" class="lookback-select">
          <option v-for="opt in lookbackOptions" :key="opt" :value="opt">
            {{ opt }}d
          </option>
        </select>
      </label>

      <button class="compute-btn" @click="compute">计算</button>
    </div>

    <!-- Stats cards -->
    <div v-if="dataReady" class="stats-row">
      <div class="stat-card">
        <span class="stat-label">{{ t('misc.mean') }}</span>
        <span class="stat-value">{{ fmtNumber(meanVal * 100, 4) }}%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('misc.stddev') }}</span>
        <span class="stat-value">{{ fmtNumber(stdVal * 100, 4) }}%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('misc.skewness') }}</span>
        <span class="stat-value">{{ fmtNumber(skewnessVal) }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('misc.kurtosis') }}</span>
        <span class="stat-value">{{ fmtNumber(kurtosisVal) }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('misc.jarque_bera') }}</span>
        <span class="stat-value">{{ fmtNumber(jarqueBeraVal, 2) }}</span>
      </div>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div class="chart-body">
      <div v-if="loading" class="chart-fallback">{{ $t('common.loading') }}</div>
      <div v-else-if="!dataReady" class="placeholder-msg">
        Enter a symbol and click 计算
      </div>

      <template v-else-if="hasECharts">
        <VChart
          v-if="chartOption"
          :option="chartOption"
          autoresize
          class="echarts-container"
        />
      </template>

      <!-- Fallback HTML table -->
      <div v-else class="fallback-table-wrap">
        <table class="dist-table">
          <thead>
            <tr>
              <th>区间中心</th>
              <th>频率</th>
              <th>正态拟合</th>
            </tr>
          </thead>
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
.distribution-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  overflow: hidden;
}

.panel-header {
  padding: 10px 14px 6px;
  border-bottom: 1px solid var(--color-border-strong);
}
.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.controls-row {
  display: flex;
  gap: 10px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--color-border-strong);
  align-items: flex-end;
}

.control-label {
  font-size: 11px;
  color: var(--color-text-secondary);
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.symbol-input {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
  font-size: 12px;
  width: 120px;
  font-family: 'Courier New', monospace;
}

.lookback-select {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: var(--radius-sm);
  padding: 5px 6px;
  font-size: 12px;
}

.compute-btn {
  padding: 5px 16px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  height: fit-content;
}
.compute-btn:hover {
  background: var(--color-border-strong);
}

/* Stats row */
.stats-row {
  display: flex;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--color-border-strong);
  flex-wrap: wrap;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 12px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  min-width: 80px;
}

.stat-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}

.stat-value {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
}

/* Chart body */
.chart-body {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-msg {
  color: var(--color-text-tertiary);
  font-size: 14px;
}

.echarts-container {
  width: 100%;
  height: 100%;
}

/* Fallback table */
.fallback-table-wrap {
  width: 100%;
  height: 100%;
  overflow: auto;
  padding: 8px 12px;
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-strong) transparent;
}

.dist-table {
  border-collapse: collapse;
  font-size: 11px;
  width: 100%;
}

.dist-table th,
.dist-table td {
  padding: 4px 10px;
  text-align: center;
  border-bottom: 1px solid var(--color-border-strong);
}

.dist-table th {
  color: var(--color-text-secondary);
  font-weight: 500;
}

.dist-table td {
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
}
</style>
