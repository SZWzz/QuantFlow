<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface WhaleTx {
  hash: string
  from: string
  to: string
  token: string
  value: number
  usd_value: number
  time: number
  symbol: string
}

const txs = ref<WhaleTx[]>([])
const loading = ref(false)
const loadError = ref('')
let loadSeq = 0
const address = ref('')
const minUsd = ref(1000000)
const sortKey = ref<string>('usd_value')
const sortDir = ref<number>(-1)
const { fetchWithCache } = usePanelCache()

const sorted = computed(() => {
  const arr = [...txs.value]
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof WhaleTx] ?? 0
    const bVal = b[sortKey.value as keyof WhaleTx] ?? 0
    return (typeof aVal === 'number' ? aVal - bVal : String(aVal).localeCompare(String(bVal))) * sortDir.value
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

function shorten(addr: string): string {
  if (!addr || addr.length < 10) return addr || '?'
  return addr.slice(0, 6) + '...' + addr.slice(-4)
}

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetWhaleTransactions) return
  const seq = ++loadSeq
  loadError.value = ''
  loading.value = true
  try {
    const { data: raw } = await fetchWithCache<any>(`whale_txs:${address.value}:${minUsd.value}`, () => app.GetWhaleTransactions(address.value), 3 * 60 * 1000)
    if (seq !== loadSeq) return
    const items = raw?.data || raw?.result || []
    txs.value = (Array.isArray(items) ? items : []).map((t: any) => ({
      hash: t.hash || '',
      from: t.from || '',
      to: t.to || '',
      token: t.tokenSymbol || t.tokenName || 'ETH',
      value: Number(t.value || 0) / 1e18,
      usd_value: t.usd_value || (Number(t.value || 0) / 1e18 * (t.tokenUSD || 2000)),
      time: Number(t.timeStamp || t.timestamp || 0) * 1000,
      symbol: t.tokenSymbol || t.tokenName || '',
    })).filter(t => t.usd_value >= minUsd.value)
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    txs.value = []
  } finally {
    loading.value = false
  }
}

function fmUSD(n: number): string {
  if (n >= 1e8) return '$' + (n / 1e8).toFixed(2) + '亿'
  if (n >= 1e4) return '$' + (n / 1e4).toFixed(2) + '万'
  return '$' + n.toLocaleString('en-US', { maximumFractionDigits: 0 })
}

function formatTime(ts: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchData)
</script>

<template>
  <div class="whale-tracking-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.whale_tracking') }}</h3>
      <input v-model="address" :placeholder="$t('misc.whale_address_hint')" class="addr-input" />
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="loadError" class="error-state" @click="fetchData">{{ loadError }} ⟳</div>

    <SkeletonPanel v-else-if="loading && txs.length === 0" type="table" :rows="6" />

    <div v-else-if="txs.length === 0 && !loading" class="empty-state">
      <div>{{ $t('misc.whale_empty') }}</div>
      <div class="hint">{{ $t('misc.whale_hint') }}</div>
    </div>

    <template v-else>
      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-time sortable" @click="toggleSort('time')">{{ $t('misc.time') }}{{ sortArrow('time') }}</span>
          <span class="col-token sortable" @click="toggleSort('token')">{{ $t('misc.token') }}{{ sortArrow('token') }}</span>
          <span class="col-from">{{ $t('misc.from') }}</span>
          <span class="col-to">{{ $t('misc.to') }}</span>
          <span class="col-value sortable" @click="toggleSort('usd_value')">{{ $t('misc.whale_value') }}{{ sortArrow('usd_value') }}</span>
        </div>
        <div class="table-body">
          <div v-for="(tx, i) in sorted" :key="tx.hash || i" class="table-row" :class="{ mega: tx.usd_value > 10e6 }">
            <span class="col-time">{{ formatTime(tx.time) }}</span>
            <span class="col-token">{{ tx.token || tx.symbol }}</span>
            <span class="col-from" :title="tx.from">{{ shorten(tx.from) }}</span>
            <span class="col-to" :title="tx.to">{{ shorten(tx.to) }}</span>
            <span class="col-value">{{ fmUSD(tx.usd_value) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.whale-tracking-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.addr-input {
  padding: 3px 8px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 12px; width: 140px;
}
.addr-input::placeholder { color: var(--color-text-tertiary); font-size: 10px; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px; gap: 4px;
}
.error-state {
  display: flex; align-items: center; justify-content: center; padding: 12px;
  color: var(--color-error); font-size: 13px; cursor: pointer;
}
.hint { font-size: 11px; opacity: 0.6; }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--color-text-primary); }
.table-body { flex: 1; overflow-y: auto; font-size: 11px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.table-row.mega { background: rgba(245,158,11,0.06); }
.col-time { width: 80px; font-size: 10px; color: var(--color-text-tertiary); }
.col-token { width: 52px; font-weight: 600; }
.col-from, .col-to { width: 80px; font-family: monospace; font-size: 10px; color: var(--color-text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-value { flex: 1; min-width: 0; text-align: right; font-weight: 600; font-variant-numeric: tabular-nums; }
</style>
