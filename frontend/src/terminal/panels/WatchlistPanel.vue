<script setup lang="ts">
import { ref } from 'vue'
import { useDataStore, type QuoteSnapshot } from '@/stores/data'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const dataStore = useDataStore()

const symbols = ref<string[]>(['AAPL', 'GOOGL', 'MSFT', 'TSLA', 'NVDA', 'AMD', 'BABA', '0700.HK'])
const newSymbol = ref('')

function addSymbol() {
  const sym = newSymbol.value.trim().toUpperCase()
  if (sym && !symbols.value.includes(sym)) {
    symbols.value.push(sym)
  }
  newSymbol.value = ''
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter((s) => s !== sym)
}

function getQuote(sym: string): QuoteSnapshot | undefined {
  return dataStore.getQuote(sym)
}

function formatPrice(p: number): string {
  return p.toFixed(2)
}

function formatChange(c: number, pct: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)} (${sign}${pct.toFixed(2)}%)`
}

// Mock data for display until M6 MarketDataHub is ready
const mockQuotes: Record<string, QuoteSnapshot> = {
  'AAPL': { symbol: 'AAPL', last: 195.32, bid: 195.30, ask: 195.35, volume: 32456789, change: 4.05, changePct: 2.12, timestamp: Date.now() },
  'GOOGL': { symbol: 'GOOGL', last: 142.15, bid: 142.10, ask: 142.20, volume: 18765432, change: -1.85, changePct: -1.28, timestamp: Date.now() },
  'MSFT': { symbol: 'MSFT', last: 378.91, bid: 378.85, ask: 378.95, volume: 22109876, change: 3.47, changePct: 0.92, timestamp: Date.now() },
  'TSLA': { symbol: 'TSLA', last: 245.30, bid: 245.25, ask: 245.40, volume: 45678901, change: -5.20, changePct: -2.08, timestamp: Date.now() },
  'NVDA': { symbol: 'NVDA', last: 875.28, bid: 875.20, ask: 875.35, volume: 28901234, change: 12.45, changePct: 1.44, timestamp: Date.now() },
}
</script>

<template>
  <div class="watchlist-panel">
    <div class="panel-toolbar">
      <input
        v-model="newSymbol"
        type="text"
        placeholder="Add symbol..."
        class="symbol-input"
        @keyup.enter="addSymbol"
      />
      <button class="add-btn" @click="addSymbol">+</button>
    </div>
    <div class="symbol-list">
      <div
        v-for="sym in symbols"
        :key="sym"
        class="symbol-row"
        :class="{ up: (mockQuotes[sym]?.change || 0) >= 0, down: (mockQuotes[sym]?.change || 0) < 0 }"
      >
        <div class="symbol-info">
          <span class="symbol-name">{{ sym }}</span>
        </div>
        <div v-if="mockQuotes[sym]" class="symbol-price">
          <span class="last">{{ formatPrice(mockQuotes[sym].last) }}</span>
          <span class="change">{{ formatChange(mockQuotes[sym].change, mockQuotes[sym].changePct) }}</span>
        </div>
        <div v-else class="symbol-price">
          <span class="no-data">--</span>
        </div>
        <button class="remove-btn" @click="removeSymbol(sym)" title="Remove">✕</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.watchlist-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
}

.panel-toolbar {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid #0f3460;
}

.symbol-input {
  flex: 1;
  padding: 4px 8px;
  background: #0f2137;
  border: 1px solid #1a3a5c;
  border-radius: 4px;
  color: #c9d1d9;
  font-size: 12px;
  outline: none;
}

.symbol-input:focus {
  border-color: #58a6ff;
}

.add-btn {
  padding: 4px 10px;
  background: #0f3460;
  border: 1px solid #1a4a7c;
  color: #58a6ff;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: bold;
}

.symbol-list {
  flex: 1;
  overflow-y: auto;
}

.symbol-row {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-bottom: 1px solid #0f2137;
  transition: background 0.1s;
}

.symbol-row:hover {
  background: rgba(88, 166, 255, 0.05);
}

.symbol-row.up .symbol-name { color: #3fb950; }
.symbol-row.down .symbol-name { color: #f85149; }
.symbol-row.up .last { color: #3fb950; }
.symbol-row.down .last { color: #f85149; }

.symbol-info {
  flex: 1;
}

.symbol-name {
  font-weight: 600;
  font-size: 13px;
  color: #e0e0e0;
}

.symbol-price {
  text-align: right;
  margin-right: 8px;
}

.last {
  font-size: 13px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.change {
  display: block;
  font-size: 11px;
  color: #5a6380;
}

.symbol-row.up .change { color: #3fb950; }
.symbol-row.down .change { color: #f85149; }

.no-data {
  font-size: 12px;
  color: #3a4a6c;
}

.remove-btn {
  background: none;
  border: none;
  color: #2a3a5c;
  cursor: pointer;
  font-size: 12px;
  padding: 2px 4px;
}

.remove-btn:hover {
  color: #f85149;
}
</style>
