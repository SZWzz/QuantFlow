<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const name = ref('')
const lastPrice = ref(0)
const change = ref(0)
const changePct = ref(0)
const bidLevels = ref<{ price: number; size: number }[]>([])
const askLevels = ref<{ price: number; size: number }[]>([])
const trades = ref<{ time: string; price: number; volume: number; side: 'B' | 'S' }[]>([])
const isSimulated = ref(true) // A-share no free L2; show simulated from bid/ask

const maxSize = computed(() => {
  const all = [...bidLevels.value, ...askLevels.value]
  const m = Math.max(...all.map(l => l.size), 1)
  return m
})

function formatSize(size: number): string {
  if (size >= 10000) return (size / 10000).toFixed(1) + '万'
  return size.toFixed(0)
}

function barWidth(size: number): string {
  return ((size / maxSize.value) * 100).toFixed(0) + '%'
}

function buildSimulatedLevels(baseBid: number, baseAsk: number) {
  // Simulate 5-level depth from single bid/ask
  const bids: { price: number; size: number }[] = []
  const asks: { price: number; size: number }[] = []
  const step = (baseAsk - baseBid) / 5 || 0.02
  for (let i = 0; i < 5; i++) {
    bids.push({ price: +(baseBid - i * step).toFixed(2), size: Math.round(1000 / (i + 1)) })
    asks.push({ price: +(baseAsk + i * step).toFixed(2), size: Math.round(800 / (i + 1)) })
  }
  bidLevels.value = bids
  askLevels.value = asks
}

async function refresh() {
  const app = (window as any).go?.main?.App
  if (!app) return
  try {
    const result = await app.GetQuote(detectMarket(symbol.value), symbol.value)
    const snapshot = Array.isArray(result) ? result[0] : result
    if (!snapshot) return
    name.value = snapshot.name || symbol.value
    lastPrice.value = snapshot.last || 0
    change.value = snapshot.change || 0
    changePct.value = snapshot.change_pct || snapshot.changePct || 0
    if (snapshot.bid > 0 && snapshot.ask > 0) {
      buildSimulatedLevels(snapshot.bid, snapshot.ask)
    }
    // No real tick data for A-shares via free APIs
    trades.value = []
    isSimulated.value = true
  } catch {
    // silent
  }
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
  refresh()
}

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    refresh()
  }
})

onMounted(refresh)
</script>

<template>
  <div class="market-depth-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.depth') }}</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          :placeholder="$t('quote.symbol') + '...'"
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="refresh">⟳</button>
      </div>
    </div>

    <div class="last-price-row">
      <span class="price-label">{{ name || symbol }}</span>
      <span class="price-value" :style="{ color: change >= 0 ? '#ef4444' : '#22c55e' }">
        {{ lastPrice.toFixed(2) }}
      </span>
      <span class="price-change" :style="{ color: change >= 0 ? '#ef4444' : '#22c55e' }">
        {{ change >= 0 ? '+' : '' }}{{ change.toFixed(2) }} ({{ changePct >= 0 ? '+' : '' }}{{ changePct.toFixed(2) }}%)
      </span>
      <span v-if="isSimulated" class="sim-badge">{{ $t('misc.simulated_depth') }}</span>
    </div>

    <!-- Section A: Order Book -->
    <div class="orderbook">
      <div class="ob-header">
        <span class="ob-cell price-col">{{ $t('quote.bid') }}</span>
        <span class="ob-cell size-col">{{ $t('common.size') }}</span>
        <span class="ob-cell bar-col"></span>
        <span class="ob-cell price-col">{{ $t('quote.ask') }}</span>
        <span class="ob-cell size-col">{{ $t('common.size') }}</span>
        <span class="ob-cell bar-col"></span>
      </div>
      <div class="ob-rows">
        <div v-for="i in 5" :key="'lvl-' + i" class="ob-row">
          <!-- Bid side -->
          <template v-if="bidLevels[5 - i]">
            <span class="ob-cell price-col bid-price">{{ bidLevels[5 - i].price.toFixed(2) }}</span>
            <span class="ob-cell size-col bid-size">{{ formatSize(bidLevels[5 - i].size) }}</span>
            <span class="ob-cell bar-col">
              <span class="ob-bar bid-bar" :style="{ width: barWidth(bidLevels[5 - i].size) }"></span>
            </span>
          </template>
          <template v-else>
            <span class="ob-cell price-col"></span>
            <span class="ob-cell size-col"></span>
            <span class="ob-cell bar-col"></span>
          </template>
          <!-- Ask side -->
          <template v-if="askLevels[i - 1]">
            <span class="ob-cell price-col ask-price">{{ askLevels[i - 1].price.toFixed(2) }}</span>
            <span class="ob-cell size-col ask-size">{{ formatSize(askLevels[i - 1].size) }}</span>
            <span class="ob-cell bar-col">
              <span class="ob-bar ask-bar" :style="{ width: barWidth(askLevels[i - 1].size) }"></span>
            </span>
          </template>
          <template v-else>
            <span class="ob-cell price-col"></span>
            <span class="ob-cell size-col"></span>
            <span class="ob-cell bar-col"></span>
          </template>
        </div>
      </div>
    </div>

    <!-- Section B: Recent Trades -->
    <div class="trades-section">
      <div class="trades-label">{{ $t('misc.recent_trades') }}</div>
      <div class="trades-list">
        <div v-for="(t, idx) in trades" :key="idx" class="trade-row">
          <span class="trade-time">{{ t.time }}</span>
          <span class="trade-price" :style="{ color: t.side === 'B' ? '#ef4444' : '#22c55e' }">{{ t.price.toFixed(2) }}</span>
          <span class="trade-volume">{{ t.volume }}</span>
          <span class="trade-side" :class="t.side === 'B' ? 'side-buy' : 'side-sell'">{{ t.side === 'B' ? 'B' : 'S' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.market-depth-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; align-items: center; }
.symbol-input {
  width: 110px; padding: 4px 8px; border: 1px solid var(--color-border-strong);
  border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}

/* Last Price */
.last-price-row {
  display: flex; align-items: baseline; gap: 10px;
  padding: 6px 0; margin-bottom: 8px; border-bottom: 1px solid var(--color-border-strong);
}
.price-label { font-size: 12px; color: var(--color-text-secondary); }
.price-value { font-size: 18px; font-weight: 700; }
.price-change { font-size: 12px; }

/* Order Book */
.orderbook { margin-bottom: 10px; }
.ob-header {
  display: flex; padding: 2px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase;
}
.ob-rows { font-size: 12px; font-variant-numeric: tabular-nums; }
.ob-row { display: flex; padding: 1px 0; }
.ob-cell { display: flex; align-items: center; }
.price-col { width: 70px; }
.size-col { width: 54px; text-align: right; }
.bar-col { flex: 1; position: relative; height: 16px; }
.ob-bar { position: absolute; right: 0; top: 2px; height: 12px; border-radius: 2px; opacity: 0.3; }
.bid-bar { background: #22c55e; }
.ask-bar { background: #ef4444; }
.bid-price { color: #22c55e; }
.ask-price { color: #ef4444; }
.bid-size { color: var(--color-text-secondary); }
.ask-size { color: var(--color-text-secondary); }

/* Trades */
.trades-section { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.trades-label { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.trades-list { flex: 1; overflow-y: auto; scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent; }
.trade-row {
  display: flex; justify-content: space-between; padding: 1px 0;
  font-size: 11px; font-variant-numeric: tabular-nums;
}
.trade-time { color: var(--color-text-tertiary); width: 56px; }
.trade-price { width: 70px; text-align: right; }
.trade-volume { width: 60px; text-align: right; color: var(--color-text-secondary); }
.trade-side { width: 24px; text-align: center; font-weight: 600; font-size: 10px; }
.side-buy { color: #ef4444; }
.side-sell { color: #22c55e; }
</style>
