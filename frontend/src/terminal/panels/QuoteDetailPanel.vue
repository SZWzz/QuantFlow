<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
    regenerateQuote(newSym)
  }
})

const quote = ref(generateQuote(symbol.value))

function generateQuote(sym: string) {
  const basePrices: Record<string, number> = { 'AAPL': 195, 'GOOGL': 142, 'MSFT': 378, 'TSLA': 245, 'NVDA': 875, 'AMD': 168, 'BABA': 88, '0700.HK': 440 }
  const base = basePrices[sym] || 100
  const last = +(base + (Math.random() - 0.5) * base * 0.05).toFixed(2)
  const prevClose = +(base + (Math.random() - 0.5) * base * 0.04).toFixed(2)
  const change = +(last - prevClose).toFixed(2)
  return {
    symbol: sym, name: sym, exchange: 'MOCK', last, bid: +(last - 0.02).toFixed(2), ask: +(last + 0.02).toFixed(2),
    open: prevClose, high: +(last * 1.02).toFixed(2), low: +(last * 0.98).toFixed(2), prevClose,
    volume: Math.floor(Math.random() * 5e7) + 1e7, avgVolume: 28500000,
    marketCap: 'N/A', pe: 0, eps: 0, dividendYield: 0, change, changePct: +((change / prevClose) * 100).toFixed(2),
  }
}

function regenerateQuote(sym: string) { quote.value = generateQuote(sym) }

const isPositive = computed(() => quote.value.change >= 0)

function fmt(n: number, decimals = 2): string { return n.toFixed(decimals) }
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
      <div class="kv-item"><span class="label">Open</span><span class="value">{{ fmt(quote.open) }}</span></div>
      <div class="kv-item"><span class="label">High</span><span class="value up-val">{{ fmt(quote.high) }}</span></div>
      <div class="kv-item"><span class="label">Low</span><span class="value down-val">{{ fmt(quote.low) }}</span></div>
      <div class="kv-item"><span class="label">Prev Close</span><span class="value">{{ fmt(quote.prevClose) }}</span></div>
    </div>
    <div class="spread-section">
      <div class="spread-row">
        <span class="side-label bid">Bid</span>
        <span class="side-price">{{ fmt(quote.bid) }}</span>
        <div class="spread-bar"><div class="spread-fill" :style="{ width: '35%' }" /></div>
        <span class="side-price">{{ fmt(quote.ask) }}</span>
        <span class="side-label ask">Ask</span>
      </div>
    </div>
    <div class="info-grid">
      <div class="kv-item"><span class="label">Volume</span><span class="value">{{ fmtVolume(quote.volume) }}</span></div>
      <div class="kv-item"><span class="label">Avg Vol</span><span class="value">{{ fmtVolume(quote.avgVolume) }}</span></div>
      <div class="kv-item"><span class="label">Market Cap</span><span class="value">{{ quote.marketCap }}</span></div>
      <div class="kv-item"><span class="label">P/E</span><span class="value">{{ quote.pe || '--' }}</span></div>
      <div class="kv-item"><span class="label">EPS</span><span class="value">{{ quote.eps || '--' }}</span></div>
      <div class="kv-item"><span class="label">Div Yield</span><span class="value">{{ quote.dividendYield || '--' }}%</span></div>
    </div>
  </div>
</template>

<style scoped>
.quote-panel {
  padding: 12px; background: var(--color-bg-panel); height: 100%; overflow-y: auto;
  font-variant-numeric: tabular-nums;
}
.quote-header { margin-bottom: 10px; }
.header-main { display: flex; align-items: center; gap: 8px; }
.symbol-badge {
  padding: 1px 6px; background: var(--color-accent-soft); color: var(--color-accent);
  border-radius: var(--radius-sm); font-weight: 700; font-size: var(--font-base);
}
.company-name { font-size: var(--font-base); color: var(--color-text-primary); font-weight: 500; }
.exchange { font-size: 10px; color: var(--color-text-tertiary); padding: 1px 4px; border: 1px solid var(--color-border); border-radius: 2px; }
.price-section { display: flex; align-items: baseline; gap: 10px; margin-bottom: 14px; }
.price-section.up .last-price { color: var(--color-up); }
.price-section.down .last-price { color: var(--color-down); }
.last-price { font-size: 28px; font-weight: 700; }
.change-info { font-size: var(--font-base); }
.price-section.up .change-info { color: var(--color-up); }
.price-section.down .change-info { color: var(--color-down); }
.change-pct { color: var(--color-text-tertiary); font-size: var(--font-xs); }
.ohlcv-grid, .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2px; margin-bottom: 12px; }
.kv-item {
  display: flex; justify-content: space-between; padding: 6px 10px;
  background: var(--color-bg-subtle); border-radius: var(--radius-sm);
}
.kv-item .label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.kv-item .value { font-size: var(--font-sm); color: var(--color-text-primary); font-weight: 500; }
.up-val { color: var(--color-up) !important; }
.down-val { color: var(--color-down) !important; }
.spread-section { margin-bottom: 12px; }
.spread-row { display: flex; align-items: center; gap: 8px; }
.side-label { font-size: 10px; padding: 1px 4px; border-radius: 2px; min-width: 28px; text-align: center; }
.side-label.bid { color: var(--color-up); border: 1px solid var(--color-up-soft); }
.side-label.ask { color: var(--color-down); border: 1px solid var(--color-down-soft); }
.side-price { font-size: var(--font-sm); color: var(--color-text-primary); }
.spread-bar { flex: 1; height: 3px; background: var(--color-border); border-radius: 2px; }
.spread-fill { height: 100%; background: var(--color-accent); border-radius: 2px; margin: 0 auto; }
</style>
