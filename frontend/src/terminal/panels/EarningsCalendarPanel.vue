<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface EarningsEvent {
  symbol: string
  date: string
  hour: string
  quarter: number
  year: number
  eps_actual: number
  eps_estimate: number
  revenue_actual: number
  revenue_estimate: number
}

const events = ref<EarningsEvent[]>([])
const loading = ref(false)
const dateRange = ref<'week' | 'month' | 'quarter'>('week')

function getDateRange(): { from: string; to: string } {
  const now = new Date()
  const from = now.toISOString().slice(0, 10)
  const to = new Date(now.getTime() + (dateRange.value === 'week' ? 7 : dateRange.value === 'month' ? 30 : 90) * 86400000).toISOString().slice(0, 10)
  return { from, to }
}

const groupedEvents = computed(() => {
  const groups: Record<string, EarningsEvent[]> = {}
  for (const e of events.value) {
    if (!groups[e.date]) groups[e.date] = []
    groups[e.date].push(e)
  }
  return groups
})

const dateKeys = computed(() => Object.keys(groupedEvents.value).sort())

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetEarningsCalendar) return
  loading.value = true
  try {
    const { from, to } = getDateRange()
    const result = await app.GetEarningsCalendar(from, to)
    events.value = (result || []).map((r: any) => ({
      symbol: r.symbol || '',
      date: r.date || '',
      hour: r.hour || '',
      quarter: r.quarter || 0,
      year: r.year || 0,
      eps_actual: r.eps_actual || 0,
      eps_estimate: r.eps_estimate || 0,
      revenue_actual: r.revenue_actual || 0,
      revenue_estimate: r.revenue_estimate || 0,
    }))
  } catch (e) {
    console.error('[EarningsCalendar]', e)
    events.value = []
  } finally {
    loading.value = false
  }
}

function formatMoney(v: number): string {
  if (v >= 1e12) return '$' + (v / 1e12).toFixed(2) + 'T'
  if (v >= 1e9) return '$' + (v / 1e9).toFixed(1) + 'B'
  if (v >= 1e6) return '$' + (v / 1e6).toFixed(0) + 'M'
  return '$' + (v / 1e3).toFixed(0) + 'K'
}

function hourLabel(h: string): string {
  if (h === 'bmo') return '盘前'
  if (h === 'amc') return '盘后'
  if (h === 'dmh') return '盘中'
  return h
}

function epsSurprise(e: EarningsEvent): number | null {
  if (!e.eps_estimate || !e.eps_actual) return null
  return ((e.eps_actual - e.eps_estimate) / Math.abs(e.eps_estimate)) * 100
}

onMounted(fetchData)
</script>

<template>
  <div class="earnings-calendar-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.earnings_calendar') }}</h3>
      <div class="range-tabs">
        <button :class="['r-tab', { active: dateRange === 'week' }]" @click="dateRange = 'week'; fetchData()">{{ $t('misc.this_week') }}</button>
        <button :class="['r-tab', { active: dateRange === 'month' }]" @click="dateRange = 'month'; fetchData()">{{ $t('misc.this_month') }}</button>
        <button :class="['r-tab', { active: dateRange === 'quarter' }]" @click="dateRange = 'quarter'; fetchData()">{{ $t('misc.this_quarter') }}</button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && events.length === 0" type="table" :rows="5" />

    <div v-else-if="events.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>

    <div v-else class="calendar-scroll">
      <div v-for="dateKey in dateKeys" :key="dateKey" class="day-group">
        <div class="day-header">{{ dateKey }}</div>
        <div v-for="evt in groupedEvents[dateKey]" :key="evt.symbol + evt.date" class="event-row">
          <span class="evt-hour">{{ hourLabel(evt.hour) }}</span>
          <span class="evt-symbol">{{ evt.symbol }}</span>
          <span class="evt-quarter">Q{{ evt.quarter }}</span>
          <span class="evt-est">{{ evt.eps_estimate ? '$' + evt.eps_estimate.toFixed(2) : '--' }}</span>
          <span class="evt-act" :class="evt.eps_actual > 0 ? 'has-value' : ''">{{ evt.eps_actual ? '$' + evt.eps_actual.toFixed(2) : '--' }}</span>
          <span v-if="epsSurprise(evt) !== null" class="evt-surprise" :class="epsSurprise(evt)! >= 0 ? 'up' : 'down'">
            {{ epsSurprise(evt)!.toFixed(1) }}%
          </span>
          <span v-else class="evt-surprise">--</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.earnings-calendar-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, #e5e7eb); background: var(--color-bg-panel, #1a1a2e); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.range-tabs { display: flex; gap: 4px; }
.r-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.r-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.refresh-btn { margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }

.calendar-scroll { flex: 1; overflow-y: auto; }
.day-group { margin-bottom: 12px; }
.day-header { font-size: 12px; font-weight: 600; padding: 6px 0; border-bottom: 1px solid var(--color-border-strong); color: var(--color-text-primary); position: sticky; top: 0; background: var(--color-bg-panel); z-index: 1; }
.event-row { display: flex; align-items: center; gap: 8px; padding: 4px 0; font-size: 12px; border-bottom: 1px solid var(--color-border-subtle); }
.event-row:hover { background: var(--color-bg-elevated); }
.evt-hour { width: 32px; font-size: 10px; color: var(--color-text-tertiary); }
.evt-symbol { width: 64px; font-weight: 600; cursor: pointer; color: #60a5fa; }
.evt-quarter { width: 24px; font-size: 10px; color: var(--color-text-tertiary); }
.evt-est, .evt-act { width: 60px; text-align: right; font-variant-numeric: tabular-nums; color: var(--color-text-tertiary); }
.evt-act.has-value { color: var(--color-text-primary); font-weight: 500; }
.evt-surprise { width: 56px; text-align: right; font-weight: 500; font-size: 11px; }
.up { color: #16a34a; }
.down { color: #dc2626; }
</style>
