<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
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
import { histogramBins } from '@/lib/stats'

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
const lookback = ref(props.params?.lookback ?? 252)
const lookbackOptions = [30, 60, 90, 252]

const binData = ref<{ x: number; y: number }[]>([])
const normalCurve = ref<{ x: number; y: number }[]>([])

const meanVal = ref(0)
const stdVal = ref(0)
const skewnessVal = ref(0)
const kurtosisVal = ref(0)
const jarqueBeraVal = ref(0)

const hasECharts = ref(false)
const dataReady = ref(false)

function generateMockReturns(length: number): number[] {
  const returns: number[] = []
  for (let i = 0; i < length; i++) {
    // Sum-of-uniforms approximates Gaussian
    const z =
      (Math.random() +
        Math.random() +
        Math.random() +
        Math.random() +
        Math.random() +
        Math.random() -
        3) /
      3
    returns.push(z * 0.03)
  }
  return returns
}

function normalPDF(x: number, mean: number, std: number): number {
  if (std === 0) return 0
  const coeff = 1 / (std * Math.sqrt(2 * Math.PI))
  const exp = -0.5 * ((x - mean) / std) ** 2
  return coeff * Math.exp(exp)
}

function compute() {
  const returns = generateMockReturns(lookback.value)
  const n = returns.length

  // Histogram bins
  const bins = histogramBins(returns, 30)
  binData.value = bins

  // Stats
  const mean = returns.reduce((a, b) => a + b, 0) / n
  const variance = returns.reduce((s, r) => s + (r - mean) ** 2, 0) / (n - 1)
  const std = Math.sqrt(variance)

  meanVal.value = mean
  stdVal.value = std

  // Higher moments
  if (std > 0) {
    const skew =
      returns.reduce((s, r) => s + ((r - mean) / std) ** 3, 0) / n
    const kurt =
      returns.reduce((s, r) => s + ((r - mean) / std) ** 4, 0) / n - 3

    skewnessVal.value = skew
    kurtosisVal.value = kurt
    jarqueBeraVal.value = (n / 6) * (skew ** 2 + kurt ** 2 / 4)
  } else {
    skewnessVal.value = 0
    kurtosisVal.value = 0
    jarqueBeraVal.value = 0
  }

  // Normal curve overlay: scale PDF so max equals histogram max
  const maxBinY = Math.max(...bins.map((b) => b.y), 1)
  const pdfValues: { x: number; y: number }[] = []
  for (const bin of bins) {
    const pdf = normalPDF(bin.x, mean, std)
    pdfValues.push({ x: bin.x, y: pdf })
  }
  const maxPdf = Math.max(...pdfValues.map((p) => p.y), 0.001)
  const scale = maxBinY / maxPdf
  normalCurve.value = pdfValues.map((p) => ({ x: p.x, y: p.y * scale }))

  dataReady.value = true
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

  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis' as const,
      backgroundColor: '#1f2937',
      borderColor: '#374151',
      textStyle: { color: '#e5e7eb', fontSize: 11 },
    },
    legend: {
      data: ['Returns', 'Normal Fit'],
      textStyle: { color: '#9ca3af', fontSize: 10 },
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
        color: '#9ca3af',
        fontSize: 10,
        formatter: (v: number) => (v * 100).toFixed(1) + '%',
      },
      axisLine: { lineStyle: { color: '#374151' } },
      splitLine: { lineStyle: { color: '#1f2937' } },
    },
    yAxis: {
      type: 'value' as const,
      axisLabel: { color: '#9ca3af', fontSize: 10 },
      axisLine: { lineStyle: { color: '#374151' } },
      splitLine: { lineStyle: { color: '#1f2937' } },
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
          lineStyle: { type: 'dashed' as const, color: '#9ca3af', width: 1 },
          label: {
            color: '#9ca3af',
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
        name: 'Normal Fit',
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
      <h3>Return Distribution</h3>
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
        Lookback
        <select v-model="lookback" class="lookback-select">
          <option v-for="opt in lookbackOptions" :key="opt" :value="opt">
            {{ opt }}d
          </option>
        </select>
      </label>

      <button class="compute-btn" @click="compute">Compute</button>
    </div>

    <!-- Stats cards -->
    <div v-if="dataReady" class="stats-row">
      <div class="stat-card">
        <span class="stat-label">Mean</span>
        <span class="stat-value">{{ fmtNumber(meanVal * 100, 4) }}%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Std Dev</span>
        <span class="stat-value">{{ fmtNumber(stdVal * 100, 4) }}%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Skewness</span>
        <span class="stat-value">{{ fmtNumber(skewnessVal) }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Kurtosis</span>
        <span class="stat-value">{{ fmtNumber(kurtosisVal) }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Jarque-Bera</span>
        <span class="stat-value">{{ fmtNumber(jarqueBeraVal, 2) }}</span>
      </div>
    </div>

    <div class="chart-body">
      <div v-if="!dataReady" class="placeholder-msg">
        Enter a symbol and click Compute
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
              <th>Bin Center</th>
              <th>Frequency</th>
              <th>Normal Fit</th>
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
  background: #111827;
  color: #e5e7eb;
  overflow: hidden;
}

.panel-header {
  padding: 10px 14px 6px;
  border-bottom: 1px solid #374151;
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
  border-bottom: 1px solid #374151;
  align-items: flex-end;
}

.control-label {
  font-size: 11px;
  color: #9ca3af;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.symbol-input {
  background: #1f2937;
  border: 1px solid #374151;
  color: #e5e7eb;
  border-radius: 4px;
  padding: 5px 8px;
  font-size: 12px;
  width: 120px;
  font-family: 'Courier New', monospace;
}

.lookback-select {
  background: #1f2937;
  border: 1px solid #374151;
  color: #e5e7eb;
  border-radius: 4px;
  padding: 5px 6px;
  font-size: 12px;
}

.compute-btn {
  padding: 5px 16px;
  border: 1px solid #374151;
  border-radius: 4px;
  background: #1f2937;
  color: #e5e7eb;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  height: fit-content;
}
.compute-btn:hover {
  background: #374151;
}

/* Stats row */
.stats-row {
  display: flex;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid #374151;
  flex-wrap: wrap;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 12px;
  background: #1f2937;
  border: 1px solid #374151;
  border-radius: 4px;
  min-width: 80px;
}

.stat-label {
  font-size: 10px;
  color: #6b7280;
  text-transform: uppercase;
}

.stat-value {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: #e5e7eb;
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
  color: #6b7280;
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
  scrollbar-color: #374151 transparent;
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
  border-bottom: 1px solid #374151;
}

.dist-table th {
  color: #9ca3af;
  font-weight: 500;
}

.dist-table td {
  font-variant-numeric: tabular-nums;
  color: #e5e7eb;
}
</style>
