<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const { t } = useI18n()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const app = (window as any).go?.main?.App
const { fetchWithCache } = usePanelCache()

const loading = ref(false)
const error = ref('')
const data = ref<any>(null)
const activeTab = ref<'arbitrage' | 'redeem' | 'put'>('arbitrage')

const columnDefs: Record<string, { key: string; label: string; align?: string }[]> = {
  arbitrage: [
    { key: 'bond_code', label: '转债代码' },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码' },
    { key: 'stock_price', label: '正股价', align: 'right' },
    { key: 'convert_price', label: '转股价', align: 'right' },
    { key: 'convert_value', label: '转股价值', align: 'right' },
    { key: 'premium_ratio', label: '溢价率%', align: 'right' },
    { key: 'ytm_ratio', label: '税前收益率', align: 'right' },
    { key: 'price', label: '转债价格', align: 'right' },
  ],
  redeem: [
    { key: 'bond_code', label: '转债代码' },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码' },
    { key: 'stock_price', label: '正股价', align: 'right' },
    { key: 'redeem_cond', label: '强赎条件' },
    { key: 'redeem_price', label: '强赎触发价', align: 'right' },
    { key: 'premium_ratio', label: '溢价率%', align: 'right' },
  ],
  put: [
    { key: 'bond_code', label: '转债代码' },
    { key: 'bond_nm', label: '转债名称' },
    { key: 'stock_code', label: '正股代码' },
    { key: 'stock_price', label: '正股价', align: 'right' },
    { key: 'put_cond', label: '回售条件' },
    { key: 'premium_ratio', label: '溢价率%', align: 'right' },
    { key: 'price', label: '转债价格', align: 'right' },
  ],
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
  return [...bonds].sort((a: any, b: any) => (a.premium_ratio ?? Infinity) - (b.premium_ratio ?? Infinity))
})

const redeemBonds = computed(() => {
  const r = data.value?.redeem?.data
  if (!Array.isArray(r)) return []
  return r
})

const putBonds = computed(() => {
  const bonds = data.value?.bonds?.data
  if (!Array.isArray(bonds)) return []
  return bonds.filter((b: any) => b.put_cond && String(b.put_cond).trim() !== '')
})

const showPythonRequired = computed(() => {
  return error.value && error.value.includes('Python sidecar not available')
})

function onStockClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
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

function premiumColor(v: any): string {
  const n = parseFloat(v)
  if (n < 0) return '#22c55e'
  if (n > 50) return '#ef4444'
  return 'inherit'
}

function switchTab(tab: 'arbitrage' | 'redeem' | 'put') {
  activeTab.value = tab
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="cb-arbitrage-panel">
    <div class="panel-header">
      <h3>{{ t('panels.cb_arbitrage') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'arbitrage' }]" @click="switchTab('arbitrage')">{{ t('panels.arbitrage_opp') }}</button>
        <button :class="['tab', { active: activeTab === 'redeem' }]" @click="switchTab('redeem')">{{ t('panels.redeem_warn') }}</button>
        <button :class="['tab', { active: activeTab === 'put' }]" @click="switchTab('put')">{{ t('panels.put_opp') }}</button>
      </div>
      <div class="header-controls">
        <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading && !data" type="table" :rows="8" />

    <div v-else-if="showPythonRequired" class="empty-state">{{ t('panels.python_required') }}</div>
    <div v-else-if="error && !showPythonRequired" class="empty-state error">{{ error }}</div>

    <template v-else-if="activeTab === 'arbitrage'">
      <div v-if="arbitrageBonds.length === 0" class="empty-state">{{ t('panels.no_data') }}</div>
      <div v-else class="table-wrapper">
        <div class="table-header">
          <span v-for="col in columnDefs.arbitrage" :key="col.key" :class="['col', col.key, { 'col-right': col.align === 'right' }]">{{ col.label }}</span>
        </div>
        <div class="table-body">
          <div v-for="row in arbitrageBonds" :key="row.bond_code" class="table-row">
            <span class="col bond_code">{{ row.bond_code }}</span>
            <span class="col bond_nm">{{ row.bond_nm }}</span>
            <span class="col stock_code clickable" @click="onStockClick(row.stock_code)">{{ row.stock_code }}</span>
            <span class="col stock_price col-right">{{ fmt(row.stock_price) }}</span>
            <span class="col convert_price col-right">{{ fmt(row.convert_price) }}</span>
            <span class="col convert_value col-right">{{ fmt(row.convert_value) }}</span>
            <span class="col premium_ratio col-right" :style="{ color: premiumColor(row.premium_ratio) }">{{ fmtPct(row.premium_ratio) }}</span>
            <span class="col ytm_ratio col-right">{{ fmtPct(row.ytm_ratio) }}</span>
            <span class="col price col-right">{{ fmt(row.price) }}</span>
          </div>
        </div>
      </div>
    </template>

    <template v-else-if="activeTab === 'redeem'">
      <div v-if="redeemBonds.length === 0" class="empty-state">{{ t('panels.no_data') }}</div>
      <div v-else class="table-wrapper">
        <div class="table-header">
          <span v-for="col in columnDefs.redeem" :key="col.key" :class="['col', col.key, { 'col-right': col.align === 'right' }]">{{ col.label }}</span>
        </div>
        <div class="table-body">
          <div v-for="row in redeemBonds" :key="row.bond_code" class="table-row">
            <span class="col bond_code">{{ row.bond_code }}</span>
            <span class="col bond_nm">{{ row.bond_nm }}</span>
            <span class="col stock_code clickable" @click="onStockClick(row.stock_code)">{{ row.stock_code }}</span>
            <span class="col stock_price col-right">{{ fmt(row.stock_price) }}</span>
            <span class="col redeem_cond">{{ row.redeem_cond }}</span>
            <span class="col redeem_price col-right">{{ fmt(row.redeem_price) }}</span>
            <span class="col premium_ratio col-right" :style="{ color: premiumColor(row.premium_ratio) }">{{ fmtPct(row.premium_ratio) }}</span>
          </div>
        </div>
      </div>
    </template>

    <template v-else>
      <div v-if="putBonds.length === 0" class="empty-state">{{ t('panels.no_data') }}</div>
      <div v-else class="table-wrapper">
        <div class="table-header">
          <span v-for="col in columnDefs.put" :key="col.key" :class="['col', col.key, { 'col-right': col.align === 'right' }]">{{ col.label }}</span>
        </div>
        <div class="table-body">
          <div v-for="row in putBonds" :key="row.bond_code" class="table-row">
            <span class="col bond_code">{{ row.bond_code }}</span>
            <span class="col bond_nm">{{ row.bond_nm }}</span>
            <span class="col stock_code clickable" @click="onStockClick(row.stock_code)">{{ row.stock_code }}</span>
            <span class="col stock_price col-right">{{ fmt(row.stock_price) }}</span>
            <span class="col put_cond">{{ row.put_cond }}</span>
            <span class="col premium_ratio col-right" :style="{ color: premiumColor(row.premium_ratio) }">{{ fmtPct(row.premium_ratio) }}</span>
            <span class="col price col-right">{{ fmt(row.price) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.cb-arbitrage-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.header-controls { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.empty-state.error { color: var(--color-error, var(--color-up)); }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex-shrink: 0; }
.col-right { text-align: right; }
.clickable { cursor: pointer; color: var(--color-accent); }
.bond_code { width: 76px; }
.bond_nm { width: 80px; }
.stock_code { width: 68px; }
.stock_price { width: 58px; }
.convert_price { width: 58px; }
.convert_value { width: 68px; }
.premium_ratio { width: 66px; font-weight: 500; }
.ytm_ratio { width: 68px; }
.price { width: 58px; }
.redeem_cond { width: 80px; }
.redeem_price { width: 74px; }
.put_cond { width: 80px; }
</style>
