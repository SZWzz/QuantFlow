<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, PanelTable, EmptyState, LoadingState, type Column } from '@/terminal/components/panel'
import KlineChart from '@/terminal/components/panel/KlineChart.vue'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { logger } from '@/lib/logger'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()

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

const headerTabs = computed(() => [
  { key: 'northbound', label: t('misc.northbound') },
  { key: 'quota', label: t('misc.quota') },
])

function onTabChange(key: string) {
  if (key !== 'northbound' && key !== 'quota') return
  activeTab.value = key
}

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

/** 历史表格行：累计列预映射（sh_cum + sz_cum），供 PanelTable colorize 直接取数 */
const historyRows = computed(() =>
  history.value.map(h => ({ ...h, cum: (h.sh_cum || 0) + (h.sz_cum || 0) })),
)

// ── 北向资金分时图（computed option，主题切换自动重绘） ──
const flowOption = computed<ECBasicOption>(() => ({
  animation: false,
  backgroundColor: 'transparent',
  tooltip: { trigger: 'axis' },
  legend: {
    data: ['沪股通', '深股通', '合计'],
    bottom: 0,
    textStyle: { color: chartTheme.axisColor, fontSize: 10 },
  },
  grid: { left: 50, right: 16, top: 8, bottom: 32 },
  xAxis: {
    type: 'category',
    data: minuteFlow.value.map(f => f.time),
    axisLabel: { color: chartTheme.axisColor, fontSize: 10 },
    axisLine: { lineStyle: { color: chartTheme.splitColor } },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: chartTheme.gridColor } },
    axisLabel: { color: chartTheme.axisColor, fontSize: 10, formatter: (v: number) => v + '亿' },
  },
  series: [
    { name: '沪股通', data: minuteFlow.value.map(f => f.sh_net), color: chartTheme.upColor },
    { name: '深股通', data: minuteFlow.value.map(f => f.sz_net), color: chartTheme.downColor },
    { name: '合计', data: minuteFlow.value.map(f => f.total), color: chartTheme.palette[0] },
  ].map(s => ({
    name: s.name,
    type: 'line',
    smooth: true,
    data: s.data,
    lineStyle: { width: 1.5, color: s.color },
    itemStyle: { color: s.color },
    areaStyle: { opacity: 0.1, color: s.color },
    symbol: 'none',
  })),
}))

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

const cols = computed<Column[]>(() => [
  { key: 'date', label: t('common.date') },
  { key: 'sh_net', label: t('misc.sh_connect'), align: 'right', formatter: formatAmount, colorize: true },
  { key: 'sz_net', label: t('misc.sz_connect'), align: 'right', formatter: formatAmount, colorize: true },
  { key: 'total_net', label: t('common.total'), align: 'right', formatter: formatAmount, colorize: true },
  { key: 'cum', label: t('misc.cumulative'), align: 'right', formatter: formatAmount, colorize: true },
])

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="hk-connect-panel">
    <PanelHeader
      :title="$t('misc.hk_connect')"
      :tabs="headerTabs"
      :active-tab="activeTab"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
      @tab-change="onTabChange"
    />

    <LoadingState v-if="loading && minuteFlow.length === 0" type="card" :rows="4" />

    <template v-else-if="activeTab === 'northbound'">
      <EmptyState v-if="minuteFlow.length === 0" :title="$t('common.no_data')" />
      <template v-else>
        <!-- 自绘统计卡：StatItem 不支持值涨跌着色，保留但 token 化 -->
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

        <div class="flow-chart">
          <KlineChart :option="flowOption" symbol="northbound" :loading="loading" />
        </div>

        <PanelTable :columns="cols" :data="historyRows" :loading="loading" sticky-header />
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
      <div class="quota-note">{{ $t('misc.quota_note') }}</div>
    </template>
  </div>
</template>

<style scoped>
.hk-connect-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.stats-row {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-sm);
  padding: var(--space-sm) var(--panel-padding);
  flex-shrink: 0;
}
.stat-card {
  padding: var(--space-sm); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); text-align: center;
}
.stat-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.stat-value { font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

.flow-chart { height: 160px; margin: 0 var(--panel-padding) var(--space-sm); flex-shrink: 0; }

.quota-section {
  display: flex; flex-direction: column; gap: var(--space-lg); padding: var(--space-lg) var(--panel-padding);
}
.quota-card { display: flex; flex-direction: column; gap: var(--space-xs); }
.quota-label { font-size: var(--font-sm); font-weight: 600; }
.quota-bar-track { height: 12px; background: var(--color-bg-elevated); border-radius: var(--radius-md); overflow: hidden; }
.quota-bar-fill { height: 100%; border-radius: var(--radius-md); }
.sh-bar { background: var(--color-up); }
.sz-bar { background: var(--color-down); }
.quota-detail { display: flex; justify-content: space-between; font-size: var(--font-xs); color: var(--color-text-secondary); }
.quota-note {
  padding: 0 var(--panel-padding);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}
</style>
