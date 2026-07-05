<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader, PanelCard, PanelTable, LoadingState } from '@/terminal/components/panel'
import type { IndexSnapshot, SectorRanking } from '@/stores/data'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()

const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const headerControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ label: autoRefresh.value ? `自动 (${countdown.value}s)` : '手动', action: toggleAutoRefresh, title: '切换自动刷新' })
  list.push({ icon: 'refresh', action: refresh, loading: loading.value, title: '刷新' })
  return list
})
const autoRefresh = ref(true)
const loadError = ref('')
const refreshInterval = ref(15)
const countdown = ref(refreshInterval.value)
let timer: ReturnType<typeof setInterval> | null = null

// Block rank data
interface BlockRankItem {
  symbol: string
  name: string
  price: number
  volume: number
  amount: number
}
const blockRank = ref<BlockRankItem[]>([])
const blockRankLoading = ref(false)

const indices = computed(() => dataStore.marketOverview?.indices ?? [])
const breadth = computed(() => dataStore.marketOverview?.breadth ?? { advancers: 0, decliners: 0, unchanged: 0 })
const sectors = computed(() => dataStore.marketOverview?.sectors ?? [])
const updatedAt = computed(() => dataStore.marketOverview?.updatedAt ?? 0)
const loading = computed(() => dataStore.marketLoading)

const topGainers = computed(() => [...sectors.value].sort((a, b) => b.changePct - a.changePct).slice(0, 8))
const topLosers = computed(() => [...sectors.value].sort((a, b) => a.changePct - b.changePct).slice(0, 8))

function refresh() {
  dataStore.fetchMarketOverview(activeMarket.value)
  countdown.value = refreshInterval.value
  fetchBlockRank()
}

async function fetchBlockRank() {
  const app = (window as any).go?.main?.App
  if (!app) return
  blockRankLoading.value = true
  loadError.value = ''
  try {
    const { data: result } = await fetchWithCache('block_rank', () => app.GetBlockRank(1, 0, 10), 5 * 60 * 1000)
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    blockRank.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      volume: i.volume || 0,
      amount: i.amount || 0,
    }))
  } catch(e: any) {
    console.warn('[MarketOverview] block rank unavailable:', e?.message || e)
    blockRank.value = []
  } finally {
    blockRankLoading.value = false
  }
}
function switchMarket(mkt: typeof activeMarket.value) {
  activeMarket.value = mkt
  refresh()
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
  if (autoRefresh.value) {
    countdown.value = refreshInterval.value
  }
}

function formatTime(ts: number): string {
  if (!ts) return '--'
  return new Date(ts).toLocaleTimeString()
}

function changeColor(pct: number): string {
  if (pct > 0) return 'var(--color-up)'
  if (pct < 0) return 'var(--color-down)'
  return 'var(--color-text-secondary)'
}

function formatPct(pct: number): string {
  return (pct >= 0 ? '+' : '') + pct.toFixed(2) + '%'
}

function formatVolume(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return String(v)
}
function onIndexClick(idx: { symbol: string; name: string }) {
  // Strip .SH / .SZ suffix for upstream index API (expects bare code like 000300)
  const code = idx.symbol.replace(/\.(SH|SZ|SS|CSI)$/i, '')
  ctx.setGroupSymbol(pg.groupId, code)
}

function formatAmount(a: number): string {
  if (a >= 1e8) return (a / 1e8).toFixed(2) + '亿'
  if (a >= 1e4) return (a / 1e4).toFixed(1) + '万'
  return a.toFixed(0)
}

onMounted(() => {
  refresh()
  timer = setInterval(() => {
    if (autoRefresh.value) {
      if (countdown.value <= 1) {
        refresh()
      } else {
        countdown.value--
      }
    }
  }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const blockRankColumns = [
  { key: 'symbol', label: '代码', align: 'left' as const, width: 70 },
  { key: 'name', label: '名称', align: 'left' as const, flex: 1 },
  { key: 'price', label: '价格', align: 'right' as const, width: 70, format: 'price' as const },
  { key: 'volume', label: '成交量', align: 'right' as const, width: 70, format: 'volume' as const },
  { key: 'amount', label: '成交额', align: 'right' as const, width: 80, formatter: (v: number) => formatAmount(v) },
]
</script>

<template>
  <div class="market-overview-panel">
    <PanelHeader
      :title="$t('misc.market_overview')"
      :subtitle="formatTime(updatedAt)"
      :tabs="[
        { key: 'CN', label: 'CN' },
        { key: 'HK', label: 'HK' },
        { key: 'US', label: 'US' },
      ]"
      :active-tab="activeMarket"
      :controls="headerControls"
      @tab-change="switchMarket"
    />

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <!-- Section A: Index Cards -->
    <div class="indices-row">
      <PanelCard
        v-for="idx in indices"
        :key="idx.symbol"
        :title="idx.name"
        :value="idx.last"
        :change="idx.changePct / 100"
        format="price"
        :sparkline="idx.sparkline"
        :ohlcv="idx.ohlcv"
        clickable
        @click="onIndexClick(idx)"
      />
    </div>

    <!-- Section B: 市场宽度 -->
    <div class="breadth-section">
      <div class="breadth-label">{{ $t('misc.market_breadth') }}</div>
      <div class="breadth-bar">
        <div class="breadth-segment up" :style="{ flex: breadth.advancers }"></div>
        <div class="breadth-segment flat" :style="{ flex: breadth.unchanged }"></div>
        <div class="breadth-segment down" :style="{ flex: breadth.decliners }"></div>
      </div>
      <div class="breadth-text">
        <span class="up-text">涨 {{ breadth.advancers }}</span>
        <span class="flat-text">平 {{ breadth.unchanged }}</span>
        <span class="down-text">跌 {{ breadth.decliners }}</span>
      </div>
    </div>

    <!-- Section C: Sector Rankings -->
    <div class="sectors-grid">
      <div class="sector-col">
        <div class="sector-col-title up-text">{{ $t('misc.gainers') }}</div>
        <div v-for="s in topGainers" :key="'g-' + s.name" class="sector-row">
          <span class="sector-name">{{ s.name }}</span>
          <span class="sector-pct" :style="{ color: changeColor(s.changePct) }">{{ formatPct(s.changePct) }}</span>
        </div>
      </div>
      <div class="sector-col">
        <div class="sector-col-title down-text">{{ $t('misc.losers') }}</div>
        <div v-for="s in topLosers" :key="'l-' + s.name" class="sector-row">
          <span class="sector-name">{{ s.name }}</span>
          <span class="sector-pct" :style="{ color: changeColor(s.changePct) }">{{ formatPct(s.changePct) }}</span>
        </div>
      </div>
    </div>

    <!-- Section D: Block Rank -->
    <div class="block-rank-section">
      <div class="block-rank-label">{{ $t('misc.block_rank') }}</div>
      <LoadingState
        v-if="blockRankLoading && blockRank.length === 0"
        type="table"
        :rows="3"
        :cols="5"
      />
      <div v-else-if="blockRank.length === 0" class="block-empty">{{ $t('common.no_data') }}</div>
      <PanelTable
        v-else
        :columns="blockRankColumns"
        :data="blockRank"
        :striped="true"
      />
    </div>

    <!-- Skeleton: shown only on initial load (no data yet) -->
    <div v-if="loading && !dataStore.marketOverview" class="skeleton-overlay">
      <LoadingState type="card" :rows="5" />
      <div class="skeleton-breadth">
        <div class="skeleton-bar" />
        <div class="skeleton-bar short" />
      </div>
      <LoadingState type="table" :rows="8" :cols="2" />
    </div>
  </div>
</template>

<style scoped>
.market-overview-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  position: relative;
  color: var(--color-text-primary);
  background: var(--color-bg-panel);
}

/* Index Cards */
.indices-row {
  display: flex;
  gap: var(--space-sm);
  overflow-x: auto;
  padding: var(--panel-padding) var(--panel-padding) 0;
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-strong) transparent;
}

.indices-row :deep(.panel-card) {
  flex: 0 0 auto;
  min-width: 130px;
}

/* Breadth */
.breadth-section {
  padding: 0 var(--panel-padding);
  margin-bottom: var(--space-md);
}

.breadth-label {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  margin-bottom: var(--space-sm);
}

.breadth-bar {
  display: flex;
  height: 8px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  margin-bottom: var(--space-xs);
}

.breadth-segment.up {
  background: var(--color-up);
}

.breadth-segment.down {
  background: var(--color-down);
}

.breadth-segment.flat {
  background: var(--color-text-tertiary);
}

.breadth-text {
  display: flex;
  gap: var(--space-lg);
  font-size: var(--font-xs);
}

.up-text {
  color: var(--color-up);
}

.down-text {
  color: var(--color-down);
}

.flat-text {
  color: var(--color-text-tertiary);
}

/* Sectors */
.sectors-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
  flex: 1;
  overflow: hidden;
  padding: 0 var(--panel-padding);
}

.sector-col {
  overflow-y: auto;
}

.sector-col-title {
  font-size: var(--font-sm);
  font-weight: 600;
  margin-bottom: var(--space-sm);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-strong);
}

.sector-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-xs) 0;
  font-size: var(--font-sm);
}

.sector-name {
  color: var(--color-text-primary);
}

.sector-pct {
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

/* Skeleton loading */
.skeleton-overlay {
  position: absolute;
  inset: 0;
  padding: var(--panel-padding);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background: var(--color-bg-panel);
  z-index: 10;
}

.skeleton-breadth {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.skeleton-bar {
  height: 12px;
  border-radius: var(--radius-sm);
  width: 60%;
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s ease-in-out infinite;
}

.skeleton-bar.short {
  width: 40%;
}

@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}

/* Block Rank */
.block-rank-section {
  margin-top: var(--space-md);
  padding: var(--space-md) var(--panel-padding);
  border-top: 1px solid var(--color-border-strong);
  flex-shrink: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.block-rank-label {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  margin-bottom: var(--space-sm);
  font-weight: 600;
}

.panel-error { padding: 8px 12px; margin: 0 var(--panel-padding); border-radius: var(--radius-sm); background: rgba(239,68,68,0.1); color: #ef4444; font-size: 12px; }
.block-empty {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  padding: var(--space-md) 0;
  text-align: center;
}

.block-rank-section :deep(.panel-table-wrapper) {
  font-size: var(--font-xs);
}
</style>
