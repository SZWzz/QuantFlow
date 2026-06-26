<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import type { IndexSnapshot, SectorRanking } from '@/stores/data'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()

const autoRefresh = ref(true)
const refreshInterval = ref(15)
const countdown = ref(refreshInterval.value)
let timer: ReturnType<typeof setInterval> | null = null

const indices = computed(() => dataStore.marketOverview?.indices ?? [])
const breadth = computed(() => dataStore.marketOverview?.breadth ?? { advancers: 0, decliners: 0, unchanged: 0 })
const sectors = computed(() => dataStore.marketOverview?.sectors ?? [])
const updatedAt = computed(() => dataStore.marketOverview?.updatedAt ?? 0)
const loading = computed(() => dataStore.marketLoading)

const topGainers = computed(() => [...sectors.value].sort((a, b) => b.changePct - a.changePct).slice(0, 8))
const topLosers = computed(() => [...sectors.value].sort((a, b) => a.changePct - b.changePct).slice(0, 8))

function refresh() {
  dataStore.fetchMarketOverview()
  countdown.value = refreshInterval.value
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
      <div v-for="idx in indices" :key="idx.symbol" class="index-card">
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
}
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
</style>
