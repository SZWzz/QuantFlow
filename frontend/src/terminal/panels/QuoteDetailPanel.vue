<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const quote = ref<any>(null)
const loading = ref(false)

async function fetchQuote(sym: string) {
  // TODO: move to store
  loading.value = true
  try {
    const result = await (window as any).go.main.App.GetQuote(detectMarket(sym), sym)
    const snapshot = Array.isArray(result) ? result[0] : result
    quote.value = {
      symbol: snapshot.symbol ?? sym,
      name: snapshot.name ?? sym,
      exchange: snapshot.exchange ?? '--',
      last: snapshot.last ?? 0,
      bid: snapshot.bid ?? 0,
      ask: snapshot.ask ?? 0,
      open: snapshot.open ?? 0,
      high: snapshot.high ?? 0,
      low: snapshot.low ?? 0,
      volume: snapshot.volume ?? 0,
      change: snapshot.change ?? 0,
      changePct: snapshot.change_pct ?? snapshot.changePct ?? 0,
      prevClose: snapshot.prevClose ?? ((snapshot.last ?? 0) - (snapshot.change ?? 0)),
      avgVolume: snapshot.avgVolume ?? '--',
      marketCap: snapshot.marketCap ?? '--',
      pe: snapshot.pe ?? '--',
      eps: snapshot.eps ?? '--',
      dividendYield: snapshot.dividendYield ?? '--',
    }
  } catch {
    quote.value = null
  } finally {
    loading.value = false
  }
}

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
    fetchQuote(newSym)
  }
})

// Fetch on mount
onMounted(() => {
  const groupSym = ctx.getGroupSymbol(pg.groupId)
  if (groupSym && groupSym !== symbol.value) {
    symbol.value = groupSym
  }
  fetchQuote(symbol.value)
})

const isPositive = computed(() => (quote.value?.change ?? 0) >= 0)

function fmt(n: any, decimals = 2): string {
  if (typeof n !== 'number') return String(n ?? '--')
  return n.toFixed(decimals)
}
function fmtVolume(n: number): string {
  if (typeof n !== 'number') return '--'
  if (n >= 1e9) return (n / 1e9).toFixed(2) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  return n.toLocaleString()
}
</script>

<template>
  <div class="quote-panel">
    <div v-if="loading && !quote" class="loading-state">{{ $t('common.loading') }}</div>
    <template v-else-if="quote">
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
      <div class="kv-item"><span class="label">{{ $t('kline.open') }}</span><span class="value">{{ fmt(quote.open) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('kline.high') }}</span><span class="value up-val">{{ fmt(quote.high) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('kline.low') }}</span><span class="value down-val">{{ fmt(quote.low) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('kline.prev_close') }}</span><span class="value">{{ fmt(quote.prevClose) }}</span></div>
    </div>
    <div class="spread-section">
      <div class="spread-row">
        <span class="side-label bid">{{ $t('trade.buy') }}</span>
        <span class="side-price">{{ fmt(quote.bid) }}</span>
        <div class="spread-bar"><div class="spread-fill" :style="{ width: '35%' }" /></div>
        <span class="side-price">{{ fmt(quote.ask) }}</span>
        <span class="side-label ask">{{ $t('trade.sell') }}</span>
      </div>
    </div>
    <div class="info-grid">
      <div class="kv-item"><span class="label">{{ $t('quote.volume') }}</span><span class="value">{{ fmtVolume(quote.volume) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.avg_volume') }}</span><span class="value">{{ quote.avgVolume }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.market_cap') }}</span><span class="value">{{ quote.marketCap }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.pe') }}</span><span class="value">{{ quote.pe }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.eps') }}</span><span class="value">{{ quote.eps }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.dividend_yield') }}</span><span class="value">{{ quote.dividendYield === '--' ? '--' : quote.dividendYield + '%' }}</span></div>
    </div>
    </template>
    <div v-else class="loading-state">--</div>
  </div>
</template>

<style scoped>
.quote-panel {
  padding: 12px; background: var(--color-bg-panel); height: 100%; overflow-y: auto;
  font-variant-numeric: tabular-nums;
}
.loading-state {
  display: flex; align-items: center; justify-content: center;
  height: 100%; color: var(--color-text-tertiary); font-size: var(--font-sm);
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
