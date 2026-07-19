<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()

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
const sortDir = ref<'asc' | 'desc' | null>('desc')
const { fetchWithCache } = usePanelCache()

const sorted = computed(() => {
  const arr = [...txs.value]
  if (!sortKey.value || !sortDir.value) return arr
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof WhaleTx] ?? 0
    const bVal = b[sortKey.value as keyof WhaleTx] ?? 0
    const cmp = typeof aVal === 'number' && typeof bVal === 'number'
      ? aVal - bVal
      : String(aVal).localeCompare(String(bVal))
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return arr
})

/** token 展示回退（token || symbol）预映射进数据行；token 排序不受影响的复制 */
const rows = computed(() => sorted.value.map(tx => ({ ...tx, token: tx.token || tx.symbol })))

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sortKey.value = dir ? key : ''
  sortDir.value = dir
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

/** 大额交易（>$10M）行级高亮 */
function rowClass(tx: WhaleTx): string {
  return tx.usd_value > 10e6 ? 'mega-row' : ''
}

const cols = computed<Column[]>(() => [
  { key: 'time', label: t('misc.time'), sortable: true, formatter: formatTime, width: 88 },
  { key: 'token', label: t('misc.token'), sortable: true },
  { key: 'from', label: t('misc.from'), mono: true, formatter: shorten, title: (row: any) => row.from },
  { key: 'to', label: t('misc.to'), mono: true, formatter: shorten, title: (row: any) => row.to },
  { key: 'usd_value', label: t('misc.whale_value'), align: 'right', sortable: true, formatter: fmUSD },
])

onMounted(fetchData)
</script>

<template>
  <div class="whale-tracking-panel">
    <PanelHeader
      :title="$t('misc.whale_tracking')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
    >
      <template #controls>
        <input v-model="address" :placeholder="$t('misc.whale_address_hint')" class="addr-input" />
      </template>
    </PanelHeader>

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && txs.length === 0" type="table" :rows="6" :cols="cols.length" />
    <EmptyState
      v-else-if="txs.length === 0"
      :title="$t('misc.whale_empty')"
      :description="$t('misc.whale_hint')"
    />
    <PanelTable
      v-else
      :columns="cols"
      :data="rows"
      :loading="loading"
      :sort-key="sortKey"
      :sort-dir="sortDir"
      :row-class="rowClass"
      :row-key="(tx: any, i: number) => tx.hash || i"
      sticky-header
      @sort-change="onSortChange"
    />
  </div>
</template>

<style scoped>
.whale-tracking-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.addr-input {
  width: 140px;
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: var(--font-xs);
}
.addr-input::placeholder { color: var(--color-text-tertiary); }

:deep(.table-row.mega-row) {
  background: var(--color-warning-soft);
}
</style>
