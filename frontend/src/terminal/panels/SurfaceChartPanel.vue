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
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

use([LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const surfaceData = ref<number[][]>([])
const loading = ref(false); const loadError = ref('')
const { fetchWithCache } = usePanelCache()
const hasEcharts = computed(() => { try { return typeof VChart !== 'undefined' } catch { return false } })

async function loadSurface() { const app = (window as any).go?.main?.App; if (!app) return; loading.value = true; loadError.value = ''; try { const { data } = await fetchWithCache<any>(`vol_surface:${symbol.value}`, () => app.GetVolatilitySurface(symbol.value), 15 * 60 * 1000); surfaceData.value = data || [] } catch(e: any) { loadError.value = e?.message || String(e); logger.error('[SurfaceChart] fetch:', e); surfaceData.value = [] } finally { loading.value = false } }

onMounted(loadSurface)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => { if (pg.linked && newSym && newSym !== symbol.value) { symbol.value = newSym; loadSurface() } })

const chartOption = computed(() => {
  if (!surfaceData.value.length) return {}
  const windows = surfaceData.value.map(r => r[0].toFixed(0) + 'd')
  const vols = surfaceData.value.map(r => +(r[1] * 100).toFixed(1))
  const theme = useChartTheme()
  return { backgroundColor: 'transparent', grid: { top: 20, right: 20, bottom: 30, left: 50 }, xAxis: { type: 'category', data: windows, name: '窗口 (天)', axisLabel: { color: theme.axisColor, fontSize: 10 } }, yAxis: { type: 'value', name: '波动率 (%)', axisLabel: { color: theme.axisColor, fontSize: 10 } }, series: [{ type: 'line', data: vols, smooth: true, lineStyle: { color: theme.palette[5], width: 2 }, areaStyle: { color: theme.palette[5] + '26' }, itemStyle: { color: theme.palette[5] } }], tooltip: { trigger: 'axis' } }
})
</script>

<template>
  <div class="surface-chart-panel">
    <PanelHeader title="波动率曲面" :controls="[{ icon: 'refresh', title: '刷新', action: loadSurface, loading }]" />
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <LoadingState v-if="loading" type="chart" />
    <VChart v-else-if="hasEcharts && surfaceData.length" :option="chartOption" autoresize class="surface-chart" />
    <EmptyState v-else title="暂无数据" />
  </div>
</template>

<style scoped>
.surface-chart-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.surface-chart { flex: 1; width: 100%; min-height: 0; }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
</style>
