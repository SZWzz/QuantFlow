<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'

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

const headerTabs = computed(() => [
  { key: 'bull', label: t('misc.bull_cbbc') },
  { key: 'bear', label: t('misc.bear_cbbc') },
  { key: 'warrants', label: t('misc.warrants_tab') },
])

const hasData = computed(() => {
  if (!rawData.value) return false
  const c = rawData.value.cbbc?.data?.length ?? 0
  const w = rawData.value.warrants?.data?.length ?? 0
  return c + w > 0
})

const bullList = computed(() => rawData.value?.cbbc?.data?.filter((item: any) => item.类型 === 'bull' || item.type === 'bull') ?? [])
const bearList = computed(() => rawData.value?.cbbc?.data?.filter((item: any) => item.类型 === 'bear' || item.type === 'bear') ?? [])
const warrantsList = computed(() => rawData.value?.warrants?.data ?? [])

/** 源数据中英文键混用，统一映射为 PanelTable 可用的规范键 */
function normalize(item: DerivativeItem) {
  return {
    code: item.代码 || item.code || '--',
    name: item.名称 || item.name || '--',
    strike: item.行使价 || item.strike,
    expiry: item.到期日 || item.expiry || '--',
    ratio: item.换股比率 || item.convert_ratio,
    premium: item.溢价率 || item.premium_ratio || item['溢价率(%)'],
    leverage: item.杠杆比率 || item.leverage_ratio,
    outstanding: item.街货量 || item.outstanding_ratio || item['街货量(%)'],
    callprice: item.收回价 || item.call_price,
  }
}

const currentList = computed(() => {
  const list = activeTab.value === 'bull' ? bullList.value : activeTab.value === 'bear' ? bearList.value : warrantsList.value
  return list.map(normalize)
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

function onRowClick(row: any) {
  if (row.code && row.code !== '--') ctx.setGroupSymbol(pg.groupId, row.code)
}

function switchTab(tab: string) {
  if (tab !== 'bull' && tab !== 'bear' && tab !== 'warrants') return
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

const cols = computed<Column[]>(() => [
  { key: 'code', label: t('common.symbol'), mono: true, cellClass: () => 'code-cell' },
  { key: 'name', label: t('common.name') },
  { key: 'strike', label: t('misc.strike_price'), align: 'right', formatter: formatVal },
  { key: 'expiry', label: t('misc.expiry_date') },
  { key: 'ratio', label: t('misc.convert_ratio'), align: 'right', formatter: formatVal },
  { key: 'premium', label: t('misc.premium_ratio'), align: 'right', formatter: formatPct },
  { key: 'leverage', label: t('misc.leverage_ratio'), align: 'right', formatter: formatVal },
  { key: 'outstanding', label: t('misc.outstanding_ratio'), align: 'right', formatter: formatPct },
  { key: 'callprice', label: t('misc.call_price'), align: 'right', formatter: formatVal },
])

onMounted(() => fetchData())
</script>

<template>
  <div class="hk-derivatives-panel">
    <PanelHeader
      :title="$t('misc.hk_derivatives')"
      :tabs="headerTabs"
      :active-tab="activeTab"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
      @tab-change="switchTab"
    >
      <template #controls>
        <span class="count-badge">{{ currentList.length }}</span>
      </template>
    </PanelHeader>

    <ErrorState v-if="error && !loading && !hasData" :description="error" @retry="fetchData" />

    <LoadingState v-else-if="loading && !hasData" type="table" :rows="6" :cols="cols.length" />

    <EmptyState v-else-if="currentList.length === 0" :title="$t('misc.no_hk_derivatives')" />

    <PanelTable
      v-else
      :columns="cols"
      :data="currentList"
      :loading="loading"
      :row-key="(row: any, idx: number) => row.code + '-' + idx"
      clickable
      sticky-header
      @row-click="onRowClick"
    />
  </div>
</template>

<style scoped>
.hk-derivatives-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.count-badge {
  font-size: var(--font-xs); font-weight: 600; padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-lg);
  color: var(--color-accent); background: var(--color-accent-soft);
}

:deep(.td.code-cell) { color: var(--color-accent); }
</style>
