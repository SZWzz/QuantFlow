<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const portfolio = usePortfolioStore()

interface TradeGroup {
  date: string
  total_pnl: number
  trades_count: number
  by_symbol: { symbol: string; name: string; pnl: number }[]
}

const groups = ref<TradeGroup[]>([])
const loading = ref(false)
const sortBy = ref<'date' | 'pnl'>('date')

const sortedGroups = computed(() => {
  const arr = [...groups.value]
  if (sortBy.value === 'pnl') arr.sort((a, b) => Math.abs(b.total_pnl) - Math.abs(a.total_pnl))
  else arr.sort((a, b) => b.date.localeCompare(a.date))
  return arr
})

const stats = computed(() => {
  let totalPnl = 0
  let winCount = 0
  let lossCount = 0
  for (const g of groups.value) {
    totalPnl += g.total_pnl
    if (g.total_pnl > 0) winCount++
    else if (g.total_pnl < 0) lossCount++
  }
  return { totalPnl, winCount, lossCount, totalDays: groups.value.length }
})

async function fetchData() {
  loading.value = true
  try {
    await portfolio.fetchTrades()
    if (portfolio.trades && portfolio.trades.length > 0) {
      const map = new Map<string, { total: number; symbols: Map<string, { name: string; pnl: number }> }>()
      for (const t of portfolio.trades) {
        const date = (t as any).date?.slice(0, 10) || new Date().toISOString().slice(0, 10)
        const sym = (t as any).symbol || ''
        const pnl = (t as any).pnl || 0
        if (!map.has(date)) map.set(date, { total: 0, symbols: new Map() })
        const g = map.get(date)!
        g.total += pnl
        if (!g.symbols.has(sym)) g.symbols.set(sym, { name: sym, pnl: 0 })
        g.symbols.get(sym)!.pnl += pnl
      }
      groups.value = Array.from(map.entries()).map(([date, g]) => ({
        date,
        total_pnl: Math.round(g.total * 100) / 100,
        trades_count: g.symbols.size,
        by_symbol: Array.from(g.symbols.entries()).map(([sym, d]) => ({ symbol: sym, name: d.name, pnl: Math.round(d.pnl * 100) / 100 })),
      }))
    }
  } catch (e) {
    console.error('[TradingJournal]', e)
    groups.value = []
  } finally {
    loading.value = false
  }
}

function formatPnl(v: number): string {
  return (v >= 0 ? '+' : '') + v.toLocaleString('en-US', { minimumFractionDigits: 2 })
}

onMounted(fetchData)
</script>

<template>
  <div class="trading-journal-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.trading_journal') }}</h3>
      <div class="sort-tabs">
        <button :class="['s-tab', { active: sortBy === 'date' }]" @click="sortBy = 'date'">{{ $t('common.date') }}</button>
        <button :class="['s-tab', { active: sortBy === 'pnl' }]" @click="sortBy = 'pnl'">P&L</button>
      </div>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-label">{{ $t('misc.total_pnl') }}</div>
        <div class="stat-value" :class="stats.totalPnl >= 0 ? 'up' : 'down'">{{ formatPnl(stats.totalPnl) }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('misc.win_days') }}</div>
        <div class="stat-value up">{{ stats.winCount }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('misc.loss_days') }}</div>
        <div class="stat-value down">{{ stats.lossCount }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('misc.win_rate') }}</div>
        <div class="stat-value">{{ stats.totalDays > 0 ? (stats.winCount / stats.totalDays * 100).toFixed(1) + '%' : '--' }}</div>
      </div>
    </div>

    <SkeletonPanel v-if="loading && groups.length === 0" type="table" :rows="6" />

    <div v-else-if="groups.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>

    <div v-else class="table-wrapper">
      <div class="table-header">
        <span class="col-date">{{ $t('common.date') }}</span>
        <span class="col-pnl">P&L</span>
        <span class="col-count">{{ $t('misc.trades_count') }}</span>
        <span class="col-breakdown">{{ $t('misc.breakdown') }}</span>
      </div>
      <div class="table-body">
        <div v-for="g in sortedGroups" :key="g.date" class="trade-group">
          <div class="group-header">
            <span class="col-date">{{ g.date }}</span>
            <span class="col-pnl" :class="g.total_pnl >= 0 ? 'up' : 'down'">{{ formatPnl(g.total_pnl) }}</span>
            <span class="col-count">{{ g.trades_count }}</span>
            <span class="col-breakdown">
              <span v-for="s in g.by_symbol" :key="s.symbol" class="breakdown-chip" :class="s.pnl >= 0 ? 'up' : 'down'">
                {{ s.symbol }} {{ formatPnl(s.pnl) }}
              </span>
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.trading-journal-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
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
.sort-tabs { display: flex; gap: 4px; }
.s-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.s-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 12px; }
.stat-card { padding: 10px; border: 1px solid var(--color-border-subtle); border-radius: 8px; text-align: center; }
.stat-label { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.stat-value { font-size: 15px; font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.trade-group { margin-bottom: 2px; }
.group-header {
  display: flex; padding: 4px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.group-header:hover { background: var(--color-bg-elevated); }
.col-date { width: 80px; }
.col-pnl { width: 80px; text-align: right; font-weight: 600; font-variant-numeric: tabular-nums; }
.col-count { width: 48px; text-align: center; color: var(--color-text-secondary); }
.col-breakdown { flex: 1; min-width: 0; display: flex; gap: 4px; flex-wrap: wrap; }
.breakdown-chip {
  font-size: 10px; padding: 1px 6px; border-radius: 4px; font-weight: 500;
}
.breakdown-chip.up { color: var(--color-up); background: rgba(220,38,38,0.1); }
.breakdown-chip.down { color: var(--color-down); background: rgba(22,163,74,0.1); }
</style>
