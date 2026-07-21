<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

const { t } = useI18n()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const app = (window as any).go?.main?.App
const { fetchWithCache } = usePanelCache()

type TabKey = 'arbitrage' | 'redeem' | 'put'

const loading = ref(false)
const error = ref('')
const data = ref<any>(null)
const activeTab = ref<TabKey>('arbitrage')

const tabs = computed(() => [
  { key: 'arbitrage', label: t('panels.arbitrage_opp') },
  { key: 'redeem', label: t('panels.redeem_warn') },
  { key: 'put', label: t('panels.put_opp') },
])

const columnDefs: Record<TabKey, Column[]> = {
  arbitrage: [
    { key: 'bond_code', label: '转债代码', mono: true },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码', mono: true },
    { key: 'stock_price', label: '正股价', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'convert_price', label: '转股价', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'convert_value', label: '转股价值', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'premium_ratio', label: '溢价率%', align: 'right', formatter: fmtPct, cellClass: premiumCellClass },
    { key: 'ytm_ratio', label: '税前收益率', align: 'right', formatter: fmtPct },
    { key: 'price', label: '转债价格', align: 'right', formatter: (v: any) => fmt(v) },
  ],
  redeem: [
    { key: 'bond_code', label: '转债代码', mono: true },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码', mono: true },
    { key: 'stock_price', label: '正股价', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'redeem_cond', label: '强赎条件' },
    { key: 'redeem_price', label: '强赎触发价', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'premium_ratio', label: '溢价率%', align: 'right', formatter: fmtPct, cellClass: premiumCellClass },
  ],
  put: [
    { key: 'bond_code', label: '转债代码', mono: true },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码', mono: true },
    { key: 'stock_price', label: '正股价', align: 'right', formatter: (v: any) => fmt(v) },
    { key: 'put_cond', label: '回售条件' },
    { key: 'premium_ratio', label: '溢价率%', align: 'right', formatter: fmtPct, cellClass: premiumCellClass },
    { key: 'price', label: '转债价格', align: 'right', formatter: (v: any) => fmt(v) },
  ],
}

/** 源数据数值可能是字符串，转成 number 供 PanelTable 的 format/colorize 使用 */
function toNum(v: any): any {
  if (v == null || v === '') return v
  const n = typeof v === 'number' ? v : parseFloat(v)
  return Number.isFinite(n) ? n : v
}

const NUM_KEYS = ['stock_price', 'convert_price', 'convert_value', 'premium_ratio', 'ytm_ratio', 'price', 'redeem_price']
function normalize(rows: any[]): any[] {
  return rows.map((r: any) => {
    const row: any = { ...r }
    for (const k of NUM_KEYS) row[k] = toNum(r[k])
    return row
  })
}

async function fetchData() {
  if (!app?.GetCBArbitrageData) return
  loading.value = true
  error.value = ''
  try {
    const { data: result } = await fetchWithCache<any>('cb_arbitrage', () => app.GetCBArbitrageData(), 15 * 60 * 1000)
    data.value = result
  } catch (e: any) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
}

const arbitrageBonds = computed(() => {
  const bonds = data.value?.bonds?.data
  if (!Array.isArray(bonds)) return []
  return normalize(bonds).sort((a: any, b: any) => (a.premium_ratio ?? Infinity) - (b.premium_ratio ?? Infinity))
})

const redeemBonds = computed(() => {
  const r = data.value?.redeem?.data
  if (!Array.isArray(r)) return []
  return normalize(r)
})

const putBonds = computed(() => {
  const bonds = data.value?.bonds?.data
  if (!Array.isArray(bonds)) return []
  return normalize(bonds.filter((b: any) => b.put_cond && String(b.put_cond).trim() !== ''))
})

const currentCols = computed<Column[]>(() => columnDefs[activeTab.value])
const currentRows = computed<any[]>(() => {
  if (activeTab.value === 'redeem') return redeemBonds.value
  if (activeTab.value === 'put') return putBonds.value
  return arbitrageBonds.value
})

const showPythonRequired = computed(() => {
  return error.value && error.value.includes('Python sidecar not available')
})

function onRowClick(row: any) {
  if (row?.stock_code) ctx.setGroupSymbol(pg.groupId, row.stock_code)
}

function fmt(v: any, decimals = 2): string {
  if (v == null || v === '') return '-'
  const n = typeof v === 'number' ? v : parseFloat(v)
  if (isNaN(n)) return String(v)
  return n.toFixed(decimals)
}

function fmtPct(v: any): string {
  if (v == null || v === '') return '-'
  const n = typeof v === 'number' ? v : parseFloat(v)
  if (isNaN(n)) return String(v)
  return (n >= 0 ? '+' : '') + n.toFixed(2) + '%'
}

/** 溢价率三段式着色（恢复迁移前 premiumColor 语义）：<0 折价=机会；>50 极端预警；中间不着色 */
function premiumCellClass(row: any): string {
  const n = typeof row.premium_ratio === 'number' ? row.premium_ratio : parseFloat(row.premium_ratio)
  if (!Number.isFinite(n)) return ''
  if (n < 0) return 'cell-neg'
  if (n > 50) return 'cell-warn'
  return ''
}

function onTabChange(key: string) {
  activeTab.value = key as TabKey
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="cb-arbitrage-panel">
    <PanelHeader
      :title="t('panels.cb_arbitrage')"
      :tabs="tabs"
      :active-tab="activeTab"
      :controls="[{ icon: 'refresh', title: '刷新', action: fetchData, loading }]"
      @tab-change="onTabChange"
    />

    <LoadingState v-if="loading && !data" type="table" :rows="8" :cols="currentCols.length" />
    <EmptyState v-else-if="showPythonRequired" :title="t('panels.python_required')" />
    <ErrorState v-else-if="error" :description="error" @retry="fetchData" />
    <EmptyState v-else-if="currentRows.length === 0" :title="t('panels.no_data')" />
    <PanelTable
      v-else
      :columns="currentCols"
      :data="currentRows"
      :loading="loading"
      :row-key="(r: any) => r.bond_code"
      clickable
      sticky-header
      @row-click="onRowClick"
    />
  </div>
</template>

<style scoped>
.cb-arbitrage-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* PanelTable cellClass 命中在子组件 td 上，需 :deep 穿透 */
:deep(.td.cell-neg) { color: var(--color-down); font-weight: 500; }
:deep(.td.cell-warn) { color: var(--color-danger); font-weight: 600; }
</style>
