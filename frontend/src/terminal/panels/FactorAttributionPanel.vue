<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { GetFactorAttribution, type FactorAttribution } from '@/lib/wails'

const totalReturn = ref(5.2)
const attr = ref<FactorAttribution | null>(null)

async function fetchData() {
  try {
    attr.value = await GetFactorAttribution(totalReturn.value)
    await nextTick()
    renderChart()
  } catch { attr.value = null }
}

function renderChart() {
  if (typeof window === 'undefined' || !(window as any).echarts || !attr.value) return
  const el = document.getElementById('attribution-chart')
  if (!el) return
  const echarts = (window as any).echarts
  const chart = echarts.init(el)
  const a = attr.value
  const data = [
    { name: '总收益', value: a.total_return, itemStyle: { color: '#3b82f6' } },
    { name: '市场β', value: -a.market_beta, itemStyle: { color: '#6b7280' } },
    ...Object.entries(a.style_factors).map(([k, v]) => ({ name: k, value: -v, itemStyle: {} })),
    ...Object.entries(a.industry_factors).map(([k, v]) => ({ name: k, value: -v, itemStyle: {} })),
    { name: 'α', value: a.alpha, itemStyle: { color: '#22c55e' } },
  ]
  chart.setOption({
    tooltip: { trigger: 'axis', formatter: (p:any) => `${p[0].name}: ${p[0].value.toFixed(2)}%` },
    xAxis: { type: 'category', data: data.map(d => d.name), axisLabel: { fontSize: 9 } },
    yAxis: { type: 'value', axisLabel: { formatter: (v:number) => v + '%' } },
    series: [{
      type: 'waterfall', data: data.map((d, i) => {
        const isTotal = i === 0
        return { name: d.name, value: d.value, itemStyle: d.itemStyle }
      }),
      stack: 'a',
    }],
  }, true)
}
</script>

<template>
  <div class="attribution-panel">
    <div class="toolbar">
      <h4>因子归因</h4>
      <div class="return-input">
        <span>组合收益</span>
        <input v-model.number="totalReturn" type="number" step="0.1" class="num-input" />
        <span>%</span>
        <button @click="fetchData" class="btn">计算</button>
      </div>
    </div>
    <div v-if="attr" class="result">
      <div class="stats-row">
        <div class="stat"><span>总收益</span><span class="val">{{ attr.total_return.toFixed(2) }}%</span></div>
        <div class="stat"><span>市场β</span><span class="val">{{ attr.market_beta.toFixed(2) }}%</span></div>
        <div class="stat high"><span>Alpha</span><span class="val">{{ attr.alpha.toFixed(2) }}%</span></div>
      </div>
      <div id="attribution-chart" class="chart" />
    </div>
    <div v-else class="empty">输入组合收益，点击计算</div>
  </div>
</template>

<style scoped>
.attribution-panel { padding: 16px; height: 100%; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.toolbar { display: flex; justify-content: space-between; align-items: center; }
.toolbar h4 { font-size: 13px; margin: 0; }
.return-input { display: flex; align-items: center; gap: 6px; font-size: 12px; }
.num-input { width: 60px; padding: 4px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 12px; font-family: 'JetBrains Mono', monospace; }
.btn { padding: 4px 12px; background: var(--color-accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.stats-row { display: flex; gap: 12px; margin-bottom: 8px; }
.stat { flex: 1; padding: 8px 12px; border: 1px solid var(--color-border); border-radius: var(--radius-sm); display: flex; flex-direction: column; gap: 4px; font-size: 11px; color: var(--color-text-secondary); }
.stat.high { border-color: var(--color-success); background: var(--color-success-soft); }
.val { font-size: 18px; font-weight: 700; font-family: 'JetBrains Mono', monospace; color: var(--color-text-primary); }
.chart { flex: 1; min-height: 280px; }
.empty { text-align: center; padding: 48px; color: var(--color-text-tertiary); }
</style>
