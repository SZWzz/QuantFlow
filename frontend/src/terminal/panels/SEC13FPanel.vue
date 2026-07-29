<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()
const app = useWailsApp()
const loading = ref(false)
const error = ref('')
const rawData = ref<any>(null)
const sortKey = ref('')
const sortDir = ref<'asc' | 'desc' | null>(null)

const SOURCE = 'sec'
const DATA_TYPE = '13f'
const MAX_ROWS = 100

const holdings = computed<any[]>(() => {
  if (!rawData.value) return []
  const data = rawData.value.data ?? rawData.value
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.holdings)) return data.holdings
  if (Array.isArray(data?.items)) return data.items
  if (Array.isArray(data?.records)) return data.records
  return []
})

const sorted = computed(() => {
  if (!sortKey.value || !sortDir.value) return holdings.value
  return [...holdings.value].sort((a, b) => {
    const av = a[sortKey.value] ?? 0; const bv = b[sortKey.value] ?? 0
    const cmp = typeof av === 'number' && typeof bv === 'number' ? av - bv : String(av).localeCompare(String(bv))
    return sortDir.value === 'asc' ? cmp : -cmp
  })
})

const visibleRows = computed(() => sorted.value.slice(0, MAX_ROWS))

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sortKey.value = dir ? key : ''
  sortDir.value = dir
}

function colLabel(key: string): string {
  const map: Record<string, string> = {
    name: '名称', name_of_issuer: '发行人', title_of_class: '证券类型',
    cusip: 'CUSIP', ticker: '代码', symbol: '代码',
    value: '市值', market_value: '市值', val: '市值',
    shares: '股数', principal_amount: '本金',
    put_call: '期权类型', investment_discretion: '决策权',
    voting_authority_sole: '独投', voting_authority_shared: '共投', voting_authority_none: '无投',
    weight_pct: '权重%', change: '变动', qty: '数量',
    filed_at: '申报日', period: '报告期', date: '日期',
  }
  return map[key] ?? key.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function fmtVal(v: any): string {
  if (v == null) return '-'
  if (typeof v === 'number') {
    if (Math.abs(v) >= 1e8) return (v / 1e8).toFixed(2) + '亿'
    if (Math.abs(v) >= 1e4) return (v / 1e4).toFixed(1) + '万'
    return v.toLocaleString()
  }
  return String(v)
}

const cols = computed<Column[]>(() => {
  if (holdings.value.length === 0) return []
  const first = holdings.value[0]
  return Object.keys(first)
    .filter(k => typeof first[k] !== 'object' || first[k] === null)
    .map(k => {
      const numeric = typeof first[k] === 'number'
      return {
        key: k,
        label: colLabel(k),
        align: numeric ? 'right' as const : 'left' as const,
        mono: numeric,
        sortable: true,
        ...(numeric ? { formatter: fmtVal } : {}),
      }
    })
})

async function loadData() {
  loading.value = true; error.value = ''
  try {
    if (app?.FetchData) {
      const { data: result } = await fetchWithCache<any>('sec_13f', () => app.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 5 * 60 * 1000)
      if (result?.data) rawData.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

onMounted(loadData)
</script>

<template>
  <div class="sec-13f-panel">
    <PanelHeader
      title="13F 机构持仓"
      :controls="[{ icon: 'refresh', title: '刷新', action: loadData, loading }]"
    />
    <LoadingState v-if="loading && holdings.length === 0" type="table" :rows="6" />
    <ErrorState v-else-if="error" :description="error" @retry="loadData" />
    <EmptyState
      v-else-if="holdings.length === 0"
      title="暂无 13F 数据"
      description="输入机构 CIK 代码查看 SEC 13F 持仓报告"
    />
    <template v-else>
      <PanelTable
        :columns="cols"
        :data="visibleRows"
        :loading="loading"
        :sort-key="sortKey"
        :sort-dir="sortDir"
        sticky-header
        @sort-change="onSortChange"
      />
      <div v-if="holdings.length > MAX_ROWS" class="table-footer">显示前 {{ MAX_ROWS }} 条，共 {{ holdings.length }} 条</div>
    </template>
  </div>
</template>

<style scoped>
.sec-13f-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

.table-footer {
  padding: var(--space-sm);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-align: center;
  flex-shrink: 0;
  border-top: 1px solid var(--color-border-subtle);
}
</style>
