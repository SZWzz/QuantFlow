<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDataFetch } from '@/lib/composables/useDataFetch'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

const { fetchWithCache } = usePanelCache()

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()

interface CryptoRow {
  symbol: string
  price: number
  changePct24h: number
}

const sortKey = ref<string>('changePct24h')
const sortDir = ref<'asc' | 'desc' | null>('desc')

const { data: cryptos, loading, error, execute: refreshExec } = useDataFetch<CryptoRow[]>(async () => {
  const { data: result } = await fetchWithCache<any>('crypto_overview', () => (window as any).go?.main?.App?.GetCryptoOverview([]), 3 * 60 * 1000)
  if (result?.cryptos) {
    return result.cryptos.map((c: any) => ({
      symbol: c.symbol?.replace('USDT', '') || c.symbol,
      price: c.price || 0,
      changePct24h: c.change_pct || 0,
    }))
  }
  return []
})

const sortedCryptos = computed(() => {
  const arr = [...(cryptos.value || [])]
  if (!sortKey.value || !sortDir.value) return arr
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof CryptoRow]
    const bVal = b[sortKey.value as keyof CryptoRow]
    const cmp = typeof aVal === 'number' && typeof bVal === 'number'
      ? aVal - bVal
      : String(aVal).localeCompare(String(bVal))
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return arr
})

/** rank 列为展示序号，排好序后预映射进数据行 */
const rows = computed(() => sortedCryptos.value.map((c, i) => ({ ...c, rank: i + 1 })))

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sortKey.value = dir ? key : ''
  sortDir.value = dir
}

function formatPrice(p: number): string {
  if (p >= 1000) return p.toLocaleString('en-US', { maximumFractionDigits: 2 })
  if (p >= 1) return p.toFixed(2)
  if (p >= 0.01) return p.toFixed(4)
  return p.toFixed(8)
}

const cols = computed<Column[]>(() => [
  { key: 'rank', label: '#', width: 28, align: 'center' },
  { key: 'symbol', label: t('quote.symbol'), sortable: true },
  { key: 'price', label: t('common.price'), align: 'right', sortable: true, formatter: formatPrice },
  { key: 'changePct24h', label: '24h涨跌%', align: 'right', format: 'percent', colorize: true, sortable: true },
])

function refresh() {
  refreshExec()
}

onMounted(refresh)
</script>

<template>
  <div class="crypto-overview-panel">
    <PanelHeader
      :title="$t('misc.crypto_overview')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: refresh, loading }]"
    />

    <LoadingState v-if="loading && !cryptos" type="table" :rows="6" :cols="cols.length" />
    <ErrorState v-else-if="error" :description="error" @retry="refresh" />
    <PanelTable
      v-else
      :columns="cols"
      :data="rows"
      :loading="loading"
      :sort-key="sortKey"
      :sort-dir="sortDir"
      sticky-header
      @sort-change="onSortChange"
    />
  </div>
</template>

<style scoped>
.crypto-overview-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
