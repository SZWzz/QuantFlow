<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import VChart from 'vue-echarts'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

// --- Mock data ---

interface Position {
  symbol: string
  market: string
  quantity: number
  avgPrice: number
  marketPrice: number
  pnl: number
  pnlPct: number
  allocPct: number
  marketValue: number
}

const positions = ref<Position[]>([
  { symbol: 'AAPL', market: 'US', quantity: 100, avgPrice: 188.5, marketPrice: 195.3, pnl: 680, pnlPct: 3.61, allocPct: 26.7, marketValue: 19530 },
  { symbol: '000001.SZ', market: 'CN', quantity: 1000, avgPrice: 12.5, marketPrice: 13.8, pnl: 1300, pnlPct: 10.40, allocPct: 18.8, marketValue: 13800 },
  { symbol: 'BTCUSDT', market: 'CRYPTO', quantity: 0.15, avgPrice: 62000, marketPrice: 65000, pnl: 450, pnlPct: 4.84, allocPct: 13.3, marketValue: 9750 },
])

const kpi = ref({
  total_value: 115300,
  cash_balance: 42000,
  market_value: 73300,
  total_pnl: 5300,
  total_pnl_pct: 4.82,
})

const equityData = ref([
  { date: 'Jan', equity: 100000 },
  { date: 'Feb', equity: 102000 },
  { date: 'Mar', equity: 105000 },
  { date: 'Apr', equity: 103000 },
  { date: 'May', equity: 110000 },
  { date: 'Jun', equity: 115300 },
])

const allocationData = ref([
  { market: 'US', value: 52, color: '#388e3c' },
  { market: 'CN', value: 25, color: '#d32f2f' },
  { market: 'CRYPTO', value: 15, color: '#f57c00' },
  { market: 'HK', value: 8, color: '#1976d2' },
])

let refreshTimer: ReturnType<typeof setInterval> | null = null

// --- setInterval 10s refresh placeholder ---
function refreshMockData() {
  // In production, this will call dataStore to fetch real portfolio data
  const jitter = (v: number, r: number) => v + (Math.random() - 0.5) * r
  kpi.value = {
    total_value: Math.round(jitter(115300, 2000)),
    cash_balance: Math.round(jitter(42000, 500)),
    market_value: Math.round(jitter(73300, 1500)),
    total_pnl: Math.round(jitter(5300, 300)),
    total_pnl_pct: +(jitter(4.82, 0.3)).toFixed(2),
  }
}

onMounted(() => {
  refreshTimer = setInterval(refreshMockData, 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

// --- Helpers ---

function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}

// Currency symbols by market suffix. TODO: read base currency from user settings.
function currencyForSymbol(symbol: string): string {
  if (symbol.endsWith('.SZ') || symbol.endsWith('.SH') || symbol.endsWith('.BJ')) return '¥'
  if (symbol.endsWith('.HK')) return 'HK$'
  if (symbol.includes('USDT') || symbol === 'BTC' || symbol === 'ETH') return 'USDT'
  return '$' // US / default
}

function fmtMoney(n: number, symbol?: string): string {
  const c = symbol ? currencyForSymbol(symbol) : '$'
  if (Math.abs(n) >= 1e6) return c + (n / 1e6).toFixed(2) + 'M'
  if (Math.abs(n) >= 1e3) return c + (n / 1e3).toFixed(1) + 'K'
  return c + n.toFixed(2)
}

function pnlClass(v: number): string {
  return v >= 0 ? 'up' : 'down'
}

function pnlSign(v: number): string {
  return v >= 0 ? '+' : ''
}

function marketClass(market: string): string {
  return 'market-badge market-' + market.toLowerCase()
}

function onPositionClick(pos: Position) {
  // Navigate to PositionDetail panel — emit custom event or use router
  console.log('[PortfolioSummary] navigate to PositionDetail:', pos.symbol)
}

// --- Equity curve chart ---

const equityChartOption = computed(() => ({
  backgroundColor: 'transparent',
  grid: { top: 10, right: 12, bottom: 30, left: 55 },
  xAxis: {
    type: 'category' as const,
    data: equityData.value.map((p) => p.date),
    axisLine: { lineStyle: { color: '#30363d' } },
    axisLabel: { color: '#5a6380', fontSize: 10 },
    axisTick: { show: false },
  },
  yAxis: {
    type: 'value' as const,
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: {
      color: '#5a6380',
      fontSize: 10,
      formatter: (v: number) => (v / 1000).toFixed(0) + 'k',
    },
    splitLine: { lineStyle: { color: '#0f2137' } },
  },
  series: [{
    type: 'line',
    data: equityData.value.map((p) => p.equity),
    smooth: true,
    symbol: 'none',
    lineStyle: { color: '#58a6ff', width: 2 },
    areaStyle: {
      color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: 'rgba(88, 166, 255, 0.28)' },
        { offset: 1, color: 'rgba(88, 166, 255, 0.02)' },
      ]),
    },
  }],
  tooltip: {
    trigger: 'axis' as const,
    backgroundColor: '#16213e',
    borderColor: '#30363d',
    textStyle: { color: '#e0e0e0', fontSize: 12 },
    formatter: (params: any) => {
      const p = params[0]
      return `${p.name}<br/>Equity: <b>$${(p.value as number).toLocaleString()}</b>`
    },
  },
}))

// --- Allocation pie chart ---

const pieChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    trigger: 'item' as const,
    backgroundColor: '#16213e',
    borderColor: '#30363d',
    textStyle: { color: '#e0e0e0', fontSize: 12 },
    formatter: (params: any) =>
      `<b>${params.name}</b><br/>Allocation: ${params.value}%`,
  },
  series: [{
    type: 'pie',
    radius: ['45%', '75%'],
    center: ['50%', '50%'],
    avoidLabelOverlap: false,
    itemStyle: { borderRadius: 2, borderColor: '#1a1a2e', borderWidth: 2 },
    label: {
      show: true,
      position: 'outside' as const,
      color: '#5a6380',
      fontSize: 10,
      formatter: '{b}\n{d}%',
    },
    labelLine: { lineStyle: { color: '#30363d' } },
    data: allocationData.value.map((a) => ({
      name: a.market,
      value: a.value,
      itemStyle: { color: a.color },
    })),
  }],
}))

// --- Positions table helpers ---

const totalMarketValue = computed(() =>
  positions.value.reduce((s, p) => s + p.marketValue, 0),
)

function positionAllocPct(pos: Position): string {
  const total = totalMarketValue.value
  if (!total) return '0.0'
  return ((pos.marketValue / total) * 100).toFixed(1)
}
</script>

<template>
  <div class="portfolio-panel">
    <!-- KPI Cards -->
    <div class="kpi-row">
      <div class="kpi-card">
        <span class="kpi-label">Total Value</span>
        <span class="kpi-value">{{ fmtMoney(kpi.total_value) }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Cash Balance</span>
        <span class="kpi-value">{{ fmtMoney(kpi.cash_balance) }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Market Value</span>
        <span class="kpi-value">{{ fmtMoney(kpi.market_value) }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Total P&amp;L</span>
        <span class="kpi-value" :class="pnlClass(kpi.total_pnl)">
          {{ pnlSign(kpi.total_pnl) }}{{ fmtMoney(kpi.total_pnl) }}
        </span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">P&amp;L %</span>
        <span class="kpi-value" :class="pnlClass(kpi.total_pnl_pct)">
          {{ pnlSign(kpi.total_pnl_pct) }}{{ fmt(kpi.total_pnl_pct) }}%
        </span>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="charts-row">
      <div class="chart-box chart-equity">
        <h3 class="section-title">Equity Curve</h3>
        <VChart class="chart-body" :option="equityChartOption" autoresize />
      </div>
      <div class="chart-box chart-pie">
        <h3 class="section-title">Allocation</h3>
        <VChart class="chart-body" :option="pieChartOption" autoresize />
      </div>
    </div>

    <!-- Positions Table -->
    <div class="positions-section">
      <h3 class="section-title">Positions</h3>
      <div class="table-wrap">
        <table class="pos-table">
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Market</th>
              <th class="num">Qty</th>
              <th class="num">Avg$</th>
              <th class="num">Mkt$</th>
              <th class="num">P&amp;L</th>
              <th class="num">%</th>
              <th class="num">Alloc%</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="pos in positions"
              :key="pos.symbol"
              class="pos-row"
              @click="onPositionClick(pos)"
            >
              <td class="pos-symbol">{{ pos.symbol }}</td>
              <td><span :class="marketClass(pos.market)">{{ pos.market }}</span></td>
              <td class="num">{{ pos.quantity }}</td>
              <td class="num">{{ pos.avgPrice.toFixed(2) }}</td>
              <td class="num">{{ fmtMoney(pos.marketValue, pos.symbol) }}</td>
              <td class="num" :class="pnlClass(pos.pnl)">
                {{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl, pos.symbol) }}
              </td>
              <td class="num" :class="pnlClass(pos.pnlPct)">
                {{ pnlSign(pos.pnlPct) }}{{ fmt(pos.pnlPct) }}%
              </td>
              <td class="num">{{ positionAllocPct(pos) }}%</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.portfolio-panel {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  overflow-y: auto;
  font-variant-numeric: tabular-nums;
  color: var(--text);
  font-size: var(--font-sm);
}

/* --- KPI Cards --- */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: var(--spacing);
  margin-bottom: 10px;
}

.kpi-card {
  background: var(--card);
  border-radius: 4px;
  padding: 10px 8px;
  text-align: center;
}

.kpi-label {
  display: block;
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  margin-bottom: 4px;
}

.kpi-value {
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
}

.kpi-value.up { color: var(--up); }
.kpi-value.down { color: var(--down); }

/* --- Charts Row --- */
.charts-row {
  display: flex;
  gap: var(--spacing);
  margin-bottom: 10px;
  min-height: 0;
}

.chart-box {
  background: var(--card);
  border-radius: 4px;
  padding: 8px;
  display: flex;
  flex-direction: column;
}

.chart-box .section-title {
  margin-bottom: 4px;
}

.chart-equity {
  flex: 0 0 60%;
  min-width: 0;
}

.chart-pie {
  flex: 0 0 40%;
  min-width: 0;
}

.chart-body {
  flex: 1;
  min-height: 150px;
}

/* --- Section Title --- */
.section-title {
  font-size: var(--font-xs);
  text-transform: uppercase;
  color: var(--muted);
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--input);
}

/* --- Positions Table --- */
.positions-section {
  background: var(--card);
  border-radius: 4px;
  padding: 8px;
}

.table-wrap {
  overflow-x: auto;
}

.pos-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}

.pos-table th {
  text-align: left;
  padding: 5px 8px;
  font-size: var(--font-xs);
  color: var(--muted);
  font-weight: 500;
  text-transform: uppercase;
  border-bottom: 1px solid var(--input);
  white-space: nowrap;
}

.pos-table th.num {
  text-align: right;
}

.pos-table td {
  padding: 5px 8px;
  border-bottom: 1px solid var(--input);
  white-space: nowrap;
}

.pos-table td.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.pos-row {
  cursor: pointer;
  transition: background 0.15s;
}

.pos-row:hover {
  background: var(--input);
}

.pos-symbol {
  font-weight: 600;
  font-size: var(--font-sm);
  color: var(--text);
}

.up { color: var(--up); }
.down { color: var(--down); }

/* --- Market Badges --- */
.market-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: var(--font-xs);
  font-weight: 600;
}

.market-us {
  color: #388e3c;
  background: rgba(56, 142, 60, 0.15);
  border: 1px solid rgba(56, 142, 60, 0.3);
}

.market-cn {
  color: #d32f2f;
  background: rgba(211, 47, 47, 0.15);
  border: 1px solid rgba(211, 47, 47, 0.3);
}

.market-hk {
  color: #1976d2;
  background: rgba(25, 118, 210, 0.15);
  border: 1px solid rgba(25, 118, 210, 0.3);
}

.market-crypto {
  color: #f57c00;
  background: rgba(245, 124, 0, 0.15);
  border: 1px solid rgba(245, 124, 0, 0.3);
}
</style>
