<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart, HeatmapChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  VisualMapComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([LineChart, HeatmapChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, VisualMapComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const daysBack = ref(30)

const hasEcharts = computed(() => {
  try {
    return typeof VChart !== 'undefined'
  } catch {
    return false
  }
})

// ---------------------------------------------------------------------------
// Generate mock volatility surface data (SVI-like parametric)
// ---------------------------------------------------------------------------
const strikes = computed<number[]>(() => {
  const s: number[] = []
  for (let i = 0; i < 13; i++) {
    s.push(0.7 + (i / 12) * 0.6) // 0.7 to 1.3, 13 steps
  }
  return s
})

const maturities = [0.1, 0.25, 0.5, 0.75, 1.0, 1.5, 2.0]

function generateIVGrid(): number[][] {
  const base = 0.22
  const a = 0.35
  const b = -0.02
  const c = -0.08

  return maturities.map((T) =>
    strikes.value.map((m) => {
      const noise = (Math.random() - 0.5) * 0.02
      let iv = base + a * (m - 1) ** 2 + b * T + c * (m - 1) * T + noise
      if (iv < 0.02) iv = 0.02
      if (iv > 1.2) iv = 1.2
      return parseFloat(iv.toFixed(4))
    })
  )
}

const ivGrid = ref<number[][]>([])

function regenerateSurface() {
  ivGrid.value = generateIVGrid()
}

onMounted(() => {
  regenerateSurface()
})

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    regenerateSurface()
  }
})

// ---------------------------------------------------------------------------
// Heatmap chart option for volatility surface
// ---------------------------------------------------------------------------
const heatmapOption = computed(() => {
  const grid = ivGrid.value
  if (!grid.length) return {}

  const heatData: [number, number, number][] = []
  for (let i = 0; i < maturities.length; i++) {
    for (let j = 0; j < strikes.value.length; j++) {
      heatData.push([j, i, grid[i][j]])
    }
  }

  return {
    backgroundColor: 'transparent',
    grid: { top: 10, right: 30, bottom: 40, left: 60 },
    xAxis: {
      type: 'category',
      data: strikes.value.map((s) => s.toFixed(2)),
      name: 'Moneyness (K/S)',
      nameTextStyle: { color: '#9ca3af', fontSize: 11 },
      axisLabel: { color: '#6b7280', fontSize: 10 },
      splitArea: { show: true },
    },
    yAxis: {
      type: 'category',
      data: maturities.map((t) => t.toFixed(2) + 'y'),
      name: 'Maturity (T)',
      nameTextStyle: { color: '#9ca3af', fontSize: 11 },
      axisLabel: { color: '#6b7280', fontSize: 10 },
      splitArea: { show: true },
    },
    visualMap: {
      min: Math.min(...grid.flat()) * 0.95,
      max: Math.max(...grid.flat()) * 1.05,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: 0,
      inRange: {
        color: ['#1d4ed8', '#22c55e', '#eab308', '#ef4444'],
      },
      textStyle: { color: '#9ca3af', fontSize: 10 },
    },
    tooltip: {
      position: 'top',
      formatter: (p: { data: [number, number, number] }) =>
        `K/S=${strikes.value[p.data[0]].toFixed(2)} T=${maturities[p.data[1]].toFixed(2)}y IV=${(p.data[2] * 100).toFixed(2)}%`,
    },
    series: [
      {
        type: 'heatmap',
        data: heatData,
        label: { show: false },
        emphasis: {
          itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' },
        },
      },
    ],
  }
})

// ---------------------------------------------------------------------------
// Slice view -- IV smile for a selected maturity
// ---------------------------------------------------------------------------
const selectedMaturityIdx = ref(0)

const sliceChartOption = computed(() => {
  const grid = ivGrid.value
  if (!grid.length || !grid[selectedMaturityIdx.value]) return {}

  const slice = grid[selectedMaturityIdx.value]
  const T = maturities[selectedMaturityIdx.value]

  return {
    backgroundColor: 'transparent',
    title: {
      text: `IV Smile — T=${T.toFixed(2)}y`,
      textStyle: { color: '#9ca3af', fontSize: 12 },
      left: 'center',
    },
    grid: { top: 35, right: 20, bottom: 30, left: 50 },
    xAxis: {
      type: 'category',
      data: strikes.value.map((s) => s.toFixed(2)),
      name: 'Moneyness',
      nameTextStyle: { color: '#6b7280', fontSize: 10 },
      axisLabel: { color: '#6b7280', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      name: 'IV',
      nameTextStyle: { color: '#6b7280', fontSize: 10 },
      axisLabel: { color: '#6b7280', fontSize: 10, formatter: (v: number) => (v * 100).toFixed(0) + '%' },
      splitLine: { lineStyle: { color: '#1f2937' } },
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
        type: 'line',
        data: slice,
        smooth: true,
        lineStyle: { color: '#58a6ff', width: 2 },
        symbol: 'circle',
        symbolSize: 6,
        itemStyle: { color: '#58a6ff' },
      },
    ],
  }
})

// ---------------------------------------------------------------------------
// Fallback: color cell by IV value
// ---------------------------------------------------------------------------
function ivColor(iv: number): string {
  const grid = ivGrid.value
  if (!grid.length) return '#1f2937'
  const all = grid.flat()
  const min = Math.min(...all)
  const max = Math.max(...all)
  if (max === min) return '#374151'
  const t = (iv - min) / (max - min)
  if (t < 0.33) return `rgba(59, 130, 246, ${0.3 + t * 2})`
  if (t < 0.66) return `rgba(34, 197, 94, ${0.3 + t * 0.5})`
  return `rgba(239, 68, 68, ${0.3 + t * 0.8})`
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
  regenerateSurface()
}
</script>

<template>
  <div class="surface-chart-panel">
    <div class="panel-header">
      <h3>Volatility Surface</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          placeholder="Symbol..."
          @keyup.enter="handleSymbolSubmit"
        />
        <input
          v-model.number="daysBack"
          type="range"
          min="7"
          max="90"
          class="days-slider"
          :title="`Lookback: ${daysBack} days`"
        />
        <span class="days-label">{{ daysBack }}d</span>
        <button class="refresh-btn" @click="regenerateSurface">&#x21bb;</button>
      </div>
    </div>

    <div class="surface-content">
      <!-- Heatmap / Surface -->
      <div class="surface-section">
        <VChart v-if="hasEcharts" :option="heatmapOption" autoresize class="surface-chart" />
        <div v-else class="fallback-heatmap">
          <table class="heatmap-table">
            <thead>
              <tr>
                <th>T \ K/S</th>
                <th v-for="s in strikes" :key="s">{{ s.toFixed(2) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, ri) in ivGrid" :key="ri">
                <td class="maturity-label">{{ maturities[ri]?.toFixed(2) || '-' }}y</td>
                <td
                  v-for="(iv, ci) in row"
                  :key="ci"
                  :style="{ background: ivColor(iv), color: iv > 0.35 ? '#fff' : '#e5e7eb' }"
                  class="heatmap-cell"
                >
                  {{ (iv * 100).toFixed(1) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Slice View -->
      <div class="slice-section">
        <div class="slice-controls">
          <label class="slice-label">Slice Maturity:</label>
          <select v-model="selectedMaturityIdx" class="slice-select">
            <option v-for="(t, idx) in maturities" :key="t" :value="idx">
              T = {{ t.toFixed(2) }}y
            </option>
          </select>
        </div>
        <VChart v-if="hasEcharts" :option="sliceChartOption" autoresize class="slice-chart" />
        <div v-else class="slice-fallback">
          <div
            v-for="(iv, idx) in ivGrid[selectedMaturityIdx] ?? []"
            :key="idx"
            class="slice-data-row"
          >
            <span>K/S = {{ strikes[idx]?.toFixed(2) }}</span>
            <span>IV = {{ (iv * 100).toFixed(2) }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.surface-chart-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, #111827);
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
.symbol-input {
  width: 90px; padding: 4px 8px; border: 1px solid #374151;
  border-radius: 4px; background: #1f2937; color: #e5e7eb; font-size: 13px;
}
.days-slider {
  width: 80px;
  accent-color: #3b82f6;
}
.days-label {
  font-size: 11px;
  color: #9ca3af;
  min-width: 30px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px;
}
.refresh-btn:hover { background: #374151; }

.surface-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}
.surface-section {
  flex: 1;
  min-height: 0;
}
.surface-chart {
  width: 100%;
  height: 100%;
}

/* Fallback heatmap table */
.fallback-heatmap {
  max-height: 100%;
  overflow: auto;
}
.heatmap-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}
.heatmap-table th {
  background: #1f2937;
  color: #9ca3af;
  padding: 3px 6px;
  text-align: center;
  border: 1px solid #374151;
  white-space: nowrap;
}
.heatmap-table td {
  text-align: center;
  border: 1px solid #374151;
  font-variant-numeric: tabular-nums;
}
.maturity-label {
  background: #1f2937;
  color: #9ca3af;
  padding: 3px 6px;
}
.heatmap-cell {
  padding: 4px 6px;
  font-size: 11px;
}

/* Slice section */
.slice-section {
  flex-shrink: 0;
  height: 40%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 120px;
}
.slice-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}
.slice-label {
  font-size: 11px;
  color: #9ca3af;
}
.slice-select {
  padding: 2px 8px;
  border: 1px solid #374151;
  border-radius: 4px;
  background: #1f2937;
  color: #e5e7eb;
  font-size: 12px;
}
.slice-chart {
  flex: 1;
  width: 100%;
}
.slice-fallback {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  overflow-y: auto;
  padding: 4px 0;
}
.slice-data-row {
  display: flex;
  gap: 12px;
  padding: 3px 8px;
  border-radius: 4px;
  background: #1f2937;
  border: 1px solid #374151;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
</style>
