<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface CryptoRow {
  symbol: string
  name: string
  price: number
  changePct24h: number
  volume24h: number
  marketCap: number
}

const btcDominance = ref(52.4)
const ethDominance = ref(17.8)
const cryptos = ref<CryptoRow[]>([])
const sortKey = ref<string>('marketCap')
const sortDir = ref<number>(-1) // -1 = desc

const mockCryptos: CryptoRow[] = [
  { symbol: 'BTC', name: 'Bitcoin', price: 68234.50, changePct24h: 1.85, volume24h: 32_500_000_000, marketCap: 1_340_000_000_000 },
  { symbol: 'ETH', name: 'Ethereum', price: 3542.80, changePct24h: 2.10, volume24h: 18_200_000_000, marketCap: 425_000_000_000 },
  { symbol: 'BNB', name: 'BNB', price: 618.25, changePct24h: -0.45, volume24h: 1_800_000_000, marketCap: 91_000_000_000 },
  { symbol: 'SOL', name: 'Solana', price: 172.44, changePct24h: 4.30, volume24h: 4_500_000_000, marketCap: 76_000_000_000 },
  { symbol: 'XRP', name: 'XRP', price: 0.5218, changePct24h: -1.20, volume24h: 1_200_000_000, marketCap: 28_500_000_000 },
  { symbol: 'ADA', name: 'Cardano', price: 0.4532, changePct24h: 0.85, volume24h: 420_000_000, marketCap: 16_000_000_000 },
  { symbol: 'DOGE', name: 'Dogecoin', price: 0.1248, changePct24h: 5.60, volume24h: 1_550_000_000, marketCap: 18_200_000_000 },
  { symbol: 'AVAX', name: 'Avalanche', price: 35.66, changePct24h: 2.75, volume24h: 680_000_000, marketCap: 12_800_000_000 },
  { symbol: 'DOT', name: 'Polkadot', price: 6.33, changePct24h: -0.90, volume24h: 310_000_000, marketCap: 8_600_000_000 },
  { symbol: 'MATIC', name: 'Polygon', price: 0.5412, changePct24h: 1.30, volume24h: 290_000_000, marketCap: 5_100_000_000 },
  { symbol: 'LINK', name: 'Chainlink', price: 14.82, changePct24h: 3.40, volume24h: 410_000_000, marketCap: 8_700_000_000 },
  { symbol: 'UNI', name: 'Uniswap', price: 9.55, changePct24h: -2.15, volume24h: 180_000_000, marketCap: 5_700_000_000 },
  { symbol: 'ATOM', name: 'Cosmos', price: 7.92, changePct24h: 0.50, volume24h: 195_000_000, marketCap: 3_100_000_000 },
  { symbol: 'APT', name: 'Aptos', price: 12.68, changePct24h: -1.80, volume24h: 320_000_000, marketCap: 4_800_000_000 },
  { symbol: 'NEAR', name: 'NEAR Protocol', price: 5.22, changePct24h: 6.10, volume24h: 440_000_000, marketCap: 5_400_000_000 },
  { symbol: 'ICP', name: 'Internet Computer', price: 13.45, changePct24h: -3.20, volume24h: 155_000_000, marketCap: 6_200_000_000 },
  { symbol: 'SHIB', name: 'Shiba Inu', price: 0.00002534, changePct24h: 7.80, volume24h: 980_000_000, marketCap: 14_900_000_000 },
  { symbol: 'TRX', name: 'TRON', price: 0.1220, changePct24h: 0.35, volume24h: 520_000_000, marketCap: 10_600_000_000 },
  { symbol: 'FIL', name: 'Filecoin', price: 5.88, changePct24h: 1.65, volume24h: 198_000_000, marketCap: 2_600_000_000 },
  { symbol: 'ARB', name: 'Arbitrum', price: 0.8523, changePct24h: -2.50, volume24h: 340_000_000, marketCap: 2_900_000_000 },
]

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
  if (sortKey.value === key) {
    sortDir.value *= -1
  } else {
    sortKey.value = key
    sortDir.value = -1
  }
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

function formatVolume(v: number): string {
  if (v >= 1_000_000_000) return '$' + (v / 1_000_000_000).toFixed(2) + 'B'
  if (v >= 1_000_000) return '$' + (v / 1_000_000).toFixed(0) + 'M'
  return '$' + v.toFixed(0)
}

function formatMarketCap(mc: number): string {
  if (mc >= 1_000_000_000_000) return '$' + (mc / 1_000_000_000_000).toFixed(2) + 'T'
  if (mc >= 1_000_000_000) return '$' + (mc / 1_000_000_000).toFixed(2) + 'B'
  return '$' + (mc / 1_000_000).toFixed(0) + 'M'
}

function pctColor(pct: number): string {
  if (pct > 0) return '#ef4444'
  if (pct < 0) return '#22c55e'
  return '#9ca3af'
}

function refresh() {
  cryptos.value = mockCryptos.map(c => ({
    ...c,
    price: c.price * (1 + (Math.random() - 0.5) * 0.01),
    changePct24h: c.changePct24h + (Math.random() - 0.5) * 0.3,
  }))
  btcDominance.value = +(52.4 + (Math.random() - 0.5) * 1).toFixed(1)
  ethDominance.value = +(17.8 + (Math.random() - 0.5) * 0.5).toFixed(1)
}

onMounted(() => {
  refresh()
})
</script>

<template>
  <div class="crypto-overview-panel">
    <div class="panel-header">
      <h3>Crypto Overview</h3>
      <button class="refresh-btn" @click="refresh">⟳</button>
    </div>

    <!-- Dominance Section -->
    <div class="dominance-section">
      <div class="dominance-item">
        <div class="dom-label">BTC Dominance</div>
        <div class="dom-bar-track">
          <div class="dom-bar-fill btc-bar" :style="{ width: btcDominance + '%' }"></div>
        </div>
        <span class="dom-value">{{ btcDominance }}%</span>
      </div>
      <div class="dominance-item">
        <div class="dom-label">ETH Dominance</div>
        <div class="dom-bar-track">
          <div class="dom-bar-fill eth-bar" :style="{ width: ethDominance + '%' }"></div>
        </div>
        <span class="dom-value">{{ ethDominance }}%</span>
      </div>
    </div>

    <!-- Crypto Table -->
    <div class="crypto-table-wrap">
      <table class="crypto-table">
        <thead>
          <tr>
            <th class="col-rank">#</th>
            <th class="col-symbol sortable" @click="toggleSort('symbol')">Symbol{{ sortArrow('symbol') }}</th>
            <th class="col-price sortable" @click="toggleSort('price')">Price{{ sortArrow('price') }}</th>
            <th class="col-change sortable" @click="toggleSort('changePct24h')">24h Chg%{{ sortArrow('changePct24h') }}</th>
            <th class="col-volume sortable" @click="toggleSort('volume24h')">Volume{{ sortArrow('volume24h') }}</th>
            <th class="col-mcap sortable" @click="toggleSort('marketCap')">Market Cap{{ sortArrow('marketCap') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(c, idx) in sortedCryptos" :key="c.symbol">
            <td class="col-rank">{{ idx + 1 }}</td>
            <td class="col-symbol">
              <span class="crypto-symbol">{{ c.symbol }}</span>
              <span class="crypto-name">{{ c.name }}</span>
            </td>
            <td class="col-price">{{ formatPrice(c.price) }}</td>
            <td class="col-change" :style="{ color: pctColor(c.changePct24h) }">
              {{ c.changePct24h >= 0 ? '+' : '' }}{{ c.changePct24h.toFixed(2) }}%
            </td>
            <td class="col-volume">{{ formatVolume(c.volume24h) }}</td>
            <td class="col-mcap">{{ formatMarketCap(c.marketCap) }}</td>
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
  background: var(--color-bg, #111827);
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
  padding: 4px 10px; border: 1px solid #374151; border-radius: 4px;
  background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px;
}

/* Dominance */
.dominance-section {
  display: flex; gap: 24px; margin-bottom: 12px;
}
.dominance-item {
  flex: 1; display: flex; align-items: center; gap: 8px;
}
.dom-label { font-size: 11px; color: #9ca3af; white-space: nowrap; }
.dom-bar-track {
  flex: 1; height: 8px; background: #1f2937; border-radius: 4px; overflow: hidden;
}
.dom-bar-fill { height: 100%; border-radius: 4px; }
.btc-bar { background: #f7931a; }
.eth-bar { background: #627eea; }
.dom-value { font-size: 12px; font-weight: 600; min-width: 44px; text-align: right; }

/* Table */
.crypto-table-wrap {
  flex: 1; overflow-y: auto;
  scrollbar-width: thin; scrollbar-color: #374151 transparent;
}
.crypto-table {
  width: 100%; border-collapse: collapse; font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.crypto-table thead {
  position: sticky; top: 0; z-index: 1;
}
.crypto-table th {
  padding: 6px 4px; text-align: right; font-size: 11px; color: #6b7280;
  font-weight: 500; border-bottom: 1px solid #374151; background: #111827;
}
.crypto-table th.sortable { cursor: pointer; user-select: none; }
.crypto-table th.sortable:hover { color: #e5e7eb; }
.crypto-table td {
  padding: 4px; text-align: right; border-bottom: 1px solid #1f2937;
}
.col-rank, th:first-child { width: 24px; text-align: center; }
.col-symbol { text-align: left !important; }
.crypto-symbol { font-weight: 600; color: #e5e7eb; }
.crypto-name { color: #6b7280; font-size: 10px; margin-left: 4px; }
.col-price { width: 90px; }
.col-change { width: 80px; font-weight: 500; }
.col-volume { width: 80px; color: #9ca3af; }
.col-mcap { width: 90px; color: #9ca3af; }
</style>
