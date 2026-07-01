<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useStockName } from '@/lib/composables/useStockName'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const symbol = ref('AAPL')
const { name } = useStockName(symbol)
const side = ref<'buy' | 'sell'>('buy')
const orderType = ref<'market' | 'limit' | 'stop'>('limit')
const quantity = ref(100)
const price = ref(195.50)
const stopPrice = ref(190.00)
const broker = ref<'paper' | 'binance' | 'futu'>('paper')
const lastPrice = ref(0)
const quoteLoading = ref(false)

const estimatedTotal = computed(() => {
  const p = orderType.value === 'market' ? (lastPrice.value || price.value) : price.value
  return quantity.value * p
})

async function fetchQuote() {
  const app = (window as any).go?.main?.App
  if (!app?.GetQuote) return
  quoteLoading.value = true
  try {
    const result = await app.GetQuote('CN', symbol.value)
    const quote = Array.isArray(result) ? result[0] : result
    if (quote?.last) {
      lastPrice.value = quote.last
      price.value = quote.last
    }
  } catch {
    // empty — keep current price
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
        symbol.value, side.value, orderType.value, quantity.value,
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
    <div class="order-form">
      <div class="form-group">
        <label>{{ $t('quote.symbol') }}
          <span v-if="name" class="quote-status">{{ name }}</span>
        </label>
        <input v-model="symbol" type="text" class="form-input" />
      </div>

      <div class="form-group">
        <label>{{ $t('trade.broker') }}</label>
        <select v-model="broker" class="form-input">
          <option value="paper">{{ $t('trade.paper') }}</option>
          <option value="binance">{{ $t('trade.binance') }}</option>
          <option value="futu">{{ $t('trade.futu') }}</option>
        </select>
      </div>

      <div class="side-toggle">
        <button :class="{ active: side === 'buy' }" @click="side = 'buy'">{{ $t('trade.buy') }}</button>
        <button :class="{ active: side === 'sell' }" @click="side = 'sell'">{{ $t('trade.sell') }}</button>
      </div>

      <div class="form-group">
        <label>{{ $t('trade.order_type') }}</label>
        <select v-model="orderType" class="form-input">
          <option value="market">{{ $t('trade.market') }}</option>
          <option value="limit">{{ $t('trade.limit') }}</option>
          <option value="stop">{{ $t('trade.stop') }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>{{ $t('trade.quantity') }}</label>
        <input v-model.number="quantity" type="number" min="1" class="form-input" />
      </div>

      <div v-if="orderType !== 'market'" class="form-group">
        <label>{{ $t('trade.price') }}
          <span v-if="quoteLoading" class="quote-status">{{ $t('common.loading') }}</span>
          <span v-else-if="lastPrice > 0" class="quote-status">(实时)</span>
        </label>
        <input v-model.number="price" type="number" step="0.01" class="form-input" />
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
      >
        {{ side === 'buy' ? $t('trade.buy') : $t('trade.sell') }} {{ symbol }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.order-panel { padding: 12px; background: var(--bg); height: 100%; }
.order-form { display: flex; flex-direction: column; gap: 10px; }

.form-group { display: flex; flex-direction: column; gap: 3px; }
.form-group label { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
.form-input {
  padding: 6px 8px; background: var(--input); border: 1px solid var(--border); border-radius: 4px;
  color: var(--color-text-primary); font-size: 13px; outline: none;
}
.form-input:focus { border-color: var(--accent); }

.quote-status { font-size: 10px; color: var(--color-accent); margin-left: 4px; }

.side-toggle { display: flex; gap: 0; }
.side-toggle button {
  flex: 1; padding: 8px; border: 1px solid var(--border); background: var(--input); color: var(--muted);
  font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.15s;
}
.side-toggle button:first-child { border-radius: 4px 0 0 4px; }
.side-toggle button:last-child { border-radius: 0 4px 4px 0; }
.side-toggle button.active.buy { background: var(--color-down); border-color: var(--up); color: var(--up); }
.side-toggle button.active.sell { background: var(--color-up-bg, rgba(220,38,38,0.08)); border-color: var(--down); color: var(--down); }

.estimated { display: flex; justify-content: space-between; padding: 8px 0; font-size: 12px; color: var(--muted); }
.total-value { color: var(--text); font-weight: 600; }

.place-order-btn {
  padding: 10px; border: none; border-radius: 6px; font-size: 14px; font-weight: 600; cursor: pointer; transition: opacity 0.15s;
}
.place-order-btn.buy { background: var(--up); color: #000; }
.place-order-btn.sell { background: var(--down); color: var(--color-text-primary); }
.place-order-btn:hover { opacity: 0.85; }
</style>
