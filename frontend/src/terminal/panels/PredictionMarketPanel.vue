<!-- frontend/src/terminal/panels/PredictionMarketPanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import VChart from 'vue-echarts'
import 'echarts'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Outcome {
  id: string
  label: string
  price: number
  change_24h: number
}

interface Event {
  id: string
  title: string
  category: string
  volume: number
  liquidity: number
  end_date: string
  status: string
  outcomes: Outcome[]
  description: string
}

interface PricePoint {
  timestamp: number
  price: number
}

const categories = ['all', 'economics', 'crypto', 'politics', 'sports', 'tech', 'entertainment'] as const
const activeCategory = ref('all')
const events = ref<Event[]>([])
const loading = ref(true)
const selectedEvent = ref<Event | null>(null)
const priceHistory = ref<PricePoint[]>([])
const signalInfo = ref<{ action: string; confidence: number; description: string } | null>(null)
const { fetchWithCache } = usePanelCache()

const categoryLabels: Record<string, string> = {
  all: t('common.all'), economics: '经济', crypto: '加密', politics: '政治',
  sports: '体育', tech: '科技', entertainment: '娱乐'
}

async function loadEvents() {
  loading.value = true
  const cat = activeCategory.value === 'all' ? '' : activeCategory.value
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetPredictionMarkets) {
      const { data: result } = await fetchWithCache<any>(`prediction_markets:${activeCategory.value}`, () => app.GetPredictionMarkets(cat, 30), 15 * 60 * 1000)
      events.value = result?.events || []
    }
  } catch(e) {
    console.error('[PredictionMarket] loadEvents:', e)
  }
  loading.value = false
}

async function loadDetail(event: Event) {
  selectedEvent.value = event
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetPredictionEventDetail) {
      const { data: result } = await fetchWithCache<any>(`prediction_detail:${event.id}`, () => app.GetPredictionEventDetail(event.id), 15 * 60 * 1000)
      if (result?.prices?.length > 0) {
        priceHistory.value = result.prices
        return
      }
    }
  } catch(e) {
    console.error('[PredictionMarket] loadDetail:', e)
  }
}

async function loadSignals() {
  try {
    const app = (window as any).go?.main?.App
    if (app?.GetPredictionSignals) {
      const { data: result } = await fetchWithCache<any>('prediction_signals', () => app.GetPredictionSignals('', 0.05), 15 * 60 * 1000)
      signalInfo.value = result?.signal || null
    }
  } catch(e) { console.error('[PredictionMarket] loadSignals:', e) }
}

onMounted(() => {
  loadEvents()
  loadSignals()
})

// Chart option for selected event's probability
const chartOption = computed(() => {
  if (!selectedEvent.value || priceHistory.value.length === 0) return {}
  const dates = priceHistory.value.map(p => new Date(p.timestamp).toLocaleDateString('zh-CN'))
  const prices = priceHistory.value.map(p => +(p.price * 100).toFixed(1))
  return {
    tooltip: {
      trigger: 'axis' as const,
      formatter: (params: any) => `${params[0].axisValue}<br/>概率: ${params[0].value}%`
    },
    grid: { left: 20, right: 20, top: 10, bottom: 20 },
    xAxis: { type: 'category' as const, data: dates, show: false },
    yAxis: {
      type: 'value' as const, min: 0, max: 100,
      axisLabel: { formatter: '{value}%', fontSize: 10 }
    },
    series: [{
      type: 'line', data: prices, smooth: true,
      areaStyle: { color: 'rgba(59, 130, 246, 0.1)' },
      lineStyle: { color: '#3b82f6', width: 2 },
      itemStyle: { color: '#3b82f6' },
      showSymbol: false
    }]
  }
})

const sortedEvents = computed(() => {
  return [...events.value].sort((a, b) => b.volume - a.volume)
})

function formatVolume(v: number): string {
  if (v >= 1_000_000) return '$' + (v / 1_000_000).toFixed(1) + 'M'
  if (v >= 1_000) return '$' + (v / 1_000).toFixed(0) + 'K'
  return '$' + v.toFixed(0)
}

function formatChange(c: number): string {
  const pct = (c * 100).toFixed(1)
  return c >= 0 ? `+${pct}%` : `${pct}%`
}

function formatEndDate(d: string): string {
  if (!d) return ''
  const date = new Date(d)
  const now = new Date()
  const days = Math.ceil((date.getTime() - now.getTime()) / 86400000)
  if (days < 0) return '已到期'
  if (days === 0) return '今日到期'
  return `${days}天后`
}

function changeClass(c: number): string {
  return c >= 0 ? 'text-green' : 'text-red'
}

</script>

<template>
  <div class="prediction-market-panel" :data-panel-id="panelId">
    <!-- Header -->
    <div class="panel-header">
      <h3>📊 {{ t('prediction.title') }}</h3>
      <div class="header-actions">
        <span v-if="signalInfo && signalInfo.action !== 'hold'" class="signal-badge" :class="signalInfo.action">
          {{ signalInfo.action === 'buy' ? '🟢' : '🔴' }} {{ signalInfo.description }}
        </span>
        <button class="btn-sm" @click="loadEvents()">🔄 {{ t('common.refresh') }}</button>
      </div>
    </div>

    <!-- Category tabs -->
    <div class="category-tabs">
      <button
        v-for="cat in categories" :key="cat"
        :class="['tab', { active: activeCategory === cat }]"
        @click="activeCategory = cat; loadEvents()"
      >
        {{ categoryLabels[cat] }}
      </button>
    </div>

    <!-- Main content: table + detail -->
    <div class="content-area">
      <!-- Events table -->
      <div class="events-table" :class="{ 'with-detail': selectedEvent }">
        <div v-if="loading" class="empty-state">{{ t('common.loading') }}</div>
        <div v-else-if="sortedEvents.length === 0" class="empty-state">{{ t('prediction.no_data') }}</div>
        <table v-else>
          <thead>
            <tr>
              <th>{{ t('prediction.event') }}</th>
              <th>{{ t('prediction.yes_prob') }}</th>
              <th>{{ t('prediction.change_24h') }}</th>
              <th>{{ t('prediction.volume') }}</th>
              <th>{{ t('prediction.expiry') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="event in sortedEvents" :key="event.id"
              :class="{ selected: selectedEvent?.id === event.id, 'signal-row': event.outcomes[0] && Math.abs(event.outcomes[0].change_24h) > 0.05 }"
              @click="loadDetail(event)"
            >
              <td class="event-title">
                <span class="category-tag">{{ categoryLabels[event.category] || event.category }}</span>
                {{ event.title }}
              </td>
              <td class="prob">{{ (event.outcomes[0]?.price * 100).toFixed(1) }}%</td>
              <td :class="event.outcomes[0] ? changeClass(event.outcomes[0].change_24h) : ''">
                {{ event.outcomes[0] ? formatChange(event.outcomes[0].change_24h) : '-' }}
              </td>
              <td class="vol">{{ formatVolume(event.volume) }}</td>
              <td class="end-date">{{ formatEndDate(event.end_date) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedEvent" class="detail-panel">
        <div class="detail-header">
          <h4>{{ selectedEvent.title }}</h4>
          <button class="btn-close" @click="selectedEvent = null">&times;</button>
        </div>
        <p class="detail-desc">{{ selectedEvent.description }}</p>

        <!-- Outcomes -->
        <div class="outcomes-grid">
          <div v-for="o in selectedEvent.outcomes" :key="o.id" class="outcome-card">
            <span class="outcome-label">{{ o.label }}</span>
            <span class="outcome-price">{{ (o.price * 100).toFixed(1) }}%</span>
            <span :class="['outcome-change', changeClass(o.change_24h)]">{{ formatChange(o.change_24h) }}</span>
          </div>
        </div>

        <!-- Probability chart -->
        <div class="chart-container" v-if="priceHistory.length > 0">
          <VChart :option="chartOption" style="height: 200px" autoresize />
        </div>
        <div v-else class="empty-state small">{{ t('prediction.no_history') }}</div>

        <!-- Meta -->
        <div class="detail-meta">
          <span>{{ t('prediction.volume') }}: {{ formatVolume(selectedEvent.volume) }}</span>
          <span>{{ t('prediction.expiry') }}: {{ formatEndDate(selectedEvent.end_date) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prediction-market-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 13px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
}
.panel-header h3 { margin: 0; font-size: 14px; }

.header-actions { display: flex; gap: 8px; align-items: center; }
.signal-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--color-bg-subtle);
}
.signal-badge.buy { color: var(--color-down); }
.signal-badge.sell { color: var(--color-up); }

.btn-sm {
  padding: 2px 8px;
  font-size: 11px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.btn-sm:hover { background: var(--color-bg-hover); }

.category-tabs {
  display: flex;
  gap: 2px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border);
  overflow-x: auto;
}
.tab {
  padding: 3px 10px;
  font-size: 11px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
}
.tab.active { background: var(--color-accent); color: var(--color-text-primary); }
.tab:hover:not(.active) { background: var(--color-bg-hover); }

.content-area { display: flex; flex: 1; overflow: hidden; }
.events-table { flex: 1; overflow-y: auto; min-width: 0; }
.events-table.with-detail { flex: 0 0 55%; }

table { width: 100%; border-collapse: collapse; }
thead { position: sticky; top: 0; background: var(--color-bg-panel); z-index: 1; }
th { padding: 6px 12px; text-align: left; font-weight: 600; font-size: 11px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border); }
td { padding: 6px 12px; border-bottom: 1px solid var(--color-border-subtle); }
tr { cursor: pointer; }
tr:hover { background: var(--color-bg-hover); }
tr.selected { background: var(--color-bg-selected); }
tr.signal-row { border-left: 3px solid var(--color-accent); }

.event-title { min-width: 200px; }
.category-tag {
  display: inline-block;
  padding: 0 4px;
  font-size: 10px;
  border-radius: 3px;
  background: var(--color-bg-subtle);
  margin-right: 4px;
}

.prob { font-weight: 600; font-variant-numeric: tabular-nums; }
.vol { color: var(--color-text-secondary); font-variant-numeric: tabular-nums; }
.end-date { color: var(--color-text-tertiary); font-size: 11px; }
.text-green { color: var(--color-down); }
.text-red { color: var(--color-up); }

.detail-panel {
  flex: 1;
  border-left: 1px solid var(--color-border);
  padding: 12px;
  overflow-y: auto;
  min-width: 280px;
}
.detail-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px; }
.detail-header h4 { margin: 0; font-size: 14px; }
.btn-close {
  background: none; border: none; font-size: 18px;
  color: var(--color-text-secondary); cursor: pointer;
}

.detail-desc { font-size: 12px; color: var(--color-text-secondary); margin-bottom: 12px; line-height: 1.5; }

.outcomes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(100px, 1fr)); gap: 8px; margin-bottom: 12px; }
.outcome-card {
  display: flex; flex-direction: column; align-items: center;
  padding: 8px; border-radius: 6px; background: var(--color-bg-subtle);
}
.outcome-label { font-size: 11px; color: var(--color-text-secondary); }
.outcome-price { font-size: 20px; font-weight: 700; }
.outcome-change { font-size: 11px; }

.chart-container { margin-bottom: 12px; }

.detail-meta {
  display: flex; gap: 16px;
  font-size: 11px; color: var(--color-text-tertiary);
}

.empty-state {
  display: flex; align-items: center; justify-content: center;
  padding: 40px; color: var(--color-text-tertiary);
}
.empty-state.small { padding: 20px; font-size: 12px; }
</style>
