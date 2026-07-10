<script setup lang="ts">
import { shallowRef, watch, onUnmounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { CandlestickChart, BarChart, LineChart, ScatterChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, MarkLineComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import type { ECBasicOption } from 'echarts/types/dist/shared'

use([CanvasRenderer, CandlestickChart, BarChart, LineChart, ScatterChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, MarkLineComponent])

const props = defineProps<{
  option: ECBasicOption
  symbol: string
  loading?: boolean
}>()

const emit = defineEmits<{ dataZoom: [params: any] }>()

const chartRef = shallowRef<InstanceType<typeof VChart>>()

function refreshSize() {
  chartRef.value?.resize?.()
}

const getEchartsInstance = () => (chartRef.value as any)?.chart ?? null
defineExpose({ refreshSize, getEchartsInstance })

// Watch for chart ready and bind datazoom event
watch(chartRef, (ref) => {
  const inst = (ref as any)?.chart
  if (inst) {
    inst.on('datazoom', (params: any) => emit('dataZoom', params))
  }
})

onUnmounted(() => {
  const inst = (chartRef.value as any)?.chart
  if (inst) inst.off('datazoom')
})
</script>

<template>
  <VChart
    ref="chartRef"
    :key="`kc-${symbol}`"
    :option="option"
    autoresize
    style="height: 100%; width: 100%"
  />
</template>
