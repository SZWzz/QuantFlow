<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { t } = useI18n()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()

type TabKey = 'bull' | 'bear' | 'warrants'

interface DerivativeItem {
  [key: string]: any
}

interface HKDerivativesResult {
  cbbc?: { data?: DerivativeItem[] }
  warrants?: { data?: DerivativeItem[] }
}

const activeTab = ref<TabKey>('bull')
const loading = ref(false)
const error = ref<string | null>(null)
const rawData = ref<HKDerivativesResult | null>(null)

const hasData = computed(() => {
  if (!rawData.value) return false
  const c = rawData.value.cbbc?.data?.length ?? 0
  const w = rawData.value.warrants?.data?.length ?? 0
  return c + w > 0
})

const bullList = computed(() => rawData.value?.cbbc?.data?.filter((item: any) => item.类型 === 'bull' || item.type === 'bull') ?? [])
const bearList = computed(() => rawData.value?.cbbc?.data?.filter((item: any) => item.类型 === 'bear' || item.type === 'bear') ?? [])
const warrantsList = computed(() => rawData.value?.warrants?.data ?? [])

const currentList = computed(() => {
  if (activeTab.value === 'bull') return bullList.value
  if (activeTab.value === 'bear') return bearList.value
  return warrantsList.value
})

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetHKDerivatives) return
  loading.value = true
  error.value = null
  try {
    const { data: result } = await fetchWithCache<any>('hk_derivatives', () => app.GetHKDerivatives(), 15 * 60 * 1000)
    rawData.value = result as HKDerivativesResult
  } catch (e) {
    console.error('[HKDerivatives]', e)
    error.value = String(e)
    rawData.value = null
  } finally {
    loading.value = false
  }
}

function onCodeClick(row: any) {
  const code = row.代码 || row.code
  if (code) ctx.setGroupSymbol(pg.groupId, code)
}

function switchTab(tab: TabKey) {
  activeTab.value = tab
}

function formatVal(v: any): string {
  if (v === null || v === undefined) return '--'
  if (typeof v === 'number') return v.toFixed(3)
  return String(v)
}

function formatPct(v: any): string {
  if (v === null || v === undefined) return '--'
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (isNaN(n)) return String(v)
  return n.toFixed(2) + '%'
}

onMounted(() => fetchData())
</script>

<template>
  <div class="hk-derivatives-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.hk_derivatives') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'bull' }]" @click="switchTab('bull')">{{ $t('misc.bull_cbbc') }}</button>
        <button :class="['tab', { active: activeTab === 'bear' }]" @click="switchTab('bear')">{{ $t('misc.bear_cbbc') }}</button>
        <button :class="['tab', { active: activeTab === 'warrants' }]" @click="switchTab('warrants')">{{ $t('misc.warrants_tab') }}</button>
      </div>
      <div class="header-controls">
        <span class="count-badge">{{ currentList.length }}</span>
        <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
      </div>
    </div>

    <template v-if="error && !loading && !hasData">
      <div class="error-state">
        <span class="error-text">{{ $t('common.panel_error') }}: {{ error }}</span>
        <button class="retry-btn" @click="fetchData">{{ $t('common.retry') }}</button>
      </div>
    </template>

    <SkeletonPanel v-else-if="loading && !hasData" type="table" :rows="6" />

    <div v-else-if="currentList.length === 0" class="empty-state">{{ $t('misc.no_hk_derivatives') }}</div>

    <div v-else class="table-wrapper">
      <div class="table-header">
        <span class="col-code">{{ $t('common.symbol') }}</span>
        <span class="col-name">{{ $t('common.name') }}</span>
        <span class="col-strike">{{ $t('misc.strike_price') }}</span>
        <span class="col-expiry">{{ $t('misc.expiry_date') }}</span>
        <span class="col-ratio">{{ $t('misc.convert_ratio') }}</span>
        <span class="col-premium">{{ $t('misc.premium_ratio') }}</span>
        <span class="col-leverage">{{ $t('misc.leverage_ratio') }}</span>
        <span class="col-outstanding">{{ $t('misc.outstanding_ratio') }}</span>
        <span class="col-callprice">{{ $t('misc.call_price') }}</span>
      </div>
      <div class="table-body">
        <div v-for="(row, idx) in currentList" :key="(row.代码 || row.code) + '-' + idx" class="table-row">
          <span class="col-code clickable" @click="onCodeClick(row)">{{ row.代码 || row.code || '--' }}</span>
          <span class="col-name">{{ row.名称 || row.name || '--' }}</span>
          <span class="col-strike">{{ formatVal(row.行使价 || row.strike) }}</span>
          <span class="col-expiry">{{ row.到期日 || row.expiry || '--' }}</span>
          <span class="col-ratio">{{ formatVal(row.换股比率 || row.convert_ratio) }}</span>
          <span class="col-premium">{{ formatPct(row.溢价率 || row.premium_ratio || row['溢价率(%)']) }}</span>
          <span class="col-leverage">{{ formatVal(row.杠杆比率 || row.leverage_ratio) }}</span>
          <span class="col-outstanding">{{ formatPct(row.街货量 || row.outstanding_ratio || row['街货量(%)']) }}</span>
          <span class="col-callprice">{{ formatVal(row.收回价 || row.call_price) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hk-derivatives-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; white-space: nowrap; }
.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px; white-space: nowrap;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.header-controls { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.count-badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-lg);
  color: var(--color-accent); background: rgba(59,130,246,0.1);
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
}
.error-text { color: var(--color-up); font-size: 12px; }
.retry-btn {
  padding: 4px 14px; border: 1px solid var(--color-up); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-up); cursor: pointer; font-size: 11px;
}
.retry-btn:hover { background: rgba(248,113,113,0.1); }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}
.table-wrapper { flex: 1; overflow: auto; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; min-width: fit-content;
}
.table-body { font-size: 11px; min-width: fit-content; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-code { width: 72px; flex-shrink: 0; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 80px; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-strike { width: 70px; flex-shrink: 0; text-align: right; }
.col-expiry { width: 80px; flex-shrink: 0; }
.col-ratio { width: 64px; flex-shrink: 0; text-align: right; }
.col-premium { width: 64px; flex-shrink: 0; text-align: right; }
.col-leverage { width: 64px; flex-shrink: 0; text-align: right; }
.col-outstanding { width: 64px; flex-shrink: 0; text-align: right; }
.col-callprice { width: 70px; flex-shrink: 0; text-align: right; }
</style>
