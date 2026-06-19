<script setup lang="ts">
import { ref } from 'vue'
import { useDataStore, type QuoteSnapshot } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbols = ref<string[]>(['AAPL', 'GOOGL', 'MSFT', 'TSLA', 'NVDA', 'AMD', 'BABA', '0700.HK'])
const newSymbol = ref('')

function addSymbol() {
  const sym = newSymbol.value.trim().toUpperCase()
  if (sym && !symbols.value.includes(sym)) symbols.value.push(sym)
  newSymbol.value = ''
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter(s => s !== sym)
}

function selectSymbol(sym: string) {
  ctx.setGroupSymbol(pg.groupId, sym)
}

function formatPrice(p: number): string { return p.toFixed(2) }

function formatChange(c: number, pct: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)} (${sign}${pct.toFixed(2)}%)`
}

const mockQuotes: Record<string, QuoteSnapshot> = {
  'AAPL': { symbol: 'AAPL', last: 195.32, bid: 195.30, ask: 195.35, volume: 32456789, change: 4.05, changePct: 2.12, timestamp: Date.now() },
  'GOOGL': { symbol: 'GOOGL', last: 142.15, bid: 142.10, ask: 142.20, volume: 18765432, change: -1.85, changePct: -1.28, timestamp: Date.now() },
  'MSFT': { symbol: 'MSFT', last: 378.91, bid: 378.85, ask: 378.95, volume: 22109876, change: 3.47, changePct: 0.92, timestamp: Date.now() },
  'TSLA': { symbol: 'TSLA', last: 245.30, bid: 245.25, ask: 245.40, volume: 45678901, change: -5.20, changePct: -2.08, timestamp: Date.now() },
  'NVDA': { symbol: 'NVDA', last: 875.28, bid: 875.20, ask: 875.35, volume: 28901234, change: 12.45, changePct: 1.44, timestamp: Date.now() },
  'AMD': { symbol: 'AMD', last: 168.77, bid: 168.70, ask: 168.80, volume: 19876543, change: 2.33, changePct: 1.40, timestamp: Date.now() },
  'BABA': { symbol: 'BABA', last: 88.45, bid: 88.40, ask: 88.50, volume: 15432109, change: -0.89, changePct: -1.00, timestamp: Date.now() },
  '0700.HK': { symbol: '0700.HK', last: 440.20, bid: 440.00, ask: 440.40, volume: 12543210, change: 8.40, changePct: 1.95, timestamp: Date.now() },
}
</script>

<template>
  <div class="watchlist-panel">
    <div class="panel-toolbar">
      <input v-model="newSymbol" type="text" placeholder="Add symbol..." class="symbol-input" @keyup.enter="addSymbol" />
      <button class="add-btn" @click="addSymbol">+</button>
    </div>
    <div class="symbol-list">
      <div
        v-for="sym in symbols" :key="sym"
        class="symbol-row"
        :class="{
          up: (mockQuotes[sym]?.change || 0) >= 0,
          down: (mockQuotes[sym]?.change || 0) < 0,
          active: ctx.getGroupSymbol(pg.groupId) === sym,
        }"
        @click="selectSymbol(sym)"
      >
        <div class="symbol-info">
          <span class="symbol-name">{{ sym }}</span>
        </div>
        <div v-if="mockQuotes[sym]" class="symbol-price">
          <span class="last">{{ formatPrice(mockQuotes[sym].last) }}</span>
          <span class="change">{{ formatChange(mockQuotes[sym].change, mockQuotes[sym].changePct) }}</span>
        </div>
        <div v-else class="symbol-price"><span class="no-data">--</span></div>
        <button class="remove-btn" @click.stop="removeSymbol(sym)" title="Remove">✕</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.watchlist-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
}
.panel-toolbar {
  display: flex; gap: 4px; padding: 8px;
  border-bottom: 1px solid var(--color-border);
}
.symbol-input {
  flex: 1; padding: 4px 8px;
  background: var(--color-bg-input); border: 1px solid var(--color-border);
  border-radius: var(--radius-sm); color: var(--color-text-primary);
  font-size: var(--font-xs); outline: none;
}
.symbol-input:focus { border-color: var(--color-accent); }
.add-btn {
  padding: 4px 10px; background: var(--color-bg-subtle);
  border: 1px solid var(--color-border); color: var(--color-accent);
  border-radius: var(--radius-sm); cursor: pointer; font-size: 14px; font-weight: bold;
}
.symbol-list { flex: 1; overflow-y: auto; }
.symbol-row {
  display: flex; align-items: center; padding: 6px 8px;
  border-bottom: 1px solid var(--color-border-subtle);
  cursor: pointer; transition: background var(--transition-fast);
}
.symbol-row:hover { background: var(--color-bg-hover); }
.symbol-row.active {
  background: var(--color-accent-soft);
  border-left: 2px solid var(--color-accent);
  padding-left: 6px;
}
.symbol-row.up .symbol-name { color: var(--color-up); }
.symbol-row.down .symbol-name { color: var(--color-down); }
.symbol-row.up .last { color: var(--color-up); }
.symbol-row.down .last { color: var(--color-down); }
.symbol-info { flex: 1; }
.symbol-name { font-weight: 600; font-size: var(--font-base); color: var(--color-text-primary); }
.symbol-price { text-align: right; margin-right: 8px; }
.last { font-size: var(--font-base); font-weight: 500; font-variant-numeric: tabular-nums; }
.change { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.symbol-row.up .change { color: var(--color-up); }
.symbol-row.down .change { color: var(--color-down); }
.no-data { font-size: var(--font-sm); color: var(--color-text-tertiary); }
.remove-btn {
  background: none; border: none; color: var(--color-text-tertiary);
  cursor: pointer; font-size: 12px; padding: 2px 4px; opacity: 0;
  transition: opacity var(--transition-fast);
}
.symbol-row:hover .remove-btn { opacity: 0.6; }
.remove-btn:hover { opacity: 1; color: var(--color-down); }
</style>
