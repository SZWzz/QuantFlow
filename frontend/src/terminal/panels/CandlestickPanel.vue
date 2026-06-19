<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CandlestickChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

use([CandlestickChart, BarChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const symbol = ref(props.params?.symbol || 'AAPL')
const interval = ref(props.params?.interval || '1d')

// Generate mock OHLCV data
function generateMockData(rows: number) {
  const data: (string | number)[][] = []
  let close = 195
  const baseDate = new Date('2024-06-01')
  for (let i = 0; i < rows; i++) {
    const change = (Math.random() - 0.5) * 4
    const open = close
    close = close + change
    const high = Math.max(open, close) + Math.random() * 2
    const low = Math.min(open, close) - Math.random() * 2
    const volume = Math.floor(Math.random() * 50000000) + 10000000
    const date = new Date(baseDate)
    date.setDate(date.getDate() + i)
    data.push([
      date.toISOString().slice(0, 10),
      +open.toFixed(2),
      +close.toFixed(2),
      +low.toFixed(2),
      +high.toFixed(2),
      volume,
    ])
  }
  return data
}

const ohlcvData = ref(generateMockData(90))

const option = computed(() => ({
  backgroundColor: 'transparent',
  grid: [
    { left: '8%', right: '3%', top: '5%', height: '60%' },
    { left: '8%', right: '3%', top: '72%', height: '18%' },
  ],
  xAxis: [
    {
      type: 'category',
      data: ohlcvData.value.map((d) => d[0]),
      gridIndex: 0,
      axisLine: { lineStyle: { color: '#30363d' } },
      axisLabel: { color: '#5a6380', fontSize: 10 },
      boundaryGap: true,
    },
    {
      type: 'category',
      data: ohlcvData.value.map((d) => d[0]),
      gridIndex: 1,
      axisLine: { lineStyle: { color: '#30363d' } },
      axisLabel: { show: false },
      boundaryGap: true,
    },
  ],
  yAxis: [
    {
      type: 'value',
      gridIndex: 0,
      scale: true,
      splitLine: { lineStyle: { color: '#1a2a3e' } },
      axisLabel: { color: '#5a6380', fontSize: 10 },
    },
    {
      type: 'value',
      gridIndex: 1,
      axisLabel: { color: '#5a6380', fontSize: 9 },
      splitLine: { lineStyle: { color: '#1a2a3e' } },
    },
  ],
  series: [
    {
      name: 'Candlestick',
      type: 'candlestick',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: ohlcvData.value.map((d) => [d[1], d[2], d[3], d[4]]),
      itemStyle: {
        color: '#3fb950',
        color0: '#f85149',
        borderColor: '#3fb950',
        borderColor0: '#f85149',
      },
    },
    {
      name: 'Volume',
      type: 'bar',
      xAxisIndex: 1,
      yAxisIndex: 1,
      data: ohlcvData.value.map((d, i) => {
        const prevClose = i > 0 ? ohlcvData.value[i - 1][2] : d[1]
        return {
          value: d[5],
          itemStyle: {
            color: d[2] >= prevClose ? 'rgba(63,185,80,0.4)' : 'rgba(248,81,73,0.4)',
          },
        }
      }),
    },
  ],
  dataZoom: [
    { type: 'inside', xAxisIndex: [0, 1], start: 60, end: 100 },
  ],
  tooltip: {
    trigger: 'axis',
    axisPointer: { type: 'cross' },
    backgroundColor: '#1c2333',
    borderColor: '#30363d',
    textStyle: { color: '#c9d1d9', fontSize: 11 },
  },
}))
</script>

<template>
  <div class="candlestick-panel">
    <div class="chart-header">
      <span class="chart-symbol">{{ symbol }}</span>
      <div class="interval-btns">
        <button
          v-for="intv in ['5m', '1h', '1d', '1w']"
          :key="intv"
          :class="{ active: interval === intv }"
          @click="interval = intv"
        >
          {{ intv }}
        </button>
      </div>
    </div>
    <div class="chart-body">
      <VChart :option="option" autoresize />
    </div>
  </div>
</template>

<style scoped>
.candlestick-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  border-bottom: 1px solid #0f3460;
}

.chart-symbol {
  font-weight: 600;
  font-size: 13px;
  color: #e0e0e0;
}

.interval-btns {
  display: flex;
  gap: 2px;
}

.interval-btns button {
  padding: 2px 8px;
  border: 1px solid #2a3a5c;
  background: transparent;
  color: #5a6380;
  border-radius: 3px;
  font-size: 10px;
  cursor: pointer;
}

.interval-btns button.active {
  background: #0f3460;
  border-color: #58a6ff;
  color: #58a6ff;
}

.chart-body {
  flex: 1;
  min-height: 0;
}
</style>
