<script setup lang="ts">
import { ref, computed } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { GetFactorAttribution, type FactorAttribution } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, EmptyState } from '@/terminal/components/panel'

const chartTheme = useChartTheme()

const totalReturn = ref(5.2)
const attr = ref<FactorAttribution | null>(null)

async function fetchData() {
  try {
    attr.value = await GetFactorAttribution(totalReturn.value)
  } catch { attr.value = null }
}

const chartOption = computed(() => {
  if (!attr.value) return null
  const a = attr.value
  const data = [
    { name: '总收益', value: a.total_return, itemStyle: { color: chartTheme.palette[0] } },
    { name: '市场β', value: -a.market_beta, itemStyle: { color: chartTheme.axisColor } },
    ...Object.entries(a.style_factors).map(([k, v]) => ({ name: k, value: -v, itemStyle: {} })),
    ...Object.entries(a.industry_factors).map(([k, v]) => ({ name: k, value: -v, itemStyle: {} })),
    { name: 'α', value: a.alpha, itemStyle: { color: chartTheme.palette[1] } },
  ]
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartTheme.tooltipBg,
      textStyle: { color: chartTheme.tooltipText },
      formatter: (p: any) => `${p[0].name}: ${p[0].value.toFixed(2)}%`,
    },
    xAxis: {
      type: 'category',
      data: data.map(d => d.name),
      axisLabel: { color: chartTheme.axisColor, fontSize: 9 },
      axisLine: { lineStyle: { color: chartTheme.splitColor } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: chartTheme.axisColor, formatter: (v: number) => v + '%' },
      splitLine: { lineStyle: { color: chartTheme.gridColor } },
    },
    series: [{
      // TODO: 'waterfall' 非 ECharts 注册类型，渲染异常为迁移前既有问题，待后续修复
      type: 'waterfall', data: data.map((d, i) => {
        const isTotal = i === 0
        return { name: d.name, value: d.value, itemStyle: d.itemStyle }
      }),
      stack: 'a',
    }],
  }
})
</script>

<template>
  <div class="attribution-panel">
    <PanelHeader title="因子归因">
      <template #controls>
        <div class="return-input">
          <span>组合收益</span>
          <input v-model.number="totalReturn" type="number" step="0.1" class="num-input" />
          <span>%</span>
          <button class="btn btn-sm btn-primary" @click="fetchData">计算</button>
        </div>
      </template>
    </PanelHeader>

    <div v-if="attr" class="result">
      <!-- 自绘统计卡片：StatItem 不支持 Alpha 高亮边框卡片样式，保留但 token 化 -->
      <div class="stats-row">
        <div class="stat"><span>总收益</span><span class="val">{{ attr.total_return.toFixed(2) }}%</span></div>
        <div class="stat"><span>市场β</span><span class="val">{{ attr.market_beta.toFixed(2) }}%</span></div>
        <div class="stat high"><span>Alpha</span><span class="val">{{ attr.alpha.toFixed(2) }}%</span></div>
      </div>
      <VChart v-if="chartOption" :option="chartOption" autoresize class="chart" />
    </div>
    <EmptyState v-else title="输入组合收益，点击计算" />
  </div>
</template>

<style scoped>
.attribution-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.return-input { display: flex; align-items: center; gap: var(--space-sm); font-size: var(--font-xs); }
.num-input {
  width: 60px; padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary);
  font-size: var(--font-xs); font-family: var(--font-mono);
}
.result {
  flex: 1; min-height: 0; overflow-y: auto;
  display: flex; flex-direction: column; gap: var(--space-md);
  padding: var(--panel-padding);
}
.stats-row { display: flex; gap: var(--space-md); }
.stat {
  flex: 1; padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  display: flex; flex-direction: column; gap: var(--space-xs);
  font-size: var(--font-xs); color: var(--color-text-secondary);
}
.stat.high { border-color: var(--color-success); background: var(--color-success-soft); }
.val { font-size: var(--font-lg); font-weight: 700; font-family: var(--font-mono); color: var(--color-text-primary); }
.chart { flex: 1; min-height: 280px; }
</style>
