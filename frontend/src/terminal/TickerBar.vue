<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { detectMarket } from '@/lib/wails'
import { marketChangeColor } from '@/lib/composables/useMarketColors'

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
let pollTimer: ReturnType<typeof setInterval> | null = null

async function resolveNames(results: TickerItem[]) {
  const app = (window as any).go?.main?.App
  if (!app?.SearchSymbols) return
  for (const item of results) {
    if (item.name !== item.symbol) continue
    try {
      const res = await app.SearchSymbols(item.symbol, 1)
      if (Array.isArray(res) && res.length > 0 && res[0].name) {
        item.name = res[0].name
      }
    } catch { /* skip */ }
  }
}

async function fetchTape() {
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
    } catch { /* skip */ }
  }
  await resolveNames(results)
  items.value = results
  loading.value = false
}

function switchMarket(mkt: 'CN' | 'HK' | 'US') {
  activeMarket.value = mkt
  fetchTape()
}

onMounted(() => {
  fetchTape()
  pollTimer = setInterval(fetchTape, 10000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
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
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 10px; line-height: 1.2;
}
.mkt-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.15); }
.tape-track-container {
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(to right, transparent 0%, black 2%, black 98%, transparent 100%);
}
.tape-loading {
  font-size: 11px;
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
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
.tape-name {
  font-weight: 600;
  color: var(--color-text-primary);
}
.tape-price {
  color: var(--color-text-primary);
}
.tape-change {
  font-weight: 500;
}
@keyframes scroll {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}
</style>
