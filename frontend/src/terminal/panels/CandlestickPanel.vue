<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CandlestickChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import * as echarts from 'echarts'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'

use([CandlestickChart, BarChart, TitleComponent, TooltipComponent, GridComponent, DataZoomComponent, CanvasRenderer])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const hasEcharts = computed(() => !!(echarts && VChart))

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const interval = ref(props.params?.interval || '1d')
const ohlcvData = ref<(string | number)[][]>([])
const loading = ref(false)

async function loadOHLCV(sym: string) {
  loading.value = true
  try {
    const end = Math.floor(Date.now() / 1000)
    const start = end - 90 * 86400
    const result = await (window as any).go.main.App.FetchOHLCV(detectMarket(sym), sym, '1d', start, end)
    const bars = Array.isArray(result) ? result[0] : result
    ohlcvData.value = (bars as any[]).map((b: any) => {
      const date = typeof b.date === 'string' ? b.date : new Date(b.date || b.Date).toISOString().slice(0, 10)
      return [date, b.open ?? b.Open ?? 0, b.close ?? b.Close ?? 0, b.low ?? b.Low ?? 0, b.high ?? b.High ?? 0, b.volume ?? b.Volume ?? 0]
    })
  } catch {
    ohlcvData.value = []
  } finally {
    loading.value = false
  }
}

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
    loadOHLCV(newSymbol)
  }
})

// Regenerate data on interval change
watch(interval, () => {
  loadOHLCV(symbol.value)
})

const option = computed(() => {
  if (ohlcvData.value.length === 0) return {}
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
  }
  loadOHLCV(symbol.value)
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
      <div v-if="loading" class="chart-fallback">加载中...</div>
      <VChart v-else-if="hasEcharts && ohlcvData.length > 0" :key="symbol" :option="option" autoresize class="kline-chart" />
      <div v-else class="chart-fallback">--</div>
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
