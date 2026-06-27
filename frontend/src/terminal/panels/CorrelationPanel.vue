<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { HeatmapChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  VisualMapComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { pearsonMatrix } from '@/lib/stats'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'

use([HeatmapChart, TitleComponent, TooltipComponent, GridComponent, VisualMapComponent, CanvasRenderer])

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbolText = ref(
  props.params?.symbols ??
    '600519\n000858\n000001\n300750\n002594\n601318\n600036\n000002',
)
const lookback = ref(props.params?.lookback ?? 60)
const matrix = ref<number[][] | null>(null)
const symbols = ref<string[]>([])
const hasECharts = ref(false)

const firstSymbol = computed(() => symbols.value.length > 0 ? symbols.value[0] : undefined)
const { name } = useStockName(firstSymbol)

const lookbackOptions = [30, 60, 90, 252]

function parseSymbols(): string[] {
  return symbolText.value
    .split('\n')
    .map((s: string) => s.trim())
    .filter((s: string) => s.length > 0)
}

async function compute() {
  const syms = parseSymbols()
  symbols.value = syms
  if (syms.length < 2) {
    matrix.value = null
    return
  }
  const app = (window as any).go?.main?.App
  if (!app) { matrix.value = null; return }
  try {
    const corrMatrix = await app.GetCorrelationMatrix(syms, lookback.value)
    // Convert map[string]map[string]float64 to 2D array ordered by syms
    const m: number[][] = syms.map(si =>
      syms.map(sj => corrMatrix?.[si]?.[sj] ?? 0)
    )
    matrix.value = m
  } catch {
    matrix.value = null
  }
}

const chartOption = computed(() => {
  if (!matrix.value || symbols.value.length === 0) return null

  const syms = symbols.value
  const n = syms.length

  // Build heatmap data showing only lower triangle
  const heatData: { value: [number, number, number]; itemStyle?: Record<string, unknown> }[] = []

  for (let i = 0; i < n; i++) {
    for (let j = 0; j < n; j++) {
      const val = matrix.value[i][j]
      if (j <= i) {
        heatData.push({
          value: [j, i, +val.toFixed(4)],
        })
      } else {
        heatData.push({
          value: [j, i, 0],
          itemStyle: { color: 'transparent' },
        })
      }
    }
  }

  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    tooltip: {
      formatter: (p: { value: [number, number, number] }) => {
        const [xj, yi, v] = p.value
        const row = syms[yi]
        const col = syms[xj]
        return `${row} × ${col}<br/>Correlation: ${v.toFixed(4)}`
      },
      backgroundColor: theme.bgColor,
      borderColor: theme.splitColor,
      textStyle: { color: '#e5e7eb', fontSize: 12 },
    },
    grid: {
      left: '12%',
      right: '8%',
      top: '5%',
      bottom: '10%',
    },
    xAxis: {
      type: 'category' as const,
      data: syms,
      axisLabel: { color: theme.axisColor, fontSize: 10, rotate: 45 },
      axisLine: { lineStyle: { color: theme.splitColor } },
      position: 'top' as const,
    },
    yAxis: {
      type: 'category' as const,
      data: syms,
      axisLabel: { color: theme.axisColor, fontSize: 10 },
      axisLine: { lineStyle: { color: theme.splitColor } },
    },
    visualMap: {
      min: -1,
      max: 1,
      calculable: false,
      orient: 'vertical' as const,
      right: '0%',
      top: 'middle',
      inRange: {
        color: ['#3b82f6', theme.bgColor, '#ef4444'],
      },
      textStyle: { color: theme.axisColor, fontSize: 10 },
    },
    series: [
      {
        type: 'heatmap',
        data: heatData,
        label: {
          show: true,
          fontSize: 10,
          color: '#e5e7eb',
          formatter: (p: { value: [number, number, number] }) => {
            const [, , v] = p.value
            if (v === 0) return ''
            return v.toFixed(2)
          },
        },
        emphasis: {
          itemStyle: {
            shadowBlur: 8,
            shadowColor: 'rgba(255,255,255,0.3)',
          },
        },
      },
    ],
  }
})

function cellBg(r: number): string {
  if (r > 0.7) return 'rgba(239,68,68,0.45)'
  if (r > 0.3) return 'rgba(239,68,68,0.25)'
  if (r > -0.3) return 'rgba(55,65,81,0.3)'
  if (r > -0.7) return 'rgba(59,130,246,0.25)'
  return 'rgba(59,130,246,0.45)'
}

onMounted(() => {
  try {
    // ECharts is already loaded via static import above,
    // but we verify it resolved (won't throw in build).
    hasECharts.value = true
  } catch {
    hasECharts.value = false
  }
})
</script>

<template>
  <div class="correlation-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.correlation') }}{{ name ? ` — ${firstSymbol} ${name}` : '' }}</h3>
    </div>

    <div class="controls-row">
      <textarea
        v-model="symbolText"
        class="symbol-input"
        rows="4"
        placeholder="输入代码，每行一个"
      ></textarea>

      <div class="controls-right">
        <label class="control-label">
          回溯
          <select v-model="lookback" class="lookback-select">
            <option v-for="opt in lookbackOptions" :key="opt" :value="opt">
              {{ opt }}d
            </option>
          </select>
        </label>

        <button class="compute-btn" @click="compute">
          计算
        </button>
      </div>
    </div>

    <div class="chart-body">
      <div v-if="!matrix" class="placeholder-msg">
        Enter symbols and click 计算
      </div>

      <template v-else-if="hasECharts">
        <VChart v-if="chartOption" :option="chartOption" autoresize class="echarts-container" />
      </template>

      <!-- Fallback HTML table -->
      <div v-else class="fallback-table-wrap">
        <table class="corr-table">
          <thead>
            <tr>
              <th></th>
              <th v-for="s in symbols" :key="s">{{ s }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in matrix" :key="symbols[i]">
              <td class="row-label">{{ symbols[i] }}</td>
              <td
                v-for="(val, j) in row"
                :key="`${i}-${j}`"
                class="corr-cell"
                :style="j <= i ? { background: cellBg(val) } : { opacity: 0.2 }"
              >
                <template v-if="j <= i">{{ val.toFixed(2) }}</template>
                <template v-else>-</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.correlation-panel {
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
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border-strong);
  align-items: stretch;
}

.symbol-input {
  flex: 1;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 12px;
  font-family: 'Courier New', monospace;
  resize: vertical;
}

.controls-right {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 6px;
}

.control-label {
  font-size: 11px;
  color: var(--color-text-secondary);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.lookback-select {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: 4px;
  padding: 4px 6px;
  font-size: 12px;
}

.compute-btn {
  padding: 5px 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}
.compute-btn:hover {
  background: var(--color-border-strong);
}

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

.corr-table {
  border-collapse: collapse;
  font-size: 11px;
  width: 100%;
}

.corr-table th,
.corr-table td {
  padding: 4px 8px;
  text-align: center;
  min-width: 56px;
  white-space: nowrap;
}

.corr-table th {
  color: var(--color-text-secondary);
  font-weight: 500;
  border-bottom: 1px solid var(--color-border-strong);
}

.row-label {
  font-weight: 600;
  color: var(--color-text-primary);
  text-align: left !important;
}

.corr-cell {
  border-radius: 2px;
  border: 1px solid transparent;
  font-variant-numeric: tabular-nums;
}
</style>
