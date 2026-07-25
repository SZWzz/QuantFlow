<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loaded')

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const loadError = ref('')
const data = ref<any>(null)
const searchQuery = ref('')

const SOURCE = 'akshare'
const DATA_TYPE = 'bonds'

const cols: Column[] = [
  { key: 'symbol', label: '代码', mono: true },
  { key: 'name', label: '名称' },
  { key: 'trade', label: '最新价', align: 'right', mono: true },
  { key: 'changepercent', label: '涨跌幅', align: 'right', format: 'percent', colorize: true },
  { key: 'volume', label: '成交量', align: 'right', format: 'volume' },
  { key: 'amount', label: '成交额', align: 'right', format: 'volume' },
  { key: 'code', label: '正股代码', mono: true },
  { key: 'ticktime', label: '时间' },
]

const subtitle = computed(() => [symbol.value, name.value].filter(Boolean).join(' '))

const filteredData = computed(() => {
  const rows = data.value?.data ?? []
  if (!searchQuery.value) return rows
  const q = searchQuery.value.toLowerCase()
  return rows.filter((r: any) =>
    (r.symbol || '').toLowerCase().includes(q) ||
    (r.code || '').toLowerCase().includes(q) ||
    (r.name || '').toLowerCase().includes(q)
  )
})

/** 源数据数值可能是字符串，转成 number 供 PanelTable 的 format/colorize 使用 */
function toNum(v: any): any {
  if (v == null || v === '') return v
  const n = typeof v === 'number' ? v : parseFloat(v)
  return Number.isFinite(n) ? n : v
}

const tableRows = computed(() =>
  filteredData.value.map((r: any) => ({
    ...r,
    changepercent: toNum(r.changepercent),
    volume: toNum(r.volume),
    amount: toNum(r.amount),
  })),
)

const hasRows = computed(() => (data.value?.data?.length ?? 0) > 0)

async function loadData() {
  loading.value = true; loadError.value = ''
  try {
    const app = useWailsApp()
    if (app?.FetchData) {
      const { data: result } = await fetchWithCache('bonds:' + symbol.value, async () => {
        return await app.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) loadError.value = result.error
    }
  } catch (e: any) { loadError.value = e.message || '加载失败' }
  finally { loading.value = false }
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
</script>

<template>
  <PanelShell :state="state">
    <template #loaded>
      <div class="bonds-panel">
        <PanelHeader
          title="可转债实时行情"
          :subtitle="subtitle"
          :controls="[{ icon: 'refresh', title: '刷新', action: loadData, loading }]"
        >
          <template #controls>
            <input class="search-input" v-model="searchQuery" placeholder="搜索代码/名称" />
          </template>
        </PanelHeader>

        <LoadingState v-if="loading && !hasRows" type="table" :rows="6" :cols="cols.length" />
        <ErrorState v-else-if="loadError" :description="loadError" @retry="loadData" />
        <EmptyState v-else-if="!data || !data.success" :title="data?.error || '暂无数据'" />
        <template v-else>
          <div class="info-row">共 {{ filteredData.length }} 只可转债</div>
          <PanelTable :columns="cols" :data="tableRows" :loading="loading" sticky-header />
        </template>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.bonds-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

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
