<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

const { fetchWithCache } = usePanelCache()

defineProps<{ panelId: string; params?: Record<string, any> }>()
const loading = ref(false)
const error = ref('')
const data = ref<any>(null)
const searchQuery = ref('')

const SOURCE = 'akshare'
const DATA_TYPE = 'futures'

const cols: Column[] = [
  { key: '代码', label: '代码', mono: true },
  { key: '名称', label: '名称' },
  { key: '最新价', label: '最新价', align: 'right', format: 'price' },
  { key: '涨跌幅', label: '涨跌幅', align: 'right', format: 'percent', colorize: true },
  { key: '涨跌额', label: '涨跌额', align: 'right', format: 'price' },
  { key: '今开', label: '今开', align: 'right', format: 'price' },
  { key: '最高', label: '最高', align: 'right', format: 'price' },
  { key: '最低', label: '最低', align: 'right', format: 'price' },
  { key: '昨结', label: '昨结', align: 'right', format: 'price' },
  { key: '成交量', label: '成交量', align: 'right', format: 'volume' },
  { key: '持仓量', label: '持仓量', align: 'right', format: 'volume' },
]
const numericKeys = ['最新价', '涨跌幅', '涨跌额', '今开', '最高', '最低', '昨结', '成交量', '持仓量']

const filteredData = computed(() => {
  const rows = data.value?.data ?? []
  if (!searchQuery.value) return rows
  const q = searchQuery.value.toLowerCase()
  return rows.filter((r: any) =>
    (String(r['代码'] || '')).toLowerCase().includes(q) ||
    (String(r['名称'] || '')).toLowerCase().includes(q)
  )
})

/** 源数据数值可能是字符串，转成 number 供 PanelTable 的 format/colorize 使用 */
function toNum(v: any): any {
  if (v == null || v === '') return v
  const n = typeof v === 'number' ? v : parseFloat(v)
  return Number.isFinite(n) ? n : v
}

const tableRows = computed(() =>
  filteredData.value.map((r: any) => {
    const row: any = { ...r }
    for (const k of numericKeys) row[k] = toNum(r[k])
    return row
  }),
)

const hasRows = computed(() => (data.value?.data?.length ?? 0) > 0)

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = useWailsApp()
    if (app?.FetchData) {
      const { data: result } = await fetchWithCache<any>('futures_data', () => app.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 15 * 60 * 1000)
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

onMounted(loadData)
</script>

<template>
  <div class="futures-panel">
    <PanelHeader
      title="全球期货实时行情"
      :controls="[{ icon: 'refresh', title: '刷新', action: loadData, loading }]"
    >
      <template #controls>
        <input class="search-input" v-model="searchQuery" placeholder="搜索代码/名称" />
      </template>
    </PanelHeader>

    <LoadingState v-if="loading && !hasRows" type="table" :rows="6" :cols="cols.length" />
    <ErrorState v-else-if="error" :description="error" @retry="loadData" />
    <EmptyState v-else-if="!data || !data.success" :title="data?.error || '暂无数据'" />
    <template v-else>
      <div class="info-row">共 {{ filteredData.length }} 个合约</div>
      <PanelTable :columns="cols" :data="tableRows" :loading="loading" sticky-header />
    </template>
  </div>
</template>

<style scoped>
.futures-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

.search-input {
  width: 130px;
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: var(--font-xs);
}
.info-row {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  padding: var(--space-sm) var(--panel-padding) var(--space-xs);
  flex-shrink: 0;
}
</style>
