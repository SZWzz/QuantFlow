<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { detectMarket } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const quote = ref<any>(null)
const loading = ref(false)
const loadError = ref('')
let loadSeq = 0
const { fetchWithCache } = usePanelCache()

// Capital flow
interface CapitalFlowData {
  main_flow: number
  super_flow: number
  large_flow: number
  medium_flow: number
  small_flow: number
}
const capitalFlow = ref<CapitalFlowData | null>(null)
const capitalFlowLoading = ref(false)

const maxFlowAbs = computed(() => {
  if (!capitalFlow.value) return 1
  const vals = [
    Math.abs(capitalFlow.value.main_flow),
    Math.abs(capitalFlow.value.super_flow),
    Math.abs(capitalFlow.value.large_flow),
    Math.abs(capitalFlow.value.medium_flow),
    Math.abs(capitalFlow.value.small_flow),
  ]
  return Math.max(...vals, 1)
})

async function fetchQuote(sym: string) {
  const seq = ++loadSeq
  loadError.value = ''
  // TODO: move to store
  loading.value = true
  try {
    const mkt = detectMarket(sym)
    const { data: result } = await fetchWithCache<any>(`quote:${mkt}:${sym}`, () => (window as any).go?.main?.App?.GetQuote(mkt, sym), 60 * 1000)
    if (seq !== loadSeq) return
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
      turnover: snapshot.turnover ?? 0,
      avgVolume: snapshot.avgVolume ?? '--',
      marketCap: snapshot.marketCap ?? '--',
      pe: snapshot.pe ?? '--',
      eps: snapshot.eps ?? '--',
      dividendYield: snapshot.dividendYield ?? '--',
    }
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    quote.value = null
  } finally {
    loading.value = false
  }
  if (seq !== loadSeq) return
  fetchCapitalFlow(sym)
}

async function fetchCapitalFlow(sym: string) {
  const seq = ++loadSeq
  const app = (window as any).go?.main?.App
  if (!app) return
  capitalFlowLoading.value = true
  try {
    const { data: result } = await fetchWithCache<any>(`cap_flow:${sym}`, () => app.GetMACCapitalFlow(sym), 5 * 60 * 1000)
    if (seq !== loadSeq) return
    const cf = Array.isArray(result) ? result[0] : result
    if (cf) {
      capitalFlow.value = {
        main_flow: cf.main_flow || 0,
        super_flow: cf.super_flow || 0,
        large_flow: cf.large_flow || 0,
        medium_flow: cf.medium_flow || 0,
        small_flow: cf.small_flow || 0,
      }
    }
  } catch(e: any) {
    loadError.value = e?.message || String(e)
    capitalFlow.value = null
  } finally {
    capitalFlowLoading.value = false
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
  if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (n >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toLocaleString()
}
function fmtMarketCap(n: any): string {
  if (typeof n !== 'number' || n === 0) return '--'
  if (n >= 1e12) return (n / 1e12).toFixed(2) + '万亿'
  if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (n >= 1e4) return (n / 1e4).toFixed(1) + '万'
  return n.toLocaleString()
}

function fmtFlow(n: number): string {
  if (typeof n !== 'number') return '--'
  const sign = n >= 0 ? '+' : ''
  const abs = Math.abs(n)
  if (abs >= 1e8) return sign + (abs / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return sign + (abs / 1e4).toFixed(1) + '万'
  return sign + n.toFixed(0)
}
function flowBarStyle(val: number): Record<string, string> {
  const pct = (Math.abs(val) / maxFlowAbs.value * 100).toFixed(0)
  const color = val >= 0 ? '#ef4444' : '#22c55e'
  return { width: pct + '%', background: color }
}
</script>

<template>
  <div class="quote-panel">
    <div v-if="loadError" class="error-state" @click="fetchQuote(symbol)">{{ loadError }} ⟳</div>
    <div v-else-if="loading && !quote" class="loading-state">{{ $t('common.loading') }}</div>
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
      <div class="kv-item"><span class="label">{{ $t('quote.turnover') }}</span><span class="value">{{ fmtVolume(quote.turnover) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.market_cap') }}</span><span class="value">{{ fmtMarketCap(quote.marketCap) }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.pe') }}</span><span class="value">{{ typeof quote.pe === 'number' && quote.pe > 0 ? quote.pe.toFixed(2) : '--' }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.eps') }}</span><span class="value">{{ typeof quote.eps === 'number' ? quote.eps.toFixed(2) : '--' }}</span></div>
      <div class="kv-item"><span class="label">{{ $t('quote.dividend_yield') }}</span><span class="value">{{ typeof quote.dividendYield === 'number' ? quote.dividendYield.toFixed(2) + '%' : '--' }}</span></div>
    </div>
    <!-- Capital Flow -->
    <div class="capital-flow-section">
      <div class="cf-label">{{ $t('misc.capital_flow') }}</div>
      <div v-if="capitalFlowLoading && !capitalFlow" class="cf-loading">{{ $t('common.loading') }}</div>
      <div v-else-if="capitalFlow" class="cf-bars">
        <div class="cf-row">
          <span class="cf-name">{{ $t('misc.main_flow') }}</span>
          <div class="cf-bar-track"><span class="cf-bar" :style="flowBarStyle(capitalFlow.main_flow)"></span></div>
          <span class="cf-value" :style="{ color: capitalFlow.main_flow >= 0 ? '#ef4444' : '#22c55e' }">{{ fmtFlow(capitalFlow.main_flow) }}</span>
        </div>
        <div class="cf-row">
          <span class="cf-name">{{ $t('misc.super_flow') }}</span>
          <div class="cf-bar-track"><span class="cf-bar" :style="flowBarStyle(capitalFlow.super_flow)"></span></div>
          <span class="cf-value" :style="{ color: capitalFlow.super_flow >= 0 ? '#ef4444' : '#22c55e' }">{{ fmtFlow(capitalFlow.super_flow) }}</span>
        </div>
        <div class="cf-row">
          <span class="cf-name">{{ $t('misc.large_flow') }}</span>
          <div class="cf-bar-track"><span class="cf-bar" :style="flowBarStyle(capitalFlow.large_flow)"></span></div>
          <span class="cf-value" :style="{ color: capitalFlow.large_flow >= 0 ? '#ef4444' : '#22c55e' }">{{ fmtFlow(capitalFlow.large_flow) }}</span>
        </div>
        <div class="cf-row">
          <span class="cf-name">{{ $t('misc.medium_flow') }}</span>
          <div class="cf-bar-track"><span class="cf-bar" :style="flowBarStyle(capitalFlow.medium_flow)"></span></div>
          <span class="cf-value" :style="{ color: capitalFlow.medium_flow >= 0 ? '#ef4444' : '#22c55e' }">{{ fmtFlow(capitalFlow.medium_flow) }}</span>
        </div>
        <div class="cf-row">
          <span class="cf-name">{{ $t('misc.small_flow') }}</span>
          <div class="cf-bar-track"><span class="cf-bar" :style="flowBarStyle(capitalFlow.small_flow)"></span></div>
          <span class="cf-value" :style="{ color: capitalFlow.small_flow >= 0 ? '#ef4444' : '#22c55e' }">{{ fmtFlow(capitalFlow.small_flow) }}</span>
        </div>
      </div>
      <div v-else class="cf-empty">--</div>
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
.error-state {
  display: flex; align-items: center; justify-content: center;
  height: 100%; color: var(--color-error); font-size: var(--font-sm); cursor: pointer;
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

/* Capital Flow */
.capital-flow-section {
  margin-top: 10px; padding-top: 10px; border-top: 1px solid var(--color-border-strong);
}
.cf-label {
  font-size: 12px; color: var(--color-text-secondary); margin-bottom: 8px; font-weight: 600;
}
.cf-loading, .cf-empty {
  font-size: 11px; color: var(--color-text-tertiary); padding: 4px 0;
}
.cf-bars { display: flex; flex-direction: column; gap: 4px; }
.cf-row {
  display: flex; align-items: center; gap: 6px;
}
.cf-name {
  font-size: 11px; color: var(--color-text-secondary); width: 72px; text-align: right; flex-shrink: 0;
}
.cf-bar-track {
  flex: 1; height: 10px; background: var(--color-bg-subtle); border-radius: 2px;
  position: relative; overflow: hidden;
}
.cf-bar {
  position: absolute; left: 50%; top: 0; height: 100%; border-radius: 2px;
  transform: translateX(-50%);
  min-width: 2px;
}
.cf-value {
  font-size: 11px; font-variant-numeric: tabular-nums; width: 68px; text-align: right; flex-shrink: 0;
}
</style>
