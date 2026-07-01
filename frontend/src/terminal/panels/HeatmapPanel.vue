<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import type { SectorRanking, MarketOverview } from '@/stores/data'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const loading = ref(false)
const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const cacheKey = computed(() => `market:overview:${activeMarket.value}`)

interface HeatmapCell {
  name: string
  changePct: number
  marketCap: number
}

const cells = computed<HeatmapCell[]>(() => {
  const sectors = dataStore.marketOverview?.sectors ?? []
  return sectors.map(s => ({
    name: s.name,
    changePct: s.changePct ?? 0,
    marketCap: 800 + Math.round((s.changePct ?? 0) * 300),
  }))
})

function changeColor(pct: number): string {
  if (pct > 2) return '#dc2626'
  if (pct > 0.5) return '#ef4444'
  if (pct > -0.5) return '#4b5563'
  if (pct > -2) return '#22c55e'
  return '#16a34a'
}

function textColor(pct: number): string {
  return Math.abs(pct) > 1.5 ? '#fff' : '#e5e7eb'
}

function switchMarket(mkt: typeof activeMarket.value) {
  activeMarket.value = mkt
  const cached = dataStore.getCached<MarketOverview>(cacheKey.value)
  if (cached) {
    dataStore.marketOverview = cached
    return
  }
  refresh()
}
async function refresh() {
  loading.value = true
  try {
    await dataStore.fetchMarketOverview(activeMarket.value)
    dataStore.setCached(cacheKey.value, dataStore.marketOverview, 30_000)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const cached = dataStore.getCached<MarketOverview>(cacheKey.value)
  if (cached) {
    dataStore.marketOverview = cached
    return
  }
  refresh()
})
</script>

<template>
  <div class="heatmap-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.heatmap') }}</h3>
      <div class="market-tabs">
        <button v-for="mkt in (['CN', 'HK', 'US'] as const)" :key="mkt"
          :class="['mkt-tab', { active: activeMarket === mkt }]"
          @click="switchMarket(mkt)"
        >{{ mkt }}</button>
      </div>
      <button class="refresh-btn" @click="refresh" :disabled="loading">
        {{ loading ? '...' : '⟳' }}
      </button>
    </div>

    <div v-if="loading" class="loading-state">{{ $t('common.loading') }}</div>

    <div v-else-if="cells.length > 0" class="heatmap-grid">
      <div
        v-for="cell in cells"
        :key="cell.name"
        class="heatmap-cell"
        :style="{
          background: changeColor(cell.changePct),
          color: textColor(cell.changePct),
          flexGrow: Math.max(1, Math.round(cell.marketCap / 1000)),
        }"
      >
        <span class="cell-name">{{ cell.name }}</span>
        <span class="cell-pct">{{ cell.changePct >= 0 ? '+' : '' }}{{ cell.changePct }}%</span>
      </div>
    </div>

    <div v-else class="empty-state">{{ $t('misc.no_sector_data') }}</div>

    <div class="legend">
      <span class="legend-item"><span class="swatch" style="background:#dc2626"></span> +2%+</span>
      <span class="legend-item"><span class="swatch" style="background:#ef4444"></span> +0.5~2%</span>
      <span class="legend-item"><span class="swatch" style="background:#4b5563"></span> -0.5~0.5%</span>
      <span class="legend-item"><span class="swatch" style="background:#22c55e"></span> -2~-0.5%</span>
      <span class="legend-item"><span class="swatch" style="background:#16a34a"></span> -2%+</span>
    </div>
  </div>
</template>

<style scoped>
.heatmap-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.market-tabs { display: flex; gap: 4px; }
.mkt-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.mkt-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.loading-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}

/* Heatmap Grid */
.heatmap-grid {
  flex: 1; display: flex; flex-wrap: wrap; align-content: flex-start;
  gap: 2px; overflow-y: auto;
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.heatmap-cell {
  min-width: 70px; min-height: 32px; padding: 6px 8px;
  border-radius: 3px; display: flex; flex-wrap: wrap;
  align-items: center; justify-content: space-between;
  font-size: 11px; transition: filter 0.15s; cursor: default;
}
.heatmap-cell:hover { filter: brightness(1.2); }
.cell-name { font-weight: 500; white-space: nowrap; }
.cell-pct { font-variant-numeric: tabular-nums; margin-left: 4px; }

.legend {
  display: flex; gap: 12px; padding-top: 8px; flex-wrap: wrap;
  border-top: 1px solid var(--color-border-strong); margin-top: 8px; font-size: 10px; color: var(--color-text-tertiary);
}
.legend-item { display: flex; align-items: center; gap: 3px; }
.swatch { width: 10px; height: 10px; border-radius: 2px; display: inline-block; }
</style>
