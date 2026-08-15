<!-- frontend/src/terminal/panels/SatellitePanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'
import 'echarts'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { logger } from '@/lib/logger'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const app = useWailsApp()

interface RegionSnapshot { id: string; name: string; name_cn: string; lat: number; lon: number; solar_ghi: number; wind_speed: number; trend: string; wildfires: number; asset_link: string }
interface EnergyPoint { date: string; value: number }

const regions = ref<RegionSnapshot[]>([]); const chartTheme = useChartTheme()
const loading = ref(true); const loadError = ref('')
const selectedRegion = ref<RegionSnapshot | null>(null); let loadSeq = 0
const solarData = ref<EnergyPoint[]>([]); const windData = ref<EnergyPoint[]>([])
const chartLoading = ref(false)
const { fetchWithCache } = usePanelCache()
const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

async function loadRegions() { const seq = ++loadSeq; loading.value = true; loadError.value = ''; try { if (app?.GetSatelliteSnapshots) { const { data: result } = await fetchWithCache<any>('satellite_snapshots', () => app.GetSatelliteSnapshots(), 30 * 60 * 1000); if (seq !== loadSeq) return; regions.value = result?.regions || [] } } catch(e: any) { logger.error('[Satellite] loadRegions:', e); loadError.value = e?.message || String(e) } if (seq === loadSeq) loading.value = false }
async function loadRegionDetail(region: RegionSnapshot) { selectedRegion.value = region; chartLoading.value = true; try { if (app?.GetSatelliteDetail) { const { data: result } = await fetchWithCache<any>(`satellite_detail:${region.id}`, () => app.GetSatelliteDetail(region.id), 30 * 60 * 1000); solarData.value = result.solar_chart || result.solar_data || []; windData.value = result.wind_chart || result.wind_data || [] } } catch(e) { logger.error('[Satellite] loadDetail:', e) } }
onMounted(() => loadRegions())

const signalCounts = computed(() => { let up = 0, down = 0, stable = 0; for (const r of regions.value) { if (r.trend === 'up') up++; else if (r.trend === 'down') down++; else stable++ } return { up, down, stable } })

const chartOption = computed(() => {
  if (!selectedRegion.value || (solarData.value.length === 0 && windData.value.length === 0)) return {}
  const solarValues = solarData.value.map(p => p.value); const windValues = windData.value.map(p => p.value)
  const solarDates = solarData.value.map(p => formatDate(p.date)); const windDates = windData.value.map(p => formatDate(p.date))
  const dates = solarDates.length >= windDates.length ? solarDates : windDates
  const solarName = `${t('satellite.solar_radiation')} (${t('satellite.energy_kwh')})`; const windName = `风速 (${t('satellite.wind_speed')})`
  const pal = chartTheme.palette
  return { tooltip: { trigger: 'axis' as const, axisPointer: { type: 'cross' as const } }, legend: { data: [solarName, windName], top: 0, textStyle: { fontSize: 11, color: chartTheme.axisColor } }, grid: { left: 50, right: 50, top: 40, bottom: 40 }, xAxis: { type: 'category' as const, data: dates, axisLabel: { rotate: 45, fontSize: 10 } }, yAxis: [{ type: 'value' as const, name: t('satellite.energy_kwh'), nameTextStyle: { fontSize: 10, color: pal[2] }, axisLabel: { fontSize: 10 }, splitLine: { lineStyle: { color: chartTheme.gridColor } } }, { type: 'value' as const, name: t('satellite.wind_speed'), nameTextStyle: { fontSize: 10, color: pal[0] }, axisLabel: { fontSize: 10 }, splitLine: { show: false } }], series: [{ name: solarName, type: 'line', data: solarValues, yAxisIndex: 0, smooth: true, areaStyle: { color: pal[2] + '1A' }, lineStyle: { color: pal[2], width: 2 }, itemStyle: { color: pal[2] }, showSymbol: false }, { name: windName, type: 'line', data: windValues.length === solarValues.length ? windValues : [...windValues], yAxisIndex: 1, smooth: true, areaStyle: { color: pal[0] + '1A' }, lineStyle: { color: pal[0], width: 2 }, itemStyle: { color: pal[0] }, showSymbol: false }] }
})

function formatDate(dateStr: string): string { if (!dateStr) return ''; if (dateStr.length === 8) return `${dateStr.slice(0,4)}-${dateStr.slice(4,6)}-${dateStr.slice(6,8)}`; return dateStr.slice(0, 10) }
function trendIcon(t: string): string { if (t === 'up') return '↑'; if (t === 'down') return '↓'; return '→' }
function trendLabel(t: string): string { if (t === 'up') return '上升'; if (t === 'down') return '下降'; return '稳定' }
function trendClass(t: string): string { return t }
function solarGaugeClass(ghi: number): string { if (ghi > 5.0) return 'text-green'; if (ghi < 3.0) return 'text-red'; return 'text-muted' }
function windGaugeClass(speed: number): string { if (speed > 8.0) return 'text-green'; if (speed < 3.0) return 'text-red'; return 'text-muted' }
function wildfireClass(count: number): string { if (count > 50) return 'text-red'; if (count > 10) return 'text-warning'; return 'text-muted' }
</script>

<template>
  <div class="satellite-panel" :data-panel-id="panelId">
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <PanelHeader title="卫星数据">
      <template #controls>
        <span class="summary-badge up" v-if="signalCounts.up > 0">↑ {{ signalCounts.up }} 上升</span>
        <span class="summary-badge down" v-if="signalCounts.down > 0">↓ {{ signalCounts.down }} 下降</span>
        <span class="summary-badge stable" v-if="signalCounts.stable > 0">→ {{ signalCounts.stable }} 稳定</span>
        <button v-if="addToWfControl" class="btn btn-sm" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <button class="btn btn-sm" @click="loadRegions()">🔄 {{ $t('common.refresh') }}</button>
      </template>
    </PanelHeader>

    <div class="content-area">
      <div class="region-grid" :class="{ 'with-detail': selectedRegion }">
        <LoadingState v-if="loading" type="card" :rows="4" />
        <EmptyState v-else-if="regions.length === 0" title="暂无数据" />
        <div v-for="region in regions" :key="region.id" :class="['region-card', { selected: selectedRegion?.id === region.id }]" @click="loadRegionDetail(region)">
          <div class="card-header"><span class="card-name">{{ region.name_cn }}</span><span :class="['trend-badge', trendClass(region.trend)]">{{ trendIcon(region.trend) }} {{ trendLabel(region.trend) }}</span></div>
          <div class="gauge-row"><span class="gauge-label">{{ $t('satellite.solar_radiation') }}</span><span :class="['gauge-value', solarGaugeClass(region.solar_ghi)]">{{ region.solar_ghi.toFixed(1) }}</span><span class="gauge-unit">{{ $t('satellite.energy_kwh') }}</span></div>
          <div class="gauge-row"><span class="gauge-label">💨 风速</span><span :class="['gauge-value', windGaugeClass(region.wind_speed)]">{{ region.wind_speed.toFixed(1) }}</span><span class="gauge-unit">{{ $t('satellite.wind_speed') }}</span></div>
          <div class="wildfire-row"><span class="gauge-label">🔥 野火</span><span :class="['wildfire-value', wildfireClass(region.wildfires)]">{{ region.wildfires }}</span><span class="gauge-unit">{{ $t('satellite.fire_count') }}</span></div>
          <div class="asset-badge">{{ region.asset_link }}</div>
        </div>
      </div>

      <div v-if="selectedRegion" class="detail-panel">
        <div class="detail-header"><div><h4>{{ selectedRegion.name_cn }}</h4><p class="detail-subtitle">{{ selectedRegion.name }} ({{ selectedRegion.lat.toFixed(1) }}, {{ selectedRegion.lon.toFixed(1) }})</p></div><button class="btn-close" @click="selectedRegion = null">&times;</button></div>
        <div class="detail-info">
          <div class="info-row"><span class="info-label">{{ $t('satellite.solar_radiation') }}</span><span :class="['info-value', solarGaugeClass(selectedRegion.solar_ghi)]">{{ selectedRegion.solar_ghi.toFixed(1) }} {{ $t('satellite.energy_kwh') }}</span></div>
          <div class="info-row"><span class="info-label">风速</span><span :class="['info-value', windGaugeClass(selectedRegion.wind_speed)]">{{ selectedRegion.wind_speed.toFixed(1) }} {{ $t('satellite.wind_speed') }}</span></div>
          <div class="info-row"><span class="info-label">趋势</span><span :class="['info-value', trendClass(selectedRegion.trend)]">{{ trendIcon(selectedRegion.trend) }} {{ trendLabel(selectedRegion.trend) }}</span></div>
          <div class="info-row"><span class="info-label">野火</span><span :class="['info-value', wildfireClass(selectedRegion.wildfires)]">{{ selectedRegion.wildfires }} {{ $t('satellite.fire_count') }}</span></div>
          <div class="info-row"><span class="info-label">{{ $t('satellite.linked_assets') }}</span><span class="info-value asset-link">{{ selectedRegion.asset_link }}</span></div>
        </div>
        <div class="chart-container" v-if="(solarData.length > 0 || windData.length > 0) && !chartLoading"><VChart :option="chartOption" style="height: 250px" autoresize /></div>
        <LoadingState v-else-if="chartLoading" type="chart" />
        <EmptyState v-else title="暂无历史数据" />
        <div class="trend-summary" v-if="selectedRegion.trend !== 'stable'"><span :class="['trend-text', trendClass(selectedRegion.trend)]">{{ selectedRegion.trend === 'up' ? '📈 能源指标上升' : '📉 能源指标下降' }}</span><span v-if="selectedRegion.trend === 'up'">— 对关联资产偏正面</span><span v-else>— 对关联资产偏负面</span></div>
        <div class="trend-summary" v-else><span class="trend-text stable-text">{{ $t('satellite.stable_indicator') }}</span><span>{{ $t('geo.neutral_signal') }}</span></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.satellite-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
.summary-badge { font-size: var(--font-xs); padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); background: var(--color-bg-subtle); }
.summary-badge.up { color: var(--color-up); }
.summary-badge.down { color: var(--color-down); }
.summary-badge.stable { color: var(--color-text-secondary); }

.content-area { display: flex; flex: 1; overflow: hidden; }
.region-grid { flex: 1; display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: var(--space-sm); padding: var(--space-md); overflow-y: auto; align-content: start; }
.region-grid.with-detail { flex: 0 0 55%; }
.region-card { display: flex; flex-direction: column; padding: var(--space-md); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); cursor: pointer; transition: border-color 0.15s, background 0.15s; min-width: 0; }
.region-card:hover { border-color: var(--color-accent); background: var(--color-bg-hover); }
.region-card.selected { border-color: var(--color-accent); background: var(--color-bg-selected); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-md); gap: var(--space-xs); }
.card-name { font-size: var(--font-sm); font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.trend-badge { font-size: var(--font-xs); padding: 0 var(--space-sm); border-radius: var(--radius-sm); background: var(--color-bg-subtle); white-space: nowrap; }
.trend-badge.up { color: var(--color-down); }
.trend-badge.down { color: var(--color-up); }
.trend-badge.stable { color: var(--color-text-tertiary); }
.gauge-row, .wildfire-row { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: var(--space-sm); gap: var(--space-xs); }
.gauge-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.gauge-value { font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; }
.gauge-unit { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.wildfire-value { font-size: var(--font-lg); font-weight: 600; font-variant-numeric: tabular-nums; }
.asset-badge { margin-top: var(--space-sm); padding: var(--space-xs) var(--space-sm); font-size: var(--font-xs); border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-secondary); text-align: center; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.text-green { color: var(--color-down); }
.text-red { color: var(--color-up); }
.text-warning { color: var(--color-accent); }
.text-muted { color: var(--color-text-tertiary); }
.detail-panel { flex: 1; border-left: 1px solid var(--color-border); padding: var(--space-md); overflow-y: auto; min-width: 300px; }
.detail-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: var(--space-sm); }
.detail-header h4 { margin: 0; font-size: var(--font-lg); }
.detail-subtitle { margin: var(--space-xs) 0 0 0; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.btn-close { background: none; border: none; font-size: var(--font-lg); color: var(--color-text-secondary); cursor: pointer; }
.detail-info { display: grid; grid-template-columns: repeat(2, 1fr); gap: var(--space-sm); margin-bottom: var(--space-md); padding: var(--space-md); background: var(--color-bg-subtle); border-radius: var(--radius-md); }
.info-row { display: flex; flex-direction: column; gap: var(--space-xs); }
.info-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.info-value { font-size: var(--font-sm); font-weight: 600; font-variant-numeric: tabular-nums; }
.info-value.asset-link { font-size: var(--font-xs); color: var(--color-accent); }
.chart-container { margin-bottom: var(--space-md); }
.trend-summary { padding: var(--space-sm) var(--space-md); border-radius: var(--radius-md); background: var(--color-bg-subtle); font-size: var(--font-xs); line-height: 1.5; }
.trend-text { font-weight: 600; }
.trend-text.stable-text { color: var(--color-text-secondary); }
</style>
