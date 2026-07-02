<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import type { Order } from '@/stores/portfolio'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

// -- Filters --
const statusFilter = ref<string>('')
const symbolSearch = ref('')

const statusOptions = ['', 'filled', 'partial', 'cancelled', 'pending', 'rejected']

// -- Auto-refresh --
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  store.fetchOrders()
  timer = setInterval(() => store.fetchOrders(), 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// -- Computed --
const filteredOrders = computed(() => {
  let rows = store.orders as Order[]
  if (statusFilter.value) {
    rows = rows.filter(o => o.status === statusFilter.value)
  }
  if (symbolSearch.value) {
    const q = symbolSearch.value.toUpperCase()
    rows = rows.filter(o => o.symbol.toUpperCase().includes(q))
  }
  return rows
})

const stats = computed(() => {
  const all = store.orders as Order[]
  const total = all.length
  const filledCount = all.filter(o => o.status === 'filled' || o.status === 'partial').length
  const fillRate = total > 0 ? ((all.reduce((s, o) => s + o.filled_qty, 0) / all.reduce((s, o) => s + o.quantity, 0)) * 100) : 0
  const totalValue = all.filter(o => o.status === 'filled').reduce((s, o) => s + o.price * o.filled_qty, 0)
  return { total, fillRate: Math.round(fillRate * 10) / 10, totalValue }
})

// -- Helpers --
function statusLabel(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

function filledPct(o: Order): string {
  return o.quantity > 0 ? ((o.filled_qty / o.quantity) * 100).toFixed(0) + '%' : '0%'
}

function cancelOrder(orderId: string) {
  store.cancelOrder(orderId)
}

function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e6) return '$' + (n / 1e6).toFixed(2) + 'M'
  if (Math.abs(n) >= 1e3) return '$' + (n / 1e3).toFixed(1) + 'K'
  return '$' + n.toFixed(2)
}
</script>

<template>
  <div class="order-blotter">
    <!-- Filter Bar -->
    <div class="filter-bar">
      <select v-model="statusFilter" class="filter-select">
        <option value="">{{ $t('trade.all_status') }}</option>
        <option v-for="s in statusOptions.filter(Boolean)" :key="s" :value="s">
          {{ statusLabel(s) }}
        </option>
      </select>
      <input
        v-model="symbolSearch"
        type="text"
        class="filter-input"
        :placeholder="$t('common.search') + '...'"
      />
    </div>

    <!-- Orders Table -->
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ $t('common.time') }}</th>
            <th>{{ $t('trade.order_id') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('trade.side') }}</th>
            <th>{{ $t('common.type') }}</th>
            <th class="num">{{ $t('trade.quantity') }}</th>
            <th class="num">{{ $t('common.price') }}</th>
            <th class="num">{{ $t('trade.filled_pct') }}</th>
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
            <td class="num">{{ o.price.toFixed(2) }}</td>
            <td class="num">{{ filledPct(o) }}</td>
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

    <!-- Stats Footer -->
    <div class="stats-footer">
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.today_orders') }}</span>
        <span class="stat-value">{{ stats.total }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.filled_pct') }}</span>
        <span class="stat-value">{{ stats.fillRate }}%</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">{{ $t('trade.total_value') }}</span>
        <span class="stat-value">{{ fmtMoney(stats.totalValue) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.order-blotter {
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

.filter-select {
  padding: 5px 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 11px;
  outline: none;
}

.filter-input {
  flex: 1;
  padding: 5px 8px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 11px;
  outline: none;
}
.filter-input:focus { border-color: var(--accent); }
.filter-input::placeholder { color: var(--muted); }

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

.muted { color: var(--muted); font-size: 11px; }
.up   { color: var(--up); font-weight: 600; }
.down { color: var(--down); font-weight: 600; }

.empty {
  text-align: center;
  color: var(--muted);
  padding: 24px;
}

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
.badge.filled    { background: var(--color-down); color: var(--up); }
.badge.partial   { background: var(--color-accent-soft); color: var(--color-accent); }
.badge.pending   { background: var(--color-accent-soft); color: var(--color-accent); }
.badge.cancelled { background: var(--bg); color: var(--muted); }
.badge.rejected  { background: var(--color-up-bg, rgba(220,38,38,0.08)); color: var(--down); }

/* -- 取消 button -- */
.cancel-btn {
  padding: 2px 8px;
  background: var(--color-up-bg, rgba(220,38,38,0.08));
  border: 1px solid var(--down);
  border-radius: var(--radius-sm);
  color: var(--down);
  font-size: 10px;
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
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
</style>
