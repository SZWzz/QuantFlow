<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import type { SectorRanking } from '@/stores/data'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const loading = ref(false)
const hasECharts = ref(false)

interface HeatmapCell {
  name: string
  changePct: number
  marketCap: number
}

const cells = ref<HeatmapCell[]>([])

const sectorsData = computed(() => dataStore.marketOverview?.sectors ?? [])

function mockMarketCap(): number {
  return Math.round(800 + Math.random() * 25000)
}

function generateMockCells(): HeatmapCell[] {
  const names = [
    '半导体', '软件开发', '光伏设备', '消费电子', '汽车整车', '医疗器械',
    '化学制药', '食品饮料', '银行', '电力', '房地产', '钢铁', '煤炭开采',
    '航运港口', '影视院线', '通信设备', '航空航天', '新能源', '人工智能',
    '物联网', '机器人', '储能', '锂电', '氢能', '生物医药', '创新药',
    'CXO', '医美', '白酒', '调味品',
  ]
  return names.map(name => ({
    name,
    changePct: +(Math.random() * 8 - 4).toFixed(2),
    marketCap: mockMarketCap(),
  }))
}

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

function formatMc(mc: number): string {
  if (mc >= 10000) return (mc / 10000).toFixed(1) + '万亿'
  return mc.toFixed(0) + '亿'
}

async function refresh() {
  loading.value = true
  try {
    dataStore.fetchMarketOverview()
    // Use data from store if available, else mock
    if (sectorsData.value.length > 0) {
      cells.value = sectorsData.value.map(s => ({
        name: s.name,
        changePct: s.changePct,
        marketCap: mockMarketCap(),
      }))
    } else {
      cells.value = generateMockCells()
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // Check if echarts is available
  try {
    await import('echarts')
    hasECharts.value = true
  } catch {
    hasECharts.value = false
  }
  refresh()
})
</script>

<template>
  <div class="heatmap-panel">
    <div class="panel-header">
      <h3>Market Heatmap</h3>
      <button class="refresh-btn" @click="refresh" :disabled="loading">
        {{ loading ? '...' : '⟳' }}
      </button>
    </div>

    <div v-if="loading" class="loading-state">Loading...</div>

    <div v-else class="heatmap-grid">
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
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, #111827);
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.loading-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: #6b7280; font-size: 13px;
}

/* Heatmap Grid Fallback (when echarts unavailable) */
.heatmap-grid {
  flex: 1; display: flex; flex-wrap: wrap; align-content: flex-start;
  gap: 2px; overflow-y: auto;
  scrollbar-width: thin; scrollbar-color: #374151 transparent;
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
  border-top: 1px solid #374151; margin-top: 8px; font-size: 10px; color: #6b7280;
}
.legend-item { display: flex; align-items: center; gap: 3px; }
.swatch { width: 10px; height: 10px; border-radius: 2px; display: inline-block; }
</style>
