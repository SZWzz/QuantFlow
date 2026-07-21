<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, LoadingState, type Column } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()

interface FundingRate {
  symbol: string
  mark_price: number
  index_price: number
  funding_rate: number
  next_funding_time: number
}

const sortKey = ref<string>('funding_rate')
const sortDir = ref<'asc' | 'desc' | null>('desc')
const rates = ref<FundingRate[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
const { fetchWithCache } = usePanelCache()
let timer: ReturnType<typeof setInterval> | null = null

const sortedRates = computed(() => {
  const arr = [...rates.value]
  if (!sortKey.value || !sortDir.value) return arr
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof FundingRate]
    const bVal = b[sortKey.value as keyof FundingRate]
    const cmp = typeof aVal === 'number' && typeof bVal === 'number'
      ? aVal - bVal
      : String(aVal).localeCompare(String(bVal))
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return arr
})

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sortKey.value = dir ? key : ''
  sortDir.value = dir
}

async function fetchRates() {
  const app = (window as any).go?.main?.App
  if (!app?.GetCryptoFundingRates) return
  loading.value = true
  try {
    const { data: result } = await fetchWithCache<any>('funding_rates', () => app.GetCryptoFundingRates([]), 60 * 1000)
    rates.value = (result || []).map((r: any) => ({
      symbol: r.symbol?.replace('USDT', '') || r.symbol || '',
      mark_price: r.mark_price || 0,
      index_price: r.index_price || 0,
      funding_rate: r.funding_rate || 0,
      next_funding_time: r.next_funding_time || 0,
    }))
  } catch (e) {
    console.error('[FundingRate]', e)
    rates.value = []
  } finally {
    loading.value = false
  }
}

function formatRate(rate: number): string {
  return (rate * 100).toFixed(4) + '%'
}

function formatPrice(p: number): string {
  if (p >= 1000) return p.toLocaleString('en-US', { maximumFractionDigits: 2 })
  if (p >= 1) return p.toFixed(2)
  return p.toFixed(4)
}

/** 费率阈值上色：正费率偏多付费、负费率空方付费，±0.01% 内视为中性 */
function rateClass(rate: number): string {
  if (rate > 0.0001) return 'rate-up'
  if (rate < -0.0001) return 'rate-down'
  return ''
}

function nextFundingTime(ts: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) + ' UTC'
}

function isExtreme(rate: number): boolean {
  return Math.abs(rate) > 0.001
}

function rowClass(r: FundingRate): string {
  return isExtreme(r.funding_rate) ? 'extreme-row' : ''
}

const hasExtreme = computed(() => rates.value.some(r => isExtreme(r.funding_rate)))

const cols = computed<Column[]>(() => [
  { key: 'symbol', label: t('quote.symbol'), mono: true, sortable: true },
  { key: 'mark_price', label: t('misc.mark_price'), align: 'right', sortable: true, formatter: formatPrice },
  { key: 'index_price', label: t('misc.index_price'), align: 'right', sortable: true, formatter: formatPrice },
  { key: 'funding_rate', label: t('misc.funding_rate_short'), align: 'right', sortable: true, formatter: formatRate, cellClass: (r: any) => rateClass(r.funding_rate) },
  { key: 'next_funding_time', label: t('misc.next_settle'), align: 'right', formatter: nextFundingTime, cellClass: () => 'muted-cell' },
])

onMounted(() => {
  fetchRates()
  timer = setInterval(() => { if (autoRefresh.value) fetchRates() }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="funding-rate-panel">
    <PanelHeader
      :title="$t('misc.funding_rate')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchRates, loading }]"
    >
      <template #controls>
        <button class="btn btn-sm btn-ghost auto-toggle" :class="{ active: autoRefresh }" @click="autoRefresh = !autoRefresh">
          {{ autoRefresh ? '自动(30s)' : '手动' }}
        </button>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading && rates.length === 0" type="table" :rows="6" :cols="cols.length" />

    <EmptyState v-else-if="rates.length === 0" :title="$t('common.no_data')" />

    <template v-else>
      <div v-if="hasExtreme" class="alert-bar">
        ⚠ {{ $t('misc.funding_extreme') }}
      </div>
      <PanelTable
        :columns="cols"
        :data="sortedRates"
        :loading="loading"
        :sort-key="sortKey"
        :sort-dir="sortDir"
        :row-class="rowClass"
        sticky-header
        @sort-change="onSortChange"
      />
    </template>
  </div>
</template>

<style scoped>
.funding-rate-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.auto-toggle.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.alert-bar {
  padding: var(--space-xs) var(--panel-padding);
  border-bottom: 1px solid var(--color-warn);
  background: var(--color-warning-soft);
  color: var(--color-warn);
  font-size: var(--font-xs);
  font-weight: 500;
  flex-shrink: 0;
}

:deep(.td.rate-up) { color: var(--color-up); font-weight: 500; }
:deep(.td.rate-down) { color: var(--color-down); font-weight: 500; }
:deep(.td.muted-cell) { color: var(--color-text-tertiary); }
:deep(.table-row.extreme-row) { background: var(--color-warning-soft); }
</style>
