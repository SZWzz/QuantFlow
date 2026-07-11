<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { logger } from '@/lib/logger'

use([LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const surfaceData = ref<number[][]>([])
const loading = ref(false)
const loadError = ref('')
const { fetchWithCache } = usePanelCache()

const hasEcharts = computed(() => { try { return typeof VChart !== 'undefined' } catch { return false } })

async function loadSurface() {
  const app = (window as any).go?.main?.App
  if (!app) return
  loading.value = true
  loadError.value = ''
  try {
    const { data } = await fetchWithCache<any>(`vol_surface:${symbol.value}`, () => app.GetVolatilitySurface(symbol.value), 15 * 60 * 1000)
    surfaceData.value = data || []
  } catch(e: any) { loadError.value = e?.message || String(e); logger.error('[SurfaceChart] fetch:', e); surfaceData.value = [] }
  finally { loading.value = false }
}

onMounted(loadSurface)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) { symbol.value = newSym; loadSurface() }
})

const chartOption = computed(() => {
  if (!surfaceData.value.length) return {}
  const windows = surfaceData.value.map(r => r[0].toFixed(0) + 'd')
  const vols = surfaceData.value.map(r => +(r[1] * 100).toFixed(1))
  const theme = useChartTheme()
  return {
    backgroundColor: 'transparent',
    grid: { top: 20, right: 20, bottom: 30, left: 50 },
    xAxis: { type: 'category', data: windows, name: '窗口 (天)', axisLabel: { color: theme.axisColor, fontSize: 10 } },
    yAxis: { type: 'value', name: '波动率 (%)', axisLabel: { color: theme.axisColor, fontSize: 10 } },
    series: [{ type: 'line', data: vols, smooth: true, lineStyle: { color: '#534ab7', width: 2 }, areaStyle: { color: 'rgba(83,74,183,0.15)' }, itemStyle: { color: '#534ab7' } }],
    tooltip: { trigger: 'axis' },
  }
})
</script>

<template>
  <div class="surface-chart-panel">
    <div class="panel-header"><h3>{{ $t('misc.volatility_surface') }}</h3><button class="refresh-btn" @click="loadSurface" :disabled="loading">&#x21bb;</button></div>
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div class="surface-content">
      <div v-if="loading" class="no-data">{{ $t('common.loading') }}</div>
      <VChart v-else-if="hasEcharts && surfaceData.length" :option="chartOption" autoresize class="surface-chart" />
      <div v-else class="no-data">{{ surfaceData.length === 0 ? '暂无数据' : '' }}</div>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: var(--color-up-soft); color: var(--color-up); font-size: 12px; }
.surface-chart-panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg, var(--color-bg-panel)); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; }
.surface-content { flex: 1; min-height: 0; }
.surface-chart { width: 100%; height: 100%; }
.no-data { color: var(--color-text-tertiary); padding: 20px; text-align: center; }
</style>
