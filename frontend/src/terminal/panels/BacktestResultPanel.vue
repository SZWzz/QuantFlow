<script setup lang="ts">
import { ref, computed } from 'vue'
import * as echarts from 'echarts'
import VChart from 'vue-echarts'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

// Mock data for development — real data comes from Go backend via dataStore
const backtestResult = ref({
  metrics: {
    total_return: 0.153,
    cagr: 0.142,
    max_drawdown: -0.087,
    sharpe_ratio: 1.42,
    sortino_ratio: 1.89,
    calmar_ratio: 1.63,
    win_rate: 0.62,
    profit_factor: 1.85,
    total_trades: 24,
    annual_volatility: 0.12,
  },
  equityCurve: [
    { date: '2024-01', equity: 100000 },
    { date: '2024-02', equity: 102000 },
    { date: '2024-03', equity: 105000 },
    { date: '2024-04', equity: 103000 },
    { date: '2024-05', equity: 108000 },
    { date: '2024-06', equity: 112000 },
    { date: '2024-07', equity: 110000 },
    { date: '2024-08', equity: 115000 },
    { date: '2024-09', equity: 113000 },
    { date: '2024-10', equity: 118000 },
    { date: '2024-11', equity: 116000 },
    { date: '2024-12', equity: 115300 },
  ],
  trades: [
    { date: '2024-03-15', symbol: '000001.SZ', side: 'buy', quantity: 1000, price: 12.50 },
    { date: '2024-05-20', symbol: '000001.SZ', side: 'sell', quantity: 1000, price: 13.80, pnl: 1300 },
    { date: '2024-06-10', symbol: '600519.SH', side: 'buy', quantity: 100, price: 1680.00 },
    { date: '2024-08-15', symbol: '600519.SH', side: 'sell', quantity: 100, price: 1750.00, pnl: 7000 },
    { date: '2024-09-01', symbol: '000001.SZ', side: 'buy', quantity: 500, price: 14.20 },
    { date: '2024-11-01', symbol: '000001.SZ', side: 'sell', quantity: 500, price: 15.10, pnl: 450 },
  ],
})

// Equity curve chart options
const equityChartOption = computed(() => ({
  backgroundColor: 'transparent',
  grid: { top: 10, right: 20, bottom: 30, left: 60 },
  xAxis: {
    type: 'category',
    data: backtestResult.value.equityCurve.map((p) => p.date),
    axisLine: { lineStyle: { color: '#30363d' } },
    axisLabel: { color: '#5a6380', fontSize: 10 },
  },
  yAxis: {
    type: 'value',
    axisLine: { lineStyle: { color: '#30363d' } },
    axisLabel: { color: '#5a6380', fontSize: 10, formatter: (v: number) => (v / 1000).toFixed(0) + 'k' },
    splitLine: { lineStyle: { color: '#1a2332' } },
  },
  series: [{
    type: 'line',
    data: backtestResult.value.equityCurve.map((p) => p.equity),
    smooth: true,
    lineStyle: { color: '#3fb950', width: 2 },
    areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
      { offset: 0, color: 'rgba(63, 185, 80, 0.3)' },
      { offset: 1, color: 'rgba(63, 185, 80, 0)' },
    ])},
    symbol: 'none',
  }],
  tooltip: {
    trigger: 'axis',
    backgroundColor: '#161b22',
    borderColor: '#30363d',
    textStyle: { color: '#c9d1d9', fontSize: 12 },
    formatter: (params: any) => {
      const p = params[0]
      return `${p.name}<br/>Equity: <b>$${(p.value as number).toLocaleString()}</b>`
    },
  },
}))

// 回撤 chart options
const drawdownChartOption = computed(() => {
  const equityData = backtestResult.value.equityCurve.map((p) => p.equity)
  let peak = equityData[0]
  const drawdowns = equityData.map((e) => {
    if (e > peak) peak = e
    return ((e - peak) / peak) * 100
  })

  return {
    backgroundColor: 'transparent',
    grid: { top: 5, right: 20, bottom: 25, left: 60 },
    xAxis: {
      type: 'category',
      data: backtestResult.value.equityCurve.map((p) => p.date),
      axisLabel: { color: '#5a6380', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#5a6380', fontSize: 10, formatter: (v: number) => v.toFixed(1) + '%' },
      splitLine: { lineStyle: { color: '#1a2332' } },
    },
    series: [{
      type: 'bar',
      data: drawdowns,
      itemStyle: { color: '#da3633' },
    }],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#161b22',
      borderColor: '#30363d',
      textStyle: { color: '#c9d1d9', fontSize: 12 },
      formatter: (params: any) => {
        const p = params[0]
        return `${p.name}<br/>回撤: <b>${(p.value as number).toFixed(2)}%</b>`
      },
    },
  }
})

function formatPct(v: number): string {
  return (v * 100).toFixed(2) + '%'
}

function formatNum(v: number, decimals = 2): string {
  return v.toFixed(decimals)
}

function pnlClass(v: number): string {
  return v > 0 ? 'pnl-positive' : v < 0 ? 'pnl-negative' : ''
}
</script>

<template>
  <div class="backtest-panel">
    <!-- Metrics Cards -->
    <div class="metrics-grid">
      <div class="metric-card">
        <span class="metric-label">总收益</span>
        <span class="metric-value" :class="backtestResult.metrics.total_return >= 0 ? 'positive' : 'negative'">
          {{ formatPct(backtestResult.metrics.total_return) }}
        </span>
      </div>
      <div class="metric-card">
        <span class="metric-label">夏普</span>
        <span class="metric-value">{{ formatNum(backtestResult.metrics.sharpe_ratio) }}</span>
      </div>
      <div class="metric-card">
        <span class="metric-label">最大回撤</span>
        <span class="metric-value negative">{{ formatPct(backtestResult.metrics.max_drawdown) }}</span>
      </div>
      <div class="metric-card">
        <span class="metric-label">胜率</span>
        <span class="metric-value">{{ formatPct(backtestResult.metrics.win_rate) }}</span>
      </div>
      <div class="metric-card">
        <span class="metric-label">盈亏比</span>
        <span class="metric-value">{{ formatNum(backtestResult.metrics.profit_factor) }}</span>
      </div>
      <div class="metric-card">
        <span class="metric-label">交易次数</span>
        <span class="metric-value">{{ backtestResult.metrics.total_trades }}</span>
      </div>
    </div>

    <!-- 净值曲线 -->
    <div class="chart-section">
      <h3 class="section-title">净值曲线</h3>
      <v-chart class="chart" :option="equityChartOption" autoresize />
    </div>

    <!-- 回撤 -->
    <div class="chart-section">
      <h3 class="section-title">回撤</h3>
      <v-chart class="chart chart-sm" :option="drawdownChartOption" autoresize />
    </div>

    <!-- Trade List -->
    <div class="section">
      <h3 class="section-title">Trade List</h3>
      <table class="trade-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Symbol</th>
            <th>Side</th>
            <th>Qty</th>
            <th>Price</th>
            <th>P&L</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(t, i) in backtestResult.trades" :key="i">
            <td>{{ t.date }}</td>
            <td>{{ t.symbol }}</td>
            <td><span :class="t.side === 'buy' ? 'side-buy' : 'side-sell'">{{ t.side }}</span></td>
            <td>{{ t.quantity }}</td>
            <td>{{ t.price.toFixed(2) }}</td>
            <td :class="pnlClass(t.pnl || 0)">{{ t.pnl ? t.pnl.toFixed(2) : '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.backtest-panel {
  padding: 10px;
  background: #0d1117;
  height: 100%;
  overflow-y: auto;
  color: #c9d1d9;
  font-size: 12px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}

.metric-card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 10px;
  text-align: center;
}

.metric-label {
  display: block;
  font-size: 10px;
  color: #5a6380;
  text-transform: uppercase;
  margin-bottom: 4px;
}

.metric-value {
  font-size: 16px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.metric-value.positive { color: #3fb950; }
.metric-value.negative { color: #f85149; }

.chart-section {
  margin-bottom: 10px;
}

.chart {
  width: 100%;
  height: 180px;
}

.chart-sm {
  height: 80px;
}

.section-title {
  font-size: 10px;
  text-transform: uppercase;
  color: #5a6380;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid #21262d;
}

.trade-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}

.trade-table th {
  text-align: left;
  padding: 4px 6px;
  color: #5a6380;
  font-weight: 500;
  border-bottom: 1px solid #21262d;
}

.trade-table td {
  padding: 3px 6px;
  border-bottom: 1px solid #1a2332;
}

.side-buy { color: #3fb950; }
.side-sell { color: #f85149; }

.pnl-positive { color: #3fb950; }
.pnl-negative { color: #f85149; }
</style>
