<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CandlestickChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import * as echarts from 'echarts'
import { useSymbolContext } from '@/stores/symbolContext'

use([CandlestickChart, BarChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const hasEcharts = computed(() => !!(echarts && VChart))

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const interval = ref(props.params?.interval || '1d')

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
    ohlcvData.value = generateMockData(90)
  }
})

// Regenerate data on interval change
watch(interval, () => {
  ohlcvData.value = generateMockData(interval.value === '1d' ? 90 : 60)
})

// Generate mock OHLCV data
function generateMockData(rows: number) {
  const data: (string | number)[][] = []
  let close = 195
  const baseDate = new Date('2026-04-01')
  for (let i = 0; i < rows; i++) {
    const change = (Math.random() - 0.5) * 4
    const open = close
    close = close + change
    const high = Math.max(open, close) + Math.random() * 2
    const low = Math.min(open, close) - Math.random() * 2
    const volume = Math.floor(Math.random() * 50000000) + 10000000
    const date = new Date(baseDate)
    date.setDate(date.getDate() + i)
    data.push([date.toISOString().slice(0, 10), +open.toFixed(2), +close.toFixed(2), +low.toFixed(2), +high.toFixed(2), volume])
  }
  return data
}

const ohlcvData = ref(generateMockData(90))

const option = computed(() => {
  const dates = ohlcvData.value.map((d: any) => d[0])
  const kdata = ohlcvData.value.map((d: any) => [d[1], d[2], d[3], d[4]]) // [open, close, low, high]
  const vdata = ohlcvData.value.map((d: any, i: number) => {
    const open = d[1] as number
    const close = d[2] as number
    return { value: d[5], itemStyle: { color: close >= open ? 'rgba(34,197,94,0.5)' : 'rgba(239,68,68,0.5)' } }
  })

  return {
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 10, top: 10, height: '62%' },
      { left: 60, right: 10, top: '78%', height: '15%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#334155' } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#334155' } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: '#64748b', fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { type: 'value', gridIndex: 1, axisLabel: { color: '#64748b', fontSize: 10 }, splitLine: { show: false } },
    ],
    series: [
      {
        type: 'candlestick', name: 'K线',
        data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        itemStyle: { color: '#22c55e', color0: '#ef4444', borderColor: '#22c55e', borderColor0: '#ef4444' },
      },
      {
        type: 'bar', name: 'Volume',
        data: vdata, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      },
    ],
    tooltip: { trigger: 'axis' as const },
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 50, end: 100 },
    ],
  }
})

onMounted(() => {
  const groupSym = ctx.getGroupSymbol(pg.groupId)
  if (groupSym && groupSym !== symbol.value) {
    symbol.value = groupSym
    ohlcvData.value = generateMockData(90)
  }
})
</script>

<template>
  <div class="candlestick-panel">
    <div class="chart-header">
      <span class="symbol-display">{{ symbol }}</span>
      <div class="interval-btns">
        <button v-for="i in ['1m','5m','15m','1h','1d','1w']" :key="i"
          :class="{ active: interval === i }" class="interval-btn"
          @click="interval = i">{{ i }}</button>
      </div>
    </div>
    <div class="chart-body">
      <VChart v-if="hasEcharts" :key="symbol" :option="option" autoresize class="kline-chart" />
      <div v-else class="chart-fallback">Chart loading...</div>
    </div>
  </div>
</template>

<style scoped>
.candlestick-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--color-bg-panel);
}
.chart-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; border-bottom: 1px solid var(--color-border);
}
.symbol-display {
  font-size: var(--font-lg); font-weight: 700;
  color: var(--color-brand);
}
.interval-btns { display: flex; gap: 2px; }
.interval-btn {
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-tertiary);
  border-radius: var(--radius-sm); cursor: pointer;
  font-size: var(--font-xs); font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}
.interval-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.interval-btn.active {
  background: var(--color-accent); color: #fff; border-color: var(--color-accent);
}
.chart-body { flex: 1; min-height: 0; padding: 8px; position: relative; }
.kline-chart { width: 100%; height: 100%; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
</style>
