<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'

defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()

const symbol = ref('AAPL')
const { name } = useStockName(symbol)
const side = ref<'buy' | 'sell'>('buy')
const orderType = ref<'market' | 'limit' | 'stop'>('limit')
const quantity = ref(100)
const price = ref(195.50)
const stopPrice = ref(190.00)
const broker = ref<'paper' | 'binance' | 'futu' | 'ibkr' | 'alpaca'>('paper')
const lastPrice = ref(0)
const quoteLoading = ref(false)
const loadError = ref('')

function detectMarket(sym: string): string {
  // Crypto: explicit pairs like BTC/USDT, ETH/USDT
  if (sym.includes('/')) return 'CRYPTO'
  // US: 1-5 uppercase letters (AAPL, TSLA, BRK.A)
  if (/^[A-Z]{1,5}(\.[A-Z])?$/.test(sym)) return 'US'
  // HK: 4-5 digits with leading zeros (00001, 00700)
  if (/^\d{4,5}$/.test(sym)) return 'HK'
  // Default: CN (6 digits, or any other format)
  return 'CN'
}

const estimatedTotal = computed(() => {
  const p = orderType.value === 'market' ? (lastPrice.value || price.value) : price.value
  return quantity.value * p
})

async function fetchQuote() {
  const app = (window as any).go?.main?.App
  if (!app?.GetQuote) return
  quoteLoading.value = true
  loadError.value = ''
  try {
    const market = detectMarket(symbol.value)
    const { data: result } = await fetchWithCache<any>(`quote:${symbol.value}`, () => app.GetQuote(market, symbol.value), 60 * 1000)
    const quote = Array.isArray(result) ? result[0] : result
    if (quote?.last) {
      lastPrice.value = quote.last
      price.value = quote.last
    }
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  } finally {
    quoteLoading.value = false
  }
}

watch(symbol, () => {
  fetchQuote()
})

onMounted(() => {
  fetchQuote()
})

function placeOrder() {
  try {
    const app = (window as any).go?.main?.App
    if (app?.PlaceOrder) {
      app.PlaceOrder(
        symbol.value, side.value, orderType.value, broker.value,
        quantity.value,
        orderType.value === 'market' ? 0 : price.value
      )
    }
  } catch (e) {
    console.warn('PlaceOrder not available:', e)
  }
}
</script>

<template>
  <div class="order-panel">
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div class="order-form">
      <div class="form-group">
        <label>{{ $t('quote.symbol') }}
          <span v-if="name" class="quote-status">{{ name }}</span>
        </label>
        <input v-model="symbol" type="text" class="form-input" data-testid="order-symbol-input" />
      </div>

      <div class="form-group">
        <label>{{ $t('trade.broker') }}</label>
        <select v-model="broker" class="form-input">
          <option value="paper">{{ $t('trade.paper') }}</option>
          <option value="binance">{{ $t('trade.binance') }}</option>
          <option value="futu">{{ $t('trade.futu') }}</option>
          <option value="ibkr">IBKR</option>
          <option value="alpaca">Alpaca</option>
        </select>
      </div>

      <div class="side-toggle">
        <button :class="{ active: side === 'buy' }" @click="side = 'buy'" data-testid="order-side-buy">{{ $t('trade.buy') }}</button>
        <button :class="{ active: side === 'sell' }" @click="side = 'sell'" data-testid="order-side-sell">{{ $t('trade.sell') }}</button>
      </div>

      <div class="form-group">
        <label>{{ $t('trade.order_type') }}</label>
        <select v-model="orderType" class="form-input" data-testid="order-type-select">
          <option value="market">{{ $t('trade.market') }}</option>
          <option value="limit">{{ $t('trade.limit') }}</option>
          <option value="stop">{{ $t('trade.stop') }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>{{ $t('trade.quantity') }}</label>
        <input v-model.number="quantity" type="number" min="1" class="form-input" data-testid="order-quantity-input" />
      </div>

      <div v-if="orderType !== 'market'" class="form-group">
        <label>{{ $t('trade.price') }}
          <span v-if="quoteLoading" class="quote-status">{{ $t('common.loading') }}</span>
          <span v-else-if="lastPrice > 0" class="quote-status">(实时)</span>
        </label>
        <input v-model.number="price" type="number" step="0.01" class="form-input" data-testid="order-price-input" />
      </div>

      <div v-if="orderType === 'stop'" class="form-group">
        <label>{{ $t('trade.stop_price') }}</label>
        <input v-model.number="stopPrice" type="number" step="0.01" class="form-input" />
      </div>

      <div class="estimated">
        <span>{{ $t('trade.estimated') }}</span>
        <span class="total-value">${{ estimatedTotal.toLocaleString() }}</span>
      </div>

      <button
        class="place-order-btn"
        :class="side"
        @click="placeOrder"
        data-testid="order-place-btn"
      >
        {{ side === 'buy' ? $t('trade.buy') : $t('trade.sell') }} {{ symbol }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: var(--color-up-soft); color: var(--color-up); font-size: 12px; }
.order-panel { padding: 12px; background: var(--bg); height: 100%; }
.order-form { display: flex; flex-direction: column; gap: 10px; }

.form-group { display: flex; flex-direction: column; gap: 3px; }
.form-group label { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
.form-input {
  padding: 6px 8px; background: var(--input); border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: 13px; outline: none;
}
.form-input:focus { border-color: var(--accent); }

.quote-status { font-size: 10px; color: var(--color-accent); margin-left: 4px; }

.side-toggle { display: flex; gap: 0; }
.side-toggle button {
  flex: 1; padding: 8px; border: 1px solid var(--border); background: var(--input); color: var(--muted);
  font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.15s;
}
.side-toggle button:first-child { border-radius: var(--radius-sm) 0 0 4px; }
.side-toggle button:last-child { border-radius: 0 4px 4px 0; }
.side-toggle button.active.buy { background: var(--color-down); border-color: var(--up); color: var(--up); }
.side-toggle button.active.sell { background: var(--color-up-bg, rgba(220,38,38,0.08)); border-color: var(--down); color: var(--down); }

.estimated { display: flex; justify-content: space-between; padding: 8px 0; font-size: 12px; color: var(--muted); }
.total-value { color: var(--text); font-weight: 600; }

.place-order-btn {
  padding: 10px; border: none; border-radius: var(--radius-md); font-size: 14px; font-weight: 600; cursor: pointer; transition: opacity 0.15s;
}
.place-order-btn.buy { background: var(--up); color: #000; }
.place-order-btn.sell { background: var(--down); color: var(--color-text-primary); }
.place-order-btn:hover { opacity: 0.85; }
</style>
