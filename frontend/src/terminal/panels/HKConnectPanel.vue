<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, shallowRef } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface MinuteFlow {
  time: string
  sh_net: number
  sz_net: number
  total: number
}

interface DailyHistory {
  date: string
  sh_net: number
  sz_net: number
  total_net: number
  sh_cum: number
  sz_cum: number
}

const { fetchWithCache } = usePanelCache()
const chartTheme = useChartTheme()
const activeTab = ref<'northbound' | 'quota'>('northbound')
const minuteFlow = ref<MinuteFlow[]>([])
const history = ref<DailyHistory[]>([])
const loading = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const todayTotal = computed(() => {
  if (minuteFlow.value.length === 0) return 0
  const last = minuteFlow.value[minuteFlow.value.length - 1]
  return last.total
})

const shToday = computed(() => {
  if (minuteFlow.value.length === 0) return 0
  return minuteFlow.value[minuteFlow.value.length - 1].sh_net
})

const szToday = computed(() => {
  if (minuteFlow.value.length === 0) return 0
  return minuteFlow.value[minuteFlow.value.length - 1].sz_net
})

const cumulativeTotal = computed(() => {
  if (history.value.length === 0) return 0
  const last = history.value[history.value.length - 1]
  return (last.sh_cum || 0) + (last.sz_cum || 0)
})

const chartData = computed(() => ({
  categories: minuteFlow.value.map(f => f.time),
  series: [
    { name: '沪股通', data: minuteFlow.value.map(f => f.sh_net), color: '#ef4444' },
    { name: '深股通', data: minuteFlow.value.map(f => f.sz_net), color: '#22c55e' },
    { name: '合计', data: minuteFlow.value.map(f => f.total), color: '#3b82f6' },
  ],
}))

let chartInstance: any = null

function renderChart() {
  if (typeof window === 'undefined' || !(window as any).echarts) return
  const echarts = (window as any).echarts
  const el = document.getElementById('hk-flow-chart')
  if (!el) return
  if (!chartInstance) chartInstance = echarts.init(el)
  const option = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['沪股通', '深股通', '合计'], bottom: 0, textStyle: { color: chartTheme.axisColor, fontSize: 10 } },
    grid: { left: 50, right: 16, top: 8, bottom: 32 },
    xAxis: { type: 'category', data: chartData.value.categories, axisLabel: { color: chartTheme.axisColor, fontSize: 10 } },
    yAxis: { type: 'value', splitLine: { lineStyle: { color: chartTheme.gridColor } }, axisLabel: { color: chartTheme.axisColor, fontSize: 10, formatter: (v: number) => v + '亿' } },
    series: chartData.value.series.map(s => ({
      name: s.name, type: 'line', smooth: true, data: s.data,
      lineStyle: { width: 1.5 },
      areaStyle: { opacity: 0.1 },
      symbol: 'none',
    })),
  }
  chartInstance.setOption(option, true)
}

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetNorthboundFlow) return
  loading.value = true
  try {
    const { data: result } = await fetchWithCache<any>('hk_northbound_flow', () => app.GetNorthboundFlow())
    if (result?.minute_flow) {
      minuteFlow.value = (result.minute_flow as any[]).map((f: any) => ({
        time: f.time || f.timestamp || '',
        sh_net: f.sh_net || 0,
        sz_net: f.sz_net || 0,
        total: (f.sh_net || 0) + (f.sz_net || 0),
      }))
    }
    if (result?.history) {
      history.value = (result.history as any[]).map((h: any) => ({
        date: h.date || '',
        sh_net: h.sh_net || 0,
        sz_net: h.sz_net || 0,
        total_net: h.total_net || (h.sh_net || 0) + (h.sz_net || 0),
        sh_cum: h.sh_cum || 0,
        sz_cum: h.sz_cum || 0,
      }))
    }
  } catch (e) {
    logger.error('[HKConnect]', e)
  } finally {
    loading.value = false
  }
}

function formatAmount(v: number): string {
  const abs = Math.abs(v)
  if (abs >= 1e8) return (v / 1e8).toFixed(1) + '亿'
  if (abs >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 60000)
  setTimeout(renderChart, 500)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (chartInstance) { chartInstance.dispose(); chartInstance = null }
})
</script>

<template>
  <div class="hk-connect-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.hk_connect') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'northbound' }]" @click="activeTab = 'northbound'">{{ $t('misc.northbound') }}</button>
        <button :class="['tab', { active: activeTab === 'quota' }]" @click="activeTab = 'quota'">{{ $t('misc.quota') }}</button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && minuteFlow.length === 0" type="card" :rows="4" />

    <template v-else-if="activeTab === 'northbound'">
      <div v-if="minuteFlow.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
      <template v-else>
        <div class="stats-row">
          <div class="stat-card">
            <div class="stat-label">{{ $t('misc.sh_connect') }}</div>
            <div class="stat-value" :class="shToday >= 0 ? 'up' : 'down'">{{ formatAmount(shToday) }}</div>
          </div>
          <div class="stat-card">
            <div class="stat-label">{{ $t('misc.sz_connect') }}</div>
            <div class="stat-value" :class="szToday >= 0 ? 'up' : 'down'">{{ formatAmount(szToday) }}</div>
          </div>
          <div class="stat-card">
            <div class="stat-label">{{ $t('misc.total') }}</div>
            <div class="stat-value" :class="todayTotal >= 0 ? 'up' : 'down'">{{ formatAmount(todayTotal) }}</div>
          </div>
          <div class="stat-card">
            <div class="stat-label">{{ $t('misc.cumulative') }}</div>
            <div class="stat-value" :class="cumulativeTotal >= 0 ? 'up' : 'down'">{{ formatAmount(cumulativeTotal) }}</div>
          </div>
        </div>

        <div id="hk-flow-chart" class="flow-chart"></div>

        <div class="table-wrapper">
          <div class="table-header">
            <span class="col-date">{{ $t('common.date') }}</span>
            <span class="col-sh">{{ $t('misc.sh_connect') }}</span>
            <span class="col-sz">{{ $t('misc.sz_connect') }}</span>
            <span class="col-total">{{ $t('common.total') }}</span>
            <span class="col-cum">{{ $t('misc.cumulative') }}</span>
          </div>
          <div class="table-body">
            <div v-for="h in history" :key="h.date" class="table-row">
              <span class="col-date">{{ h.date }}</span>
              <span class="col-sh" :class="h.sh_net >= 0 ? 'up' : 'down'">{{ formatAmount(h.sh_net) }}</span>
              <span class="col-sz" :class="h.sz_net >= 0 ? 'up' : 'down'">{{ formatAmount(h.sz_net) }}</span>
              <span class="col-total" :class="h.total_net >= 0 ? 'up' : 'down'">{{ formatAmount(h.total_net) }}</span>
              <span class="col-cum" :class="(h.sh_cum + h.sz_cum) >= 0 ? 'up' : 'down'">{{ formatAmount(h.sh_cum + h.sz_cum) }}</span>
            </div>
          </div>
        </div>
      </template>
    </template>

    <template v-else>
      <div class="quota-section">
        <div class="quota-card">
          <div class="quota-label">{{ $t('misc.sh_connect') }}</div>
          <div class="quota-bar-track">
            <div class="quota-bar-fill sh-bar" style="width:66%"></div>
          </div>
          <div class="quota-detail">
            <span>345亿 / 520亿</span>
            <span>66%</span>
          </div>
        </div>
        <div class="quota-card">
          <div class="quota-label">{{ $t('misc.sz_connect') }}</div>
          <div class="quota-bar-track">
            <div class="quota-bar-fill sz-bar" style="width:79%"></div>
          </div>
          <div class="quota-detail">
            <span>412亿 / 520亿</span>
            <span>79%</span>
          </div>
        </div>
      </div>
      <div class="empty-state" style="flex:0;padding-top:8px">{{ $t('misc.quota_note') }}</div>
    </template>
  </div>
</template>

<style scoped>
.hk-connect-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.stats-row {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 12px;
}
.stat-card {
  padding: 10px; border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); text-align: center;
}
.stat-label { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.stat-value { font-size: 16px; font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

.flow-chart { height: 160px; margin-bottom: 12px; flex-shrink: 0; }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-date { width: 80px; }
.col-sh, .col-sz, .col-total, .col-cum { width: 80px; text-align: right; font-weight: 500; }

.quota-section {
  display: flex; flex-direction: column; gap: 16px; padding: 16px 0;
}
.quota-card { display: flex; flex-direction: column; gap: 6px; }
.quota-label { font-size: 13px; font-weight: 600; }
.quota-bar-track { height: 12px; background: var(--color-bg-elevated); border-radius: var(--radius-md); overflow: hidden; }
.quota-bar-fill { height: 100%; border-radius: var(--radius-md); }
.sh-bar { background: var(--color-up); }
.sz-bar { background: var(--color-down); }
.quota-detail { display: flex; justify-content: space-between; font-size: 11px; color: var(--color-text-secondary); }
</style>
