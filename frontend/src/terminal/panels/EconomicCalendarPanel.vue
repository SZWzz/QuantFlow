<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface CalendarEvent {
  date: string
  time: string
  country: 'CN' | 'US'
  title: string
  previous: string
  forecast: string
  actual: string
  impact: 'high' | 'medium' | 'low'
}

const { fetchWithCache } = usePanelCache()
const filter = ref<string>('all')
const events = ref<CalendarEvent[]>([])
const loading = ref(false)
const loadError = ref('')

const filteredEvents = computed(() => {
  if (filter.value === 'all') return events.value
  return events.value.filter(e => e.country === filter.value)
})

const groupedEvents = computed(() => {
  const groups: Record<string, CalendarEvent[]> = {}
  for (const e of filteredEvents.value) {
    if (!groups[e.date]) groups[e.date] = []
    groups[e.date].push(e)
  }
  return groups
})

const dateKeys = computed(() => Object.keys(groupedEvents.value).sort())

function deriveImpact(title: string): 'high' | 'medium' | 'low' {
  const high = ['CPI', '非农', 'FOMC', '利率决议', 'GDP', 'NFP', 'FED', 'PMI', '核心PCE', 'PCE']
  for (const h of high) {
    if (title.includes(h)) return 'high'
  }
  const medium = ['PPI', '零售销售', '工业产出', '失业金', '初请', '消费者信心', '房屋', '耐用品']
  for (const m of medium) {
    if (title.includes(m)) return 'medium'
  }
  return 'low'
}

function buildMockEvents(): CalendarEvent[] {
  const now = new Date()
  const today = now.toISOString().slice(0, 10)
  const tomorrow = new Date(now.getTime() + 86400000).toISOString().slice(0, 10)
  const day3 = new Date(now.getTime() + 3 * 86400000).toISOString().slice(0, 10)

  return [
    { date: today, time: '20:30', country: 'US', title: '核心PCE物价指数 (4月)', previous: '2.8%', forecast: '2.8%', actual: '2.7%', impact: 'high' },
    { date: today, time: '22:00', country: 'US', title: '密歇根消费者信心指数 (5月)', previous: '69.1', forecast: '68.5', actual: '68.9', impact: 'medium' },
    { date: tomorrow, time: '09:30', country: 'CN', title: '中国官方制造业PMI (6月)', previous: '51.7', forecast: '51.5', actual: '', impact: 'high' },
    { date: tomorrow, time: '20:30', country: 'US', title: '初请失业金人数', previous: '23.5万', forecast: '23.8万', actual: '', impact: 'medium' },
    { date: day3, time: '20:30', country: 'US', title: '非农就业人数 (6月)', previous: '27.5万', forecast: '25.0万', actual: '', impact: 'high' },
    { date: day3, time: '20:30', country: 'US', title: '失业率 (6月)', previous: '3.9%', forecast: '4.0%', actual: '', impact: 'high' },
    { date: day3, time: '09:30', country: 'CN', title: '中国财新服务业PMI (6月)', previous: '52.5', forecast: '52.0', actual: '', impact: 'medium' },
  ]
}

const dataSource = ref<'api' | 'mock'>('mock')

async function fetchData() {
  loading.value = true
  loadError.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetEconomicIndicators) {
      const { data: result } = await fetchWithCache('economic_indicators', () => app.GetEconomicIndicators())
      if (result && typeof result === 'object') {
        const entries = Object.entries(result).slice(0, 30)
        events.value = entries.map(([key, val]: [string, any]) => ({
          date: val?.date || new Date().toISOString().slice(0, 10),
          time: val?.time || '--',
          country: key.startsWith('CN_') ? 'CN' as const : 'US' as const,
          title: val?.name || key,
          previous: val?.previous_value?.toFixed?.(1) ?? String(val?.previous_value ?? '--'),
          forecast: val?.forecast?.toFixed?.(1) ?? '--',
          actual: val?.last_value?.toFixed?.(1) ?? String(val?.last_value ?? '--'),
          impact: deriveImpact(val?.name || key),
        }))
        dataSource.value = 'api'
      }
    }
    // Fallback to mock
    if (events.value.length === 0) {
      events.value = buildMockEvents()
    }
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    events.value = buildMockEvents()
  } finally {
    loading.value = false
  }
}

function impactClass(impact: string): string {
  if (impact === 'high') return 'impact-high'
  if (impact === 'medium') return 'impact-medium'
  return 'impact-low'
}

function formatDateKey(key: string): string {
  const d = new Date(key)
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const weekdaysEn = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const day = weekdays[d.getDay()]
  return `${key} ${day}`
}

onMounted(fetchData)
</script>

<template>
  <div class="economic-calendar-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.economic_calendar') }}</h3>
      <div class="filter-tabs">
        <button :class="['f-tab', { active: filter === 'all' }]" @click="filter = 'all'">{{ $t('common.all') }}</button>
        <button :class="['f-tab', { active: filter === 'CN' }]" @click="filter = 'CN'">CN</button>
        <button :class="['f-tab', { active: filter === 'US' }]" @click="filter = 'US'">US</button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <SkeletonPanel v-if="loading && events.length === 0" type="table" :rows="5" />

    <div v-else-if="events.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>

    <div v-else class="calendar-scroll">
      <div v-for="dateKey in dateKeys" :key="dateKey" class="day-group">
        <div class="day-header">{{ formatDateKey(dateKey) }}</div>
        <div v-for="evt in groupedEvents[dateKey]" :key="evt.title + evt.time" class="event-row">
          <span class="event-time">{{ evt.time }}</span>
          <span class="event-country" :class="evt.country === 'CN' ? 'cn' : 'us'">{{ evt.country }}</span>
          <span class="event-impact" :class="impactClass(evt.impact)">
            {{ evt.impact === 'high' ? '🔴' : evt.impact === 'medium' ? '🟡' : '⚪' }}
          </span>
          <span class="event-title">{{ evt.title }}</span>
          <span class="event-prev" :title="$t('misc.previous')">{{ evt.previous }}</span>
          <span class="event-forecast" :title="$t('misc.forecast')">{{ evt.forecast }}</span>
          <span class="event-actual" :class="evt.actual && evt.actual !== '--' ? 'has-value' : ''" :title="$t('misc.actual')">
            {{ evt.actual || '--' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: rgba(239,68,68,0.1); color: #ef4444; font-size: 12px; }
.economic-calendar-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.filter-tabs { display: flex; gap: 4px; }
.f-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.f-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}

.calendar-scroll {
  flex: 1; overflow-y: auto;
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.day-group { margin-bottom: 12px; }
.day-header {
  font-size: 12px; font-weight: 600; padding: 6px 0; margin-bottom: 4px;
  border-bottom: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  position: sticky; top: 0; background: var(--color-bg-panel); z-index: 1;
}
.event-row {
  display: flex; align-items: center; gap: 8px;
  padding: 4px 0; font-size: 12px;
  border-bottom: 1px solid var(--color-border-subtle);
}
.event-row:hover { background: var(--color-bg-elevated); }
.event-time { width: 40px; color: var(--color-text-tertiary); font-variant-numeric: tabular-nums; }
.event-country { width: 24px; font-size: 10px; font-weight: 600; text-align: center; }
.event-country.cn { color: var(--color-up); }
.event-country.us { color: var(--color-accent); }
.event-impact { width: 20px; text-align: center; }
.event-title { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.event-prev, .event-forecast, .event-actual {
  width: 60px; text-align: right; font-variant-numeric: tabular-nums;
  color: var(--color-text-tertiary);
}
.event-actual.has-value { color: var(--color-text-primary); font-weight: 500; }
.impact-high { opacity: 1; }
.impact-medium { opacity: 0.7; }
.impact-low { opacity: 0.4; }
</style>
