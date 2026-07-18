<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const rawData = ref<any>(null)
const sortKey = ref('')
const sortAsc = ref(true)

const SOURCE = 'sec'
const DATA_TYPE = '13f'

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
  if (!sortKey.value) return holdings.value
  return [...holdings.value].sort((a, b) => {
    const av = a[sortKey.value] ?? 0; const bv = b[sortKey.value] ?? 0
    const cmp = typeof av === 'number' && typeof bv === 'number' ? av - bv : String(av).localeCompare(String(bv))
    return sortAsc.value ? cmp : -cmp
  })
})

function toggleSort(key: string) {
  if (sortKey.value === key) { sortAsc.value = !sortAsc.value; return }
  sortKey.value = key; sortAsc.value = false
}

function colKeys(): string[] {
  if (holdings.value.length === 0) return []
  return Object.keys(holdings.value[0]).filter(k => typeof holdings.value[0][k] !== 'object' || holdings.value[0][k] === null)
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

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
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
    <div class="panel-header">
      <h3>13F 机构持仓</h3>
      <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
    </div>
    <SkeletonPanel v-if="loading && holdings.length === 0" type="table" :rows="6" />
    <div v-else-if="error" class="status error">{{ error }}</div>
    <div v-else-if="!loading && holdings.length === 0" class="status">暂无 13F 数据 — 输入机构 CIK 代码查看 SEC 13F 持仓报告</div>
    <div v-else class="table-wrapper">
      <div class="table-header">
        <span v-for="key in colKeys()" :key="key" class="th-cell" @click="toggleSort(key)">
          {{ colLabel(key) }}
          <span v-if="sortKey === key" class="sort-arrow">{{ sortAsc ? '▲' : '▼' }}</span>
        </span>
      </div>
      <div class="table-body">
        <div v-for="(row, i) in sorted.slice(0, 100)" :key="i" class="table-row">
          <span v-for="key in colKeys()" :key="key" class="td-cell">{{ fmtVal(row[key]) }}</span>
        </div>
      </div>
      <div v-if="holdings.length > 100" class="table-footer">显示前 100 条，共 {{ holdings.length }} 条</div>
    </div>
  </div>
</template>

<style scoped>
.sec-13f-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden; }

.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-danger); }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: 6px 0; border-bottom: 2px solid var(--color-border-strong); font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; overflow-x: auto; }
.th-cell { flex: 1; min-width: 80px; padding: 0 6px; cursor: pointer; user-select: none; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.th-cell:hover { color: var(--color-accent); }
.sort-arrow { font-size: 8px; margin-left: 2px; }
.table-body { flex: 1; overflow: auto; font-size: 12px; }
.table-row { display: flex; padding: 3px 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-elevated); }
.td-cell { flex: 1; min-width: 80px; padding: 0 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-variant-numeric: tabular-nums; }
.table-footer { padding: 6px; font-size: 10px; color: var(--color-text-tertiary); text-align: center; flex-shrink: 0; }
</style>
