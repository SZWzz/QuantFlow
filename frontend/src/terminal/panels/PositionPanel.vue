<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import type { PositionDetail } from '@/stores/portfolio'
import { logger } from '@/lib/logger'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()
const loading = ref(false)
const expandedSymbol = ref<string | null>(null)

async function loadPositions() {
  loading.value = true
  try { await store.fetchPositions() } catch (e) { logger.error('[Position] fetch:', e) } finally { loading.value = false }
}

const positions = computed<PositionDetail[]>(() => store.positions)

const totalPnl = computed(() => positions.value.reduce((s, p) => s + (p.pnl || 0), 0))

function fmt(n: number, dec = 2): string { return n.toFixed(dec) }
function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(1) + '万'
  return n.toFixed(2)
}
function pnlClass(v: number): string { return v >= 0 ? 'up' : 'down' }
function pnlSign(v: number): string { return v >= 0 ? '+' : '' }

function toggleExpand(symbol: string) {
  expandedSymbol.value = expandedSymbol.value === symbol ? null : symbol
}

const expandedPosition = computed(() => {
  if (!expandedSymbol.value) return null
  return positions.value.find(p => p.symbol === expandedSymbol.value) || null
})

onMounted(loadPositions)
</script>

<template>
  <div class="position-panel" data-testid="position-panel">
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
    <div v-else-if="!positions.length" class="empty-state" data-testid="position-empty">{{ $t('portfolio.no_positions') }}</div>

    <div v-else class="position-list">
      <div
        v-for="pos in positions"
        :key="pos.symbol"
        :data-testid="'position-row'"
      >
        <div class="position-row" :class="{ expanded: expandedSymbol === pos.symbol }" @click="toggleExpand(pos.symbol)">
          <div class="pos-main">
            <span class="pos-symbol">{{ pos.symbol }} - {{ pos.name || '' }}</span>
            <span class="pos-qty">{{ pos.quantity }} 手</span>
            <span class="pos-expand-icon">{{ expandedSymbol === pos.symbol ? '▴' : '▾' }}</span>
          </div>
          <div class="pos-prices">
            <span class="pos-avg">{{ $t('portfolio.avg_price') }} {{ fmt(pos.avg_price) }}</span>
            <span class="pos-mkt">{{ $t('portfolio.market_price') }} {{ fmt(pos.market_price) }}</span>
          </div>
          <div class="pos-pnl" :class="pnlClass(pos.pnl)">
            <span class="pnl-val">{{ pnlSign(pos.pnl) }}{{ fmtMoney(pos.pnl) }}</span>
            <span class="pnl-pct">({{ pnlSign(pos.pnl_pct) }}{{ fmt(pos.pnl_pct) }}%)</span>
          </div>
        </div>

        <!-- Inline detail (expanded) -->
        <div v-if="expandedSymbol === pos.symbol" class="pos-detail">
          <div class="detail-kpi-grid">
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.quantity') }}</span>
              <span class="detail-kpi-value">{{ pos.quantity }}</span>
            </div>
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.avg_price') }}</span>
              <span class="detail-kpi-value">${{ fmt(pos.avg_price) }}</span>
            </div>
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.market_price') }}</span>
              <span class="detail-kpi-value">${{ fmt(pos.market_price) }}</span>
            </div>
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.market_value') }}</span>
              <span class="detail-kpi-value">${{ fmtMoney(pos.market_price * pos.quantity) }}</span>
            </div>
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.pnl') }}</span>
              <span :class="['detail-kpi-value', pos.pnl >= 0 ? 'up' : 'down']">{{ pnlSign(pos.pnl) }}${{ fmtMoney(pos.pnl) }}</span>
            </div>
            <div class="detail-kpi-item">
              <span class="detail-kpi-label">{{ $t('portfolio.alloc') }}</span>
              <span class="detail-kpi-value">{{ fmt(pos.alloc_pct) }}%</span>
            </div>
          </div>
          <div class="detail-pnl-summary">
            <span :class="pos.pnl >= 0 ? 'up' : 'down'">{{ pos.pnl >= 0 ? '+' : '' }}${{ fmtMoney(pos.pnl) }} ({{ pos.pnl >= 0 ? '+' : '' }}{{ fmt(pos.pnl_pct) }}%)</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.position-panel { padding: 10px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.summary-row { display: flex; gap: 8px; margin-bottom: 10px; }
.summary-item { flex: 1; padding: 8px; background: var(--color-bg-subtle); border-radius: var(--radius-sm); text-align: center; }
.s-label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.s-value { font-size: 16px; font-weight: 700; }
.s-value.up { color: var(--color-down); } .s-value.down { color: var(--color-up); }

.position-row {
  padding: 8px; border-bottom: 1px solid var(--color-bg-input);
  cursor: pointer; transition: background 0.15s;
}
.position-row:hover { background: var(--color-bg-subtle); }
.position-row.expanded { background: var(--color-bg-subtle); }

.pos-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; }
.pos-symbol { font-weight: 600; font-size: 13px; color: var(--color-text-primary); }
.pos-qty { font-size: 11px; color: var(--color-text-tertiary); }
.pos-expand-icon { font-size: 10px; color: var(--color-text-tertiary); margin-left: 4px; }
.pos-prices { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 2px; display: flex; gap: 12px; }
.pos-pnl { font-size: 12px; font-weight: 500; }
.pos-pnl.up { color: var(--color-down); } .pos-pnl.down { color: var(--color-up); }
.pnl-pct { font-size: 10px; opacity: 0.7; }

/* Inline detail */
.pos-detail {
  padding: 8px 12px 12px;
  border-bottom: 1px solid var(--color-bg-input);
  background: var(--color-bg-subtle);
}

.detail-kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; margin-bottom: 8px; }
.detail-kpi-item { padding: 8px; background: var(--color-bg-panel); border-radius: var(--radius-sm); text-align: center; }
.detail-kpi-label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 3px; }
.detail-kpi-value { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.detail-kpi-value.up { color: var(--color-down); } .detail-kpi-value.down { color: var(--color-up); }
.detail-pnl-summary { text-align: center; font-size: 16px; font-weight: 700; }
.detail-pnl-summary .up { color: var(--color-down); } .detail-pnl-summary .down { color: var(--color-up); }

.up { color: var(--color-down); }
.down { color: var(--color-up); }
</style>
