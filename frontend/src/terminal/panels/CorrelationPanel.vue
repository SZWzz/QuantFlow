<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, shallowRef } from 'vue'
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
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

use([HeatmapChart, TitleComponent, TooltipComponent, GridComponent, VisualMapComponent, CanvasRenderer])

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// ── Tabs: Custom / Presets ──
type CorrelationTab = 'custom' | 'presets'
const activeView = ref<CorrelationTab>('custom')

// ══════ Custom correlation ══════
const symbolText = ref(
  props.params?.symbols ??
    '600519\n000858\n000001\n300750\n002594\n601318\n600036\n000002',
)
const lookback = ref(props.params?.lookback ?? 60)
const matrix = ref<number[][] | null>(null)
const symbols = ref<string[]>([])
const { fetchWithCache } = usePanelCache()
const customLoading = ref(false)
const customError = ref('')
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

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

async function compute() {
  const syms = parseSymbols()
  symbols.value = syms
  if (syms.length < 2) {
    matrix.value = null
    return
  }
  const app = (window as any).go?.main?.App
  if (!app) { matrix.value = null; return }
  customLoading.value = true
  customError.value = ''
  try {
    const key = 'correlation:' + syms.join(',') + ':' + lookback.value
    type CorrMap = Record<string, Record<string, number>>
    const { data: corrMatrix } = await fetchWithCache<CorrMap>(key, () => app.GetCorrelationMatrix(syms, lookback.value))
    const m: number[][] = syms.map(si =>
      syms.map(sj => corrMatrix?.[si]?.[sj] ?? 0)
    )
    matrix.value = m
  } catch (e: any) {
    customError.value = e?.message || String(e)
    matrix.value = null
  } finally {
    customLoading.value = false
  }
}

const chartOption = computed(() => {
  if (!matrix.value || symbols.value.length === 0) return null

  const syms = symbols.value
  const n = syms.length

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

// ══════ Presets (cross-asset) correlation ══════
interface AssetGroup {
  label: string
  symbols: string[]
  color: string
}

const assetPresets: AssetGroup[] = [
  { label: '美股-科技', symbols: ['AAPL', 'MSFT', 'GOOGL', 'AMZN', 'NVDA'], color: '#3b82f6' },
  { label: '美股-金融', symbols: ['JPM', 'GS', 'BAC', 'WFC', 'C'], color: '#8b5cf6' },
  { label: '中美', symbols: ['AAPL', 'BABA', 'JD', 'NIO', 'MSFT'], color: '#22c55e' },
  { label: '跨资产', symbols: ['SPY', 'QQQ', 'TLT', 'GLD', 'USO'], color: '#f59e0b' },
  { label: 'A股-板块', symbols: ['000001', '600519', '300750', '601318', '600036'], color: '#ef4444' },
  { label: '加密+美股', symbols: ['BTCUSDT', 'ETHUSDT', 'MSTR', 'COIN', 'SQ'], color: '#ec4899' },
]

const presetSymbols = ref<string[]>(assetPresets[0].symbols)
const presetLookback = ref(60)
const presetLoading = ref(false)
const presetError = ref('')
const presetMatrix = ref<Record<string, Record<string, number>>>({})
const presetAssetList = ref<string[]>([])
const activePreset = ref(0)
let chartInstance: any = null
let renderTimer: ReturnType<typeof setTimeout> | null = null

const colorScale = computed(() => {
  const values = Object.values(presetMatrix.value).flatMap(r => Object.values(r))
  const max = Math.max(...values.map(Math.abs), 0.01)
  return { max }
})

async function fetchPresetData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetCorrelationMatrix || presetSymbols.value.length < 2) return
  presetLoading.value = true
  presetError.value = ''
  try {
    const key = 'correlation:' + presetSymbols.value.join(',') + ':' + presetLookback.value
    const result = await fetchWithCache<any>(key, () => app.GetCorrelationMatrix(presetSymbols.value, presetLookback.value)).then(r => r.data)
    presetMatrix.value = result || {}
    presetAssetList.value = presetSymbols.value
  } catch (e) {
    presetError.value = (e as any)?.message || String(e)
    presetMatrix.value = {}
  } finally {
    presetLoading.value = false
  }
  renderTimer = setTimeout(renderPresetChart, 300)
}

function renderPresetChart() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const echarts = (window as any).echarts
  const el = document.getElementById('correlation-preset-chart')
  if (!el) return
  if (!chartInstance) chartInstance = echarts.init(el)

  const syms = presetAssetList.value
  const data: number[] = []
  const source: string[] = []
  const target: string[] = []

  for (let i = 0; i < syms.length; i++) {
    for (let j = 0; j < syms.length; j++) {
      if (i === j) continue
      const val = presetMatrix.value[syms[i]]?.[syms[j]]
      if (val !== undefined) {
        data.push(val)
        source.push(syms[i])
        target.push(syms[j])
      }
    }
  }

  const option = {
    tooltip: {
      formatter: (params: any) => {
        const idx = params.dataIndex
        return `${source[idx]} → ${target[idx]}: ${(data[idx] || 0).toFixed(3)}`
      },
    },
    series: [{
      type: 'heatmap',
      data: syms.flatMap((s, i) =>
        syms.map((t, j) => [i, j, presetMatrix.value[s]?.[t] || 0])
      ),
      label: { show: true, formatter: (p: any) => (p.data[2] || 0).toFixed(2), fontSize: 9, color: '#9ca3af' },
      itemStyle: {
        color: (p: any) => {
          const v = p.data[2]
          if (v > 0.7) return '#dc2626'
          if (v > 0.4) return '#f97316'
          if (v > 0.1) return '#fbbf24'
          if (v < -0.3) return '#3b82f6'
          if (v < -0.1) return '#60a5fa'
          return '#374151'
        },
      },
    }],
    xAxis: { type: 'category', data: syms, axisLabel: { fontSize: 9, color: '#9ca3af', rotate: 45 } },
    yAxis: { type: 'category', data: syms, axisLabel: { fontSize: 9, color: '#9ca3af' } },
    grid: { left: 60, right: 20, top: 20, bottom: 60 },
  }
  chartInstance.setOption(option, true)
}

function selectPreset(idx: number) {
  activePreset.value = idx
  presetSymbols.value = assetPresets[idx].symbols
  fetchPresetData()
}

onMounted(() => {
  try {
    hasECharts.value = true
  } catch {
    hasECharts.value = false
  }
})

onUnmounted(() => {
  if (renderTimer) clearTimeout(renderTimer)
})
</script>

<template>
  <div class="correlation-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.correlation') }}{{ activeView === 'custom' && name ? ` — ${firstSymbol} ${name}` : '' }}</h3>
      <div class="view-tabs">
        <button :class="['view-tab', { active: activeView === 'custom' }]" @click="activeView = 'custom'">自定义</button>
        <button :class="['view-tab', { active: activeView === 'presets' }]" @click="activeView = 'presets'; if (!presetAssetList.length) fetchPresetData()">预设</button>
      </div>
      <button v-if="addToWfControl" class="wf-btn" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
    </div>

    <!-- ── Custom ── -->
    <template v-if="activeView === 'custom'">
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

      <div v-if="customError" class="panel-error">{{ customError }}</div>
      <div class="chart-body">
        <div v-if="customLoading" class="chart-fallback">{{ $t('common.loading') }}</div>
        <div v-else-if="!matrix" class="placeholder-msg">
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
    </template>

    <!-- ── Presets ── -->
    <template v-if="activeView === 'presets'">
      <div class="presets-header">
        <div class="preset-scroll">
          <button v-for="(p, idx) in assetPresets" :key="p.label"
            :class="['preset-tab', { active: activePreset === idx }]"
            :style="activePreset === idx ? { borderColor: p.color, color: p.color } : {}"
            @click="selectPreset(idx)">
            {{ p.label }}
          </button>
        </div>
        <button class="refresh-btn" @click="fetchPresetData" :disabled="presetLoading">⟳</button>
      </div>

      <div v-if="presetError" class="panel-error">{{ presetError }}</div>
      <SkeletonPanel v-if="presetLoading && presetAssetList.length === 0" type="chart" />

      <template v-else-if="presetAssetList.length > 0">
        <div id="correlation-preset-chart" class="corr-chart"></div>
        <div class="corr-legend">
          <span class="legend-item"><span class="legend-dot" style="background:#dc2626" />&gt;0.7</span>
          <span class="legend-item"><span class="legend-dot" style="background:#f97316" />0.4~0.7</span>
          <span class="legend-item"><span class="legend-dot" style="background:#fbbf24" />0.1~0.4</span>
          <span class="legend-item"><span class="legend-dot" style="background:#60a5fa" />-0.3~-0.1</span>
          <span class="legend-item"><span class="legend-dot" style="background:#3b82f6" />&lt;-0.3</span>
        </div>
      </template>

      <div v-else class="empty-state">{{ $t('common.no_data') }}</div>
    </template>
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
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px 6px;
  border-bottom: 1px solid var(--color-border-strong);
}
.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

/* View tabs */
.view-tabs {
  display: flex;
  gap: 0;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.view-tab {
  padding: 3px 12px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 11px;
}
.view-tab + .view-tab { border-left: 1px solid var(--color-border-strong); }
.view-tab.active { color: var(--color-accent); background: rgba(59,130,246,0.1); }

/* Custom controls */
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
  border-radius: var(--radius-sm);
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
  border-radius: var(--radius-sm);
  padding: 4px 6px;
  font-size: 12px;
}

.compute-btn {
  padding: 5px 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
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

/* Presets */
.presets-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border-strong);
  flex-shrink: 0;
}
.preset-scroll { display: flex; gap: 4px; overflow-x: auto; flex: 1; }
.preset-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 10px; white-space: nowrap;
}
.preset-tab.active { background: rgba(59,130,246,0.1); }

.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.corr-chart { flex: 1; min-height: 200px; }
.corr-legend { display: flex; gap: 16px; justify-content: center; padding: 8px 0; font-size: 10px; color: var(--color-text-tertiary); flex-shrink: 0; }
.legend-item { display: flex; align-items: center; gap: 4px; }
.legend-dot { width: 8px; height: 8px; border-radius: 2px; }
.panel-error { color: var(--color-warning, var(--color-up)); font-size: 11px; padding: 8px 14px; flex-shrink: 0; }
.chart-fallback { color: var(--color-text-tertiary); font-size: 13px; }

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
  font-size: 16px;
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
</style>
