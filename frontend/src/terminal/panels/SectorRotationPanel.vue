<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '@/stores/session'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const session = useSessionStore()
const chartTheme = useChartTheme()
const { fetchWithCache } = usePanelCache()
const market = ref<'CN' | 'HK' | 'US'>(props.params?.market || session.ui.activeMarket || 'CN')
const lookback = ref(20)
const loading = ref(false); const loadError = ref('')

interface SectorStrength { name: string; change_pct: number; rs_ratio: number; rs_momentum: number }
const sectors = ref<SectorStrength[]>([])
let chartInstance: any = null

function calculateRRG(changePct: number, allPcts: number[]): { rs_ratio: number; rs_momentum: number } {
  const mean = allPcts.reduce((a, b) => a + b, 0) / (allPcts.length || 1)
  const std = Math.sqrt(allPcts.reduce((a, b) => a + (b - mean) ** 2, 0) / (allPcts.length || 1))
  return { rs_ratio: Math.round((std > 0 ? ((changePct - mean) / std) * 10 + 100 : 100) * 10) / 10, rs_momentum: Math.round((std > 0 ? ((changePct - mean) / std) * 10 + 100 : 100) - 100 + (changePct > 0 ? 2 : -2) * 10) / 10 }
}

async function fetchData() { loading.value = true; loadError.value = ''; try { const app = (window as any).go?.main?.App; if (!app?.GetIndustryRanks) return; const { data: ranks } = await fetchWithCache<any>(`industry_ranks:${market.value}:${lookback.value}`, () => app.GetIndustryRanks(market.value, lookback.value)); if (ranks && ranks.length > 0) { const pcts = ranks.map((r: any) => r.changePct || 0); sectors.value = ranks.map((r: any) => { const rrg = calculateRRG(r.changePct || 0, pcts); return { name: r.name, change_pct: r.changePct || 0, rs_ratio: rrg.rs_ratio, rs_momentum: rrg.rs_momentum } }) } } catch (e: any) { loadError.value = e?.message || String(e) } finally { loading.value = false }; setTimeout(renderChart, 300) }

function renderChart() { if (typeof window === 'undefined' || !(window as any).echarts) return; const echarts = (window as any).echarts; const el = document.getElementById('rrg-chart'); if (!el) return; if (!chartInstance) chartInstance = echarts.init(el); const pal = chartTheme.palette; const option = { tooltip: { trigger: 'item', formatter: (params: any) => { const s = sectors.value[params.dataIndex]; if (!s) return ''; return `<b>${s.name}</b><br/>涨幅: ${s.change_pct >= 0 ? '+' : ''}${s.change_pct.toFixed(2)}%<br/>RS-Ratio: ${s.rs_ratio}<br/>RS-Momentum: ${s.rs_momentum}` } }, grid: { left: 50, right: 50, top: 40, bottom: 40 }, xAxis: { min: 85, max: 115, splitLine: { show: true, lineStyle: { color: chartTheme.gridColor, type: 'dashed' } }, axisLabel: { color: chartTheme.axisColor, fontSize: 10 } }, yAxis: { min: 85, max: 115, splitLine: { show: true, lineStyle: { color: chartTheme.gridColor, type: 'dashed' } }, axisLabel: { color: chartTheme.axisColor, fontSize: 10 } }, series: [{ type: 'scatter', data: sectors.value.map(s => [s.rs_ratio, s.rs_momentum, s.name]), symbolSize: 14, itemStyle: { color: (params: any) => { const x = params.data[0]; const y = params.data[1]; if (x >= 100 && y >= 100) return pal[1]; if (x < 100 && y >= 100) return pal[2]; if (x < 100 && y < 100) return pal[3]; return pal[0] } }, label: { show: true, formatter: (params: any) => params.data[2], fontSize: 9, color: chartTheme.axisColor, position: 'right' } }] }; chartInstance.setOption(option, true) }
function switchMarket(mkt: 'CN' | 'HK' | 'US') { market.value = mkt; fetchData() }
function switchLookback(days: number) { lookback.value = days; fetchData() }
onMounted(fetchData)
onUnmounted(() => chartInstance?.dispose())
</script>

<template>
  <div class="sector-rotation-panel">
    <PanelHeader title="板块轮动">
      <template #controls>
        <button v-for="mkt in (['CN','HK','US'] as const)" :key="mkt" :class="['btn btn-sm', { 'btn-primary': market === mkt }]" @click="switchMarket(mkt)">{{ mkt }}</button>
        <button class="btn btn-sm" @click="fetchData" :disabled="loading">⟳</button>
      </template>
    </PanelHeader>

    <div class="lookback-bar">
      <button :class="['btn btn-sm', { 'btn-primary': lookback === 5 }]" @click="switchLookback(5)">5d</button>
      <button :class="['btn btn-sm', { 'btn-primary': lookback === 20 }]" @click="switchLookback(20)">20d</button>
      <button :class="['btn btn-sm', { 'btn-primary': lookback === 60 }]" @click="switchLookback(60)">60d</button>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <LoadingState v-if="loading && sectors.length === 0" type="chart" />

    <template v-else>
      <div class="rrg-hint"><span class="quadrant-label leading">领先</span><span class="quadrant-label weakening">减弱</span><span class="quadrant-label lagging">滞后</span><span class="quadrant-label improving">改善</span></div>
      <div id="rrg-chart" class="rrg-chart"></div>

      <div class="table-wrapper">
        <div class="table-header"><span class="col-sector">{{ $t('misc.sector') }}</span><span class="col-pct">{{ $t('quote.change_pct') }}</span><span class="col-ratio">RS-Ratio</span><span class="col-momentum">RS-Momentum</span><span class="col-signal">{{ $t('misc.signal') }}</span></div>
        <div class="table-body">
          <div v-for="s in sectors" :key="s.name" class="table-row">
            <span class="col-sector">{{ s.name }}</span><span class="col-pct" :class="s.change_pct >= 0 ? 'up' : 'down'">{{ s.change_pct >= 0 ? '+' : '' }}{{ s.change_pct.toFixed(2) }}%</span>
            <span class="col-ratio">{{ s.rs_ratio.toFixed(1) }}</span><span class="col-momentum">{{ s.rs_momentum.toFixed(1) }}</span>
            <span class="col-signal" :class="s.rs_ratio >= 100 && s.rs_momentum >= 100 ? 'leading' : s.rs_ratio < 100 && s.rs_momentum < 100 ? 'lagging' : 'neutral'">{{ s.rs_ratio >= 100 && s.rs_momentum >= 100 ? $t('misc.leading') : s.rs_ratio < 100 && s.rs_momentum < 100 ? $t('misc.lagging') : s.rs_ratio >= 100 ? $t('misc.improving') : $t('misc.weakening') }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.sector-rotation-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.lookback-bar { display: flex; gap: var(--space-xs); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
.rrg-hint { display: flex; justify-content: space-around; margin-bottom: var(--space-xs); padding: var(--space-xs) var(--panel-padding); font-size: var(--font-xs); }
.quadrant-label { padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); }
.quadrant-label.leading { color: var(--color-down); }
.quadrant-label.weakening { color: var(--color-accent); }
.quadrant-label.lagging { color: var(--color-up); }
.quadrant-label.improving { color: var(--color-accent); }
.rrg-chart { height: 200px; margin-bottom: var(--space-sm); flex-shrink: 0; }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: var(--space-xs) var(--panel-padding); border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: var(--font-xs); }
.table-row { display: flex; padding: var(--space-xs) var(--panel-padding); align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-hover); }
.col-sector { width: 80px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-pct { width: 64px; text-align: right; font-weight: 500; }
.col-ratio, .col-momentum { width: 64px; text-align: right; color: var(--color-text-secondary); }
.col-signal { width: 60px; text-align: center; font-size: var(--font-xs); font-weight: 500; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }
.leading { color: var(--color-down); }
.lagging { color: var(--color-up); }
.neutral { color: var(--color-accent); }
</style>
