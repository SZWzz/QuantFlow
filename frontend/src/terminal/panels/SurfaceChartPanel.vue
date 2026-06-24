<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([LineChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const surfaceData = ref<number[][]>([])

const hasEcharts = computed(() => { try { return typeof VChart !== 'undefined' } catch { return false } })

async function loadSurface() {
  const app = (window as any).go?.main?.App
  if (!app) return
  try {
    const data = await app.GetVolatilitySurface(symbol.value)
    surfaceData.value = data || []
  } catch { surfaceData.value = [] }
}

onMounted(loadSurface)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) { symbol.value = newSym; loadSurface() }
})

const chartOption = computed(() => {
  if (!surfaceData.value.length) return {}
  const windows = surfaceData.value.map(r => r[0].toFixed(0) + 'd')
  const vols = surfaceData.value.map(r => +(r[1] * 100).toFixed(1))
  return {
    backgroundColor: 'transparent',
    grid: { top: 20, right: 20, bottom: 30, left: 50 },
    xAxis: { type: 'category', data: windows, name: '窗口 (天)', axisLabel: { color: '#6b7280', fontSize: 10 } },
    yAxis: { type: 'value', name: '波动率 (%)', axisLabel: { color: '#6b7280', fontSize: 10 } },
    series: [{ type: 'line', data: vols, smooth: true, lineStyle: { color: '#534ab7', width: 2 }, areaStyle: { color: 'rgba(83,74,183,0.15)' }, itemStyle: { color: '#534ab7' } }],
    tooltip: { trigger: 'axis' },
  }
})
</script>

<template>
  <div class="surface-chart-panel">
    <div class="panel-header"><h3>波动率期限结构</h3><button class="refresh-btn" @click="loadSurface">&#x21bb;</button></div>
    <div class="surface-content">
      <VChart v-if="hasEcharts && surfaceData.length" :option="chartOption" autoresize class="surface-chart" />
      <div v-else class="no-data">{{ surfaceData.length === 0 ? '加载中或暂无数据' : '' }}</div>
    </div>
  </div>
</template>

<style scoped>
.surface-chart-panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg, #111827); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.refresh-btn { padding: 4px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; cursor: pointer; }
.surface-content { flex: 1; min-height: 0; }
.surface-chart { width: 100%; height: 100%; }
.no-data { color: #6b7280; padding: 20px; text-align: center; }
</style>
