<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, PanelTable, StatItem, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '000001')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const app = useWailsApp()
const loading = ref(false)
const error = ref('')
const rawData = ref<any>(null)

const SOURCE = 'akshare'
const DATA_TYPE = 'margin'

interface MarginRow {
  date: string
  margin_balance: number
  short_balance: number
  margin_balance_day: number
  short_balance_day: number
  [key: string]: any
}

const rows = computed<MarginRow[]>(() => {
  if (!rawData.value) return []
  const data = rawData.value.data ?? rawData.value
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.items)) return data.items
  if (Array.isArray(data?.records)) return data.records
  return []
})

const latest = computed(() => rows.value[0] || null)

/** 源数据键大小写不稳定，统一兜底取值 */
function colValue(row: MarginRow, key: string) {
  const v = row[key] ?? row[key.toLowerCase()] ?? row[key.toUpperCase()]
  return v
}

/** 表格行预映射（前 30 条），数值兜底为 0 供 formatter 计算 */
const tableRows = computed(() =>
  rows.value.slice(0, 30).map(r => ({
    date: colValue(r, 'date'),
    margin_balance: colValue(r, 'margin_balance') || 0,
    short_balance: colValue(r, 'short_balance') || 0,
    margin_balance_day: colValue(r, 'margin_balance_day') || 0,
    short_balance_day: colValue(r, 'short_balance_day') || 0,
  })),
)

function inYi(v: number): string {
  return (v / 1e8).toFixed(2)
}

function statInYi(key: string): string {
  if (!latest.value) return '--'
  return ((colValue(latest.value, key) || 0) / 1e8).toFixed(1) + '亿'
}

const diffInYi = computed(() => {
  if (!latest.value) return '--'
  const diff = (colValue(latest.value, 'margin_balance') || 0) - (colValue(latest.value, 'short_balance') || 0)
  return (diff / 1e8).toFixed(1) + '亿'
})

const subtitle = computed(() => [symbol.value, name.value].filter(Boolean).join(' '))

const cols: Column[] = [
  { key: 'date', label: '日期' },
  { key: 'margin_balance', label: '融资余额(亿)', align: 'right', formatter: inYi },
  { key: 'short_balance', label: '融券余额(亿)', align: 'right', formatter: inYi },
  { key: 'margin_balance_day', label: '日融资买入(亿)', align: 'right', formatter: inYi },
  { key: 'short_balance_day', label: '日融券卖出(亿)', align: 'right', formatter: inYi },
]

async function loadData() {
  loading.value = true; error.value = ''
  try {
    if (app?.FetchData) {
      const { data: result } = await fetchWithCache('margin:' + symbol.value, async () => {
        return await app.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) rawData.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
</script>

<template>
  <div class="margin-panel">
    <PanelHeader
      title="融资融券"
      :subtitle="subtitle"
      :controls="[{ icon: 'refresh', title: '刷新', action: loadData, loading }]"
    />

    <LoadingState v-if="loading && rows.length === 0" type="table" :rows="5" :cols="cols.length" />

    <ErrorState v-else-if="error" :description="error" @retry="loadData" />
    <EmptyState v-else-if="!loading && rows.length === 0" title="暂无融资融券数据" />

    <template v-else>
      <div v-if="latest" class="stats-row">
        <StatItem label="融资余额" :value="statInYi('margin_balance')" />
        <StatItem label="融券余额" :value="statInYi('short_balance')" />
        <StatItem label="余额差值" :value="diffInYi" />
      </div>

      <PanelTable :columns="cols" :data="tableRows" :loading="loading" sticky-header />
    </template>
  </div>
</template>

<style scoped>
.margin-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.stats-row {
  display: flex;
  gap: var(--space-xl);
  padding: var(--space-sm) var(--panel-padding);
  flex-shrink: 0;
}
</style>
