<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, EmptyState, LoadingState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()
const chartTheme = useChartTheme()

interface DepthLevel {
  price: number
  qty: number
  total: number
}

interface DepthData {
  bids: DepthLevel[]
  asks: DepthLevel[]
  spread: number
  spreadPct: number
}

const exchanges = ['binance', 'okx', 'gateio']
const symbol = ref('BTC/USDT')
const limit = ref(20)
const depths = ref<Record<string, DepthData>>({})
const loading = ref(false)
const error = ref('')
let timer: ReturnType<typeof setInterval> | null = null

async function fetchAll() {
  const app = useWailsApp()
  if (!app?.GetCryptoDepth) return
  loading.value = true
  error.value = ''
  const results: Record<string, DepthData> = {}
  for (const ex of exchanges) {
    try {
      const { data: raw } = await fetchWithCache<any>(`crypto_depth:${ex}:${symbol.value}:${limit.value}`, () => app.GetCryptoDepth(ex, symbol.value, limit.value), 60 * 1000)
      if (raw?.success === false) {
        throw new Error(raw.error || 'fetch failed')
      }
      const d = raw?.data || raw || {}
      const bids: DepthLevel[] = (d.bids || []).map((b: any) => ({
        price: Array.isArray(b) ? Number(b[0]) : b.price,
        qty: Array.isArray(b) ? Number(b[1]) : b.qty,
        total: 0,
      }))
      const asks: DepthLevel[] = (d.asks || []).map((a: any) => ({
        price: Array.isArray(a) ? Number(a[0]) : a.price,
        qty: Array.isArray(a) ? Number(a[1]) : a.qty,
        total: 0,
      }))
      let cumBid = 0
      bids.sort((a, b) => b.price - a.price)
      for (const b of bids) { cumBid += b.qty; b.total = cumBid }
      let cumAsk = 0
      asks.sort((a, b) => a.price - b.price)
      for (const a of asks) { cumAsk += a.qty; a.total = cumAsk }
      const bestBid = bids.length > 0 ? bids[0].price : 0
      const bestAsk = asks.length > 0 ? asks[0].price : 0
      const spread = bestAsk - bestBid
      const spreadPct = bestBid > 0 ? spread / bestBid : 0
      results[ex] = { bids, asks, spread, spreadPct }
    } catch (e: any) {
      results[ex] = { bids: [], asks: [], spread: 0, spreadPct: 0 }
    }
  }
  depths.value = results
  loading.value = false
}

/** 交易所品牌色取图表主题分类色板（--chart-1..3），随主题切换 */
const allDepths = computed(() => exchanges.map((ex, i) => ({
  ex, color: chartTheme.palette[i],
  bids: depths.value[ex]?.bids || [],
  asks: depths.value[ex]?.asks || [],
  spread: depths.value[ex]?.spread || 0,
  spreadPct: depths.value[ex]?.spreadPct || 0,
})))

function formatPrice(p: number): string {
  if (p >= 1000) return p.toLocaleString('en-US', { maximumFractionDigits: 2 })
  if (p >= 1) return p.toFixed(2)
  return p.toFixed(6)
}

function formatQty(q: number): string {
  if (q >= 1000) return q.toLocaleString('en-US', { maximumFractionDigits: 1 })
  return q.toFixed(4)
}

/** 价差阈值着色：小=好（success），中=警示（chart-3 琥珀），大=差（danger） */
function spreadColor(pct: number): string {
  if (pct < 0.0005) return 'var(--color-success)'
  if (pct < 0.002) return 'var(--chart-3)'
  return 'var(--color-danger)'
}

onMounted(() => {
  fetchAll()
  timer = setInterval(fetchAll, 15000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="depth-comparison-panel">
    <PanelHeader
      :title="$t('misc.depth_comparison')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchAll, loading }]"
    >
      <template #controls>
        <select v-model="symbol" @change="fetchAll" class="sym-select">
          <option>BTC/USDT</option><option>ETH/USDT</option><option>SOL/USDT</option>
          <option>BNB/USDT</option><option>DOGE/USDT</option><option>XRP/USDT</option>
        </select>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading && Object.keys(depths).length === 0" type="table" :rows="6" />
    <EmptyState v-else-if="Object.keys(depths).length === 0" :title="$t('common.no_data')" />

    <template v-else>
      <div class="exchanges-grid">
        <div v-for="ex in allDepths" :key="ex.ex" class="ex-card">
          <div class="ex-header" :style="{ borderColor: ex.color }">
            <span class="ex-name">{{ ex.ex.toUpperCase() }}</span>
            <span class="ex-spread" :style="{ color: spreadColor(ex.spreadPct) }">
              {{ $t('misc.spread') }}: {{ (ex.spreadPct * 100).toFixed(3) }}%
            </span>
          </div>
          <div class="depth-cols">
            <div class="bid-col">
              <div class="col-label">{{ $t('misc.bids') }}</div>
              <div v-for="(b, i) in ex.bids.slice(0, 8)" :key="i" class="depth-row bid">
                <span class="dp">{{ formatPrice(b.price) }}</span>
                <span class="dq">{{ formatQty(b.qty) }}</span>
              </div>
            </div>
            <div class="ask-col">
              <div class="col-label">{{ $t('misc.asks') }}</div>
              <div v-for="(a, i) in ex.asks.slice(0, 8)" :key="i" class="depth-row ask">
                <span class="dp">{{ formatPrice(a.price) }}</span>
                <span class="dq">{{ formatQty(a.qty) }}</span>
              </div>
            </div>
          </div>
          <div class="cum-bar">
            <div class="cum-bid" :style="{ width: (ex.bids[0]?.total || 0) > 0 ? '50%' : '0%', background: ex.color }"></div>
            <div class="cum-ask" :style="{ width: (ex.asks[0]?.total || 0) > 0 ? '50%' : '0%', background: ex.color }"></div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.depth-comparison-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sym-select {
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: var(--font-xs);
}

.exchanges-grid {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--panel-padding);
}
.ex-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  overflow: hidden;
  flex-shrink: 0;
}
.ex-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-xs) var(--space-sm);
  border-bottom: 2px solid;
  font-size: var(--font-xs);
}
.ex-name { font-weight: 700; font-size: var(--font-xs); }
.ex-spread { font-variant-numeric: tabular-nums; }
.depth-cols { display: flex; }
.bid-col, .ask-col { flex: 1; padding: var(--space-xs) var(--space-sm); }
.col-label {
  font-size: var(--font-xs);
  text-transform: uppercase;
  color: var(--color-text-tertiary);
  margin-bottom: var(--space-xs);
}
.depth-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-xs);
  padding: var(--space-xs) 0;
  font-variant-numeric: tabular-nums;
}
.depth-row.bid .dp { color: var(--color-down); }
.depth-row.ask .dp { color: var(--color-up); }
.depth-row .dq { color: var(--color-text-tertiary); }
.cum-bar { height: 3px; display: flex; }
.cum-bid { height: 100%; border-radius: 0 0 0 var(--radius-md); opacity: 0.6; }
.cum-ask { height: 100%; border-radius: 0 0 var(--radius-md) 0; opacity: 0.6; }
</style>
