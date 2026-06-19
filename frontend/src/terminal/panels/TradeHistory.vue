<script setup lang="ts">
import { ref, computed } from 'vue'
import { exportCSV } from '@/lib/export'

defineProps<{ panelId: string; params?: Record<string, any> }>()

// ── Mock data ──────────────────────────────────────────────────────────

interface Trade {
  date: string
  symbol: string
  side: 'buy' | 'sell'
  qty: number
  price: number
  total: number
  orderId: string
}

interface Order {
  placed: string
  symbol: string
  side: 'buy' | 'sell'
  type: string
  qty: number
  filled: number
  price: number
  status: 'filled' | 'pending' | 'cancelled' | 'rejected'
}

const trades = ref<Trade[]>([
  { date: '2026-06-17 09:32:15', symbol: '000001.SZ', side: 'buy',  qty: 1000, price: 12.58, total: 12580.00, orderId: 'ORD-001' },
  { date: '2026-06-17 10:15:42', symbol: '000001.SZ', side: 'sell', qty: 500,  price: 12.72, total: 6360.00,  orderId: 'ORD-002' },
  { date: '2026-06-17 13:05:08', symbol: '600519.SH', side: 'buy',  qty: 200,  price: 1780.00, total: 356000.00, orderId: 'ORD-003' },
])

const orders = ref<Order[]>([
  { placed: '2026-06-17 09:32:15', symbol: '000001.SZ', side: 'buy',  type: 'Limit', qty: 1000, filled: 1000, price: 12.58, status: 'filled' },
  { placed: '2026-06-17 11:20:30', symbol: '600519.SH', side: 'buy',  type: 'Limit', qty: 200,  filled: 200,  price: 1780.00, status: 'filled' },
  { placed: '2026-06-17 14:45:00', symbol: '300750.SZ', side: 'sell', type: 'Limit', qty: 300,  filled: 0,    price: 210.00, status: 'cancelled' },
])

// ── State ──────────────────────────────────────────────────────────────

const activeTab = ref<'trades' | 'orders'>('trades')
const symbolFilter = ref('')
const orderStatusFilter = ref('')

const orderStatusOptions = ['', 'filled', 'pending', 'cancelled', 'rejected']

// ── Computed ───────────────────────────────────────────────────────────

const filteredTrades = computed(() => {
  let rows = trades.value
  if (symbolFilter.value) {
    const q = symbolFilter.value.toUpperCase()
    rows = rows.filter(t => t.symbol.toUpperCase().includes(q))
  }
  return rows
})

const filteredOrders = computed(() => {
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

// ── Helpers ────────────────────────────────────────────────────────────

function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}

function statusLabel(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function exportData() {
  if (activeTab.value === 'trades') {
    const headers = ['Date', 'Symbol', 'Side', 'Qty', 'Price', 'Total', 'OrderID']
    const rows = filteredTrades.value.map(t => [
      t.date, t.symbol, t.side, String(t.qty), fmt(t.price), fmt(t.total), t.orderId,
    ])
    exportCSV('trades.csv', headers, rows)
  } else {
    const headers = ['Placed', 'Symbol', 'Side', 'Type', 'Qty', 'Filled', 'Price', 'Status']
    const rows = filteredOrders.value.map(o => [
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
        placeholder="Search symbol..."
      />

      <div class="tab-switch">
        <button
          :class="{ active: activeTab === 'trades' }"
          @click="activeTab = 'trades'"
        >Trades</button>
        <button
          :class="{ active: activeTab === 'orders' }"
          @click="activeTab = 'orders'"
        >Orders</button>
      </div>

      <select v-if="activeTab === 'orders'" v-model="orderStatusFilter" class="filter-select">
        <option value="">All Status</option>
        <option v-for="s in orderStatusOptions.filter(Boolean)" :key="s" :value="s">
          {{ statusLabel(s) }}
        </option>
      </select>

      <button class="export-btn" @click="exportData">CSV</button>
    </div>

    <!-- Trades Table -->
    <div v-if="activeTab === 'trades'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Symbol</th>
            <th>Side</th>
            <th class="num">Qty</th>
            <th class="num">Price</th>
            <th class="num">Total</th>
            <th>Order ID</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in filteredTrades" :key="t.orderId">
            <td class="muted">{{ t.date }}</td>
            <td class="symbol">{{ t.symbol }}</td>
            <td :class="t.side === 'buy' ? 'up' : 'down'">{{ t.side === 'buy' ? 'Buy' : 'Sell' }}</td>
            <td class="num">{{ t.qty.toLocaleString() }}</td>
            <td class="num">{{ fmt(t.price) }}</td>
            <td class="num">{{ fmt(t.total) }}</td>
            <td class="muted">{{ t.orderId }}</td>
          </tr>
          <tr v-if="filteredTrades.length === 0">
            <td colspan="7" class="empty">No trades match</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Orders Table -->
    <div v-if="activeTab === 'orders'" class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Placed</th>
            <th>Symbol</th>
            <th>Side</th>
            <th>Type</th>
            <th class="num">Qty/Filled</th>
            <th class="num">Price</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in filteredOrders" :key="`${o.symbol}-${o.placed}`">
            <td class="muted">{{ o.placed }}</td>
            <td class="symbol">{{ o.symbol }}</td>
            <td :class="o.side === 'buy' ? 'up' : 'down'">{{ o.side === 'buy' ? 'Buy' : 'Sell' }}</td>
            <td>{{ o.type }}</td>
            <td class="num">{{ o.qty }}<span class="muted">/{{ o.filled }}</span></td>
            <td class="num">{{ fmt(o.price) }}</td>
            <td>
              <span :class="['badge', o.status]">{{ statusLabel(o.status) }}</span>
            </td>
          </tr>
          <tr v-if="filteredOrders.length === 0">
            <td colspan="7" class="empty">No orders match</td>
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

/* ── Filter bar ──────────────────────────────── */

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

/* ── Tab switch ──────────────────────────────── */

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

/* ── Export button ───────────────────────────── */

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

/* ── Table ───────────────────────────────────── */

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

/* ── Status badges ───────────────────────────── */

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
