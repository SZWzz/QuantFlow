<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()

interface Protocol {
  name: string
  chain: string
  tvl: number
  change_1d: number
  change_7d: number
  mcap: number
  category: string
}

const protocols = ref<Protocol[]>([])
const loading = ref(false)
const loadError = ref('')
const { fetchWithCache } = usePanelCache()
const search = ref('')
const sortKey = ref<string>('tvl')
const sortDir = ref<'asc' | 'desc' | null>('desc')

const filtered = computed(() => {
  const kw = search.value.toLowerCase()
  const arr = kw ? protocols.value.filter(p =>
    p.name.toLowerCase().includes(kw) || p.chain.toLowerCase().includes(kw)
  ) : [...protocols.value]
  if (sortKey.value && sortDir.value) {
    arr.sort((a, b) => {
      const aVal = a[sortKey.value as keyof Protocol] ?? 0
      const bVal = b[sortKey.value as keyof Protocol] ?? 0
      const cmp = typeof aVal === 'number' && typeof bVal === 'number'
        ? aVal - bVal
        : String(aVal).localeCompare(String(bVal))
      return sortDir.value === 'asc' ? cmp : -cmp
    })
  }
  return arr
})

/** rank 列为展示序号，筛选+排序后预映射进数据行 */
const rows = computed(() => filtered.value.map((p, i) => ({ ...p, rank: i + 1 })))

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sortKey.value = dir ? key : ''
  sortDir.value = dir
}

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDeFiTVL) return
  loadError.value = ''
  loading.value = true
  try {
    const { data: result } = await fetchWithCache<any>('defi_tvl', () => app.GetDeFiTVL(), 3 * 60 * 1000)
    const items = result?.data || []
    protocols.value = items.slice(0, 150).map((p: any) => ({
      name: p.name || p.id || '?',
      chain: p.chain || (p.chains?.[0] || 'multi'),
      tvl: p.tvl || p.tvl30d?.at(-1)?.[1] || 0,
      change_1d: p.change_1d || 0,
      change_7d: p.change_7d || 0,
      mcap: p.mcap || 0,
      category: p.category || '',
    }))
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    protocols.value = []
  } finally {
    loading.value = false
  }
}

function fmTVL(n: number): string {
  if (n >= 1e8) return '$' + (n / 1e8).toFixed(2) + '亿'
  if (n >= 1e4) return '$' + (n / 1e4).toFixed(1) + '万'
  return '$' + n.toFixed(0)
}

function fmtChange(n: number): string {
  return (n > 0 ? '+' : '') + (n * 100).toFixed(2) + '%'
}

const cols = computed<Column[]>(() => [
  { key: 'rank', label: '#', width: 28, align: 'center' },
  { key: 'name', label: t('quote.name'), flex: 2, sortable: true },
  { key: 'chain', label: t('misc.chain'), sortable: true },
  { key: 'tvl', label: t('misc.tvl'), align: 'right', sortable: true, formatter: fmTVL },
  { key: 'change_1d', label: '1d', align: 'right', sortable: true, formatter: fmtChange, colorize: true },
  { key: 'change_7d', label: '7d', align: 'right', sortable: true, formatter: fmtChange, colorize: true },
])

onMounted(fetchData)
</script>

<template>
  <div class="defi-tvl-panel">
    <PanelHeader
      :title="$t('misc.defi_tvl')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
    >
      <template #controls>
        <input v-model="search" :placeholder="$t('common.search')" class="search-input" />
      </template>
    </PanelHeader>

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && protocols.length === 0" type="table" :rows="10" :cols="cols.length" />
    <EmptyState v-else-if="protocols.length === 0" :title="$t('common.no_data')" />
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
.defi-tvl-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-input {
  width: 120px;
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: var(--font-xs);
}
</style>
