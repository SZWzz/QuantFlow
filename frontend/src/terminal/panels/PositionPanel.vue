<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Position {
  Symbol: string; Quantity: number; AvgPrice: number; MarketPrice: number
  PnL: number; PnLPct: number
}

const positions = ref<Position[]>([])
const loading = ref(false)

async function loadPositions() {
  loading.value = true
  try {
    const result = await (window as any).go.main.App.GetPositions()
    positions.value = Array.isArray(result) ? result : []
  } catch {
    positions.value = []
  } finally {
    loading.value = false
  }
}

const totalPnl = computed(() => positions.value.reduce((s, p) => s + (p.PnL || 0), 0))
function fmt(n: number, dec = 2): string { return n.toFixed(dec) }
function pnlClass(v: number) { return v >= 0 ? 'up' : 'down' }
function pnlSign(v: number) { return v >= 0 ? '+' : '' }

onMounted(loadPositions)
</script>

<template>
  <div class="position-panel">
    <div class="summary-row" v-if="!loading && positions.length > 0">
      <div class="summary-item">
        <span class="s-label">{{ $t('portfolio.total_pnl') }}</span>
        <span :class="['s-value', pnlClass(totalPnl)]">{{ pnlSign(totalPnl) }}{{ fmt(totalPnl) }}</span>
      </div>
      <div class="summary-item">
        <span class="s-label">{{ $t('portfolio.position_count') }}</span>
        <span class="s-value" style="color:var(--color-text-primary)">{{ positions.length }}</span>
      </div>
    </div>

    <div v-if="loading" class="empty-state">{{ $t('common.loading') }}</div>
    <div v-else-if="!positions.length" class="empty-state">{{ $t('portfolio.no_positions') }}</div>

    <div v-else class="position-list">
      <div v-for="pos in positions" :key="pos.Symbol" class="position-row">
        <div class="pos-main">
          <span class="pos-symbol">{{ pos.Symbol }}</span>
          <span class="pos-qty">{{ pos.Quantity }} 手</span>
        </div>
        <div class="pos-prices">
          <span class="pos-avg">{{ $t('portfolio.avg_price') }} {{ fmt(pos.AvgPrice) }}</span>
          <span class="pos-mkt">{{ $t('portfolio.market_price') }} {{ fmt(pos.MarketPrice) }}</span>
        </div>
        <div class="pos-pnl" :class="pnlClass(pos.PnL)">
          <span class="pnl-val">{{ pnlSign(pos.PnL) }}{{ fmt(pos.PnL) }}</span>
          <span class="pnl-pct">({{ pnlSign(pos.PnLPct) }}{{ fmt(pos.PnLPct) }}%)</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.position-panel { padding: 10px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.summary-row { display: flex; gap: 8px; margin-bottom: 10px; }
.summary-item { flex: 1; padding: 8px; background: var(--color-bg-subtle); border-radius: 4px; text-align: center; }
.s-label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.s-value { font-size: 16px; font-weight: 700; }
.s-value.up { color: #3fb950; } .s-value.down { color: #f85149; }
.empty-state { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); font-size: 13px; }

.position-row { padding: 8px; border-bottom: 1px solid var(--color-bg-input); }
.pos-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; }
.pos-symbol { font-weight: 600; font-size: 13px; color: var(--color-text-primary); }
.pos-qty { font-size: 11px; color: var(--color-text-tertiary); }
.pos-prices { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 2px; display: flex; gap: 12px; }
.pos-pnl { font-size: 12px; font-weight: 500; }
.pos-pnl.up { color: #3fb950; } .pos-pnl.down { color: #f85149; }
.pnl-pct { font-size: 10px; opacity: 0.7; }
</style>
