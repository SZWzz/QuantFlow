<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { GetStyleQuadrant, GetMarketSentiment, type StyleQuadrant, type MarketSentimentGauge } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader } from '@/terminal/components/panel'

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
  } catch { /* empty */ }
}

const quadrantOption = computed(() => ({
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
}))
</script>

<template>
  <div class="market-style-panel">
    <PanelHeader title="市场风格" />

    <div class="style-grid">
      <div class="quadrant-panel">
        <h5 class="section-title">规模 × 风格</h5>
        <VChart :option="quadrantOption" autoresize class="chart" />
      </div>
      <div class="sentiment-panel">
        <h5 class="section-title">情绪</h5>
        <!-- 自绘情绪仪表行：带边框的键值卡片，共享组件无等价物，保留但 token 化 -->
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
.market-style-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.style-grid {
  flex: 1; min-height: 0; overflow-y: auto;
  display: grid; grid-template-columns: 1fr 1fr;
  gap: var(--space-md); padding: var(--panel-padding);
}
.quadrant-panel h5, .sentiment-panel h5 { margin-bottom: var(--space-sm); }
.chart { height: 320px; }
.sentiment-gauges { display: flex; flex-direction: column; gap: var(--space-sm); }
.gauge {
  display: flex; justify-content: space-between; align-items: center;
  padding: var(--space-sm) var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: var(--color-bg-subtle); font-size: var(--font-xs);
}
.val { font-weight: 700; font-family: var(--font-mono); font-size: var(--font-lg); }
.val.up { color: var(--color-up); }
.val.down { color: var(--color-down); }
</style>
