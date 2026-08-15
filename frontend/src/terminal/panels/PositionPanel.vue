<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePortfolioStore } from '@/stores/portfolio'
import type { PositionDetail } from '@/stores/portfolio'
import { logger } from '@/lib/logger'
import { PanelHeader, PanelTable, StatItem, EmptyState, ErrorState, type Column } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const store = usePortfolioStore()
const loading = ref(false)
const fetchError = ref<string | null>(null)
const expandedSymbol = ref<string | null>(null)

async function loadPositions() {
  loading.value = true
  fetchError.value = null
  try {
    await store.fetchPositions()
    fetchError.value = store.error
  } catch (e) {
    logger.error('[Position] fetch:', e)
    fetchError.value = String(e)
  } finally {
    loading.value = false
  }
}

const positions = computed<PositionDetail[]>(() => store.positions)

interface PositionRow extends PositionDetail { label: string }
const rows = computed<PositionRow[]>(() =>
  positions.value.map(p => ({ ...p, label: p.name ? `${p.symbol} - ${p.name}` : p.symbol })),
)

const totalPnl = computed(() => positions.value.reduce((s, p) => s + (p.pnl || 0), 0))
const totalPnlPct = computed(() => {
  const cost = positions.value.reduce((s, p) => s + (p.cost_basis || 0), 0)
  return cost > 0 ? (totalPnl.value / cost) * 100 : undefined
})

const columns = computed<Column[]>(() => [
  { key: 'label', label: t('portfolio.symbol'), flex: 2 },
  { key: 'quantity', label: t('portfolio.quantity'), align: 'right', mono: true, formatter: (v: number) => `${v} 手` },
  { key: 'avg_price', label: t('portfolio.avg_price'), align: 'right', format: 'price' },
  { key: 'market_price', label: t('portfolio.market_price'), align: 'right', format: 'price' },
  { key: 'pnl', label: t('portfolio.pnl'), align: 'right', mono: true, colorize: true, formatter: fmtSignedMoney },
  { key: 'pnl_pct', label: t('portfolio.pnl_pct'), align: 'right', format: 'percent', colorize: true },
])

function fmt(n: number, dec = 2): string { return n.toFixed(dec) }
function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(1) + '万'
  return n.toFixed(2)
}
function fmtSignedMoney(n: number): string { return (n >= 0 ? '+' : '') + fmtMoney(n) }
function pnlClass(v: number): string { return v >= 0 ? 'up' : 'down' }
function pnlSign(v: number): string { return v >= 0 ? '+' : '' }

function onRowClick(row: PositionRow) {
  expandedSymbol.value = expandedSymbol.value === row.symbol ? null : row.symbol
}

const expandedPosition = computed(() => {
  if (!expandedSymbol.value) return null
  return positions.value.find(p => p.symbol === expandedSymbol.value) || null
})

onMounted(loadPositions)
</script>

<template>
  <div class="position-panel" data-testid="position-panel">
    <PanelHeader
      :title="t('portfolio.positions')"
      :controls="[{ icon: 'refresh', title: t('common.refresh'), action: loadPositions, loading }]"
    />

    <div v-if="!loading && positions.length > 0" class="summary-row">
      <StatItem
        :label="t('portfolio.total_pnl')"
        :value="pnlSign(totalPnl) + fmt(totalPnl)"
        :delta="totalPnlPct"
      />
      <StatItem :label="t('portfolio.position_count')" :value="positions.length" />
    </div>

    <ErrorState
      v-if="fetchError && !positions.length"
      :title="t('common.panel_error')"
      :description="fetchError"
      :retry-label="t('common.retry')"
      @retry="loadPositions"
    />
    <EmptyState
      v-else-if="!loading && !positions.length"
      :title="t('portfolio.no_positions')"
      data-testid="position-empty"
    />
    <template v-else>
      <PanelTable
        :columns="columns"
        :data="rows"
        :loading="loading"
        clickable
        row-test-id="position-row"
        @row-click="onRowClick"
      >
        <template #action="{ row }">
          <span class="expand-icon">{{ expandedSymbol === row.symbol ? '▴' : '▾' }}</span>
        </template>
      </PanelTable>

      <!-- Inline detail (expanded) -->
      <div v-if="expandedPosition" class="pos-detail">
        <div class="detail-kpi-grid">
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.quantity') }}</span>
            <span class="detail-kpi-value">{{ expandedPosition.quantity }}</span>
          </div>
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.avg_price') }}</span>
            <span class="detail-kpi-value">${{ fmt(expandedPosition.avg_price) }}</span>
          </div>
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.market_price') }}</span>
            <span class="detail-kpi-value">${{ fmt(expandedPosition.market_price) }}</span>
          </div>
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.market_value') }}</span>
            <span class="detail-kpi-value">${{ fmtMoney(expandedPosition.market_price * expandedPosition.quantity) }}</span>
          </div>
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.pnl') }}</span>
            <span :class="['detail-kpi-value', pnlClass(expandedPosition.pnl)]">{{ pnlSign(expandedPosition.pnl) }}${{ fmtMoney(expandedPosition.pnl) }}</span>
          </div>
          <div class="detail-kpi-item">
            <span class="detail-kpi-label">{{ t('portfolio.alloc') }}</span>
            <span class="detail-kpi-value">{{ fmt(expandedPosition.alloc_pct) }}%</span>
          </div>
        </div>
        <div class="detail-pnl-summary">
          <span :class="pnlClass(expandedPosition.pnl)">{{ pnlSign(expandedPosition.pnl) }}${{ fmtMoney(expandedPosition.pnl) }} ({{ pnlSign(expandedPosition.pnl_pct) }}{{ fmt(expandedPosition.pnl_pct) }}%)</span>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.position-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

.summary-row {
  display: flex;
  gap: var(--space-xl);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}

.expand-icon { font-size: var(--font-xs); color: var(--color-text-tertiary); }

/* Inline detail */
.pos-detail {
  flex-shrink: 0;
  max-height: 45%;
  overflow-y: auto;
  padding: var(--space-sm) var(--panel-padding) var(--space-md);
  border-top: 1px solid var(--color-border-subtle);
  background: var(--color-bg-subtle);
}

.detail-kpi-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-xs); margin-bottom: var(--space-sm); }
.detail-kpi-item { padding: var(--space-sm); background: var(--color-bg-panel); border-radius: var(--radius-sm); text-align: center; }
.detail-kpi-label { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: var(--space-xs); }
.detail-kpi-value { font-size: var(--font-lg); font-weight: 600; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.detail-pnl-summary { text-align: center; font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; }

.up { color: var(--color-up); }
.down { color: var(--color-down); }
</style>
