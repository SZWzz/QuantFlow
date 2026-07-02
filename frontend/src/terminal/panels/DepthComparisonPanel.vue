<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()

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

const exchangeColors: Record<string, string> = {
  binance: '#f0b90b',
  okx: '#1a8cff',
  gateio: '#21a179',
}

async function fetchAll() {
  const app = (window as any).go?.main?.App
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

const allDepths = computed(() => exchanges.map(ex => ({
  ex, color: exchangeColors[ex],
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

function spreadColor(pct: number): string {
  if (pct < 0.0005) return '#16a34a'
  if (pct < 0.002) return '#eab308'
  return '#dc2626'
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
    <div class="panel-header">
      <h3>{{ $t('misc.depth_comparison') }}</h3>
      <select v-model="symbol" @change="fetchAll" class="sym-select">
        <option>BTC/USDT</option><option>ETH/USDT</option><option>SOL/USDT</option>
        <option>BNB/USDT</option><option>DOGE/USDT</option><option>XRP/USDT</option>
      </select>
      <button class="refresh-btn" @click="fetchAll" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && Object.keys(depths).length === 0" type="table" :rows="6" />

    <div v-else-if="Object.keys(depths).length === 0" class="empty-state">{{ $t('common.no_data') }}</div>

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
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.sym-select {
  padding: 2px 6px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 12px;
}
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer;
  font-size: 13px; margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}
.exchanges-grid { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 8px; }
.ex-card {
  border: 1px solid var(--color-border-strong); border-radius: var(--radius-md);
  background: var(--color-bg-elevated); overflow: hidden;
}
.ex-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px; border-bottom: 2px solid; font-size: 11px;
}
.ex-name { font-weight: 700; font-size: 12px; }
.ex-spread { font-variant-numeric: tabular-nums; }
.depth-cols { display: flex; gap: 0; }
.bid-col, .ask-col { flex: 1; padding: 4px 8px; }
.col-label { font-size: 9px; text-transform: uppercase; color: var(--color-text-tertiary); margin-bottom: 2px; }
.depth-row { display: flex; justify-content: space-between; font-size: 10px; padding: 1px 0; font-variant-numeric: tabular-nums; }
.depth-row.bid .dp { color: var(--color-down); }
.depth-row.ask .dp { color: var(--color-up); }
.depth-row .dq { color: var(--color-text-tertiary); }
.cum-bar { height: 3px; display: flex; }
.cum-bid { height: 100%; border-radius: 0 0 0 6px; opacity: 0.6; }
.cum-ask { height: 100%; border-radius: 0 0 6px 0; opacity: 0.6; }
</style>
