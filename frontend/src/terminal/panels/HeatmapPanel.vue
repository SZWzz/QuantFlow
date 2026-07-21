<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import type { MarketOverview } from '@/stores/data'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const loading = ref(false)
const loadError = ref('')
const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const cacheKey = computed(() => `market:overview:${activeMarket.value}`)

const marketTabs = [
  { key: 'CN', label: 'CN' },
  { key: 'HK', label: 'HK' },
  { key: 'US', label: 'US' },
]

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

/** 涨跌分档 → 色带 class（token 化：强档实底反白，弱档 soft 底 + 同向文字） */
function bandClass(pct: number): string {
  if (pct > 2) return 'band-up-strong'
  if (pct > 0.5) return 'band-up'
  if (pct > -0.5) return 'band-flat'
  if (pct > -2) return 'band-down'
  return 'band-down-strong'
}

function switchMarket(mkt: string) {
  if (mkt !== 'CN' && mkt !== 'HK' && mkt !== 'US') return
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
  loadError.value = ''
  try {
    await dataStore.fetchMarketOverview(activeMarket.value)
    dataStore.setCached(cacheKey.value, dataStore.marketOverview, 30_000)
  } catch (e: any) {
    loadError.value = e?.message || String(e)
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
    <PanelHeader
      :title="$t('misc.heatmap')"
      :tabs="marketTabs"
      :active-tab="activeMarket"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: refresh, loading }]"
      @tab-change="switchMarket"
    />

    <ErrorState v-if="loadError" :description="loadError" @retry="refresh" />
    <LoadingState v-else-if="loading" type="card" :rows="2" />

    <!-- 自绘热力网格：PanelTable 表达不了按市值伸缩的色块，保留但 token 化 -->
    <div v-else-if="cells.length > 0" class="heatmap-grid">
      <div
        v-for="cell in cells"
        :key="cell.name"
        class="heatmap-cell"
        :class="bandClass(cell.changePct)"
        :style="{ flexGrow: Math.max(1, Math.round(cell.marketCap / 1000)) }"
      >
        <span class="cell-name">{{ cell.name }}</span>
        <span class="cell-pct">{{ cell.changePct >= 0 ? '+' : '' }}{{ cell.changePct }}%</span>
      </div>
    </div>

    <EmptyState
      v-else
      :title="activeMarket === 'HK' ? $t('misc.no_hk_sector_data') :
             activeMarket === 'US' ? $t('misc.no_us_sector_data') :
             $t('misc.no_sector_data')"
    />

    <div class="legend">
      <span class="legend-item"><span class="swatch band-up-strong"></span> +2%+</span>
      <span class="legend-item"><span class="swatch band-up"></span> +0.5~2%</span>
      <span class="legend-item"><span class="swatch band-flat"></span> -0.5~0.5%</span>
      <span class="legend-item"><span class="swatch band-down"></span> -2~-0.5%</span>
      <span class="legend-item"><span class="swatch band-down-strong"></span> -2%+</span>
    </div>
  </div>
</template>

<style scoped>
.heatmap-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Heatmap Grid */
.heatmap-grid {
  flex: 1; display: flex; flex-wrap: wrap; align-content: flex-start;
  gap: 2px; overflow-y: auto; padding: var(--panel-padding);
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.heatmap-cell {
  min-width: 70px; min-height: 32px; padding: var(--space-xs) var(--space-sm);
  border-radius: var(--radius-sm); display: flex; flex-wrap: wrap;
  align-items: center; justify-content: space-between;
  font-size: var(--font-xs); transition: filter var(--transition-fast); cursor: default;
}
.heatmap-cell:hover { filter: brightness(1.2); }
.cell-name { font-weight: 500; white-space: nowrap; }
.cell-pct { font-variant-numeric: tabular-nums; margin-left: var(--space-xs); }

/* 涨跌色带（heatmap 单元格与 legend 色块共用） */
.band-up-strong { background: var(--color-up); color: var(--color-text-inverse); }
.band-up { background: var(--color-up-soft); color: var(--color-up); }
.band-flat { background: var(--color-bg-elevated); color: var(--color-text-secondary); }
.band-down { background: var(--color-down-soft); color: var(--color-down); }
.band-down-strong { background: var(--color-down); color: var(--color-text-inverse); }

.legend {
  display: flex; gap: var(--space-md); padding: var(--space-sm) var(--panel-padding); flex-wrap: wrap;
  border-top: 1px solid var(--color-border-subtle); font-size: var(--font-xs); color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.legend-item { display: flex; align-items: center; gap: var(--space-xs); }
.swatch { width: 10px; height: 10px; border-radius: var(--radius-sm); display: inline-block; }
</style>
