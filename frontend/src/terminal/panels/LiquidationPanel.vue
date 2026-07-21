<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, PanelTable, EmptyState, ErrorState, LoadingState, type Column } from '@/terminal/components/panel'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { t } = useI18n()

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
const liquidations = ref<Liquidation[]>([])
const loading = ref(false)
const loadError = ref('')
const autoRefresh = ref(true)
const { fetchWithCache } = usePanelCache()
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
  loadError.value = ''
  try {
    const { data: result } = await fetchWithCache<any>(`liquidations:${symbol.value}:100`, () => app.GetCryptoLiquidations(symbol.value, 100), 60 * 1000)
    liquidations.value = (result || []).map((l: any) => ({
      symbol: l.symbol?.replace('USDT', '') || l.symbol || '',
      side: l.side || '',
      price: l.price || 0,
      qty: l.qty || 0,
      amount: l.amount || 0,
      time: l.time || 0,
      order_side: l.order_side || '',
    }))
  } catch (e: any) {
    logger.error('[Liquidation]', e)
    loadError.value = e?.message || String(e)
    liquidations.value = []
  } finally {
    loading.value = false
  }
}

function formatAmount(v: number): string {
  if (v >= 1e8) return '$' + (v / 1e8).toFixed(1) + '亿'
  if (v >= 1e4) return '$' + (v / 1e4).toFixed(0) + '万'
  return '$' + v.toFixed(0)
}

function formatTime(ts: number): string {
  if (!ts) return '--'
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

/** SELL = 多单被强平（多→空），BUY = 空单被强平（空→多） */
function directionLabel(side: string): string {
  return side === 'SELL' ? '多→空' : '空→多'
}

function directionClass(l: Liquidation): string {
  return l.order_side === 'SELL' ? 'dir-up' : 'dir-down'
}

const cols = computed<Column[]>(() => [
  { key: 'time', label: t('common.time'), formatter: formatTime, cellClass: () => 'muted-cell' },
  { key: 'symbol', label: t('quote.symbol') },
  { key: 'order_side', label: t('misc.direction'), align: 'center', formatter: directionLabel, cellClass: directionClass },
  { key: 'price', label: t('common.price'), align: 'right', formatter: (v: number) => '$' + v.toFixed(2) },
  { key: 'qty', label: t('common.size'), align: 'right', formatter: (v: number) => v.toFixed(4) },
  { key: 'amount', label: t('common.amount'), align: 'right', formatter: formatAmount },
])

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
    <PanelHeader
      :title="$t('misc.liquidation')"
      :controls="[{ icon: 'refresh', title: $t('common.refresh'), action: fetchData, loading }]"
    >
      <template #controls>
        <input v-model="symbol" class="sym-input" :placeholder="$t('misc.symbol_filter')" @change="fetchData" />
        <button class="btn btn-sm btn-ghost auto-toggle" :class="{ active: autoRefresh }" @click="autoRefresh = !autoRefresh">
          {{ autoRefresh ? '自动(30s)' : '手动' }}
        </button>
      </template>
    </PanelHeader>

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && liquidations.length === 0" type="card" :rows="3" />

    <template v-else-if="liquidations.length > 0">
      <!-- 自绘统计卡：StatItem 不支持值涨跌着色，保留但 token 化 -->
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

      <PanelTable
        :columns="cols"
        :data="liquidations"
        :loading="loading"
        :row-key="(l: any) => l.time + l.symbol + l.price"
        sticky-header
      />
    </template>

    <EmptyState v-else :title="$t('common.no_data')" />
  </div>
</template>

<style scoped>
.liquidation-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sym-input {
  width: 80px;
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
}
.auto-toggle.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.stats-row {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--space-sm);
  padding: var(--space-sm) var(--panel-padding);
  flex-shrink: 0;
}
.stat-card {
  padding: var(--space-sm); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); text-align: center;
}
.stat-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.stat-value { font-size: var(--font-sm); font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

:deep(.td.muted-cell) { color: var(--color-text-tertiary); }
:deep(.td.dir-up) { color: var(--color-up); font-weight: 500; }
:deep(.td.dir-down) { color: var(--color-down); font-weight: 500; }
</style>
