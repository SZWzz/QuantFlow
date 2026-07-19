<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'
import { ComputeEventStudy, type EventStudyResult } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader, EmptyState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const chartTheme = useChartTheme()

const panelGroup = ctx.getOrCreatePanelGroup(props.panelId)
const groupId = computed(() => panelGroup.groupId)
const linkedSymbol = computed(() => ctx.linkGroups[groupId.value]?.activeSymbol)

const symbol = ref(props.params?.symbol || linkedSymbol.value || '')
const eventDate = ref(new Date().toISOString().slice(0, 10))
const windowDays = ref(10)

watch(linkedSymbol, (s) => { if (s) { symbol.value = s } })
const result = ref<EventStudyResult | null>(null)
const loading = ref(false)

async function runStudy() {
  loading.value = true
  try {
    result.value = await ComputeEventStudy(symbol.value, 'CN', '1d', eventDate.value, windowDays.value)
  } catch { result.value = null }
  finally { loading.value = false }
}

const chartOption = computed(() => {
  if (!result.value) return null
  const ar = result.value.daily_ar
  return {
    tooltip: { trigger: 'axis' },
    xAxis: { data: ar.map(d => `D${d.day}`), axisLabel: { fontSize: 9 } },
    yAxis: { axisLabel: { formatter: (v: number) => v + '%' } },
    series: [
      { name: 'AR', type: 'bar', data: ar.map(d => d.ar), itemStyle: { color: chartTheme.palette[0] } },
      { name: 'CAR', type: 'line', data: ar.map(d => d.car), lineStyle: { color: chartTheme.palette[3] }, symbol: 'circle' },
    ],
  }
})
</script>

<template>
  <div class="event-study-panel">
    <PanelHeader title="事件研究">
      <template #controls>
        <input v-model="symbol" placeholder="股票代码" class="sym-input" />
        <input v-model="eventDate" type="date" class="date-input" />
        <select v-model.number="windowDays" class="win-select">
          <option :value="5">±5</option><option :value="10">±10</option>
          <option :value="20">±20</option><option :value="30">±30</option>
        </select>
        <button class="btn btn-sm btn-primary" :disabled="loading" @click="runStudy">{{ loading ? '计算中...' : '计算' }}</button>
      </template>
    </PanelHeader>

    <div v-if="result" class="result">
      <div class="car-stats">
        CAR(±{{ windowDays }}): <span :class="result.car>=0?'up':'down'">{{ result.car }}%</span>
      </div>
      <VChart v-if="chartOption" :option="chartOption" autoresize class="chart" />
    </div>
    <EmptyState v-else title="输入股票代码和事件日期，点击计算" />
  </div>
</template>

<style scoped>
.event-study-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.sym-input, .date-input, .win-select {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
}
.sym-input { font-family: var(--font-mono); width: 110px; }
.result {
  flex: 1; min-height: 0; overflow-y: auto;
  display: flex; flex-direction: column; gap: var(--space-md);
  padding: var(--panel-padding);
}
.car-stats { font-size: var(--font-base); font-weight: 600; }
.car-stats .up { color: var(--color-up); }
.car-stats .down { color: var(--color-down); }
.chart { flex: 1; min-height: 250px; }
</style>
