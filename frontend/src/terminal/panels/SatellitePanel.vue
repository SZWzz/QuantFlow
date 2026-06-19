<!-- frontend/src/terminal/panels/SatellitePanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface RegionSnapshot {
  id: string
  name: string
  name_cn: string
  lat: number
  lon: number
  solar_ghi: number    // kWh/m^2/day
  wind_speed: number   // m/s
  trend: string        // up, down, stable
  wildfires: number
  asset_link: string
}

interface EnergyPoint {
  date: string
  value: number
}

const regions = ref<RegionSnapshot[]>([])
const loading = ref(true)
const selectedRegion = ref<RegionSnapshot | null>(null)
const solarData = ref<EnergyPoint[]>([])
const windData = ref<EnergyPoint[]>([])
const chartLoading = ref(false)

async function loadRegions() {
  loading.value = true
  try {
    const go = (window as any).go
    if (go?.main?.App?.GetSatelliteSnapshots) {
      const result = await go.main.App.GetSatelliteSnapshots()
      regions.value = result.regions || []
    } else {
      regions.value = getMockRegions()
    }
  } catch {
    regions.value = getMockRegions()
  }
  loading.value = false
}

async function loadRegionDetail(region: RegionSnapshot) {
  selectedRegion.value = region
  chartLoading.value = true
  try {
    const go = (window as any).go
    if (go?.main?.App?.GetSatelliteDetail) {
      const result = await go.main.App.GetSatelliteDetail(region.id)
      solarData.value = result.solar_chart || result.solar_data || []
      windData.value = result.wind_chart || result.wind_data || []
    } else {
      const mock = generateMockPoints(region)
      solarData.value = mock.solar
      windData.value = mock.wind
    }
  } catch {
    const mock = generateMockPoints(region)
    solarData.value = mock.solar
    windData.value = mock.wind
  }
  chartLoading.value = false
}

onMounted(() => {
  loadRegions()
})

// Signal counts
const signalCounts = computed(() => {
  let up = 0, down = 0, stable = 0
  for (const r of regions.value) {
    if (r.trend === 'up') up++
    else if (r.trend === 'down') down++
    else stable++
  }
  return { up, down, stable }
})

// Dual-axis chart option for the selected region
const chartOption = computed(() => {
  if (!selectedRegion.value || (solarData.value.length === 0 && windData.value.length === 0)) return {}

  const solarDates = solarData.value.map(p => formatDate(p.date))
  const solarValues = solarData.value.map(p => p.value)
  const windDates = windData.value.map(p => formatDate(p.date))
  const windValues = windData.value.map(p => p.value)

  // Use solar dates as primary (most complete)
  const dates = solarDates.length >= windDates.length ? solarDates : windDates

  return {
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: { type: 'cross' as const },
    },
    legend: {
      data: ['太阳辐射 (kWh/m²/天)', '风速 (m/s)'],
      top: 0,
      textStyle: { fontSize: 11, color: '#9ca3af' },
    },
    grid: { left: 50, right: 50, top: 40, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: dates,
      axisLabel: { rotate: 45, fontSize: 10 },
    },
    yAxis: [
      {
        type: 'value' as const,
        name: 'kWh/m²/天',
        nameTextStyle: { fontSize: 10, color: '#f59e0b' },
        axisLabel: { fontSize: 10 },
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.05)' } },
      },
      {
        type: 'value' as const,
        name: 'm/s',
        nameTextStyle: { fontSize: 10, color: '#3b82f6' },
        axisLabel: { fontSize: 10 },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: '太阳辐射 (kWh/m²/天)',
        type: 'line',
        data: solarValues,
        yAxisIndex: 0,
        smooth: true,
        areaStyle: { color: 'rgba(245, 158, 11, 0.1)' },
        lineStyle: { color: '#f59e0b', width: 2 },
        itemStyle: { color: '#f59e0b' },
        showSymbol: false,
      },
      {
        name: '风速 (m/s)',
        type: 'line',
        data: windValues.length === solarValues.length ? windValues : [...windValues], // pad if needed
        yAxisIndex: 1,
        smooth: true,
        areaStyle: { color: 'rgba(59, 130, 246, 0.1)' },
        lineStyle: { color: '#3b82f6', width: 2 },
        itemStyle: { color: '#3b82f6' },
        showSymbol: false,
      },
    ],
  }
})

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  if (dateStr.length === 8) {
    // Convert YYYYMMDD to YYYY-MM-DD
    return `${dateStr.slice(0,4)}-${dateStr.slice(4,6)}-${dateStr.slice(6,8)}`
  }
  return dateStr.slice(0, 10)
}

function trendIcon(t: string): string {
  if (t === 'up') return '↑'
  if (t === 'down') return '↓'
  return '→'
}

function trendLabel(t: string): string {
  if (t === 'up') return '上升'
  if (t === 'down') return '下降'
  return '稳定'
}

function trendClass(t: string): string {
  return t
}

function solarGaugeClass(ghi: number): string {
  if (ghi > 5.0) return 'text-green'
  if (ghi < 3.0) return 'text-red'
  return 'text-muted'
}

function windGaugeClass(speed: number): string {
  if (speed > 8.0) return 'text-green'
  if (speed < 3.0) return 'text-red'
  return 'text-muted'
}

function wildfireClass(count: number): string {
  if (count > 50) return 'text-red'
  if (count > 10) return 'text-warning'
  return 'text-muted'
}

// ── Mock data ─────────────────────────────────────────────────────
function getMockRegions(): RegionSnapshot[] {
  return [
    { id: 'texas', name: 'Texas Wind Corridor', name_cn: '德州风能走廊', lat: 32.8, lon: -100.1, solar_ghi: 5.2, wind_speed: 8.7, trend: 'up', wildfires: 15, asset_link: '天然气/电力' },
    { id: 'north-sea', name: 'North Sea Wind Farm', name_cn: '北海风电场', lat: 56.0, lon: 3.0, solar_ghi: 2.8, wind_speed: 9.4, trend: 'up', wildfires: 0, asset_link: '欧洲电力/天然气' },
    { id: 'gobi', name: 'Gobi Solar Base', name_cn: '戈壁太阳能基地', lat: 40.5, lon: 100.0, solar_ghi: 6.1, wind_speed: 4.8, trend: 'stable', wildfires: 0, asset_link: '中国新能源/多晶硅' },
    { id: 'sahara', name: 'Sahara Solar Belt', name_cn: '撒哈拉太阳能带', lat: 23.0, lon: 13.0, solar_ghi: 7.2, wind_speed: 3.5, trend: 'up', wildfires: 3, asset_link: '欧洲碳配额' },
    { id: 'midwest', name: 'US Midwest Agricultural Belt', name_cn: '美国中西部农业带', lat: 41.0, lon: -93.0, solar_ghi: 4.5, wind_speed: 6.2, trend: 'down', wildfires: 22, asset_link: '玉米/大豆/小麦期货' },
  ]
}

function generateMockPoints(region: RegionSnapshot): { solar: EnergyPoint[]; wind: EnergyPoint[] } {
  const solar: EnergyPoint[] = []
  const wind: EnergyPoint[] = []
  const now = new Date()
  const solarBase = region.solar_ghi
  const windBase = region.wind_speed

  for (let i = 0; i < 30; i++) {
    const date = new Date(now)
    date.setDate(date.getDate() - (29 - i))
    const dateStr = date.toISOString().slice(0, 10).replace(/-/g, '')

    const solarSeasonal = Math.sin(i / 30 * 2 * Math.PI) * 1.5
    const solarNoise = (i % 7 - 3) * 0.3
    const solarVal = Math.max(0, Math.round((solarBase + solarSeasonal + solarNoise) * 100) / 100)
    solar.push({ date: dateStr, value: solarVal })

    const windNoise = (i % 5 - 2) * 2.0
    const windVal = Math.max(0, Math.round((windBase + windNoise) * 100) / 100)
    wind.push({ date: dateStr, value: windVal })
  }
  return { solar, wind }
}
</script>

<template>
  <div class="satellite-panel" :data-panel-id="panelId">
    <!-- Header -->
    <div class="panel-header">
      <h3>🛰️ 卫星数据 (NASA POWER)</h3>
      <div class="header-summary">
        <span class="summary-badge up" v-if="signalCounts.up > 0">↑ {{ signalCounts.up }} 上升</span>
        <span class="summary-badge down" v-if="signalCounts.down > 0">↓ {{ signalCounts.down }} 下降</span>
        <span class="summary-badge stable" v-if="signalCounts.stable > 0">→ {{ signalCounts.stable }} 稳定</span>
        <button class="btn-sm" @click="loadRegions()">🔄 刷新</button>
      </div>
    </div>

    <!-- Main content: region grid + detail -->
    <div class="content-area">
      <!-- Region cards grid -->
      <div class="region-grid" :class="{ 'with-detail': selectedRegion }">
        <div v-if="loading" class="empty-state">加载中...</div>
        <div v-else-if="regions.length === 0" class="empty-state">暂无数据</div>
        <div
          v-for="region in regions"
          :key="region.id"
          :class="['region-card', { selected: selectedRegion?.id === region.id }]"
          @click="loadRegionDetail(region)"
        >
          <div class="card-header">
            <span class="card-name">{{ region.name_cn }}</span>
            <span :class="['trend-badge', trendClass(region.trend)]">
              {{ trendIcon(region.trend) }} {{ trendLabel(region.trend) }}
            </span>
          </div>

          <!-- Solar GHI gauge -->
          <div class="gauge-row">
            <div class="gauge">
              <span class="gauge-label">☀️ 太阳辐射</span>
              <span :class="['gauge-value', solarGaugeClass(region.solar_ghi)]">
                {{ region.solar_ghi.toFixed(1) }}
              </span>
              <span class="gauge-unit">kWh/m²/天</span>
            </div>
          </div>

          <!-- Wind speed gauge -->
          <div class="gauge-row">
            <div class="gauge">
              <span class="gauge-label">💨 风速</span>
              <span :class="['gauge-value', windGaugeClass(region.wind_speed)]">
                {{ region.wind_speed.toFixed(1) }}
              </span>
              <span class="gauge-unit">m/s</span>
            </div>
          </div>

          <!-- Wildfire count -->
          <div class="wildfire-row">
            <span class="gauge-label">🔥 野火</span>
            <span :class="['wildfire-value', wildfireClass(region.wildfires)]">
              {{ region.wildfires }}
            </span>
            <span class="gauge-unit">次/周</span>
          </div>

          <!-- Asset link badge -->
          <div class="asset-badge">
            {{ region.asset_link }}
          </div>
        </div>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedRegion" class="detail-panel">
        <div class="detail-header">
          <div>
            <h4>{{ selectedRegion.name_cn }}</h4>
            <p class="detail-subtitle">{{ selectedRegion.name }} ({{ selectedRegion.lat.toFixed(1) }}, {{ selectedRegion.lon.toFixed(1) }})</p>
          </div>
          <button class="btn-close" @click="selectedRegion = null">&times;</button>
        </div>

        <!-- Gauges summary -->
        <div class="detail-info">
          <div class="info-row">
            <span class="info-label">太阳辐射</span>
            <span :class="['info-value', solarGaugeClass(selectedRegion.solar_ghi)]">
              {{ selectedRegion.solar_ghi.toFixed(1) }} kWh/m²/天
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">风速</span>
            <span :class="['info-value', windGaugeClass(selectedRegion.wind_speed)]">
              {{ selectedRegion.wind_speed.toFixed(1) }} m/s
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">趋势</span>
            <span :class="['info-value', trendClass(selectedRegion.trend)]">
              {{ trendIcon(selectedRegion.trend) }} {{ trendLabel(selectedRegion.trend) }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">野火</span>
            <span :class="['info-value', wildfireClass(selectedRegion.wildfires)]">
              {{ selectedRegion.wildfires }} 次/周
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">关联资产</span>
            <span class="info-value asset-link">{{ selectedRegion.asset_link }}</span>
          </div>
        </div>

        <!-- 30-day dual-axis line chart -->
        <div class="chart-container" v-if="(solarData.length > 0 || windData.length > 0) && !chartLoading">
          <VChart :option="chartOption" style="height: 250px" autoresize />
        </div>
        <div v-else-if="chartLoading" class="empty-state small">加载图表中...</div>
        <div v-else class="empty-state small">暂无历史数据</div>

        <!-- Trend summary -->
        <div class="trend-summary" v-if="selectedRegion.trend !== 'stable'">
          <span :class="['trend-text', trendClass(selectedRegion.trend)]">
            {{ selectedRegion.trend === 'up' ? '📈 能源指标上升' : '📉 能源指标下降' }}
          </span>
          <span v-if="selectedRegion.trend === 'up'">— 对关联资产偏正面</span>
          <span v-else>— 对关联资产偏负面</span>
        </div>
        <div class="trend-summary" v-else>
          <span class="trend-text stable-text">→ 能源指标稳定</span>
          <span>— 对关联资产中性</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.satellite-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 13px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
  gap: 4px;
}
.panel-header h3 { margin: 0; font-size: 14px; }

.header-summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.summary-badge {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--color-bg-subtle);
}
.summary-badge.up { color: #16a34a; }
.summary-badge.down { color: #dc2626; }
.summary-badge.stable { color: var(--color-text-secondary); }

.btn-sm {
  padding: 2px 8px;
  font-size: 11px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.btn-sm:hover { background: var(--color-bg-hover); }

.content-area { display: flex; flex: 1; overflow: hidden; }

/* Region grid */
.region-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
  padding: 12px;
  overflow-y: auto;
  align-content: start;
}
.region-grid.with-detail {
  flex: 0 0 55%;
}

.region-card {
  display: flex;
  flex-direction: column;
  padding: 12px;
  border: 1px solid var(--color-border-subtle);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  min-width: 0;
}
.region-card:hover {
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}
.region-card.selected {
  border-color: var(--color-accent);
  background: var(--color-bg-selected);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  gap: 4px;
}
.card-name {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.trend-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--color-bg-subtle);
  white-space: nowrap;
}
.trend-badge.up { color: #16a34a; }
.trend-badge.down { color: #dc2626; }
.trend-badge.stable { color: var(--color-text-tertiary); }

.gauge-row, .wildfire-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 6px;
  gap: 4px;
}
.gauge-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
}
.gauge-value {
  font-size: 18px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.gauge-unit {
  font-size: 10px;
  color: var(--color-text-tertiary);
}

.wildfire-value {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.asset-badge {
  margin-top: 8px;
  padding: 3px 8px;
  font-size: 10px;
  border-radius: 4px;
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.text-green { color: #16a34a; }
.text-red { color: #dc2626; }
.text-warning { color: #f59e0b; }
.text-muted { color: var(--color-text-tertiary); }

/* Detail panel */
.detail-panel {
  flex: 1;
  border-left: 1px solid var(--color-border);
  padding: 12px;
  overflow-y: auto;
  min-width: 300px;
}
.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}
.detail-header h4 { margin: 0; font-size: 15px; }
.detail-subtitle {
  margin: 2px 0 0 0;
  font-size: 11px;
  color: var(--color-text-tertiary);
}
.btn-close {
  background: none; border: none; font-size: 18px;
  color: var(--color-text-secondary); cursor: pointer;
}

.detail-info {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px;
  background: var(--color-bg-subtle);
  border-radius: 6px;
}
.info-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.info-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.info-value {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.info-value.asset-link {
  font-size: 12px;
  color: var(--color-accent);
}

.chart-container { margin-bottom: 12px; }

.trend-summary {
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--color-bg-subtle);
  font-size: 12px;
  line-height: 1.5;
}
.trend-text { font-weight: 600; }
.trend-text.stable-text { color: var(--color-text-secondary); }

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--color-text-tertiary);
  grid-column: 1 / -1;
}
.empty-state.small { padding: 20px; font-size: 12px; }
</style>
