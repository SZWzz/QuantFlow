<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface CryptoRow {
  symbol: string
  price: number
  changePct24h: number
}

const cryptos = ref<CryptoRow[]>([])
const sortKey = ref<string>('changePct24h')
const sortDir = ref<number>(-1)

const sortedCryptos = computed(() => {
  const arr = [...cryptos.value]
  arr.sort((a, b) => {
    const aVal = a[sortKey.value as keyof CryptoRow]
    const bVal = b[sortKey.value as keyof CryptoRow]
    if (typeof aVal === 'number' && typeof bVal === 'number') {
      return (aVal - bVal) * sortDir.value
    }
    return 0
  })
  return arr
})

function toggleSort(key: string) {
  if (sortKey.value === key) { sortDir.value *= -1 }
  else { sortKey.value = key; sortDir.value = -1 }
}

function sortArrow(key: string): string {
  if (sortKey.value !== key) return ''
  return sortDir.value === -1 ? ' ▼' : ' ▲'
}

function formatPrice(p: number): string {
  if (p >= 1000) return p.toLocaleString('en-US', { maximumFractionDigits: 2 })
  if (p >= 1) return p.toFixed(2)
  if (p >= 0.01) return p.toFixed(4)
  return p.toFixed(8)
}

function pctColor(pct: number): string {
  if (pct > 0) return '#ef4444'
  if (pct < 0) return '#22c55e'
  return 'var(--color-text-secondary)'
}

async function refresh() {
  const app = (window as any).go?.main?.App
  if (!app) return
  try {
    const result = await app.GetCryptoOverview([])
    if (result?.cryptos) {
      cryptos.value = result.cryptos.map((c: any) => ({
        symbol: c.symbol?.replace('USDT', '') || c.symbol,
        price: c.price || 0,
        changePct24h: c.change_pct || 0,
      }))
    }
  } catch { /* silent */ }
}

onMounted(refresh)
</script>

<template>
  <div class="crypto-overview-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.crypto_overview') }}</h3>
      <button class="refresh-btn" @click="refresh">⟳</button>
    </div>

    <!-- Crypto Table -->
    <div class="crypto-table-wrap">
      <table class="crypto-table">
        <thead>
          <tr>
            <th class="col-rank">#</th>
            <th class="col-symbol sortable" @click="toggleSort('symbol')">{{ $t('quote.symbol') }}{{ sortArrow('symbol') }}</th>
            <th class="col-price sortable" @click="toggleSort('price')">{{ $t('common.price') }}{{ sortArrow('price') }}</th>
            <th class="col-change sortable" @click="toggleSort('changePct24h')">24h涨跌%{{ sortArrow('changePct24h') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(c, idx) in sortedCryptos" :key="c.symbol">
            <td class="col-rank">{{ idx + 1 }}</td>
            <td class="col-symbol">{{ c.symbol }}</td>
            <td class="col-price">{{ formatPrice(c.price) }}</td>
            <td class="col-change" :style="{ color: pctColor(c.changePct24h) }">
              {{ c.changePct24h >= 0 ? '+' : '' }}{{ c.changePct24h.toFixed(2) }}%
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.crypto-overview-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: #e5e7eb; cursor: pointer; font-size: 13px;
}

/* Dominance */
.dominance-section {
  display: flex; gap: 24px; margin-bottom: 12px;
}
.dominance-item {
  flex: 1; display: flex; align-items: center; gap: 8px;
}
.dom-label { font-size: 11px; color: var(--color-text-secondary); white-space: nowrap; }
.dom-bar-track {
  flex: 1; height: 8px; background: var(--color-bg-elevated); border-radius: 4px; overflow: hidden;
}
.dom-bar-fill { height: 100%; border-radius: 4px; }
.btc-bar { background: #f7931a; }
.eth-bar { background: #627eea; }
.dom-value { font-size: 12px; font-weight: 600; min-width: 44px; text-align: right; }

/* Table */
.crypto-table-wrap {
  flex: 1; overflow-y: auto;
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.crypto-table {
  width: 100%; border-collapse: collapse; font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.crypto-table thead {
  position: sticky; top: 0; z-index: 1;
}
.crypto-table th {
  padding: 6px 4px; text-align: right; font-size: 11px; color: var(--color-text-tertiary);
  font-weight: 500; border-bottom: 1px solid var(--color-border-strong); background: var(--color-bg-panel);
}
.crypto-table th.sortable { cursor: pointer; user-select: none; }
.crypto-table th.sortable:hover { color: #e5e7eb; }
.crypto-table td {
  padding: 4px; text-align: right; border-bottom: 1px solid var(--color-bg-elevated);
}
.col-rank, th:first-child { width: 24px; text-align: center; }
.col-symbol { text-align: left !important; }
.crypto-symbol { font-weight: 600; color: #e5e7eb; }
.crypto-name { color: var(--color-text-tertiary); font-size: 10px; margin-left: 4px; }
.col-price { width: 90px; }
.col-change { width: 80px; font-weight: 500; }
.col-volume { width: 80px; color: var(--color-text-secondary); }
.col-mcap { width: 90px; color: var(--color-text-secondary); }
</style>
