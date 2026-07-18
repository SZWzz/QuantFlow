<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { GetStyleQuadrant, GetMarketSentiment, type StyleQuadrant, type MarketSentimentGauge } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'

const chartTheme = useChartTheme()

const market = ref('CN')
const quadrants = ref<StyleQuadrant[]>([])
const sentiment = ref<MarketSentimentGauge | null>(null)

onMounted(() => fetchData())

async function fetchData() {
  try {
    const [q, s] = await Promise.all([
      GetStyleQuadrant(market.value).catch(() => []),
      GetMarketSentiment(market.value).catch(() => null),
    ])
    quadrants.value = q || []
    sentiment.value = s
    await nextTick()
    renderQuadrant()
  } catch { /* empty */ }
}

function renderQuadrant() {
  if (typeof window === 'undefined' || !(window as any).echarts || quadrants.value.length === 0) return
  const el = document.getElementById('style-quadrant')
  if (!el) return
  const echarts = (window as any).echarts
  const chart = echarts.init(el)
  chart.setOption({
    tooltip: { formatter: (p: any) => `${p.data[2]}<br/>规模: ${p.data[0]}<br/>风格: ${p.data[1]}` },
    grid: { left: 50, right: 30, top: 30, bottom: 40 },
    xAxis: { name: '规模 (大→小)', min: -0.1, max: 1.1, axisLabel: { fontSize: 10 } },
    yAxis: { name: '风格 (价值→成长)', min: -0.1, max: 1.1, axisLabel: { fontSize: 10 } },
    series: [{
      type: 'scatter', symbolSize: 16,
      data: quadrants.value.map(q => [q.size, q.style, q.index]),
      label: { show: true, formatter: (p: any) => p.data[2], fontSize: 10, position: 'right' },
      itemStyle: { color: chartTheme.palette[0] },
    }],
  }, true)
}
</script>

<template>
  <div class="market-style-panel">
    <h4>市场风格</h4>
    <div class="style-grid">
      <div class="quadrant-panel">
        <h5>规模 × 风格</h5>
        <div id="style-quadrant" class="chart" />
      </div>
      <div class="sentiment-panel">
        <h5>情绪</h5>
        <div v-if="sentiment" class="sentiment-gauges">
          <div class="gauge"><span>涨停</span><span class="val up">{{ sentiment.limit_up }}</span></div>
          <div class="gauge"><span>跌停</span><span class="val down">{{ sentiment.limit_down }}</span></div>
          <div class="gauge"><span>成交额</span><span class="val">{{ ((sentiment.turnover||0)/1e8).toFixed(0) }}亿</span></div>
          <div class="gauge"><span>北向30日</span><span :class="'val '+(sentiment.northbound_cum>=0?'up':'down')">{{ sentiment.northbound_cum?.toFixed(1) }}亿</span></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.market-style-panel { padding: 16px; height: 100%; overflow-y: auto; }
.market-style-panel h4 { font-size: 13px; margin-bottom: 12px; }
.style-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.quadrant-panel h5, .sentiment-panel h5 { font-size: 11px; color: var(--color-text-secondary); margin-bottom: 8px; }
.chart { height: 320px; }
.sentiment-gauges { display: flex; flex-direction: column; gap: 8px; }
.gauge {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 14px; border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-subtle); font-size: 12px;
}
.val { font-weight: 700; font-family: 'JetBrains Mono', monospace; font-size: 16px; }
.val.up { color: var(--color-success); }
.val.down { color: var(--color-danger); }
</style>
