<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import * as echarts from 'echarts'
import VChart from 'vue-echarts'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const market = ref('US')
const currency = ref('USD')

const detail = ref({ quantity: 100, avg_price: 188.50, market_price: 195.32, market_value: 19532, pnl: 682, pnl_pct: 3.62, alloc_pct: 26.7 })

const fmt = (n: number, dec = 2) => n.toFixed(dec)

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

const priceChartOption = computed(() => ({
  backgroundColor: 'transparent',
  grid: { top: 10, right: 20, bottom: 30, left: 60 },
  xAxis: { type: 'category', data: ['Jan','Feb','Mar','Apr','May','Jun'], axisLabel: { color: '#5a6380', fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { color: '#5a6380', fontSize: 10 } },
  series: [
    { type: 'line', data: [185,190,188,192,194,195.32], smooth: true, lineStyle: { color: '#58a6ff', width: 2 }, symbol: 'none' },
    { type: 'line', data: [188.5,188.5,188.5,188.5,188.5,188.5], lineStyle: { color: '#f0883e', width: 1, type: 'dashed' }, symbol: 'none', name: 'Cost Basis' }
  ]
}))
</script>

<template>
  <div class="position-detail-panel">
    <div class="header">
      <span class="symbol-name">{{ symbol }}</span>
      <span class="market-badge">{{ market }}</span>
      <span class="currency">{{ currency }}</span>
    </div>
    <div class="kpi-grid">
      <div class="kpi-item"><span class="kpi-label">Quantity</span><span class="kpi-value">{{ detail.quantity }}</span></div>
      <div class="kpi-item"><span class="kpi-label">Avg Price</span><span class="kpi-value">${{ fmt(detail.avg_price) }}</span></div>
      <div class="kpi-item"><span class="kpi-label">Market Price</span><span class="kpi-value">${{ fmt(detail.market_price) }}</span></div>
      <div class="kpi-item"><span class="kpi-label">Market Value</span><span class="kpi-value">${{ fmt(detail.market_value).replace(/\B(?=(\d{3})+(?!\d))/g, ',') }}</span></div>
      <div class="kpi-item"><span class="kpi-label">P&amp;L</span><span :class="['kpi-value', detail.pnl >= 0 ? 'up' : 'down']">${{ fmt(detail.pnl) }}</span></div>
      <div class="kpi-item"><span class="kpi-label">Allocation</span><span class="kpi-value">{{ fmt(detail.alloc_pct) }}%</span></div>
    </div>
    <div class="chart-section">
      <div class="chart-title">Price History (30d)</div>
      <VChart :option="priceChartOption" autoresize style="height:180px" />
    </div>
    <div class="pnl-summary">
      <span :class="detail.pnl >= 0 ? 'up' : 'down'">{{ detail.pnl >= 0 ? '+' : '' }}${{ fmt(detail.pnl) }} ({{ detail.pnl >= 0 ? '+' : '' }}{{ fmt(detail.pnl_pct) }}%)</span>
    </div>
  </div>
</template>

<style scoped>
.position-detail-panel { padding: 12px; background: var(--bg); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.header { margin-bottom: 12px; }
.symbol-name { font-size: 20px; font-weight: 700; color: var(--text); }
.market-badge { display: inline-block; margin-left: 8px; padding: 2px 8px; background: var(--input); border-radius: 3px; font-size: 11px; color: var(--accent); }
.currency { margin-left: 6px; font-size: var(--font-sm); color: var(--muted); }
.kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; margin-bottom: 12px; }
.kpi-item { padding: 8px; background: var(--card); border-radius: 4px; text-align: center; }
.kpi-label { display: block; font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 3px; }
.kpi-value { font-size: 15px; font-weight: 600; color: var(--text); }
.kpi-value.up { color: var(--up); } .kpi-value.down { color: var(--down); }
.chart-section { background: var(--card); border-radius: 4px; padding: 8px; margin-bottom: 12px; }
.chart-title { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 4px; }
.pnl-summary { text-align: center; font-size: 18px; font-weight: 700; }
.pnl-summary .up { color: var(--up); } .pnl-summary .down { color: var(--down); }
</style>
