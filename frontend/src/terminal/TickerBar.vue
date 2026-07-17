<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { detectMarket } from '@/lib/wails'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { useWebSocket } from '@/lib/composables/useWebSocket'

interface TickerItem {
  symbol: string
  name: string
  price: number
  changePct: number
}

const MARKET_SYMBOLS: Record<string, string[]> = {
  CN: ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276'],
  HK: ['00700.HK', '09988.HK', '00388.HK', '00941.HK', '03690.HK', '09888.HK', '01810.HK', '01211.HK'],
  US: ['AAPL', 'MSFT', 'NVDA', 'TSLA', 'AMZN', 'GOOGL', 'META', 'SPY'],
}
const activeMarket = ref<'CN' | 'HK' | 'US'>('CN')
const SYMBOLS = computed(() => MARKET_SYMBOLS[activeMarket.value])

const items = ref<TickerItem[]>([])
const loading = ref(true)
const ws = useWebSocket()
const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws/market`
const wsTopics = computed(() => SYMBOLS.value.map(sym => `market:quote:${detectMarket(sym)}:${sym}`))

// Resolve names for symbols that don't have them yet
async function resolveNames() {
  const app = (window as any).go?.main?.App
  if (!app?.SearchSymbols) return
  for (const item of items.value) {
    if (item.name !== item.symbol) continue
    try {
      const res = await app.SearchSymbols(item.symbol, 1)
      if (Array.isArray(res) && res.length > 0 && res[0].name) {
        item.name = res[0].name
      }
    } catch { /* skip */ }
  }
}

// Initial load via IPC (gets full data + names)
async function initialLoad() {
  const results: TickerItem[] = []
  for (const sym of SYMBOLS.value) {
    try {
      const result = await (window as any).go.main.App.GetQuote(detectMarket(sym), sym)
      const snapshot = Array.isArray(result) ? result[0] : result
      results.push({
        symbol: snapshot.symbol ?? sym,
        name: snapshot.name || sym,
        price: snapshot.last ?? 0,
        changePct: snapshot.change_pct ?? snapshot.changePct ?? 0,
      })
    } catch { results.push({ symbol: sym, name: sym, price: 0, changePct: 0 }) }
  }
  items.value = results
  loading.value = false
  resolveNames()
}

function switchMarket(mkt: 'CN' | 'HK' | 'US') {
  activeMarket.value = mkt
  initialLoad()
}

// WS handler: update individual quote in place
function handleWSQuote(topic: string, data: any) {
  const parts = topic.split(':')
  const symbol = parts[parts.length - 1]
  const idx = items.value.findIndex(i => i.symbol === symbol)
  if (idx < 0) return
  if (data.last !== undefined) items.value[idx].price = data.last
  if (data.changePct !== undefined) items.value[idx].changePct = data.changePct
  if (data.name && items.value[idx].name === items.value[idx].symbol) {
    items.value[idx].name = data.name
  }
}

onMounted(() => {
  initialLoad()
  ws.connect(wsUrl, wsTopics.value)
  ws.onMessage('*', (msg: any) => {
    if (msg.topic?.startsWith('market:quote:')) {
      handleWSQuote(msg.topic, msg.data)
    }
  })
})

onUnmounted(() => {
  ws.disconnect()
})
</script>

<template>
  <div class="ticker-bar">
    <div class="market-tabs">
      <button v-for="mkt in (['CN', 'HK', 'US'] as const)" :key="mkt"
        :class="['mkt-tab', { active: activeMarket === mkt }]"
        @click="switchMarket(mkt)"
      >{{ mkt }}</button>
    </div>
    <div class="tape-track-container">
      <div v-if="loading" class="tape-loading">...</div>
      <div v-else class="tape-track">
        <span v-for="(item, idx) in items" :key="idx" class="tape-item">
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :style="{ color: marketChangeColor(item.symbol, item.changePct) }">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
        <span v-for="(item, idx) in items" :key="'dup-' + idx" class="tape-item">
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :style="{ color: marketChangeColor(item.symbol, item.changePct) }">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ticker-bar {
  display: flex;
  align-items: center;
  height: 30px;
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border);
  padding: 0 12px;
  gap: 10px;
  overflow: hidden;
  flex-shrink: 0;
}
.market-tabs { display: flex; gap: 2px; flex-shrink: 0; }
.mkt-tab {
  padding: 2px 6px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px; line-height: 1.2;
}
.mkt-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: var(--color-accent-soft); }
.tape-track-container {
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(to right, transparent 0%, black 2%, black 98%, transparent 100%);
}
.tape-loading {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}
.tape-track {
  display: inline-flex;
  gap: 20px;
  white-space: nowrap;
  animation: scroll 40s linear infinite;
}
.tape-track:hover {
  animation-play-state: paused;
}
.tape-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: var(--font-xs);
  font-variant-numeric: tabular-nums;
}
.tape-name {
  font-weight: 600;
  color: var(--color-text-primary);
}
.tape-price {
  color: var(--color-text-primary);
  font-family: var(--font-mono);
}
.tape-change {
  font-weight: 500;
}
@keyframes scroll {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}
</style>
