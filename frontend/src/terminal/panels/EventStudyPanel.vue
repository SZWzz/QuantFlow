<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { ComputeEventStudy, type EventStudyResult } from '@/lib/wails'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useSymbolContext } from '@/stores/symbolContext'

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
    await nextTick()
    renderChart()
  } catch { result.value = null }
  finally { loading.value = false }
}

function renderChart() {
  if (typeof window === 'undefined' || !(window as any).echarts || !result.value) return
  const el = document.getElementById('event-chart')
  if (!el) return
  const echarts = (window as any).echarts
  const chart = echarts.init(el)
  const ar = result.value.daily_ar
  chart.setOption({
    tooltip: { trigger: 'axis' },
    xAxis: { data: ar.map(d => `D${d.day}`), axisLabel: { fontSize: 9 } },
    yAxis: { axisLabel: { formatter: (v:number) => v + '%' } },
    series: [
      { name: 'AR', type: 'bar', data: ar.map(d => d.ar), itemStyle: { color: chartTheme.palette[0] } },
      { name: 'CAR', type: 'line', data: ar.map(d => d.car), lineStyle: { color: chartTheme.palette[3] }, symbol: 'circle' },
    ],
  }, true)
}
</script>

<template>
  <div class="event-study-panel">
    <div class="toolbar">
      <input v-model="symbol" placeholder="股票代码" class="sym-input" />
      <input v-model="eventDate" type="date" class="date-input" />
      <select v-model.number="windowDays" class="win-select">
        <option :value="5">±5</option><option :value="10">±10</option>
        <option :value="20">±20</option><option :value="30">±30</option>
      </select>
      <button @click="runStudy" :disabled="loading" class="btn">{{ loading ? '计算中...' : '计算' }}</button>
    </div>

    <div v-if="result" class="result">
      <div class="car-stats">
        CAR(±{{ windowDays }}): <span :class="result.car>=0?'up':'down'">{{ result.car }}%</span>
      </div>
      <div id="event-chart" class="chart" />
    </div>
    <div v-else class="empty">输入股票代码和事件日期，点击计算</div>
  </div>
</template>

<style scoped>
.event-study-panel { padding: 16px; height: 100%; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.toolbar { display: flex; gap: 8px; align-items: center; }
.sym-input { padding: 6px 10px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 12px; font-family: 'JetBrains Mono', monospace; width: 120px; }
.date-input { padding: 6px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 12px; }
.win-select { padding: 6px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 12px; }
.btn { padding: 6px 16px; background: var(--color-accent); color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 12px; font-weight: 600; }
.btn:disabled { opacity: 0.5; }
.car-stats { font-size: 14px; font-weight: 600; }
.car-stats .up { color: var(--color-success); }
.car-stats .down { color: var(--color-danger); }
.chart { flex: 1; min-height: 250px; }
.empty { text-align: center; padding: 64px; color: var(--color-text-tertiary); }
</style>
