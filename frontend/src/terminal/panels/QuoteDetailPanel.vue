<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const symbol = ref(props.params?.symbol || 'AAPL')

// Mock quote data
const quote = ref({
  symbol: 'AAPL',
  name: 'Apple Inc.',
  exchange: 'NASDAQ',
  last: 195.32,
  bid: 195.30,
  ask: 195.35,
  open: 191.27,
  high: 196.50,
  low: 190.80,
  prevClose: 191.27,
  volume: 32456789,
  avgVolume: 28500000,
  marketCap: '3.02T',
  pe: 32.5,
  eps: 6.43,
  dividendYield: 0.48,
  change: 4.05,
  changePct: 2.12,
})

const isPositive = computed(() => quote.value.change >= 0)

function fmt(n: number, decimals = 2): string {
  return n.toFixed(decimals)
}

function fmtVolume(n: number): string {
  if (n >= 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  return n.toLocaleString()
}
</script>

<template>
  <div class="quote-panel">
    <div class="quote-header">
      <div class="header-main">
        <span class="symbol-badge">{{ symbol }}</span>
        <span class="company-name">{{ quote.name }}</span>
        <span class="exchange">{{ quote.exchange }}</span>
      </div>
    </div>

    <div class="price-section" :class="{ up: isPositive, down: !isPositive }">
      <span class="last-price">{{ fmt(quote.last) }}</span>
      <span class="change-info">
        <span class="change-val">{{ isPositive ? '+' : '' }}{{ fmt(quote.change) }}</span>
        <span class="change-pct">({{ isPositive ? '+' : '' }}{{ fmt(quote.changePct) }}%)</span>
      </span>
    </div>

    <div class="ohlcv-grid">
      <div class="ohlcv-item">
        <span class="label">Open</span>
        <span class="value">{{ fmt(quote.open) }}</span>
      </div>
      <div class="ohlcv-item">
        <span class="label">High</span>
        <span class="value" style="color:#3fb950">{{ fmt(quote.high) }}</span>
      </div>
      <div class="ohlcv-item">
        <span class="label">Low</span>
        <span class="value" style="color:#f85149">{{ fmt(quote.low) }}</span>
      </div>
      <div class="ohlcv-item">
        <span class="label">Prev Close</span>
        <span class="value">{{ fmt(quote.prevClose) }}</span>
      </div>
    </div>

    <div class="spread-section">
      <div class="spread-row">
        <span class="side-label bid">Bid</span>
        <span class="side-price">{{ fmt(quote.bid) }}</span>
        <div class="spread-bar">
          <div class="spread-fill" :style="{ width: '35%' }" />
        </div>
        <span class="side-price">{{ fmt(quote.ask) }}</span>
        <span class="side-label ask">Ask</span>
      </div>
      <div class="spread-val">Spread: {{ fmt(quote.ask - quote.bid, 4) }}</div>
    </div>

    <div class="info-grid">
      <div class="info-item">
        <span class="label">Volume</span>
        <span class="value">{{ fmtVolume(quote.volume) }}</span>
      </div>
      <div class="info-item">
        <span class="label">Avg Vol</span>
        <span class="value">{{ fmtVolume(quote.avgVolume) }}</span>
      </div>
      <div class="info-item">
        <span class="label">Market Cap</span>
        <span class="value">{{ quote.marketCap }}</span>
      </div>
      <div class="info-item">
        <span class="label">P/E</span>
        <span class="value">{{ quote.pe }}</span>
      </div>
      <div class="info-item">
        <span class="label">EPS</span>
        <span class="value">{{ fmt(quote.eps) }}</span>
      </div>
      <div class="info-item">
        <span class="label">Div Yield</span>
        <span class="value">{{ fmt(quote.dividendYield) }}%</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.quote-panel {
  padding: 12px;
  background: #1a1a2e;
  height: 100%;
  overflow-y: auto;
  font-variant-numeric: tabular-nums;
}

.quote-header {
  margin-bottom: 10px;
}

.header-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.symbol-badge {
  padding: 1px 6px;
  background: #0f3460;
  color: #58a6ff;
  border-radius: 3px;
  font-weight: 700;
  font-size: 13px;
}

.company-name {
  font-size: 13px;
  color: #c9d1d9;
  font-weight: 500;
}

.exchange {
  font-size: 10px;
  color: #3a4a6c;
  padding: 1px 4px;
  border: 1px solid #2a3a5c;
  border-radius: 2px;
}

.price-section {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 14px;
}

.price-section.up .last-price { color: #3fb950; }
.price-section.down .last-price { color: #f85149; }

.last-price {
  font-size: 28px;
  font-weight: 700;
}

.change-info {
  font-size: 13px;
}

.change-val {
  color: inherit;
}

.price-section.up .change-info { color: #3fb950; }
.price-section.down .change-info { color: #f85149; }

.change-pct {
  color: #5a6380;
  font-size: 11px;
}

.ohlcv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2px;
  margin-bottom: 12px;
}

.ohlcv-item {
  display: flex;
  justify-content: space-between;
  padding: 6px 10px;
  background: #16213e;
  border-radius: 3px;
}

.ohlcv-item .label {
  font-size: 11px;
  color: #5a6380;
}

.ohlcv-item .value {
  font-size: 12px;
  color: #c9d1d9;
  font-weight: 500;
}

.spread-section {
  margin-bottom: 12px;
}

.spread-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.side-label {
  font-size: 10px;
  padding: 1px 4px;
  border-radius: 2px;
  min-width: 28px;
  text-align: center;
}

.side-label.bid { color: #3fb950; border: 1px solid #1a4a2e; }
.side-label.ask { color: #f85149; border: 1px solid #4a1a2e; }

.side-price {
  font-size: 12px;
  color: #c9d1d9;
}

.spread-bar {
  flex: 1;
  height: 3px;
  background: #2a3a5c;
  border-radius: 2px;
}

.spread-fill {
  height: 100%;
  background: #58a6ff;
  border-radius: 2px;
  margin: 0 auto;
}

.spread-val {
  text-align: center;
  font-size: 10px;
  color: #5a6380;
  margin-top: 2px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  padding: 5px 10px;
  background: #16213e;
  border-radius: 3px;
}

.info-item .label {
  font-size: 11px;
  color: #5a6380;
}

.info-item .value {
  font-size: 12px;
  color: #c9d1d9;
  font-weight: 500;
}
</style>
