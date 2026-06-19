<script setup lang="ts">
import { ref, computed } from 'vue'
import { simulateGBM, histogramBins } from '@/lib/stats'
import type { GBMInput, GBMOutput } from '@/lib/stats'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

// Inputs
const initialCapital = ref(100000)
const annualReturn = ref(8)
const annualVol = ref(20)
const years = ref(5)
const simulations = ref(500)
const confidence = ref(95)

// State
const result = ref<GBMOutput | null>(null)
const running = ref(false)
const hasEcharts = ref(false)

// Detect echarts availability
async function detectEcharts() {
  try {
    await import('echarts')
    hasEcharts.value = true
  } catch {
    hasEcharts.value = false
  }
}
detectEcharts()

function formatCurrency(v: number): string {
  if (Math.abs(v) >= 1e6) return '$' + (v / 1e6).toFixed(2) + 'M'
  if (Math.abs(v) >= 1e3) return '$' + (v / 1e3).toFixed(1) + 'K'
  return '$' + v.toFixed(0)
}

// Downsample paths for chart rendering
function downsamplePaths(paths: number[][], maxPaths: number, stepsPerYear: number): number[][] {
  if (paths.length <= maxPaths) return paths
  const step = Math.floor(paths.length / maxPaths)
  return paths.filter((_, i) => i % step === 0).slice(0, maxPaths)
}

// Median path
function computeMedianPath(paths: number[][], totalSteps: number): number[] {
  const medians: number[] = []
  for (let t = 0; t <= totalSteps; t++) {
    const vals = paths.map(p => p[t]).sort((a, b) => a - b)
    medians.push(vals[Math.floor(vals.length / 2)])
  }
  return medians
}

// Confidence band at each time step
function computeConfidenceBand(paths: number[][], totalSteps: number, ci: number): { upper: number[]; lower: number[] } {
  const upper: number[] = []
  const lower: number[] = []
  const lowPct = (100 - ci) / 100 / 2
  const highPct = 1 - lowPct
  for (let t = 0; t <= totalSteps; t++) {
    const vals = paths.map(p => p[t]).sort((a, b) => a - b)
    lower.push(vals[Math.floor(vals.length * lowPct)])
    upper.push(vals[Math.floor(vals.length * highPct)])
  }
  return { upper, lower }
}

// Chart options
const pathsChartOption = computed(() => {
  if (!result.value) return {}
  const r = result.value
  const totalSteps = r.paths[0].length - 1
  const stepsPerYear = 252
  const displayedPaths = downsamplePaths(r.paths, 200, stepsPerYear)
  const medianPath = computeMedianPath(displayedPaths, totalSteps)
  const band = computeConfidenceBand(r.paths, totalSteps, confidence.value)

  // Time axis labels
  const xLabels: string[] = []
  for (let t = 0; t <= totalSteps; t++) {
    const yr = t / stepsPerYear
    if (t % Math.floor(stepsPerYear) === 0 || t === totalSteps) {
      xLabels.push('Y' + yr.toFixed(1))
    } else {
      xLabels.push('')
    }
  }

  const series: any[] = []

  // Individual paths (thin, low opacity)
  for (const path of displayedPaths) {
    series.push({
      type: 'line',
      data: path,
      lineStyle: { width: 0.5, opacity: 0.08, color: '#58a6ff' },
      showSymbol: false,
      silent: true,
    })
  }

  // Median path
  series.push({
    type: 'line',
    data: medianPath,
    lineStyle: { width: 2, color: '#f59e0b' },
    showSymbol: false,
    name: 'Median',
  })

  // Confidence band (area)
  series.push({
    type: 'line',
    data: band.lower,
    lineStyle: { width: 0, opacity: 0 },
    areaStyle: { color: 'rgba(88,166,255,0.1)' },
    showSymbol: false,
    name: 'Lower ' + confidence.value + '%',
    stack: 'band',
  })
  series.push({
    type: 'line',
    data: band.upper.map((v, i) => v - band.lower[i]),
    lineStyle: { width: 0, opacity: 0 },
    areaStyle: { color: 'rgba(88,166,255,0.15)' },
    showSymbol: false,
    name: 'Upper ' + confidence.value + '%',
    stack: 'band',
  })

  return {
    backgroundColor: 'transparent',
    grid: { left: 60, right: 20, top: 10, bottom: 30 },
    xAxis: {
      type: 'category',
      data: xLabels,
      axisLabel: { color: '#6b7280', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#6b7280', fontSize: 10, formatter: (v: number) => formatCurrency(v) },
      splitLine: { lineStyle: { color: '#1f2937' } },
    },
    tooltip: { trigger: 'axis' as const },
    series,
  }
})

const histogramOption = computed(() => {
  if (!result.value) return {}
  const r = result.value
  const bins = histogramBins(r.terminalValues, 50)
  const var5Idx = r.terminalValues.sort((a, b) => a - b)[Math.floor(r.terminalValues.length * 0.05)]

  return {
    backgroundColor: 'transparent',
    grid: { left: 60, right: 20, top: 10, bottom: 30 },
    xAxis: {
      type: 'category',
      data: bins.map(b => b.x.toFixed(0)),
      axisLabel: { color: '#6b7280', fontSize: 9, rotate: 45 },
      interval: 9,
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#6b7280', fontSize: 10 },
      splitLine: { lineStyle: { color: '#1f2937' } },
    },
    tooltip: { trigger: 'axis' as const },
    series: [
      {
        type: 'bar',
        data: bins.map(b => b.y),
        itemStyle: { color: '#58a6ff', borderRadius: [1, 1, 0, 0] },
      },
    ],
  }
})

async function runSimulation() {
  running.value = true
  result.value = null
  // Yield to let UI update
  await new Promise(r => setTimeout(r, 10))

  const input: GBMInput = {
    initialCapital: initialCapital.value,
    annualReturn: annualReturn.value / 100,
    annualVol: annualVol.value / 100,
    years: years.value,
    simulations: simulations.value,
  }
  result.value = simulateGBM(input)
  running.value = false
}
</script>

<template>
  <div class="montecarlo-panel">
    <div class="panel-header">
      <h3>Monte Carlo Simulation</h3>
    </div>
    <div class="panel-body">
      <div class="sidebar">
        <div class="param-group">
          <label class="param-label">Initial Capital ($)</label>
          <input v-model.number="initialCapital" type="number" class="param-input" min="1000" step="1000" />
        </div>
        <div class="param-group">
          <label class="param-label">Annual Return (%)</label>
          <input v-model.number="annualReturn" type="number" class="param-input" min="-50" max="100" step="0.5" />
        </div>
        <div class="param-group">
          <label class="param-label">Annual Vol (%)</label>
          <input v-model.number="annualVol" type="number" class="param-input" min="1" max="200" step="1" />
        </div>
        <div class="param-group">
          <label class="param-label">Years</label>
          <input v-model.number="years" type="number" class="param-input" min="1" max="30" step="1" />
        </div>
        <div class="param-group">
          <label class="param-label">Simulations</label>
          <input v-model.number="simulations" type="number" class="param-input" min="100" max="5000" step="100" />
        </div>
        <div class="param-group">
          <label class="param-label">Confidence (%)</label>
          <input v-model.number="confidence" type="number" class="param-input" min="80" max="99" step="1" />
        </div>
        <button class="run-btn" :disabled="running" @click="runSimulation">
          {{ running ? 'Running...' : 'Run Simulation' }}
        </button>
      </div>
      <div class="content">
        <template v-if="result">
          <!-- Paths Chart -->
          <div class="chart-section chart-a">
            <div v-if="hasEcharts" class="chart-container">
              <v-chart :option="pathsChartOption" autoresize />
            </div>
            <div v-else class="fallback-table">
              <div class="fallback-title">Paths Percentile Table (per year)</div>
              <table>
                <thead>
                  <tr>
                    <th>Year</th>
                    <th>P5</th>
                    <th>P25</th>
                    <th>Median</th>
                    <th>P75</th>
                    <th>P95</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="y in years" :key="y">
                    <td>Y{{ y }}</td>
                    <td>{{ formatCurrency(result.paths.map(p => p[y * 252] || p[p.length - 1]).sort((a, b) => a - b)[Math.floor(result.paths.length * 0.05)]) }}</td>
                    <td>{{ formatCurrency(result.paths.map(p => p[y * 252] || p[p.length - 1]).sort((a, b) => a - b)[Math.floor(result.paths.length * 0.25)]) }}</td>
                    <td class="median">{{ formatCurrency(result.paths.map(p => p[y * 252] || p[p.length - 1]).sort((a, b) => a - b)[Math.floor(result.paths.length * 0.5)]) }}</td>
                    <td>{{ formatCurrency(result.paths.map(p => p[y * 252] || p[p.length - 1]).sort((a, b) => a - b)[Math.floor(result.paths.length * 0.75)]) }}</td>
                    <td>{{ formatCurrency(result.paths.map(p => p[y * 252] || p[p.length - 1]).sort((a, b) => a - b)[Math.floor(result.paths.length * 0.95)]) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Histogram Chart -->
          <div class="chart-section chart-b">
            <div v-if="hasEcharts" class="chart-container">
              <v-chart :option="histogramOption" autoresize />
            </div>
            <div v-else class="fallback-stats">
              <div class="fallback-title">Terminal Value Distribution</div>
              <div class="stat-row"><span>Median</span><span>{{ formatCurrency(result.medianTerminal) }}</span></div>
              <div class="stat-row"><span>VaR (95%)</span><span class="negative">{{ formatCurrency(result.var95) }}</span></div>
              <div class="stat-row"><span>CVaR (95%)</span><span class="negative">{{ formatCurrency(result.cvar95) }}</span></div>
            </div>
          </div>

          <!-- Stats Cards -->
          <div class="stats-row">
            <div class="stat-card">
              <div class="stat-value accent">{{ formatCurrency(result.medianTerminal) }}</div>
              <div class="stat-label">Median Terminal</div>
            </div>
            <div class="stat-card">
              <div class="stat-value negative">{{ formatCurrency(result.var95) }}</div>
              <div class="stat-label">95% VaR</div>
            </div>
            <div class="stat-card">
              <div class="stat-value negative">{{ formatCurrency(result.cvar95) }}</div>
              <div class="stat-label">95% CVaR</div>
            </div>
            <div class="stat-card">
              <div class="stat-value">{{ (result.probLoss * 100).toFixed(1) }}%</div>
              <div class="stat-label">Prob(Loss)</div>
            </div>
            <div class="stat-card">
              <div class="stat-value positive">{{ (result.probDouble * 100).toFixed(1) }}%</div>
              <div class="stat-label">Prob(Double)</div>
            </div>
          </div>
        </template>
        <div v-else class="empty-state">
          <p v-if="running">Running simulation...</p>
          <p v-else>Set parameters and click Run Simulation to begin</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.montecarlo-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #111827;
  color: #e5e7eb;
}

.panel-header {
  padding: 8px 12px;
  border-bottom: 1px solid #374151;
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.panel-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.sidebar {
  width: 220px;
  flex-shrink: 0;
  padding: 12px;
  border-right: 1px solid #374151;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
}

.param-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.param-label {
  font-size: 11px;
  color: #6b7280;
  font-weight: 500;
}

.param-input {
  padding: 6px 8px;
  background: #1f2937;
  border: 1px solid #374151;
  color: #e5e7eb;
  border-radius: 4px;
  font-size: 13px;
  outline: none;
  font-variant-numeric: tabular-nums;
}

.param-input:focus {
  border-color: #58a6ff;
}

.run-btn {
  padding: 8px 0;
  background: #1f2937;
  border: 1px solid #58a6ff;
  color: #58a6ff;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  margin-top: 4px;
  transition: background 0.15s;
}

.run-btn:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.1);
}

.run-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 8px;
}

.chart-section {
  flex-shrink: 0;
}

.chart-a {
  height: 55%;
}

.chart-b {
  height: 25%;
}

.chart-container {
  width: 100%;
  height: 100%;
}

.fallback-table {
  padding: 8px;
  overflow-y: auto;
  height: 100%;
}

.fallback-stats {
  padding: 8px;
}

.fallback-title {
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 6px;
  font-weight: 500;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}

th, td {
  padding: 3px 6px;
  text-align: right;
  border-bottom: 1px solid #1f2937;
}

th {
  color: #6b7280;
  font-weight: 500;
}

.median {
  color: #f59e0b;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  padding: 3px 0;
  font-size: 12px;
  border-bottom: 1px solid #1f2937;
}

.stats-row {
  display: flex;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid #374151;
  flex-wrap: wrap;
}

.stat-card {
  flex: 1;
  min-width: 100px;
  background: #1f2937;
  border: 1px solid #374151;
  border-radius: 4px;
  padding: 8px 10px;
  text-align: center;
}

.stat-value {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.stat-value.accent { color: #58a6ff; }
.stat-value.positive { color: #22c55e; }
.stat-value.negative { color: #f87171; }

.stat-label {
  font-size: 10px;
  color: #6b7280;
  margin-top: 2px;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #6b7280;
  font-size: 13px;
}
</style>
