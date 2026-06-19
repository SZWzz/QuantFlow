<!-- frontend/src/terminal/panels/GovDataPanel.vue -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import VChart from 'vue-echarts'
import 'echarts'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface MacroSignal {
  indicator_id: string
  name: string
  name_cn: string
  latest_value: number
  change: number
  direction: string   // up, down, flat
  signal: string      // bullish, bearish, neutral
  unit: string
  category: string
  updated_at: number
}

interface IndicatorPoint {
  date: string
  value: number
}

const signals = ref<MacroSignal[]>([])
const loading = ref(true)
const selectedSignal = ref<MacroSignal | null>(null)
const indicatorData = ref<IndicatorPoint[]>([])
const chartLoading = ref(false)

const filterCategories = ['all', 'gdp', 'inflation', 'employment', 'rates', 'energy', 'housing'] as const
const activeCategory = ref('all')

const categoryLabels: Record<string, string> = {
  all: '全部', gdp: 'GDP/增长', inflation: '通胀', employment: '就业',
  rates: '利率', energy: '能源', housing: '房地产'
}

async function loadSignals() {
  loading.value = true
  try {
    const go = (window as any).go
    if (go?.main?.App?.GetEconomicIndicators) {
      const result = await go.main.App.GetEconomicIndicators()
      signals.value = result.signals || []
    } else {
      signals.value = getMockSignals()
    }
  } catch {
    signals.value = getMockSignals()
  }
  loading.value = false
}

async function loadIndicatorDetail(signal: MacroSignal) {
  selectedSignal.value = signal
  chartLoading.value = true
  try {
    const go = (window as any).go
    if (go?.main?.App?.GetIndicatorData) {
      const result = await go.main.App.GetIndicatorData(signal.indicator_id, 12)
      indicatorData.value = result.data || []
    } else {
      indicatorData.value = generateMockPoints(signal)
    }
  } catch {
    indicatorData.value = generateMockPoints(signal)
  }
  chartLoading.value = false
}

onMounted(() => {
  loadSignals()
})

const filteredSignals = computed(() => {
  if (activeCategory.value === 'all') return signals.value
  return signals.value.filter(s => s.category === activeCategory.value)
})

// Count signals by type
const signalCounts = computed(() => {
  let bullish = 0, bearish = 0, neutral = 0
  for (const s of signals.value) {
    if (s.signal === 'bullish') bullish++
    else if (s.signal === 'bearish') bearish++
    else neutral++
  }
  return { bullish, bearish, neutral }
})

// Chart option for selected indicator's time series
const chartOption = computed(() => {
  if (!selectedSignal.value || indicatorData.value.length === 0) return {}
  const dates = indicatorData.value.map(p => p.date)
  const values = indicatorData.value.map(p => p.value)
  return {
    tooltip: {
      trigger: 'axis' as const,
      formatter: (params: any) => `${params[0].axisValue}<br/>${selectedSignal.value!.name_cn}: ${params[0].value}`
    },
    grid: { left: 60, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'category' as const,
      data: dates,
      axisLabel: { rotate: 45, fontSize: 10 }
    },
    yAxis: {
      type: 'value' as const,
      name: selectedSignal.value?.unit || '',
      axisLabel: { fontSize: 10 }
    },
    series: [{
      type: 'line',
      data: values,
      smooth: true,
      areaStyle: {
        color: selectedSignal.value?.signal === 'bullish'
          ? 'rgba(22, 163, 74, 0.1)' : selectedSignal.value?.signal === 'bearish'
          ? 'rgba(220, 38, 38, 0.1)' : 'rgba(156, 163, 175, 0.1)'
      },
      lineStyle: {
        color: selectedSignal.value?.signal === 'bullish'
          ? '#16a34a' : selectedSignal.value?.signal === 'bearish'
          ? '#dc2626' : '#9ca3af',
        width: 2
      },
      itemStyle: {
        color: selectedSignal.value?.signal === 'bullish'
          ? '#16a34a' : selectedSignal.value?.signal === 'bearish'
          ? '#dc2626' : '#9ca3af'
      },
      markLine: {
        silent: true,
        data: [{ type: 'average', name: '均值' }],
        lineStyle: { color: '#f59e0b', type: 'dashed' }
      },
      showSymbol: false
    }]
  }
})

function formatValue(v: number, unit: string): string {
  if (unit === '%') return v.toFixed(2) + '%'
  if (v >= 1_000_000) return (v / 1_000_000).toFixed(2) + 'M'
  if (v >= 1_000) return (v / 1_000).toFixed(1) + 'K'
  if (v >= 1) return v.toFixed(2)
  return v.toFixed(4)
}

function formatChange(c: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)}%`
}

function directionIcon(d: string): string {
  if (d === 'up') return '↑'
  if (d === 'down') return '↓'
  return '→'
}

function signalEmoji(s: string): string {
  if (s === 'bullish') return '🟢'
  if (s === 'bearish') return '🔴'
  return '⚪'
}

function signalLabel(s: string): string {
  if (s === 'bullish') return '看涨'
  if (s === 'bearish') return '看跌'
  return '中性'
}

function signalClass(s: string): string {
  return s
}

function changeClass(c: number): string {
  if (c > 0) return 'text-green'
  if (c < 0) return 'text-red'
  return 'text-muted'
}

// ── Mock data ─────────────────────────────────────────────────────
function getMockSignals(): MacroSignal[] {
  return [
    { indicator_id: 'GDP', name: 'Gross Domestic Product', name_cn: '国内生产总值(GDP)', latest_value: 29250.3, change: 2.8, direction: 'up', signal: 'bullish', unit: 'Billions of Dollars', category: 'gdp', updated_at: Date.now() },
    { indicator_id: 'GDPC1', name: 'Real Gross Domestic Product', name_cn: '实际GDP', latest_value: 23250.1, change: 2.1, direction: 'up', signal: 'bullish', unit: 'Billions of Chained 2017 Dollars', category: 'gdp', updated_at: Date.now() },
    { indicator_id: 'CPIAUCSL', name: 'Consumer Price Index', name_cn: '消费者价格指数(CPI)', latest_value: 316.8, change: 0.3, direction: 'up', signal: 'bearish', unit: 'Index 1982-1984=100', category: 'inflation', updated_at: Date.now() },
    { indicator_id: 'PCEPI', name: 'PCE Price Index', name_cn: '个人消费支出价格指数(PCE)', latest_value: 125.6, change: 0.2, direction: 'up', signal: 'bearish', unit: 'Index 2017=100', category: 'inflation', updated_at: Date.now() },
    { indicator_id: 'PPIACO', name: 'Producer Price Index', name_cn: '生产者价格指数(PPI)', latest_value: 267.2, change: 0.5, direction: 'up', signal: 'bearish', unit: 'Index 1982=100', category: 'inflation', updated_at: Date.now() },
    { indicator_id: 'UNRATE', name: 'Unemployment Rate', name_cn: '失业率', latest_value: 3.8, change: -2.6, direction: 'down', signal: 'bullish', unit: '%', category: 'employment', updated_at: Date.now() },
    { indicator_id: 'PAYEMS', name: 'Total Nonfarm Payrolls', name_cn: '非农就业人数', latest_value: 159850, change: 0.15, direction: 'up', signal: 'bullish', unit: 'Thousands', category: 'employment', updated_at: Date.now() },
    { indicator_id: 'IC4WSA', name: 'Initial Claims', name_cn: '初请失业金人数', latest_value: 215, change: -3.2, direction: 'down', signal: 'bullish', unit: 'Number', category: 'employment', updated_at: Date.now() },
    { indicator_id: 'FEDFUNDS', name: 'Federal Funds Effective Rate', name_cn: '联邦基金利率', latest_value: 4.25, change: 0, direction: 'flat', signal: 'neutral', unit: '%', category: 'rates', updated_at: Date.now() },
    { indicator_id: 'DGS10', name: '10-Year Treasury Rate', name_cn: '10年期国债收益率', latest_value: 4.32, change: -1.8, direction: 'down', signal: 'bullish', unit: '%', category: 'rates', updated_at: Date.now() },
    { indicator_id: 'T10Y2Y', name: '10Y-2Y Treasury Spread', name_cn: '美债10Y-2Y利差', latest_value: -0.25, change: 10.0, direction: 'up', signal: 'bearish', unit: '%', category: 'rates', updated_at: Date.now() },
    { indicator_id: 'DCOILWTICO', name: 'Crude Oil WTI', name_cn: 'WTI原油价格', latest_value: 72.5, change: -3.5, direction: 'down', signal: 'bullish', unit: 'Dollars per Barrel', category: 'energy', updated_at: Date.now() },
    { indicator_id: 'NGDPRPI', name: 'Natural Gas Spot Price', name_cn: '天然气现货价格', latest_value: 3.85, change: 8.5, direction: 'up', signal: 'bearish', unit: 'Dollars per Million BTU', category: 'energy', updated_at: Date.now() },
    { indicator_id: 'HOUST', name: 'Housing Starts', name_cn: '新屋开工', latest_value: 1480, change: 1.2, direction: 'up', signal: 'bullish', unit: 'Thousands of Units', category: 'housing', updated_at: Date.now() },
    { indicator_id: 'MSPUS', name: 'Median Sales Price of Houses', name_cn: '房屋销售中位价', latest_value: 428500, change: 0.8, direction: 'up', signal: 'bullish', unit: 'Dollars', category: 'housing', updated_at: Date.now() },
  ]
}

function generateMockPoints(signal: MacroSignal): IndicatorPoint[] {
  const points: IndicatorPoint[] = []
  const base = signal.latest_value
  const noise = signal.signal === 'bearish' ? base * 0.02 : base * 0.01
  const now = new Date()
  for (let i = 0; i < 12; i++) {
    const date = new Date(now)
    date.setMonth(date.getMonth() - (11 - i))
    const dateStr = date.toISOString().slice(0, 10)
    const value = base - (11 - i) * (signal.change / 100 * base / 12) + (Math.random() - 0.5) * noise
    points.push({ date: dateStr, value: Math.round(value * 100) / 100 })
  }
  return points
}
</script>

<template>
  <div class="govdata-panel" :data-panel-id="panelId">
    <!-- Header -->
    <div class="panel-header">
      <h3>📈 宏观指标 (FRED)</h3>
      <div class="header-summary">
        <span class="summary-badge bullish" v-if="signalCounts.bullish > 0">🟢 {{ signalCounts.bullish }} 看涨</span>
        <span class="summary-badge bearish" v-if="signalCounts.bearish > 0">🔴 {{ signalCounts.bearish }} 看跌</span>
        <span class="summary-badge neutral" v-if="signalCounts.neutral > 0">⚪ {{ signalCounts.neutral }} 中性</span>
        <button class="btn-sm" @click="loadSignals()">🔄 刷新</button>
      </div>
    </div>

    <!-- Filter tabs -->
    <div class="category-tabs">
      <button
        v-for="cat in filterCategories" :key="cat"
        :class="['tab', { active: activeCategory === cat }]"
        @click="activeCategory = cat; selectedSignal = null"
      >
        {{ categoryLabels[cat] }}
      </button>
    </div>

    <!-- Main content: indicator grid + detail -->
    <div class="content-area">
      <!-- Indicator cards grid -->
      <div class="indicator-grid" :class="{ 'with-detail': selectedSignal }">
        <div v-if="loading" class="empty-state">加载中...</div>
        <div v-else-if="filteredSignals.length === 0" class="empty-state">暂无数据</div>
        <div
          v-for="signal in filteredSignals"
          :key="signal.indicator_id"
          :class="['indicator-card', { selected: selectedSignal?.indicator_id === signal.indicator_id }]"
          @click="loadIndicatorDetail(signal)"
        >
          <div class="card-header">
            <span class="card-name">{{ signal.name_cn }}</span>
            <span :class="['signal-badge', signalClass(signal.signal)]">
              {{ signalEmoji(signal.signal) }} {{ signalLabel(signal.signal) }}
            </span>
          </div>
          <div class="card-value">
            <span class="value">{{ formatValue(signal.latest_value, signal.unit) }}</span>
          </div>
          <div class="card-change">
            <span :class="['direction-icon', changeClass(signal.change)]">
              {{ directionIcon(signal.direction) }}
            </span>
            <span :class="['change-text', changeClass(signal.change)]">
              {{ formatChange(signal.change) }}
            </span>
            <span class="card-unit">{{ signal.unit }}</span>
          </div>
        </div>
      </div>

      <!-- Detail panel -->
      <div v-if="selectedSignal" class="detail-panel">
        <div class="detail-header">
          <div>
            <h4>{{ selectedSignal.name_cn }}</h4>
            <p class="detail-subtitle">{{ selectedSignal.name }}</p>
          </div>
          <button class="btn-close" @click="selectedSignal = null">&times;</button>
        </div>

        <!-- Signal info -->
        <div class="detail-info">
          <div class="info-row">
            <span class="info-label">最新值</span>
            <span class="info-value">{{ formatValue(selectedSignal.latest_value, selectedSignal.unit) }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">变化</span>
            <span :class="['info-value', changeClass(selectedSignal.change)]">
              {{ directionIcon(selectedSignal.direction) }} {{ formatChange(selectedSignal.change) }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">信号</span>
            <span :class="['info-value', signalClass(selectedSignal.signal)]">
              {{ signalEmoji(selectedSignal.signal) }} {{ signalLabel(selectedSignal.signal) }}
            </span>
          </div>
          <div class="info-row">
            <span class="info-label">单位</span>
            <span class="info-value">{{ selectedSignal.unit }}</span>
          </div>
        </div>

        <!-- Time series chart -->
        <div class="chart-container" v-if="indicatorData.length > 0 && !chartLoading">
          <VChart :option="chartOption" style="height: 250px" autoresize />
        </div>
        <div v-else-if="chartLoading" class="empty-state small">加载图表中...</div>
        <div v-else class="empty-state small">暂无历史数据</div>

        <!-- Trend summary -->
        <div class="trend-summary" v-if="selectedSignal.direction !== 'flat'">
          <span :class="['trend-text', changeClass(selectedSignal.change)]">
            {{ selectedSignal.direction === 'up' ? '📈 上升趋势' : '📉 下降趋势' }}
          </span>
          <span v-if="selectedSignal.signal === 'bullish'">— 对市场偏正面</span>
          <span v-else-if="selectedSignal.signal === 'bearish'">— 对市场偏负面</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.govdata-panel {
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
.summary-badge.bullish { color: #16a34a; }
.summary-badge.bearish { color: #dc2626; }
.summary-badge.neutral { color: var(--color-text-secondary); }

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

.category-tabs {
  display: flex;
  gap: 2px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border);
  overflow-x: auto;
}
.tab {
  padding: 3px 10px;
  font-size: 11px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
}
.tab.active { background: var(--color-accent); color: #fff; }
.tab:hover:not(.active) { background: var(--color-bg-hover); }

.content-area { display: flex; flex: 1; overflow: hidden; }

/* Indicator grid */
.indicator-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  padding: 12px;
  overflow-y: auto;
  align-content: start;
}
.indicator-grid.with-detail {
  flex: 0 0 55%;
}

.indicator-card {
  display: flex;
  flex-direction: column;
  padding: 10px;
  border: 1px solid var(--color-border-subtle);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  min-width: 0;
}
.indicator-card:hover {
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}
.indicator-card.selected {
  border-color: var(--color-accent);
  background: var(--color-bg-selected);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  gap: 4px;
}
.card-name {
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-value { margin-bottom: 4px; }
.value {
  font-size: 20px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.card-change {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
}
.direction-icon { font-weight: 700; font-size: 13px; }
.change-text { font-variant-numeric: tabular-nums; }
.card-unit {
  color: var(--color-text-tertiary);
  font-size: 10px;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.signal-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--color-bg-subtle);
  white-space: nowrap;
}
.signal-badge.bullish { color: #16a34a; }
.signal-badge.bearish { color: #dc2626; }

.text-green { color: #16a34a; }
.text-red { color: #dc2626; }
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

.chart-container { margin-bottom: 12px; }

.trend-summary {
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--color-bg-subtle);
  font-size: 12px;
  line-height: 1.5;
}
.trend-text { font-weight: 600; }

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
