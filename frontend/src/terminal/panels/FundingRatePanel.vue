<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface FundingRate {
  symbol: string
  mark_price: number
  index_price: number
  funding_rate: number
  next_funding_time: number
}

const sortKey = ref<string>('funding_rate')
const sortDir = ref<number>(-1)
const rates = ref<FundingRate[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
const { fetchWithCache } = usePanelCache()
let timer: ReturnType<typeof setInterval> | null = null

const sortedRates = computed(() => {
  const arr = [...rates.value]
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof FundingRate]
    const bVal = b[sortKey.value as keyof FundingRate]
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return (aVal - bVal) * sortDir.value
    }
    return 0
  })
  return arr
})

function toggleSort(key: string) {
  if (sortKey.value === key) sortDir.value *= -1
  else { sortKey.value = key; sortDir.value = -1 }
}

function sortArrow(key: string): string {
  if (sortKey.value !== key) return ''
  return sortDir.value === -1 ? ' ▼' : ' ▲'
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

function rateColor(rate: number): string {
  if (rate > 0.0001) return '#dc2626'
  if (rate < -0.0001) return '#16a34a'
  return 'var(--color-text-primary)'
}

function nextFundingTime(ts: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) + ' UTC'
}

function isExtreme(rate: number): boolean {
  return Math.abs(rate) > 0.001
}

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
    <div class="panel-header">
      <h3>{{ $t('misc.funding_rate') }}</h3>
      <button class="auto-btn" :class="{ active: autoRefresh }" @click="autoRefresh = !autoRefresh">
        {{ autoRefresh ? '自动(30s)' : '手动' }}
      </button>
      <button class="refresh-btn" @click="fetchRates" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && rates.length === 0" type="table" :rows="6" />

    <div v-else-if="rates.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>

    <template v-else>
      <div v-if="sortedRates.some(r => isExtreme(r.funding_rate))" class="alert-bar">
        ⚠ {{ $t('misc.funding_extreme') }}
      </div>
      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-sym sortable" @click="toggleSort('symbol')">{{ $t('quote.symbol') }}{{ sortArrow('symbol') }}</span>
          <span class="col-mp sortable" @click="toggleSort('mark_price')">{{ $t('misc.mark_price') }}{{ sortArrow('mark_price') }}</span>
          <span class="col-ip sortable" @click="toggleSort('index_price')">{{ $t('misc.index_price') }}{{ sortArrow('index_price') }}</span>
          <span class="col-fr sortable" @click="toggleSort('funding_rate')">{{ $t('misc.funding_rate_short') }}{{ sortArrow('funding_rate') }}</span>
          <span class="col-next">{{ $t('misc.next_settle') }}</span>
        </div>
        <div class="table-body">
          <div v-for="r in sortedRates" :key="r.symbol" class="table-row" :class="{ extreme: isExtreme(r.funding_rate) }">
            <span class="col-sym">{{ r.symbol }}</span>
            <span class="col-mp">{{ formatPrice(r.mark_price) }}</span>
            <span class="col-ip">{{ formatPrice(r.index_price) }}</span>
            <span class="col-fr" :style="{ color: rateColor(r.funding_rate) }">{{ formatRate(r.funding_rate) }}</span>
            <span class="col-next">{{ nextFundingTime(r.next_funding_time) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.funding-rate-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.auto-btn {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.auto-btn.active { color: var(--color-accent); border-color: var(--color-accent); }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.alert-bar {
  padding: 6px 10px; margin-bottom: 8px; border-radius: var(--radius-sm);
  background: rgba(245,158,11,0.1); border: 1px solid rgba(245,158,11,0.3);
  color: var(--color-accent); font-size: 11px; font-weight: 500;
}
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--color-text-primary); }
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.table-row.extreme { background: rgba(245,158,11,0.05); }
.col-sym { width: 56px; font-weight: 600; }
.col-mp, .col-ip { width: 72px; text-align: right; font-variant-numeric: tabular-nums; }
.col-fr { width: 76px; text-align: right; font-weight: 500; font-variant-numeric: tabular-nums; }
.col-next { flex: 1; min-width: 0; text-align: right; color: var(--color-text-tertiary); font-size: 11px; }
</style>
