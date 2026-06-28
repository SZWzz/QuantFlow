<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { useStockName } from '@/lib/composables/useStockName'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const market = ref('')
const currency = ref('')
const loading = ref(false)

interface PositionData {
  symbol: string
  quantity: number
  avg_price: number
  market_price: number
  pnl: number
  pnl_pct: number
  market: string
  currency: string
  cost_basis: number
  alloc_pct: number
}
const position = ref<PositionData | null>(null)

const fmt = (n: number, dec = 2) => n.toFixed(dec)

async function fetchPosition() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetPositions) return
    const all: PositionData[] = await app.GetPositions()
    const found = all?.find((p: PositionData) => p.symbol === symbol.value)
    if (found) {
      position.value = found
      market.value = found.market || detectMarket(symbol.value)
      currency.value = found.currency
    } else {
      position.value = null
      market.value = detectMarket(symbol.value)
      currency.value = market.value === 'CN' ? 'CNY' : market.value === 'HK' ? 'HKD' : 'USD'
    }
  } catch {
    position.value = null
  } finally {
    loading.value = false
  }
}

watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    fetchPosition()
  }
})

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; fetchPosition() }
})
onMounted(fetchPosition)
</script>

<template>
  <div class="position-detail-panel">
    <div v-if="loading" class="loading-text">{{ $t('common.loading') }}</div>
    <template v-else-if="position">
      <div class="header">
        <span class="symbol-name">{{ symbol }} {{ name }}</span>
        <span class="market-badge">{{ market }}</span>
        <span class="currency">{{ currency }}</span>
      </div>
      <div class="kpi-grid">
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.quantity') }}</span><span class="kpi-value">{{ position.quantity }}</span></div>
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.avg_price') }}</span><span class="kpi-value">${{ fmt(position.avg_price) }}</span></div>
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.market_price') }}</span><span class="kpi-value">${{ fmt(position.market_price) }}</span></div>
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.market_value') }}</span><span class="kpi-value">${{ fmt(position.market_price * position.quantity).replace(/\B(?=(\d{3})+(?!\d))/g, ',') }}</span></div>
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.pnl') }}</span><span :class="['kpi-value', position.pnl >= 0 ? 'up' : 'down']">${{ fmt(position.pnl) }}</span></div>
        <div class="kpi-item"><span class="kpi-label">{{ $t('portfolio.alloc') }}</span><span class="kpi-value">{{ fmt(position.alloc_pct) }}%</span></div>
      </div>
      <div class="pnl-summary">
        <span :class="position.pnl >= 0 ? 'up' : 'down'">{{ position.pnl >= 0 ? '+' : '' }}${{ fmt(position.pnl) }} ({{ position.pnl >= 0 ? '+' : '' }}{{ fmt(position.pnl_pct) }}%)</span>
      </div>
    </template>
    <div v-else class="empty-state">
      <div class="empty-text">{{ $t('portfolio.no_positions') }}</div>
    </div>
  </div>
</template>

<style scoped>
.position-detail-panel { padding: 12px; background: var(--bg); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.header { margin-bottom: 12px; }
.symbol-name { font-size: 20px; font-weight: 700; color: var(--text); }
.market-badge { display: inline-block; margin-left: 8px; padding: 2px 8px; background: var(--input); border-radius: 3px; font-size: 11px; color: var(--accent); }
.currency { margin-left: 6px; font-size: var(--font-sm); color: var(--muted); }
.kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; margin-bottom: 12px; }
.kpi-item { padding: 8px; background: var(--card); border-radius: 4px; text-align: center; }
.kpi-label { display: block; font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 3px; }
.kpi-value { font-size: 15px; font-weight: 600; color: var(--text); }
.kpi-value.up { color: var(--up); } .kpi-value.down { color: var(--down); }
.chart-section { background: var(--card); border-radius: 4px; padding: 8px; margin-bottom: 12px; }
.chart-title { font-size: var(--font-xs); color: var(--muted); text-transform: uppercase; margin-bottom: 4px; }
.pnl-summary { text-align: center; font-size: 18px; font-weight: 700; }
.pnl-summary .up { color: var(--up); } .pnl-summary .down { color: var(--down); }
</style>
