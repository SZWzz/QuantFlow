<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useDataFetch } from '@/lib/composables/useDataFetch'
import { detectMarket } from '@/lib/wails'
import { marketChangeColor } from '@/lib/composables/useMarketColors'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

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
const activeMarket = ref<'CN' | 'HK' | 'US'>(
  (props.params?.market as 'CN' | 'HK' | 'US') || 'CN'
)
const SYMBOLS = computed(() => MARKET_SYMBOLS[activeMarket.value])

const { data: items, loading, execute } = useDataFetch<TickerItem[]>(async () => {
  const results: TickerItem[] = []
  for (const sym of SYMBOLS.value) {
    try {
      const result = await (window as any).go.main.App.GetQuote(detectMarket(sym), sym)
      const snapshot = Array.isArray(result) ? result[0] : result
      results.push({
        symbol: snapshot.symbol ?? sym,
        name: snapshot.name ?? sym,
        price: snapshot.last ?? 0,
        changePct: snapshot.change_pct ?? snapshot.changePct ?? 0,
      })
    } catch {
      // skip failed symbols
    }
  }
  return results
})

onMounted(() => execute())
</script>

<template>
  <div class="ticker-tape-panel">
    <span class="tape-title">{{ $t('watchlist.ticker_tape') }}</span>
    <div class="market-tabs">
      <button v-for="mkt in (['CN', 'HK', 'US'] as const)" :key="mkt"
        :class="['mkt-tab', { active: activeMarket === mkt }]"
        @click="activeMarket = mkt; execute()"
      >{{ mkt }}</button>
    </div>
    <div v-if="loading && !items" class="tape-loading">{{ $t('common.loading') }}</div>
    <div v-else-if="!items" class="tape-loading">{{ $t('common.no_data') }}</div>
    <div v-else class="tape-track-container">
      <div class="tape-track">
        <span v-for="(item, idx) in items" :key="idx" class="tape-item">
          <span class="tape-symbol">{{ item.symbol }}</span>
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :style="{ color: marketChangeColor(item.symbol, item.changePct) }">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
        <!-- Duplicate for seamless loop -->
        <span v-for="(item, idx) in items" :key="'dup-' + idx" class="tape-item">
          <span class="tape-symbol">{{ item.symbol }}</span>
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
.ticker-tape-panel {
  height: 36px;
  display: flex;
  align-items: center;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-base);
  border-bottom: 1px solid var(--color-border, var(--color-border-strong));
  overflow: hidden;
  padding: 0 8px;
  gap: 12px;
}
.tape-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  white-space: nowrap;
  flex-shrink: 0;
}
.market-tabs { display: flex; gap: 2px; flex-shrink: 0; }
.mkt-tab {
  padding: 2px 6px; border: 1px solid var(--color-border-strong); border-radius: 3px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 10px; line-height: 1.2;
}
.mkt-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.15); }
.tape-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.tape-track-container {
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(to right, transparent 0%, black 2%, black 98%, transparent 100%);
}
.tape-track {
  display: inline-flex;
  gap: 24px;
  white-space: nowrap;
  animation: scroll 40s linear infinite;
}
.tape-track:hover {
  animation-play-state: paused;
}
.tape-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.tape-symbol {
  font-weight: 600;
  color: var(--color-text-primary);
}
.tape-name {
  color: var(--color-text-secondary);
  font-size: 11px;
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
