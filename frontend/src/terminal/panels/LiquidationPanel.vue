<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Liquidation {
  symbol: string
  side: string
  price: number
  qty: number
  amount: number
  time: number
  order_side: string
}

const symbol = ref(props.params?.symbol || '')
const range = ref<'24h' | '7d'>('24h')
const liquidations = ref<Liquidation[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
let timer: ReturnType<typeof setInterval> | null = null

const stats = computed(() => {
  let totalAmount = 0
  let maxAmount = 0
  let longLiq = 0
  let shortLiq = 0
  for (const l of liquidations.value) {
    totalAmount += l.amount
    if (l.amount > maxAmount) maxAmount = l.amount
    if (l.order_side === 'SELL') longLiq += l.amount
    else shortLiq += l.amount
  }
  return { totalAmount, maxAmount, longLiq, shortLiq, count: liquidations.value.length }
})

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetCryptoLiquidations) return
  loading.value = true
  try {
    const result = await app.GetCryptoLiquidations(symbol.value, 100)
    liquidations.value = (result || []).map((l: any) => ({
      symbol: l.symbol?.replace('USDT', '') || l.symbol || '',
      side: l.side || '',
      price: l.price || 0,
      qty: l.qty || 0,
      amount: l.amount || 0,
      time: l.time || 0,
      order_side: l.order_side || '',
    }))
  } catch (e) {
    console.error('[Liquidation]', e)
    liquidations.value = []
  } finally {
    loading.value = false
  }
}

function formatAmount(v: number): string {
  if (v >= 1e6) return '$' + (v / 1e6).toFixed(1) + 'M'
  if (v >= 1e3) return '$' + (v / 1e3).toFixed(0) + 'K'
  return '$' + v.toFixed(0)
}

function formatTime(ts: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function isLongLiq(orderSide: string): boolean {
  return orderSide === 'SELL'
}

function directionLabel(l: Liquidation): string {
  if (l.order_side === 'SELL') return '多→空'
  return '空→多'
}

function directionColor(l: Liquidation): string {
  return isLongLiq(l.order_side) ? '#dc2626' : '#16a34a'
}

onMounted(() => {
  fetchData()
  timer = setInterval(() => { if (autoRefresh.value) fetchData() }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="liquidation-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.liquidation') }}</h3>
      <input v-model="symbol" class="sym-input" :placeholder="$t('misc.symbol_filter')" @change="fetchData" />
      <button class="auto-btn" :class="{ active: autoRefresh }" @click="autoRefresh = !autoRefresh">
        {{ autoRefresh ? '自动(30s)' : '手动' }}
      </button>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <SkeletonPanel v-if="loading && liquidations.length === 0" type="card" :rows="3" />

    <template v-else-if="liquidations.length > 0">
      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-label">24h {{ $t('misc.liquidation_total') }}</div>
          <div class="stat-value">{{ formatAmount(stats.totalAmount) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.max_single') }}</div>
          <div class="stat-value">{{ formatAmount(stats.maxAmount) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.long_liq') }}</div>
          <div class="stat-value up">{{ formatAmount(stats.longLiq) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">{{ $t('misc.short_liq') }}</div>
          <div class="stat-value down">{{ formatAmount(stats.shortLiq) }}</div>
        </div>
      </div>

      <div class="table-wrapper">
        <div class="table-header">
          <span class="col-time">{{ $t('common.time') }}</span>
          <span class="col-sym">{{ $t('quote.symbol') }}</span>
          <span class="col-dir">{{ $t('misc.direction') }}</span>
          <span class="col-price">{{ $t('common.price') }}</span>
          <span class="col-qty">{{ $t('common.size') }}</span>
          <span class="col-amt">{{ $t('common.amount') }}</span>
        </div>
        <div class="table-body">
          <div v-for="l in liquidations" :key="l.time + l.symbol + l.price" class="table-row">
            <span class="col-time">{{ formatTime(l.time) }}</span>
            <span class="col-sym">{{ l.symbol }}</span>
            <span class="col-dir" :style="{ color: directionColor(l) }">{{ directionLabel(l) }}</span>
            <span class="col-price">${{ l.price.toFixed(2) }}</span>
            <span class="col-qty">{{ l.qty.toFixed(4) }}</span>
            <span class="col-amt">{{ formatAmount(l.amount) }}</span>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="empty-state">{{ $t('common.no_data') }}</div>
  </div>
</template>

<style scoped>
.liquidation-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg-panel, #1a1a2e);
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.sym-input {
  padding: 2px 6px; font-size: 11px; border: 1px solid var(--color-border-strong);
  border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); width: 80px;
}
.auto-btn {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.auto-btn.active { color: #60a5fa; border-color: #3b82f6; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
  margin-left: auto;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}

.stats-row {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 12px;
}
.stat-card {
  padding: 10px; border: 1px solid var(--color-border-subtle); border-radius: 8px; text-align: center;
}
.stat-label { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.stat-value { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: #dc2626; }
.down { color: #16a34a; }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-time { width: 56px; color: var(--color-text-tertiary); font-variant-numeric: tabular-nums; }
.col-sym { width: 48px; font-weight: 600; }
.col-dir { width: 48px; text-align: center; font-weight: 500; font-size: 11px; }
.col-price { width: 64px; text-align: right; font-variant-numeric: tabular-nums; }
.col-qty { width: 64px; text-align: right; color: var(--color-text-secondary); }
.col-amt { flex: 1; text-align: right; font-weight: 500; }
</style>
