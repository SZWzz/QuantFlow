<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useDataFetch } from '@/lib/composables/useDataFetch'
import { detectMarket } from '@/lib/wails'
import { PanelHeader, LoadingState } from '@/terminal/components/panel'

// 涨跌色走 CSS class + token，避免高频渲染下每项一次 getComputedStyle
const changeClass = (pct: number) => (pct > 0 ? 'is-up' : pct < 0 ? 'is-down' : 'is-flat')

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
      const result = await (window as any).go?.main?.App?.GetQuote(detectMarket(sym), sym)
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

function switchMarket(mkt: 'CN' | 'HK' | 'US') {
  activeMarket.value = mkt
  execute()
}

onMounted(() => execute())
</script>

<template>
  <div class="ticker-tape-panel">
    <PanelHeader title="快讯">
      <template #controls>
        <button v-for="mkt in (['CN', 'HK', 'US'] as const)" :key="mkt"
          :class="['btn btn-sm', { 'btn-primary': activeMarket === mkt }]"
          @click="switchMarket(mkt)"
        >{{ mkt }}</button>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading && !items" type="inline" />
    <div v-else-if="!items" class="tape-empty">{{ $t('common.no_data') }}</div>
    <div v-else class="tape-track-container">
      <div class="tape-track">
        <span v-for="(item, idx) in items" :key="idx" class="tape-item">
          <span class="tape-symbol">{{ item.symbol }}</span>
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :class="changeClass(item.changePct)">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
        <!-- Duplicate for seamless loop -->
        <span class="tape-clone" aria-hidden="true">
          <span v-for="(item, idx) in items" :key="'dup-' + idx" class="tape-item">
            <span class="tape-symbol">{{ item.symbol }}</span>
            <span class="tape-name">{{ item.name }}</span>
            <span class="tape-price">{{ item.price.toFixed(2) }}</span>
            <span class="tape-change" :class="changeClass(item.changePct)">
              {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
            </span>
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ticker-tape-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.tape-empty { flex: 1; display: flex; align-items: center; justify-content: center; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.tape-track-container { flex: 1; overflow: hidden; mask-image: linear-gradient(to right, transparent 0%, black 2%, black 98%, transparent 100%); }
.tape-track { display: inline-flex; gap: var(--space-xl); white-space: nowrap; animation: scroll 40s linear infinite; }
.tape-track:hover { animation-play-state: paused; }
.tape-item { display: inline-flex; align-items: center; gap: var(--space-sm); font-size: var(--font-xs); font-variant-numeric: tabular-nums; }
.tape-symbol { font-weight: 600; color: var(--color-text-primary); }
.tape-name { color: var(--color-text-secondary); font-size: var(--font-xs); }
.tape-price { color: var(--color-text-primary); }
.tape-change { font-weight: 500; }
.tape-change.is-up { color: var(--color-up); }
.tape-change.is-down { color: var(--color-down); }
.tape-change.is-flat { color: var(--color-text-secondary); }

.tape-clone { display: contents; }

@keyframes scroll {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}

@media (prefers-reduced-motion: reduce) {
  .tape-track { animation: none; }
  .tape-clone { display: none; }
}
</style>
