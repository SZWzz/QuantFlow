<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import type { Trade } from '@/stores/portfolio'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()

// -- Pagination --
const pageSize = 20
const visibleCount = ref(pageSize)

// -- Auto-refresh (5s) --
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  store.fetchTrades()
  timer = setInterval(() => store.fetchTrades(), 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// -- Computed --
const visibleTrades = computed(() => {
  return (store.trades as Trade[]).slice(0, visibleCount.value)
})

const hasMore = computed(() => {
  return visibleCount.value < store.trades.length
})

function loadMore() {
  visibleCount.value = Math.min(visibleCount.value + pageSize, store.trades.length)
}

// -- Helpers --
function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

function fmt(n: number, dec = 2): string {
  return n.toFixed(dec)
}
</script>

<template>
  <div class="execution-panel">
    <!-- Table -->
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Symbol</th>
            <th>Side</th>
            <th class="num">Price</th>
            <th class="num">Qty</th>
            <th class="num">Value</th>
            <th>Order ID</th>
            <th class="num">Fee</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in visibleTrades" :key="t.trade_id">
            <td class="muted">{{ formatTime(t.executed_at) }}</td>
            <td class="symbol">{{ t.symbol }}</td>
            <td :class="t.side === 'buy' ? 'up' : 'down'">
              {{ t.side === 'buy' ? '买入' : '卖出' }}
            </td>
            <td class="num">{{ fmt(t.price) }}</td>
            <td class="num">{{ t.quantity.toLocaleString() }}</td>
            <td class="num">{{ fmt(t.value) }}</td>
            <td class="muted">{{ t.order_id }}</td>
            <td class="num muted">{{ fmt(t.fee, 4) }}</td>
          </tr>
          <tr v-if="visibleTrades.length === 0">
            <td colspan="8" class="empty">暂无成交</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div v-if="hasMore" class="load-more-bar">
      <span class="load-count">
        Showing {{ visibleCount }} of {{ store.trades.length }}
      </span>
      <button class="load-btn" @click="loadMore">加载更多</button>
    </div>
  </div>
</template>

<style scoped>
.execution-panel {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--spacing);
  font-variant-numeric: tabular-nums;
  color: var(--text);
}

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
  border-radius: 4px;
}

.load-count {
  font-size: 11px;
  color: var(--muted);
}

.load-btn {
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
.load-btn:hover { background: var(--card); }
</style>
