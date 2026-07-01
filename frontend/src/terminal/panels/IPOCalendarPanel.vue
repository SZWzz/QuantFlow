<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import ErrorBoundary from '@/terminal/components/ErrorBoundary.vue'

const { t } = useI18n()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface IPOItem {
  code: string
  name: string
  issue_price: number
  pe: number
  subscription_date: string
  listing_date: string
  lottery_rate: number
  issue_volume: number
  status: string
}

type TabKey = 'today_apply' | 'upcoming' | 'recent'

const { fetchWithCache } = usePanelCache()
const activeTab = ref<TabKey>('today_apply')
const allData = ref<IPOItem[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

function toDateStr(d: string): string {
  return d ? new Date(d).toLocaleDateString('zh-CN') : '--'
}

function toLocalDate(d: string): Date {
  const parts = d.split('-').map(Number)
  return new Date(parts[0], parts[1] - 1, parts[2])
}

function todayStr(): string {
  const d = new Date()
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
}

function inDays(d: Date, n: number): Date {
  return new Date(d.getTime() + n * 86400000)
}

const filteredData = computed(() => {
  const today = new Date()
  const todayLocal = todayStr()
  switch (activeTab.value) {
    case 'today_apply':
      return allData.value.filter(item => item.subscription_date === todayLocal)
    case 'upcoming': {
      const limit = inDays(today, 14)
      return allData.value.filter(item => {
        if (!item.listing_date) return false
        const ld = toLocalDate(item.listing_date)
        return ld >= today && ld <= limit
      })
    }
    case 'recent': {
      const past = inDays(today, -7)
      return allData.value.filter(item => {
        if (!item.listing_date) return false
        const ld = toLocalDate(item.listing_date)
        return ld >= past && ld <= today
      })
    }
    default:
      return []
  }
})

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetIPOCalendar) return
  loading.value = true
  error.value = null
  try {
    const start = inDays(new Date(), -30).toISOString().slice(0, 10)
    const end = inDays(new Date(), 30).toISOString().slice(0, 10)
    const { data: result } = await fetchWithCache<any>(`ipo_calendar:${start}:${end}`, () => app.GetIPOCalendar(start, end))
    allData.value = (result || []).map((r: any) => ({
      code: r.code || '',
      name: r.name || '',
      issue_price: r.issue_price || 0,
      pe: r.pe || 0,
      subscription_date: r.subscription_date || '',
      listing_date: r.listing_date || '',
      lottery_rate: r.lottery_rate || 0,
      issue_volume: r.issue_volume || 0,
      status: r.status || '',
    }))
  } catch (e: any) {
    console.error('[IPOCalendar]', e)
    error.value = e.message || String(e)
    allData.value = []
  } finally {
    loading.value = false
  }
}

function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function switchTab(tab: TabKey) {
  activeTab.value = tab
}

function formatLotteryRate(rate: number): string {
  if (!rate) return '--'
  return (rate * 100).toFixed(3) + '%'
}

function formatVolume(v: number): string {
  if (!v) return '--'
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return String(v)
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <ErrorBoundary :panel-id="panelId">
    <div class="ipo-calendar-panel">
      <div class="panel-header">
        <h3>{{ $t('panels.ipo_calendar') }}</h3>
        <div class="header-tabs">
          <button :class="['tab', { active: activeTab === 'today_apply' }]" @click="switchTab('today_apply')">{{ $t('panels.today_apply') }}</button>
          <button :class="['tab', { active: activeTab === 'upcoming' }]" @click="switchTab('upcoming')">{{ $t('panels.upcoming') }}</button>
          <button :class="['tab', { active: activeTab === 'recent' }]" @click="switchTab('recent')">{{ $t('panels.recent') }}</button>
        </div>
        <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
      </div>

      <div v-if="error" class="error-state">
        <span class="error-icon">⚠</span>
        <span class="error-text">{{ error }}</span>
        <button class="retry-btn" @click="fetchData">{{ $t('common.retry') }}</button>
      </div>

      <SkeletonPanel v-else-if="loading && !allData.length" type="table" :rows="8" />

      <div v-else-if="filteredData.length === 0" class="empty-state">
        <span class="empty-icon">📋</span>
        <span>{{ $t('panels.no_data') }}</span>
      </div>

      <div v-else class="table-wrapper">
        <div class="table-header">
          <span class="col-code">{{ $t('common.symbol') }}</span>
          <span class="col-name">{{ $t('common.name') }}</span>
          <span class="col-price">{{ $t('common.price') }}</span>
          <span class="col-pe">PE</span>
          <span class="col-sub-date">{{ $t('common.subscription_date') }}</span>
          <span class="col-list-date">{{ $t('common.listing_date') }}</span>
          <span class="col-lottery">{{ $t('panels.lottery_rate') }}</span>
          <span class="col-status">{{ $t('common.status') }}</span>
        </div>
        <div class="table-body">
          <div v-for="row in filteredData" :key="row.code + row.subscription_date" class="table-row">
            <span class="col-code clickable" @click="onSymbolClick(row.code)">{{ row.code }}</span>
            <span class="col-name">{{ row.name }}</span>
            <span class="col-price">{{ row.issue_price ? row.issue_price.toFixed(2) : '--' }}</span>
            <span class="col-pe">{{ row.pe ? row.pe.toFixed(2) : '--' }}</span>
            <span class="col-sub-date">{{ toDateStr(row.subscription_date) }}</span>
            <span class="col-list-date">{{ toDateStr(row.listing_date) }}</span>
            <span class="col-lottery">{{ formatLotteryRate(row.lottery_rate) }}</span>
            <span class="col-status">{{ row.status || '--' }}</span>
          </div>
        </div>
      </div>
    </div>
  </ErrorBoundary>
</template>

<style scoped>
.ipo-calendar-panel {
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
.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
}
.error-icon { font-size: 24px; }
.error-text { max-width: 300px; text-align: center; word-break: break-all; color: var(--color-warning, var(--color-accent)); }
.retry-btn {
  padding: 6px 16px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 12px;
}
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
}
.empty-icon { font-size: 24px; }
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
.col-code { width: 72px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 64px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-price { width: 56px; text-align: right; }
.col-pe { width: 52px; text-align: right; }
.col-sub-date { width: 80px; text-align: center; }
.col-list-date { width: 80px; text-align: center; }
.col-lottery { width: 64px; text-align: right; }
.col-status { flex: 1; min-width: 0; text-align: center; color: var(--color-text-secondary); }
</style>
