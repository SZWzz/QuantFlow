<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'
import VChart from 'vue-echarts'
import { usePortfolioStore } from '@/stores/portfolio'
import type { PositionDetail } from '@/stores/portfolio'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const store = usePortfolioStore()

// --- Data (reactive via store) ---

const positions = computed<PositionDetail[]>(() => store.positions ?? [])

const kpi = computed(() => {
  const s = store.summary
  if (!s) return null
  return {
    total_value: s.total_value,
    cash_balance: s.cash_balance,
    market_value: s.market_value,
    total_pnl: s.total_pnl,
    total_pnl_pct: s.total_pnl_pct,
  }
})

const equityData = computed(() => {
  const curve = store.equityCurve
  if (!curve || curve.length === 0) return []
  return curve.map((p) => ({ date: p.date, equity: p.nav }))
})

const marketColors: Record<string, string> = {
  US: '#388e3c',
  CN: '#d32f2f',
  CRYPTO: '#f57c00',
  HK: '#1976d2',
}
let paletteIdx = 0
const fallbackPalette = ['#388e3c', '#d32f2f', '#f57c00', '#1976d2', '#8e24aa', '#00838f']

function colorForMarket(market: string): string {
  if (marketColors[market]) return marketColors[market]
  return fallbackPalette[paletteIdx++ % fallbackPalette.length]
}

const allocationData = computed(() => {
  paletteIdx = 0
  const byMarket = store.allocation?.by_market
  if (!byMarket) return []
  return Object.entries(byMarket).map(([market, value]) => ({
    market,
    value,
    color: colorForMarket(market),
  }))
})

// --- Lifecycle ---

onMounted(async () => {
  store.startAutoRefresh()
  store.fetchEquityCurve()
})

onUnmounted(() => {
  store.stopAutoRefresh()
})

// --- Helpers ---

function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}

function currencyForSymbol(symbol: string): string {
  if (symbol.endsWith('.SZ') || symbol.endsWith('.SH') || symbol.endsWith('.BJ')) return '¥'
  if (symbol.endsWith('.HK')) return 'HK$'
  if (symbol.includes('USDT') || symbol === 'BTC' || symbol === 'ETH') return 'USDT'
  return '$'
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

function onPositionClick(pos: PositionDetail) {
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

// --- 配置分布 pie chart ---

const pieChartOption = computed(() => ({
  backgroundColor: 'transparent',
  tooltip: {
    trigger: 'item' as const,
    backgroundColor: '#16213e',
    borderColor: '#30363d',
    textStyle: { color: '#e0e0e0', fontSize: 12 },
    formatter: (params: any) =>
      `<b>${params.name}</b><br/>配置分布: ${params.value}%`,
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

// --- 持仓 table helpers ---

const totalMarketValue = computed(() =>
  positions.value.reduce((s, p) => s + p.market_price * p.quantity, 0),
)

function positionAllocPct(pos: PositionDetail): string {
  const total = totalMarketValue.value
  if (!total) return '0.0'
  return ((pos.market_price * pos.quantity / total) * 100).toFixed(1)
}
</script>

<template>
  <div class="portfolio-panel">
    <!-- KPI Cards -->
    <div class="kpi-row">
      <div class="kpi-card">
        <span class="kpi-label">总价值</span>
        <span class="kpi-value">{{ kpi ? fmtMoney(kpi.total_value) : '--' }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">现金余额</span>
        <span class="kpi-value">{{ kpi ? fmtMoney(kpi.cash_balance) : '--' }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">市值</span>
        <span class="kpi-value">{{ kpi ? fmtMoney(kpi.market_value) : '--' }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Total P&amp;L</span>
        <span v-if="kpi" class="kpi-value" :class="pnlClass(kpi.total_pnl)">
          {{ pnlSign(kpi.total_pnl) }}{{ fmtMoney(kpi.total_pnl) }}
        </span>
        <span v-else class="kpi-value">--</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">P&amp;L %</span>
        <span v-if="kpi" class="kpi-value" :class="pnlClass(kpi.total_pnl_pct)">
          {{ pnlSign(kpi.total_pnl_pct) }}{{ fmt(kpi.total_pnl_pct) }}%
        </span>
        <span v-else class="kpi-value">--</span>
      </div>
    </div>

    <!-- Charts Row -->
    <div class="charts-row">
      <div class="chart-box chart-equity">
        <h3 class="section-title">净值曲线</h3>
        <VChart v-if="equityData.length > 0" class="chart-body" :option="equityChartOption" autoresize />
        <div v-else class="chart-empty">--</div>
      </div>
      <div class="chart-box chart-pie">
        <h3 class="section-title">配置分布</h3>
        <VChart v-if="allocationData.length > 0" class="chart-body" :option="pieChartOption" autoresize />
        <div v-else class="chart-empty">--</div>
      </div>
    </div>

    <!-- 持仓 Table -->
    <div class="positions-section">
      <h3 class="section-title">持仓</h3>
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
              <td class="num">{{ pos.avg_price.toFixed(2) }}</td>
              <td class="num">{{ fmtMoney(pos.market_price * pos.quantity, pos.symbol) }}</td>
              <td class="num" :class="pnlClass(pos.pnl)">
                {{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl, pos.symbol) }}
              </td>
              <td class="num" :class="pnlClass(pos.pnl_pct)">
                {{ pnlSign(pos.pnl_pct) }}{{ fmt(pos.pnl_pct) }}%
              </td>
              <td class="num">{{ positionAllocPct(pos) }}%</td>
            </tr>
            <tr v-if="positions.length === 0">
              <td colspan="8" class="empty-state-cell">--</td>
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

.chart-empty {
  flex: 1;
  min-height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--muted);
  font-size: var(--font-sm);
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

/* --- 持仓 Table --- */
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

.empty-state-cell {
  text-align: center;
  color: var(--muted);
  padding: 24px;
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
