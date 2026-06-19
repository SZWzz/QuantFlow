<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const symbol = ref('AAPL')
const side = ref<'buy' | 'sell'>('buy')
const orderType = ref<'market' | 'limit' | 'stop'>('limit')
const quantity = ref(100)
const price = ref(195.50)
const stopPrice = ref(190.00)
const broker = ref<'paper' | 'binance' | 'futu'>('paper')

const estimatedTotal = computed(() => {
  if (orderType.value === 'market') return quantity.value * 195.32 // mock market price
  return quantity.value * price.value
})

import { computed } from 'vue'

function placeOrder() {
  try {
    ;(window as any).go.main.App.PlaceOrder(
      symbol.value, side.value, orderType.value, quantity.value,
      orderType.value === 'market' ? 0 : price.value
    )
  } catch (e) {
    console.warn('PlaceOrder not available:', e)
  }
  console.log('Place order:', { symbol: symbol.value, side: side.value, type: orderType.value, broker: broker.value, qty: quantity.value })
}
</script>

<template>
  <div class="order-panel">
    <div class="order-form">
      <div class="form-group">
        <label>Symbol</label>
        <input v-model="symbol" type="text" class="form-input" />
      </div>

      <div class="form-group">
        <label>Broker</label>
        <select v-model="broker" class="form-input">
          <option value="paper">Paper (Simulation)</option>
          <option value="binance">Binance</option>
          <option value="futu">Futu</option>
        </select>
      </div>

      <div class="side-toggle">
        <button :class="{ active: side === 'buy' }" @click="side = 'buy'">Buy</button>
        <button :class="{ active: side === 'sell' }" @click="side = 'sell'">Sell</button>
      </div>

      <div class="form-group">
        <label>Order Type</label>
        <select v-model="orderType" class="form-input">
          <option value="market">Market</option>
          <option value="limit">Limit</option>
          <option value="stop">Stop</option>
        </select>
      </div>

      <div class="form-group">
        <label>Quantity</label>
        <input v-model.number="quantity" type="number" min="1" class="form-input" />
      </div>

      <div v-if="orderType !== 'market'" class="form-group">
        <label>Price</label>
        <input v-model.number="price" type="number" step="0.01" class="form-input" />
      </div>

      <div v-if="orderType === 'stop'" class="form-group">
        <label>Stop Price</label>
        <input v-model.number="stopPrice" type="number" step="0.01" class="form-input" />
      </div>

      <div class="estimated">
        <span>Est. Total</span>
        <span class="total-value">${{ estimatedTotal.toLocaleString() }}</span>
      </div>

      <button
        class="place-order-btn"
        :class="side"
        @click="placeOrder"
      >
        {{ side === 'buy' ? 'Buy' : 'Sell' }} {{ symbol }}
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
  color: #c9d1d9; font-size: 13px; outline: none;
}
.form-input:focus { border-color: var(--accent); }

.side-toggle { display: flex; gap: 0; }
.side-toggle button {
  flex: 1; padding: 8px; border: 1px solid var(--border); background: var(--input); color: var(--muted);
  font-size: 13px; font-weight: 600; cursor: pointer; transition: all 0.15s;
}
.side-toggle button:first-child { border-radius: 4px 0 0 4px; }
.side-toggle button:last-child { border-radius: 0 4px 4px 0; }
.side-toggle button.active.buy { background: #0a3d1a; border-color: var(--up); color: var(--up); }
.side-toggle button.active.sell { background: #3d0a0a; border-color: var(--down); color: var(--down); }

.estimated { display: flex; justify-content: space-between; padding: 8px 0; font-size: 12px; color: var(--muted); }
.total-value { color: var(--text); font-weight: 600; }

.place-order-btn {
  padding: 10px; border: none; border-radius: 6px; font-size: 14px; font-weight: 600; cursor: pointer; transition: opacity 0.15s;
}
.place-order-btn.buy { background: var(--up); color: #000; }
.place-order-btn.sell { background: var(--down); color: #fff; }
.place-order-btn:hover { opacity: 0.85; }
</style>
