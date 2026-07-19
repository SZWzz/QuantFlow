<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
ctx.getOrCreatePanelGroup(props.panelId)

const { t } = useI18n()

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

const { fetchWithCache } = usePanelCache()
const events = ref<EarningsEvent[]>([])
const loading = ref(false)
const loadError = ref('')
const dateRange = ref<'week' | 'month' | 'quarter'>('week')

const tabs = computed(() => [
  { key: 'week', label: t('misc.this_week') },
  { key: 'month', label: t('misc.this_month') },
  { key: 'quarter', label: t('misc.this_quarter') },
])

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
  loadError.value = ''
  try {
    const { from, to } = getDateRange()
    const { data: result } = await fetchWithCache<any>(`earnings_calendar:${from}:${to}`, () => app.GetEarningsCalendar(from, to))
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
  } catch (e: any) {
    console.error('[EarningsCalendar]', e)
    loadError.value = e?.message || String(e)
    events.value = []
  } finally {
    loading.value = false
  }
}

function onTabChange(key: string) {
  dateRange.value = key as 'week' | 'month' | 'quarter'
  fetchData()
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
    <PanelHeader
      :title="$t('misc.earnings_calendar')"
      :tabs="tabs"
      :active-tab="dateRange"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
      @tab-change="onTabChange"
    />

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && events.length === 0" type="table" :rows="5" />
    <EmptyState v-else-if="events.length === 0" :title="$t('common.no_data')" />

    <!-- 按日期分组的自定义列表：粘性日期头 + 多段行，PanelTable 无法表达，保留自绘 -->
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
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.calendar-scroll { flex: 1; overflow-y: auto; padding: 0 var(--panel-padding); }
.day-group { margin-bottom: var(--space-md); }
.day-header {
  font-size: var(--font-xs);
  font-weight: 600;
  padding: var(--space-xs) 0;
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  position: sticky;
  top: 0;
  background: var(--color-bg-panel);
  z-index: 1;
}
.event-row {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs) 0;
  font-size: var(--font-xs);
  border-bottom: 1px solid var(--color-border-subtle);
}
.event-row:hover { background: var(--color-bg-elevated); }
.evt-hour { width: 32px; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.evt-symbol { width: 64px; font-weight: 600; color: var(--color-accent); }
.evt-quarter { width: 24px; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.evt-est, .evt-act {
  width: 60px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-tertiary);
}
.evt-act.has-value { color: var(--color-text-primary); font-weight: 500; }
.evt-surprise { width: 56px; text-align: right; font-weight: 500; font-size: var(--font-xs); }
.up { color: var(--color-down); }
.down { color: var(--color-up); }
</style>
