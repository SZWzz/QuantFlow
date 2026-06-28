<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'
import type { IndexSnapshot, SectorRanking } from '@/stores/data'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const autoRefresh = ref(true)
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
  try {
    const result = await app.GetBlockRank(1, 0, 10)
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    blockRank.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      volume: i.volume || 0,
      amount: i.amount || 0,
    }))
  } catch(e) {
    console.error('[MarketOverview] block rank:', e)
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

function sparklinePoints(data: number[]): string {
  if (!data.length) return ''
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const w = 60
  const h = 24
  return data.map((v, i) => {
    const x = (i / (data.length - 1)) * w
    const y = h - ((v - min) / range) * h
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

function changeColor(pct: number): string {
  if (pct > 0) return '#ef4444'
  if (pct < 0) return '#22c55e'
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
</script>

<template>
  <div class="market-overview-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.market_overview') }}</h3>
      <div class="market-tabs">
        <button v-for="mkt in (['CN', 'HK', 'US'] as const)" :key="mkt"
          :class="['mkt-tab', { active: activeMarket === mkt }]"
          @click="switchMarket(mkt)"
        >{{ mkt }}</button>
      </div>
      <div class="header-controls">
        <span class="update-time">{{ formatTime(updatedAt) }}</span>
        <button class="auto-btn" :class="{ active: autoRefresh }" @click="toggleAutoRefresh">
          自动 {{ autoRefresh ? `(${countdown}s)` : '' }}
        </button>
        <button class="refresh-btn" @click="refresh" :disabled="loading">
          {{ loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <!-- Section A: Index Cards -->
    <div class="indices-row">
      <div v-for="idx in indices" :key="idx.symbol" class="index-card" @click="onIndexClick(idx)">
        <div class="index-name">{{ idx.name }}</div>
        <div class="index-price">{{ idx.last.toLocaleString() }}</div>
        <div class="index-change" :style="{ color: changeColor(idx.changePct) }">
          {{ formatPct(idx.changePct) }}
        </div>
        <svg class="index-sparkline" viewBox="0 0 60 24" preserveAspectRatio="none">
          <polyline
            :points="sparklinePoints(idx.sparkline)"
            fill="none"
            :stroke="changeColor(idx.changePct)"
            stroke-width="1.5"
          />
        </svg>
      </div>
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
      <div v-if="blockRankLoading" class="block-loading">{{ $t('common.loading') }}</div>
      <div v-else-if="blockRank.length === 0" class="block-empty">{{ $t('common.no_data') }}</div>
      <div v-else class="block-rank-table">
        <div class="br-header-row">
          <span class="br-col symbol-col">{{ $t('common.symbol') }}</span>
          <span class="br-col name-col">{{ $t('common.name') }}</span>
          <span class="br-col price-col">{{ $t('common.price') }}</span>
          <span class="br-col vol-col">{{ $t('common.volume') }}</span>
          <span class="br-col amt-col">{{ $t('common.amount') }}</span>
        </div>
        <div v-for="(item, idx) in blockRank" :key="idx" class="br-row">
          <span class="br-col symbol-col">{{ item.symbol }}</span>
          <span class="br-col name-col">{{ item.name }}</span>
          <span class="br-col price-col">{{ item.price.toFixed(2) }}</span>
          <span class="br-col vol-col">{{ formatVolume(item.volume) }}</span>
          <span class="br-col amt-col">{{ formatAmount(item.amount) }}</span>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">{{ $t('common.loading') }}</div>
  </div>
</template>

<style scoped>
.market-overview-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.market-tabs { display: flex; gap: 4px; margin: 0 8px; }
.mkt-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.mkt-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.header-controls { display: flex; gap: 8px; align-items: center; }
.update-time { font-size: 11px; color: var(--color-text-tertiary); }
.auto-btn {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.auto-btn.active { color: #60a5fa; border-color: #3b82f6; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* Index Cards */
.indices-row {
  display: flex; gap: 8px; overflow-x: auto;
  padding-bottom: 4px; margin-bottom: 12px;
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.index-card {
  flex: 0 0 auto; min-width: 130px;
  padding: 10px 12px; border-radius: 6px;
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong);
  cursor: pointer; transition: background 0.15s, border-color 0.15s;
}
.index-card:hover { background: var(--color-bg-hover); border-color: var(--color-primary); }
.index-name { font-size: 11px; color: var(--color-text-secondary); margin-bottom: 2px; }
.index-price { font-size: 16px; font-weight: 600; margin-bottom: 2px; }
.index-change { font-size: 12px; font-weight: 500; margin-bottom: 4px; }
.index-sparkline { width: 100%; height: 24px; display: block; }

/* Breadth */
.breadth-section { margin-bottom: 12px; }
.breadth-label { font-size: 12px; color: var(--color-text-secondary); margin-bottom: 6px; }
.breadth-bar { display: flex; height: 8px; border-radius: 4px; overflow: hidden; margin-bottom: 4px; }
.breadth-segment.up { background: #ef4444; }
.breadth-segment.down { background: #22c55e; }
.breadth-segment.flat { background: #4b5563; }
.breadth-text { display: flex; gap: 16px; font-size: 11px; }
.up-text { color: #ef4444; }
.down-text { color: #22c55e; }
.flat-text { color: var(--color-text-tertiary); }

/* Sectors */
.sectors-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
  flex: 1; overflow: hidden;
}
.sector-col { overflow-y: auto; }
.sector-col-title { font-size: 12px; font-weight: 600; margin-bottom: 6px; padding-bottom: 4px; border-bottom: 1px solid var(--color-border-strong); }
.sector-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 4px 0; font-size: 12px;
}
.sector-name { color: var(--color-text-primary); }
.sector-pct { font-weight: 500; font-variant-numeric: tabular-nums; }

.loading-overlay {
  position: absolute; top: 0; left: 0; right: 0; bottom: 0;
  display: flex; align-items: center; justify-content: center;
  background: rgba(17, 24, 39, 0.7); font-size: 14px; color: var(--color-text-tertiary);
}

/* Block Rank */
.block-rank-section {
  margin-top: 10px; padding-top: 10px; border-top: 1px solid var(--color-border-strong);
  flex-shrink: 0;
}
.block-rank-label {
  font-size: 12px; color: var(--color-text-secondary); margin-bottom: 6px; font-weight: 600;
}
.block-loading, .block-empty {
  font-size: 11px; color: var(--color-text-tertiary); padding: 8px 0; text-align: center;
}
.block-rank-table { font-size: 11px; font-variant-numeric: tabular-nums; }
.br-header-row {
  display: flex; padding: 2px 0; border-bottom: 1px solid var(--color-border-strong);
  color: var(--color-text-tertiary); font-size: 10px;
}
.br-row {
  display: flex; padding: 1px 0;
}
.br-row:nth-child(odd) { background: rgba(255,255,255,0.02); }
.br-col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.symbol-col { width: 70px; }
.name-col { flex: 1; min-width: 0; }
.price-col { width: 70px; text-align: right; }
.vol-col { width: 70px; text-align: right; }
.amt-col { width: 80px; text-align: right; }
</style>
