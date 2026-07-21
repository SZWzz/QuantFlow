<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useToast } from '@/lib/composables/useToast'
import { useSymbolContext } from '@/stores/symbolContext'
import { useTerminalStore } from '@/stores/terminal'
import { detectMarket } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()
const toast = useToast()
const terminal = useTerminalStore()
const isLive = computed(() => terminal.tradingMode === 'live')

// ── Symbol linkage ─────────────────────────────────────────────────────
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// Priority: explicit open params > linked group symbol > empty (no demo default)
const paramSymbol = typeof props.params?.symbol === 'string' ? props.params.symbol.trim().toUpperCase() : ''
const symbol = ref(paramSymbol || ctx.getActiveSymbolForPanel(props.panelId) || '')
const { name } = useStockName(symbol)

// Follow the linked group's active symbol (e.g. picked in SymbolBar)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSymbol) => {
  if (newSymbol && newSymbol !== symbol.value) {
    symbol.value = newSymbol
  }
})

const side = ref<'buy' | 'sell'>('buy')
const orderType = ref<'market' | 'limit' | 'stop'>('limit')
const quantity = ref(100)
const price = ref(0)
const stopPrice = ref(0)
const broker = ref<'paper' | 'binance' | 'futu' | 'ibkr' | 'alpaca'>('paper')
const lastPrice = ref(0)
const quoteLoading = ref(false)
const loadError = ref('')

// ── Submit state machine: edit → confirm → submitting ──────────────────
const confirming = ref(false)
const submitting = ref(false)

const quantityPresets = [100, 200, 500, 1000]

const currencySymbol = computed(() => {
  const m = detectMarket(symbol.value || '')
  return m === 'CN' ? '¥' : m === 'HK' ? 'HK$' : '$'
})

const estimatedTotal = computed(() => {
  const p = orderType.value === 'market' ? (lastPrice.value || price.value) : price.value
  return quantity.value * p
})

const priceDisplay = computed(() =>
  orderType.value === 'market' ? '市价' : price.value.toFixed(2)
)

const canSubmit = computed(() => {
  if (!symbol.value.trim()) return false
  if (!(quantity.value > 0)) return false
  if (orderType.value !== 'market' && !(price.value > 0)) return false
  if (orderType.value === 'stop' && !(stopPrice.value > 0)) return false
  return true
})

function fmtMoney(v: number): string {
  return v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function fetchQuote() {
  const sym = symbol.value.trim()
  if (!sym) return
  const app = (window as any).go?.main?.App
  if (!app?.GetQuote) return
  quoteLoading.value = true
  loadError.value = ''
  try {
    const market = detectMarket(sym)
    const { data: result } = await fetchWithCache<any>(`quote:${sym}`, () => app.GetQuote(market, sym), 60 * 1000)
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
  lastPrice.value = 0
  confirming.value = false
  fetchQuote()
})

onMounted(() => {
  fetchQuote()
})

// Step 1: show inline confirmation summary
function placeOrder() {
  if (submitting.value || !canSubmit.value) return
  confirming.value = true
}

function cancelConfirm() {
  if (submitting.value) return
  confirming.value = false
}

// Step 2: actually submit
async function confirmOrder() {
  if (submitting.value || !canSubmit.value) return
  const app = (window as any).go?.main?.App
  const sideLabel = side.value === 'buy' ? '买入' : '卖出'
  const sym = symbol.value.trim().toUpperCase()
  const effStop = stopPrice.value > 0 ? stopPrice.value : 0
  submitting.value = true
  try {
    const args: [string, string, string, string, number, number, number] = [
      sym, side.value, orderType.value, broker.value,
      quantity.value,
      orderType.value === 'market' ? 0 : price.value,
      effStop,
    ]
    if (app?.PlaceOrderWithStop) {
      await app.PlaceOrderWithStop(...args)
    } else if (app?.PlaceOrder) {
      // Fallback for backends without the stop-price binding
      await app.PlaceOrder(...args.slice(0, 6))
    } else {
      throw new Error('下单接口不可用')
    }
    toast.success(`已提交 ${sideLabel} ${sym} ${quantity.value} 股 @ ${orderType.value === 'market' ? '市价' : currencySymbol.value + price.value.toFixed(2)}`)
    confirming.value = false
  } catch (e: any) {
    toast.error(`下单失败：${e?.message || String(e)}`)
  } finally {
    submitting.value = false
  }
}

// Ctrl+Enter: submit (or confirm when in confirmation state)
function onSubmitKey() {
  if (confirming.value) confirmOrder()
  else placeOrder()
}
</script>

<template>
  <div class="order-panel" @keydown.ctrl.enter="onSubmitKey" @keydown.meta.enter="onSubmitKey">
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>

    <!-- Inline confirmation state (replaces the form) -->
    <div v-if="confirming" class="confirm-view" data-testid="order-confirm-view">
      <div class="confirm-title">确认订单</div>
      <div class="confirm-summary">
        <div class="confirm-row"><span>标的</span><span class="confirm-value">{{ symbol.toUpperCase() }}<template v-if="name"> · {{ name }}</template></span></div>
        <div class="confirm-row"><span>方向</span><span class="confirm-value" :class="side">{{ side === 'buy' ? $t('trade.buy') : $t('trade.sell') }}</span></div>
        <div class="confirm-row"><span>类型</span><span class="confirm-value">{{ $t(`trade.${orderType}`) }}</span></div>
        <div class="confirm-row"><span>数量</span><span class="confirm-value">{{ quantity }}</span></div>
        <div class="confirm-row"><span>价格</span><span class="confirm-value">{{ orderType === 'market' ? '市价' : currencySymbol + price.toFixed(2) }}</span></div>
        <div class="confirm-row"><span>预估金额</span><span class="confirm-value">{{ currencySymbol }}{{ fmtMoney(estimatedTotal) }}</span></div>
        <div v-if="stopPrice > 0" class="confirm-row"><span>止损价</span><span class="confirm-value">{{ currencySymbol }}{{ stopPrice.toFixed(2) }}</span></div>
      </div>
      <div class="confirm-actions">
        <button
          class="confirm-btn"
          :class="side"
          :disabled="submitting"
          @click="confirmOrder"
          data-testid="order-confirm-btn"
        >{{ submitting ? '提交中…' : '确认下单' }}</button>
        <button
          class="cancel-btn"
          :disabled="submitting"
          @click="cancelConfirm"
          data-testid="order-cancel-btn"
        >取消</button>
      </div>
    </div>

    <!-- Edit state -->
    <div v-else class="order-form">
      <div class="form-group">
        <label>{{ $t('quote.symbol') }}
          <span v-if="name" class="quote-status">{{ name }}</span>
        </label>
        <input v-model="symbol" type="text" class="form-input" placeholder="输入代码，或在 SymbolBar 选择" data-testid="order-symbol-input" />
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
        <button :class="{ active: side === 'buy' }" class="buy" @click="side = 'buy'" data-testid="order-side-buy">{{ $t('trade.buy') }}</button>
        <button :class="{ active: side === 'sell' }" class="sell" @click="side = 'sell'" data-testid="order-side-sell">{{ $t('trade.sell') }}</button>
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
        <div class="qty-chips">
          <button
            v-for="q in quantityPresets"
            :key="q"
            class="qty-chip"
            :class="{ active: quantity === q }"
            @click="quantity = q"
            :data-testid="`order-qty-chip-${q}`"
          >{{ q }}</button>
        </div>
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
        <input v-model.number="stopPrice" type="number" step="0.01" class="form-input" data-testid="order-stop-price-input" />
        <div v-if="isLive" class="stop-hint">实盘模式下止损价仅记录在本地订单，不会转发给券商</div>
      </div>

      <div class="estimated">
        <span>{{ $t('trade.estimated') }}</span>
        <span class="total-value">{{ currencySymbol }}{{ fmtMoney(estimatedTotal) }}</span>
      </div>

      <button
        class="place-order-btn"
        :class="side"
        :disabled="!canSubmit"
        @click="placeOrder"
        data-testid="order-place-btn"
      >
        {{ side === 'buy' ? $t('trade.buy') : $t('trade.sell') }} {{ symbol.toUpperCase() || '—' }}
      </button>
      <div class="submit-hint">Ctrl+Enter 提交</div>
    </div>
  </div>
</template>

<style scoped>
.order-panel { padding: 12px; background: var(--bg); height: 100%; }
.order-form { display: flex; flex-direction: column; gap: 10px; }

.form-group { display: flex; flex-direction: column; gap: 3px; }
.form-group label { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
.stop-hint { font-size: var(--font-xs); color: var(--color-warn); }
.form-input {
  padding: 6px 8px; background: var(--input); border: 1px solid var(--border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: var(--font-sm); outline: none;
}
.form-input:focus { border-color: var(--accent); }

.quote-status { font-size: var(--font-xs); color: var(--color-accent); margin-left: 4px; }

.side-toggle { display: flex; gap: 0; }
.side-toggle button {
  flex: 1; padding: 8px; border: 1px solid var(--border); background: var(--input); color: var(--muted);
  font-size: var(--font-sm); font-weight: 600; cursor: pointer; transition: all 0.15s;
}
.side-toggle button:first-child { border-radius: var(--radius-sm) 0 0 4px; }
.side-toggle button:last-child { border-radius: 0 4px 4px 0; }
.side-toggle button.buy.active { background: var(--color-up-soft); border-color: var(--color-up); color: var(--color-up); }
.side-toggle button.sell.active { background: var(--color-down-soft); border-color: var(--color-down); color: var(--color-down); }

.qty-chips { display: flex; gap: 6px; margin-top: 4px; }
.qty-chip {
  flex: 1; padding: 4px 0; border: 1px solid var(--border); border-radius: var(--radius-sm);
  background: var(--input); color: var(--muted); font-size: var(--font-xs); font-weight: 600; cursor: pointer;
}
.qty-chip:hover { border-color: var(--accent); color: var(--text); }
.qty-chip.active { border-color: var(--accent); color: var(--color-accent); background: var(--bg); }

.estimated { display: flex; justify-content: space-between; padding: 8px 0; font-size: var(--font-sm); color: var(--muted); }
.total-value { color: var(--text); font-weight: 600; }

.place-order-btn {
  padding: 10px; border: 0; border-radius: var(--radius-md); font-size: var(--font-sm); font-weight: 600; cursor: pointer; transition: opacity 0.15s;
}
.place-order-btn.buy { background: var(--color-up); color: var(--color-text-inverse); }
.place-order-btn.sell { background: var(--color-down); color: var(--color-text-inverse); }
.place-order-btn:hover:not(:disabled) { opacity: 0.85; }
.place-order-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.submit-hint { text-align: center; font-size: var(--font-xs); color: var(--muted); }

/* Inline confirmation view */
.confirm-view { display: flex; flex-direction: column; gap: 12px; }
.confirm-title { font-size: var(--font-sm); font-weight: 600; color: var(--text); }
.confirm-summary { display: flex; flex-direction: column; gap: 6px; padding: 10px; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--input); }
.confirm-row { display: flex; justify-content: space-between; font-size: var(--font-sm); color: var(--muted); }
.confirm-value { color: var(--text); font-weight: 600; }
.confirm-value.buy { color: var(--color-up); }
.confirm-value.sell { color: var(--color-down); }
.confirm-actions { display: flex; gap: 8px; }
.confirm-btn {
  flex: 1; padding: 10px; border: 0; border-radius: var(--radius-md);
  font-size: var(--font-sm); font-weight: 600; cursor: pointer; transition: opacity 0.15s;
}
.confirm-btn.buy { background: var(--color-up); color: var(--color-text-inverse); }
.confirm-btn.sell { background: var(--color-down); color: var(--color-text-inverse); }
.confirm-btn:hover:not(:disabled) { opacity: 0.85; }
.confirm-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.cancel-btn {
  padding: 10px 16px; border: 1px solid var(--border); border-radius: var(--radius-md);
  background: var(--input); color: var(--muted); font-size: var(--font-sm); font-weight: 600; cursor: pointer;
}
.cancel-btn:hover:not(:disabled) { color: var(--text); }
.cancel-btn:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
