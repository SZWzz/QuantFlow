<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, shallowRef } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { HeatmapChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, VisualMapComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { pearsonMatrix } from '@/lib/stats'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

use([HeatmapChart, TitleComponent, TooltipComponent, GridComponent, VisualMapComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

type CorrelationTab = 'custom' | 'presets'
const activeView = ref<CorrelationTab>('custom')
const viewTabs = [{ key: 'custom', label: '自定义' }, { key: 'presets', label: '预设' }]

const symbolText = ref(props.params?.symbols ?? '600519\n000858\n000001\n300750\n002594\n601318\n600036\n000002')
const lookback = ref(props.params?.lookback ?? 60)
const matrix = ref<number[][] | null>(null)
const symbols = ref<string[]>([])
const { fetchWithCache } = usePanelCache()
const chartTheme = useChartTheme()
const customLoading = ref(false)
const customError = ref('')
const hasECharts = ref(false)

const firstSymbol = computed(() => symbols.value.length > 0 ? symbols.value[0] : undefined)
const { name } = useStockName(firstSymbol)
const lookbackOptions = [30, 60, 90, 252]

function parseSymbols(): string[] { return symbolText.value.split('\n').map((s: string) => s.trim()).filter((s: string) => s.length > 0) }

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

async function compute() {
  const syms = parseSymbols(); symbols.value = syms
  if (syms.length < 2) { matrix.value = null; return }
  const app = useWailsApp()
  if (!app) { matrix.value = null; return }
  customLoading.value = true; customError.value = ''
  try {
    const key = 'correlation:' + syms.join(',') + ':' + lookback.value
    type CorrMap = Record<string, Record<string, number>>
    const { data: corrMatrix } = await fetchWithCache<CorrMap>(key, () => app.GetCorrelationMatrix(syms, lookback.value))
    const m: number[][] = syms.map(si => syms.map(sj => corrMatrix?.[si]?.[sj] ?? 0))
    matrix.value = m
  } catch (e: any) { customError.value = e?.message || String(e); matrix.value = null }
  finally { customLoading.value = false }
}

const chartOption = computed(() => {
  if (!matrix.value || symbols.value.length === 0) return null
  const syms = symbols.value; const n = syms.length
  const heatData: { value: [number, number, number]; itemStyle?: Record<string, unknown> }[] = []
  for (let i = 0; i < n; i++) {
    for (let j = 0; j < n; j++) {
      const val = matrix.value[i][j]
      if (j <= i) { heatData.push({ value: [j, i, +val.toFixed(4)] }) }
      else { heatData.push({ value: [j, i, 0], itemStyle: { color: 'transparent' } }) }
    }
  }
  const theme = chartTheme
  return {
    backgroundColor: 'transparent',
    tooltip: { formatter: (p: { value: [number, number, number] }) => { const [xj, yi, v] = p.value; return `${syms[yi]} × ${syms[xj]}<br/>Correlation: ${v.toFixed(4)}` }, backgroundColor: theme.bgColor, borderColor: theme.splitColor, textStyle: { color: theme.tooltipText, fontSize: 12 } },
    grid: { left: '12%', right: '8%', top: '5%', bottom: '10%' },
    xAxis: { type: 'category' as const, data: syms, axisLabel: { color: theme.axisColor, fontSize: 10, rotate: 45 }, axisLine: { lineStyle: { color: theme.splitColor } }, position: 'top' as const },
    yAxis: { type: 'category' as const, data: syms, axisLabel: { color: theme.axisColor, fontSize: 10 }, axisLine: { lineStyle: { color: theme.splitColor } } },
    visualMap: { min: -1, max: 1, calculable: false, orient: 'vertical' as const, right: '0%', top: 'middle', inRange: { color: [theme.downColor, theme.bgColor, theme.upColor] }, textStyle: { color: theme.axisColor, fontSize: 10 } },
    series: [{ type: 'heatmap', data: heatData, label: { show: true, fontSize: 10, color: theme.textColor, formatter: (p: { value: [number, number, number] }) => { const [, , v] = p.value; if (v === 0) return ''; return v.toFixed(2) } }, emphasis: { itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.3)' } } }],
  }
})

function cellBg(r: number): string {
  const pal = chartTheme.palette
  if (r > 0.7) return pal[3] + '73'
  if (r > 0.3) return pal[3] + '40'
  if (r > -0.3) return 'var(--color-bg-subtle)'
  if (r > -0.7) return pal[0] + '40'
  return pal[0] + '73'
}

// ── Presets ──
interface AssetGroup { label: string; symbols: string[]; color: string }
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

async function fetchPresetData() {
  const app = useWailsApp()
  if (!app?.GetCorrelationMatrix || presetSymbols.value.length < 2) return
  presetLoading.value = true; presetError.value = ''
  try {
    const key = 'correlation:' + presetSymbols.value.join(',') + ':' + presetLookback.value
    const result = await fetchWithCache<any>(key, () => app.GetCorrelationMatrix(presetSymbols.value, presetLookback.value)).then(r => r.data)
    presetMatrix.value = result || {}; presetAssetList.value = presetSymbols.value
  } catch (e) { presetError.value = (e as any)?.message || String(e); presetMatrix.value = {} }
  finally { presetLoading.value = false }
  renderTimer = setTimeout(renderPresetChart, 300)
}

function presetHeatColor(v: number, pal: string[]): string {
  if (v > 0.7) return pal[3]
  if (v > 0.4) return pal[2]
  if (v > 0.1) return pal[2]
  if (v < -0.3) return pal[0]
  if (v < -0.1) return pal[0]
  return 'var(--color-bg-subtle)'
}

function renderPresetChart() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const echarts = (window as any).echarts
  const el = document.getElementById('correlation-preset-chart')
  if (!el) return
  if (!chartInstance) chartInstance = echarts.init(el)
  const syms = presetAssetList.value; const pal = chartTheme.palette
  const option = {
    tooltip: { formatter: (params: any) => { const [i, j, v] = params.data; return `${syms[i]} → ${syms[j]}: ${(v || 0).toFixed(3)}` } },
    series: [{ type: 'heatmap', data: syms.flatMap((s, i) => syms.map((t, j) => [i, j, presetMatrix.value[s]?.[t] || 0])),
      label: { show: true, formatter: (p: any) => (p.data[2] || 0).toFixed(2), fontSize: 9, color: chartTheme.axisColor },
      itemStyle: { color: (p: any) => presetHeatColor(p.data[2], pal) },
    }],
    xAxis: { type: 'category', data: syms, axisLabel: { fontSize: 9, color: chartTheme.axisColor, rotate: 45 } },
    yAxis: { type: 'category', data: syms, axisLabel: { fontSize: 9, color: chartTheme.axisColor } },
    grid: { left: 60, right: 20, top: 20, bottom: 60 },
  }
  chartInstance.setOption(option, true)
}

function selectPreset(idx: number) { activePreset.value = idx; presetSymbols.value = assetPresets[idx].symbols; fetchPresetData() }

onMounted(() => { try { hasECharts.value = true } catch { hasECharts.value = false } })
onUnmounted(() => { if (renderTimer) clearTimeout(renderTimer) })
</script>

<template>
  <div class="correlation-panel">
    <PanelHeader title="相关性分析" :tabs="viewTabs" :active-tab="activeView" @tab-change="(k: string) => { activeView = k as CorrelationTab; if (k === 'presets' && !presetAssetList.length) fetchPresetData() }">
      <template #controls>
        <button v-if="addToWfControl" class="btn btn-sm" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
      </template>
    </PanelHeader>

    <!-- Custom -->
    <div v-if="activeView === 'custom'" class="corr-content">
      <div class="controls-row">
        <textarea v-model="symbolText" class="symbol-input" rows="4" placeholder="输入代码，每行一个"></textarea>
        <div class="controls-right">
          <label class="control-label">回溯<select v-model="lookback" class="lookback-select"><option v-for="opt in lookbackOptions" :key="opt" :value="opt">{{ opt }}d</option></select></label>
          <button class="btn btn-primary" @click="compute">计算</button>
        </div>
      </div>

      <div v-if="customError" class="panel-error">{{ customError }}</div>
      <LoadingState v-if="customLoading" type="chart" />
      <EmptyState v-else-if="!matrix" title="输入标的并点击计算" description="输入股票代码后点击计算查看相关性矩阵" />
      <div v-else class="chart-body">
        <template v-if="hasECharts">
          <VChart v-if="chartOption" :option="chartOption" autoresize class="echarts-container" />
        </template>
        <div v-else class="fallback-table-wrap">
          <table class="corr-table">
            <thead><tr><th></th><th v-for="s in symbols" :key="s">{{ s }}</th></tr></thead>
            <tbody><tr v-for="(row, i) in matrix" :key="symbols[i]"><td class="row-label">{{ symbols[i] }}</td><td v-for="(val, j) in row" :key="`${i}-${j}`" class="corr-cell" :style="j <= i ? { background: cellBg(val) } : { opacity: 0.2 }"><template v-if="j <= i">{{ val.toFixed(2) }}</template><template v-else>-</template></td></tr></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Presets -->
    <div v-if="activeView === 'presets'" class="corr-content">
      <div class="presets-header">
        <div class="preset-scroll">
          <button v-for="(p, idx) in assetPresets" :key="p.label" :class="['btn btn-sm', { 'btn-primary': activePreset === idx }]" @click="selectPreset(idx)">{{ p.label }}</button>
        </div>
        <button class="btn btn-sm" @click="fetchPresetData" :disabled="presetLoading">⟳</button>
      </div>

      <div v-if="presetError" class="panel-error">{{ presetError }}</div>
      <LoadingState v-if="presetLoading && presetAssetList.length === 0" type="chart" />
      <template v-else-if="presetAssetList.length > 0">
        <div id="correlation-preset-chart" class="corr-chart"></div>
        <div class="corr-legend">
          <span class="legend-item"><span class="legend-dot" :style="{ background: chartTheme.palette[3] }" />&gt;0.7</span>
          <span class="legend-item"><span class="legend-dot" :style="{ background: chartTheme.palette[2] }" />0.4~0.7</span>
          <span class="legend-item"><span class="legend-dot" :style="{ background: chartTheme.palette[2] }" />0.1~0.4</span>
          <span class="legend-item"><span class="legend-dot" :style="{ background: chartTheme.palette[0] }" />-0.3~-0.1</span>
          <span class="legend-item"><span class="legend-dot" :style="{ background: chartTheme.palette[0] }" />&lt;-0.3</span>
        </div>
      </template>
      <EmptyState v-else title="暂无数据" />
    </div>
  </div>
</template>

<style scoped>
.correlation-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.corr-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.controls-row { display: flex; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); align-items: stretch; }
.symbol-input { flex: 1; background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); color: var(--color-text-primary); border-radius: var(--radius-sm); padding: var(--space-sm); font-size: var(--font-xs); font-family: var(--font-mono); resize: vertical; }
.controls-right { display: flex; flex-direction: column; justify-content: space-between; gap: var(--space-sm); }
.control-label { font-size: var(--font-xs); color: var(--color-text-secondary); display: flex; flex-direction: column; gap: var(--space-xs); }
.lookback-select { background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); color: var(--color-text-primary); border-radius: var(--radius-sm); padding: var(--space-xs); font-size: var(--font-xs); }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
.chart-body { flex: 1; min-height: 0; display: flex; align-items: center; justify-content: center; }
.echarts-container { width: 100%; height: 100%; }
.fallback-table-wrap { width: 100%; height: 100%; overflow: auto; padding: var(--space-sm) var(--panel-padding); scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent; }
.corr-table { border-collapse: collapse; font-size: var(--font-xs); width: 100%; }
.corr-table th, .corr-table td { padding: var(--space-xs) var(--space-sm); text-align: center; min-width: 56px; white-space: nowrap; }
.corr-table th { color: var(--color-text-secondary); font-weight: 500; border-bottom: 1px solid var(--color-border-strong); }
.row-label { font-weight: 600; color: var(--color-text-primary); text-align: left !important; }
.corr-cell { border-radius: var(--radius-xs, 2px); border: 1px solid transparent; font-variant-numeric: tabular-nums; }
.presets-header { display: flex; align-items: center; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.preset-scroll { display: flex; gap: var(--space-xs); overflow-x: auto; flex: 1; }

.corr-chart { flex: 1; min-height: 200px; }
.corr-legend { display: flex; gap: var(--space-lg); justify-content: center; padding: var(--space-sm) 0; font-size: var(--font-xs); color: var(--color-text-tertiary); flex-shrink: 0; }
.legend-item { display: flex; align-items: center; gap: var(--space-xs); }
.legend-dot { width: 8px; height: 8px; border-radius: 2px; }
</style>
