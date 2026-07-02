<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, shallowRef } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

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

const symbols = ref<string[]>(assetPresets[0].symbols)
const lookback = ref(60)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const loadError = ref('')
const matrix = ref<Record<string, Record<string, number>>>({})
const assetList = ref<string[]>([])
const activePreset = ref(0)
let chartInstance: any = null
let renderTimer: ReturnType<typeof setTimeout> | null = null

const colorScale = computed(() => {
  const values = Object.values(matrix.value).flatMap(r => Object.values(r))
  const max = Math.max(...values.map(Math.abs), 0.01)
  return { max }
})

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetCorrelationMatrix || symbols.value.length < 2) return
  loading.value = true
  loadError.value = ''
  try {
    const key = 'correlation:' + symbols.value.join(',') + ':' + lookback.value
    const result = await fetchWithCache<any>(key, () => app.GetCorrelationMatrix(symbols.value, lookback.value)).then(r => r.data)
    matrix.value = result || {}
    assetList.value = symbols.value
  } catch (e) {
    loadError.value = (e as any)?.message || String(e)
    matrix.value = {}
  } finally {
    loading.value = false
  }
  renderTimer = setTimeout(renderChart, 300)
}

function renderChart() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const echarts = (window as any).echarts
  const el = document.getElementById('correlation-chart')
  if (!el) return
  if (!chartInstance) chartInstance = echarts.init(el)

  const symbols = assetList.value
  const data: number[] = []
  const source: string[] = []
  const target: string[] = []

  for (let i = 0; i < symbols.length; i++) {
    for (let j = 0; j < symbols.length; j++) {
      if (i === j) continue
      const val = matrix.value[symbols[i]]?.[symbols[j]]
      if (val !== undefined) {
        data.push(val)
        source.push(symbols[i])
        target.push(symbols[j])
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
      data: symbols.flatMap((s, i) =>
        symbols.map((t, j) => [i, j, matrix.value[s]?.[t] || 0])
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
    xAxis: { type: 'category', data: symbols, axisLabel: { fontSize: 9, color: '#9ca3af', rotate: 45 } },
    yAxis: { type: 'category', data: symbols, axisLabel: { fontSize: 9, color: '#9ca3af' } },
    grid: { left: 60, right: 20, top: 20, bottom: 60 },
  }
  chartInstance.setOption(option, true)
}

function selectPreset(idx: number) {
  activePreset.value = idx
  symbols.value = assetPresets[idx].symbols
  fetchData()
}

onUnmounted(() => {
  if (renderTimer) clearTimeout(renderTimer)
})
onMounted(fetchData)
</script>

<template>
  <div class="cross-asset-correlation-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.cross_asset_corr') }}</h3>
      <div class="preset-scroll">
        <button v-for="(p, idx) in assetPresets" :key="p.label"
          :class="['preset-tab', { active: activePreset === idx }]"
          :style="activePreset === idx ? { borderColor: p.color, color: p.color } : {}"
          @click="selectPreset(idx)">
          {{ p.label }}
        </button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <SkeletonPanel v-if="loading && assetList.length === 0" type="chart" />

    <template v-else-if="assetList.length > 0">
      <div id="correlation-chart" class="corr-chart"></div>
      <div class="corr-legend">
        <span class="legend-item"><span class="legend-dot" style="background:#dc2626" />&gt;0.7</span>
        <span class="legend-item"><span class="legend-dot" style="background:#f97316" />0.4~0.7</span>
        <span class="legend-item"><span class="legend-dot" style="background:#fbbf24" />0.1~0.4</span>
        <span class="legend-item"><span class="legend-dot" style="background:#60a5fa" />-0.3~-0.1</span>
        <span class="legend-item"><span class="legend-dot" style="background:#3b82f6" />&lt;-0.3</span>
      </div>
    </template>

    <div v-else class="empty-state">{{ $t('common.no_data') }}</div>
  </div>
</template>

<style scoped>
.cross-asset-correlation-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; flex-wrap: wrap; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
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
.corr-legend { display: flex; gap: 16px; justify-content: center; margin-top: 8px; font-size: 10px; color: var(--color-text-tertiary); flex-shrink: 0; }
.legend-item { display: flex; align-items: center; gap: 4px; }
.legend-dot { width: 8px; height: 8px; border-radius: 2px; }
</style>
