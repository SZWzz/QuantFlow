<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { exportCSV } from '@/lib/export'
import { confirmDialog } from '@/lib/wails'
import { usePortfolioStore } from '@/stores/portfolio'
import type { Trade, Order } from '@/stores/portfolio'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

// -- Pagination --
const pageSize = 20
const tradeVisibleCount = ref(pageSize)

// -- Auto-refresh --
let tradesTimer: ReturnType<typeof setInterval> | null = null
let ordersTimer: ReturnType<typeof setInterval> | null = null

const loadError = ref('')
const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')

onMounted(async () => {
  loadError.value = ''
  state.value = 'loading'
  try {
    await store.fetchOrders()
    await store.fetchTrades()
    state.value = 'loaded'
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    state.value = 'error'
  }
  tradesTimer = setInterval(async () => {
    try { await store.fetchTrades() } catch (e: any) { loadError.value = e?.message || String(e) }
  }, 5000)
  ordersTimer = setInterval(async () => {
    try { await store.fetchOrders() } catch (e: any) { loadError.value = e?.message || String(e) }
  }, 10000)
})

onUnmounted(() => {
  if (tradesTimer) clearInterval(tradesTimer)
  if (ordersTimer) clearInterval(ordersTimer)
})

// -- State --
const activeTab = ref<'trades' | 'orders'>('trades')
const symbolFilter = ref('')
const orderStatusFilter = ref('')
const orderStatusOptions = ['', 'filled', 'partial', 'cancelled', 'pending', 'rejected']

// -- Computed --
const visibleTrades = computed(() => {
  return (store.trades as Trade[]).slice(0, tradeVisibleCount.value)
})

const hasMoreTrades = computed(() => {
  return tradeVisibleCount.value < store.trades.length
})

function loadMoreTrades() {
  tradeVisibleCount.value = Math.min(tradeVisibleCount.value + pageSize, store.trades.length)
}

const filteredTrades = computed(() => {
  let rows = visibleTrades.value
  if (symbolFilter.value) {
    const q = symbolFilter.value.toUpperCase()
    rows = rows.filter(t => t.symbol.toUpperCase().includes(q))
  }
  return rows
})

const filteredOrders = computed(() => {
  let rows = store.orders as Order[]
  if (symbolFilter.value) {
    const q = symbolFilter.value.toUpperCase()
    rows = rows.filter(o => o.symbol.toUpperCase().includes(q))
  }
  if (orderStatusFilter.value) {
    rows = rows.filter(o => o.status === orderStatusFilter.value)
  }
  return rows
})

const orderStats = computed(() => {
  const all = store.orders as Order[]
  const total = all.length
  const filledQty = all.reduce((s, o) => s + o.filled_qty, 0)
  const totalQty = all.reduce((s, o) => s + o.quantity, 0)
  const fillRate = totalQty > 0 ? ((filledQty / totalQty) * 100) : 0
  const totalValue = all.filter(o => o.status === 'filled' || o.status === 'partial')
    .reduce((s, o) => s + o.price * o.filled_qty, 0)
  return { total, fillRate: Math.round(fillRate * 10) / 10, totalValue }
})

// -- Helpers --
function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}

function statusLabel(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function filledPct(o: Order): string {
  return o.quantity > 0 ? ((o.filled_qty / o.quantity) * 100).toFixed(0) + '%' : '0%'
}

function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e8) return '$' + (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return '$' + (n / 1e4).toFixed(1) + '万'
  return '$' + n.toFixed(2)
}

async function cancelOrder(orderId: string) {
  const ok = await confirmDialog('确认撤销此委托订单？', '撤销订单')
  if (!ok) return
  store.cancelOrder(orderId)
}

// -- CSV export --
function exportData() {
  if (activeTab.value === 'trades') {
    const headers = ['Time', 'Symbol', 'Side', 'Price', 'Qty', 'Amount', 'Fee', 'OrderID']
    const rows = filteredTrades.value.map(t => [
      formatTime(t.executed_at), t.symbol, t.side, fmt(t.price),
      String(t.quantity), fmt(t.value), fmt(t.fee, 4), t.order_id,
    ])
    exportCSV('trades.csv', headers, rows)
  } else {
    const headers = ['Time', 'OrderID', 'Symbol', 'Side', 'Type', 'Qty', 'Filled', 'Price', 'Status']
    const rows = filteredOrders.value.map(o => [
      formatTime(o.created_at), o.order_id, o.symbol, o.side, o.type,
      String(o.quantity), String(o.filled_qty), fmt(o.price), o.status,
    ])
    exportCSV('orders.csv', headers, rows)
  }
}

async function retryLoad() {
  state.value = 'loading'
  loadError.value = ''
  try {
    await store.fetchOrders()
    await store.fetchTrades()
    state.value = 'loaded'
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    state.value = 'error'
  }
}
</script>

<template>
  <PanelShell :state="state" :error="loadError" @retry="retryLoad">
    <template #loaded>
      <div class="trade-history">
        <!-- Filters -->
    <div class="filter-bar">
      <input
        v-model="symbolFilter"
        type="text"
        class="filter-input"
        :placeholder="$t('common.search') + '...'"
      />

      <div class="tab-switch">
        <button
          :class="{ active: activeTab === 'trades' }"
          @click="activeTab = 'trades'"
        >成交</button>
        <button
          :class="{ active: activeTab === 'orders' }"
          @click="activeTab = 'orders'"
        >委托</button>
      </div>

      <select v-if="activeTab === 'orders'" v-model="orderStatusFilter" class="filter-select">
        <option value="">{{ $t('trade.all_status') }}</option>
        <option v-for="s in orderStatusOptions.filter(Boolean)" :key="s" :value="s">
          {{ statusLabel(s) }}
        </option>
      </select>

      <button class="export-btn" @click="exportData">{{ $t('misc.csv_export') }}</button>
    </div>

    <!-- Trades Table -->
    <div v-if="activeTab === 'trades'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ $t('common.time') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('trade.side') }}</th>
            <th class="num">{{ $t('common.price') }}</th>
            <th class="num">{{ $t('trade.quantity') }}</th>
            <th class="num">{{ $t('common.amount') }}</th>
            <th class="num">{{ $t('workflow.fee') }}</th>
            <th>{{ $t('trade.order_id') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in filteredTrades" :key="t.trade_id">
            <td class="muted">{{ formatTime(t.executed_at) }}</td>
            <td class="symbol">{{ t.symbol }} - {{ t.name || '' }}</td>
            <td :class="t.side === 'buy' ? 'up' : 'down'">
              {{ t.side === 'buy' ? $t('trade.buy') : $t('trade.sell') }}
            </td>
            <td class="num">{{ fmt(t.price) }}</td>
            <td class="num">{{ t.quantity.toLocaleString() }}</td>
            <td class="num">{{ fmt(t.value) }}</td>
            <td class="num muted">{{ fmt(t.fee, 4) }}</td>
            <td class="muted">{{ t.order_id }}</td>
          </tr>
          <tr v-if="filteredTrades.length === 0">
            <td colspan="8" class="empty">{{ $t('common.no_data') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Trades pagination -->
    <div v-if="activeTab === 'trades' && hasMoreTrades" class="load-more-bar">
      <span class="load-count">
        Showing {{ tradeVisibleCount }} of {{ store.trades.length }}
      </span>
      <button class="load-btn" @click="loadMoreTrades">{{ $t('workflow.load_more') }}</button>
    </div>

    <!-- Orders Table -->
    <div v-if="activeTab === 'orders'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ $t('common.time') }}</th>
            <th>{{ $t('trade.order_id') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('trade.side') }}</th>
            <th>{{ $t('common.type') }}</th>
            <th class="num">{{ $t('trade.quantity') }}</th>
            <th class="num">{{ $t('trade.filled_pct') }}</th>
            <th class="num">{{ $t('common.price') }}</th>
            <th>{{ $t('common.status') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in filteredOrders" :key="o.order_id">
            <td class="muted">{{ formatTime(o.created_at) }}</td>
            <td class="muted">{{ o.order_id }}</td>
            <td class="symbol">{{ o.symbol }} - {{ o.name || '' }}</td>
            <td :class="o.side === 'buy' ? 'up' : 'down'">
              {{ o.side === 'buy' ? $t('trade.buy') : $t('trade.sell') }}
            </td>
            <td>{{ o.type }}</td>
            <td class="num">{{ o.quantity.toLocaleString() }}</td>
            <td class="num">{{ filledPct(o) }}</td>
            <td class="num">{{ fmt(o.price) }}</td>
            <td>
              <span :class="['badge', o.status]">{{ statusLabel(o.status) }}</span>
            </td>
            <td>
              <button
                v-if="o.status === 'pending' || o.status === 'partial'"
                class="cancel-btn"
                @click="cancelOrder(o.order_id)"
              >
                {{ $t('trade.cancel_order') }}
              </button>
            </td>
          </tr>
          <tr v-if="filteredOrders.length === 0">
            <td colspan="10" class="empty">{{ $t('trade.no_orders') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Orders stats footer -->
    <div v-if="activeTab === 'orders'" class="stats-footer">
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.today_orders') }}</span>
        <span class="stat-value">{{ orderStats.total }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.filled_pct') }}</span>
        <span class="stat-value">{{ orderStats.fillRate }}%</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.total_value') }}</span>
        <span class="stat-value">{{ fmtMoney(orderStats.totalValue) }}</span>
      </div>
    </div>
    </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.trade-history {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--spacing);
  font-variant-numeric: tabular-nums;
  color: var(--text);
}

/* -- Filter bar -- */
.filter-bar {
  display: flex;
  gap: 6px;
  align-items: center;
}

.filter-input {
  flex: 1;
  padding: 5px 8px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: var(--font-xs);
  outline: none;
}
.filter-input:focus { border-color: var(--accent); }
.filter-input::placeholder { color: var(--muted); }

.filter-select {
  padding: 5px 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: var(--font-xs);
  outline: none;
}

/* -- Tab switch -- */
.tab-switch {
  display: flex;
  gap: 0;
}
.tab-switch button {
  padding: 5px 12px;
  background: var(--input);
  border: 1px solid var(--border);
  color: var(--muted);
  font-size: var(--font-xs);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.tab-switch button:first-child { border-radius: var(--radius-sm) 0 0 4px; }
.tab-switch button:last-child  { border-radius: 0 4px 4px 0; }
.tab-switch button.active {
  background: var(--card);
  border-color: var(--accent);
  color: var(--accent);
}

/* -- Export button -- */
.export-btn {
  padding: 5px 14px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--accent);
  font-size: var(--font-xs);
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.export-btn:hover { background: var(--card); }

/* -- Table -- */
.table-wrap {
  flex: 1;
  overflow-y: auto;
  background: var(--card);
  border-radius: var(--radius-sm);
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-sm);
}

thead { position: sticky; top: 0; z-index: 1; }

th {
  padding: 7px 10px;
  background: var(--input);
  color: var(--muted);
  font-size: var(--font-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  text-align: left;
  white-space: nowrap;
}

td {
  padding: 7px 10px;
  color: var(--text);
  border-bottom: 1px solid var(--input);
}

.num { text-align: right; }

.symbol { font-weight: 600; color: var(--text); }

.muted { color: var(--muted); font-size: var(--font-xs); }
.up   { color: var(--up); font-weight: 600; }
.down { color: var(--down); font-weight: 600; }

.empty {
  text-align: center;
  color: var(--muted);
  padding: 24px;
}

/* -- Pagination -- */
.load-more-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: var(--card);
  border-radius: var(--radius-sm);
}

.load-count {
  font-size: var(--font-xs);
  color: var(--muted);
}

.load-btn {
  padding: 5px 14px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--accent);
  font-size: var(--font-xs);
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.load-btn:hover { background: var(--card); }

/* -- Status badges -- */
.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.badge.filled {
  background: var(--color-down);
  color: var(--up);
}

.badge.partial {
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.badge.pending {
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.badge.cancelled {
  background: var(--bg);
  color: var(--muted);
}

.badge.rejected {
  background: var(--color-danger-soft);
  color: var(--color-danger);
}

/* -- Cancel button -- */
.cancel-btn {
  padding: 2px 8px;
  background: var(--color-danger-soft);
  border: 1px solid var(--color-danger);
  border-radius: var(--radius-sm);
  color: var(--color-danger);
  font-size: var(--font-xs);
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}
.cancel-btn:hover { opacity: 0.75; }

/* -- Stats footer -- */
.stats-footer {
  display: flex;
  gap: 16px;
  padding: 8px 12px;
  background: var(--card);
  border-radius: var(--radius-sm);
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.stat-value {
  font-size: var(--font-sm);
  font-weight: 600;
  color: var(--text);
}
</style>
