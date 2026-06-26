<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { exportCSV } from '@/lib/export'
import { usePortfolioStore } from '@/stores/portfolio'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

// -- Raw Go types (from Wails IPC) --
interface GoTrade {
  ID: string; Symbol: string; Side: string; Quantity: number
  Price: number; Timestamp: string; PnL: number
}

interface GoOrder {
  ID: string; Symbol: string; Side: string; Quantity: number
  Price: number; Status: string; PlacedAt: string
}

// -- Display types (panel-native) --
interface Trade {
  date: string; symbol: string; side: 'buy' | 'sell'
  qty: number; price: number; total: number; orderId: string
}

interface Order {
  placed: string; symbol: string; side: 'buy' | 'sell'
  type: string; qty: number; filled: number
  price: number; status: 'filled' | 'pending' | 'cancelled' | 'rejected'
}

const DISPLAY_LIMIT = 20

// -- Convert Go Trade → panel Trade --
function adaptTrade(t: GoTrade): Trade {
  const side = (t.Side || '').toLowerCase() as 'buy' | 'sell'
  return {
    date: t.Timestamp || '--',
    symbol: t.Symbol || '--',
    side,
    qty: t.Quantity ?? 0,
    price: t.Price ?? 0,
    total: (t.Price ?? 0) * (t.Quantity ?? 0),
    orderId: t.ID || '--',
  }
}

// -- Convert Go Order → panel Order --
function adaptOrder(o: GoOrder): Order {
  const side = (o.Side || '').toLowerCase() as 'buy' | 'sell'
  const status = (o.Status || 'pending').toLowerCase() as Order['status']
  const isFilled = status === 'filled'
  return {
    placed: o.PlacedAt || '--',
    symbol: o.Symbol || '--',
    side,
    type: '--',
    qty: o.Quantity ?? 0,
    filled: isFilled ? (o.Quantity ?? 0) : 0,
    price: o.Price ?? 0,
    status,
  }
}

// -- Computed data from store --
const trades = computed<Trade[]>(() => {
  const raw = (store.trades as unknown) as GoTrade[] | null
  if (!raw || raw.length === 0) return []
  return raw.slice(0, DISPLAY_LIMIT).map(adaptTrade)
})

const orders = computed<Order[]>(() => {
  const raw = (store.orders as unknown) as GoOrder[] | null
  if (!raw || raw.length === 0) return []
  return raw.slice(0, DISPLAY_LIMIT).map(adaptOrder)
})

// -- Lifecycle --
onMounted(async () => {
  store.fetchOrders()
  store.fetchTrades()
})

// -- State --
const activeTab = ref<'trades' | 'orders'>('trades')
const symbolFilter = ref('')
const orderStatusFilter = ref('')

const orderStatusOptions = ['', 'filled', 'pending', 'cancelled', 'rejected']

// -- Computed filters --
const filtered成交 = computed(() => {
  let rows = trades.value
  if (symbolFilter.value) {
    const q = symbolFilter.value.toUpperCase()
    rows = rows.filter(t => t.symbol.toUpperCase().includes(q))
  }
  return rows
})

const filtered委托 = computed(() => {
  let rows = orders.value
  if (symbolFilter.value) {
    const q = symbolFilter.value.toUpperCase()
    rows = rows.filter(o => o.symbol.toUpperCase().includes(q))
  }
  if (orderStatusFilter.value) {
    rows = rows.filter(o => o.status === orderStatusFilter.value)
  }
  return rows
})

// -- Helpers --
function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}

function statusLabel(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function exportData() {
  if (activeTab.value === 'trades') {
    const headers = ['Date', 'Symbol', 'Side', 'Qty', 'Price', 'Total', 'OrderID']
    const rows = filtered成交.value.map(t => [
      t.date, t.symbol, t.side, String(t.qty), fmt(t.price), fmt(t.total), t.orderId,
    ])
    exportCSV('trades.csv', headers, rows)
  } else {
    const headers = ['Placed', 'Symbol', 'Side', 'Type', 'Qty', 'Filled', 'Price', 'Status']
    const rows = filtered委托.value.map(o => [
      o.placed, o.symbol, o.side, o.type, String(o.qty), String(o.filled), fmt(o.price), o.status,
    ])
    exportCSV('orders.csv', headers, rows)
  }
}
</script>

<template>
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

    <!-- 成交 Table -->
    <div v-if="activeTab === 'trades'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ $t('common.date') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('trade.side') }}</th>
            <th class="num">{{ $t('trade.quantity') }}</th>
            <th class="num">{{ $t('common.price') }}</th>
            <th class="num">{{ $t('common.total') }}</th>
            <th>{{ $t('trade.order_id') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in filtered成交" :key="t.orderId">
            <td class="muted">{{ t.date }}</td>
            <td class="symbol">{{ t.symbol }}</td>
            <td :class="t.side === 'buy' ? 'up' : 'down'">{{ t.side === 'buy' ? $t('trade.buy') : $t('trade.sell') }}</td>
            <td class="num">{{ t.qty.toLocaleString() }}</td>
            <td class="num">{{ fmt(t.price) }}</td>
            <td class="num">{{ fmt(t.total) }}</td>
            <td class="muted">{{ t.orderId }}</td>
          </tr>
          <tr v-if="filtered成交.length === 0">
            <td colspan="7" class="empty">{{ $t('common.no_data') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 委托 Table -->
    <div v-if="activeTab === 'orders'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ $t('common.date') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('trade.side') }}</th>
            <th>{{ $t('common.type') }}</th>
            <th class="num">{{ $t('trade.quantity') }}</th>
            <th class="num">{{ $t('common.price') }}</th>
            <th>{{ $t('common.status') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in filtered委托" :key="`${o.symbol}-${o.placed}`">
            <td class="muted">{{ o.placed }}</td>
            <td class="symbol">{{ o.symbol }}</td>
            <td :class="o.side === 'buy' ? 'up' : 'down'">{{ o.side === 'buy' ? $t('trade.buy') : $t('trade.sell') }}</td>
            <td>{{ o.type }}</td>
            <td class="num">{{ o.qty }}<span class="muted">/{{ o.filled }}</span></td>
            <td class="num">{{ fmt(o.price) }}</td>
            <td>
              <span :class="['badge', o.status]">{{ statusLabel(o.status) }}</span>
            </td>
          </tr>
          <tr v-if="filtered委托.length === 0">
            <td colspan="7" class="empty">{{ $t('common.no_data') }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
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
  border-radius: 4px;
  color: var(--text);
  font-size: 11px;
  outline: none;
}
.filter-input:focus { border-color: var(--accent); }
.filter-input::placeholder { color: var(--muted); }

.filter-select {
  padding: 5px 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
  font-size: 11px;
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
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.tab-switch button:first-child { border-radius: 4px 0 0 4px; }
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
  border-radius: 4px;
  color: var(--accent);
  font-size: 11px;
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
  border-radius: 4px;
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

.muted { color: var(--muted); font-size: 11px; }
.up   { color: var(--up); }
.down { color: var(--down); }

.empty {
  text-align: center;
  color: var(--muted);
  padding: 24px;
}

/* -- Status badges -- */
.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: var(--font-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.badge.filled {
  background: #0a3d1a;
  color: var(--up);
}

.badge.pending {
  background: #3d2e0a;
  color: #f0883e;
}

.badge.cancelled {
  background: var(--bg);
  color: var(--muted);
}

.badge.rejected {
  background: #3d0a0a;
  color: var(--down);
}
</style>
