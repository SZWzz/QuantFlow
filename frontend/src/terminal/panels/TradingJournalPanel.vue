<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import type { Trade } from '@/stores/portfolio'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { logger } from '@/lib/logger'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const portfolio = usePortfolioStore()

// -- Tab state --
const activeTab = ref<'journal' | 'events'>('journal')

// ===========================================================================
// Journal tab
// ===========================================================================
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

// ===========================================================================
// Events tab
// ===========================================================================
interface TradeEvent {
  id: string
  title: string
  status: 'info'
  time: string
  detail: string
}

const DISPLAY_LIMIT = 12
const events = ref<TradeEvent[]>([])

// ===========================================================================
// Data fetching (shared)
// ===========================================================================
async function fetchData() {
  loading.value = true
  try {
    await portfolio.fetchTrades()
    const trades = portfolio.trades as Trade[]

    // -- Journal: group by date --
    if (trades && trades.length > 0) {
      const map = new Map<string, { total: number; symbols: Map<string, { name: string; pnl: number }> }>()
      for (const t of trades) {
        const date = t.executed_at?.slice(0, 10) || new Date().toISOString().slice(0, 10)
        const sym = t.symbol || ''
        const pnl = (t as any).pnl || 0
        if (!map.has(date)) map.set(date, { total: 0, symbols: new Map() })
        const g = map.get(date)!
        g.total += pnl
        if (!g.symbols.has(sym)) g.symbols.set(sym, { name: t.name || sym, pnl: 0 })
        g.symbols.get(sym)!.pnl += pnl
      }
      groups.value = Array.from(map.entries()).map(([date, g]) => ({
        date,
        total_pnl: Math.round(g.total * 100) / 100,
        trades_count: g.symbols.size,
        by_symbol: Array.from(g.symbols.entries()).map(([sym, d]) => ({ symbol: sym, name: d.name, pnl: Math.round(d.pnl * 100) / 100 })),
      }))
    }

    // -- Events: derive from trades --
    if (trades && trades.length > 0) {
      events.value = trades.slice(0, DISPLAY_LIMIT).map((t, i) => ({
        id: t.trade_id || `evt-${i}`,
        title: `${t.side} ${t.symbol}`,
        status: 'info' as const,
        time: t.executed_at || new Date().toISOString(),
        detail: `${t.quantity.toLocaleString()}股 @ ${t.price}`,
      }))
    } else {
      events.value = []
    }
  } catch (e) {
    logger.error('[TradingJournal]', e)
    groups.value = []
    events.value = []
  } finally {
    loading.value = false
  }
}

// -- Helpers --
function formatPnl(v: number): string {
  return (v >= 0 ? '+' : '') + v.toLocaleString('en-US', { minimumFractionDigits: 2 })
}

function formatEventTime(iso: string): string {
  const d = new Date(iso)
  const now = Date.now()
  const diffMin = Math.floor((now - d.getTime()) / 60000)
  if (diffMin < 1) return 'Just now'
  if (diffMin < 60) return diffMin + 'm ago'
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return diffHrs + 'h ago'
  const diffDays = Math.floor(diffHrs / 24)
  return diffDays + 'd ago'
}

function dismissEvent(id: string) {
  events.value = events.value.filter(e => e.id !== id)
}

// -- Sorted events (newest first) --
const sortedEvents = computed(() => {
  return [...events.value].sort((a, b) =>
    new Date(b.time).getTime() - new Date(a.time).getTime()
  )
})

onMounted(fetchData)
</script>

<template>
  <div class="trading-journal-panel">
    <!-- Tab bar -->
    <div class="tab-bar">
      <button
        :class="['tab-btn', { active: activeTab === 'journal' }]"
        @click="activeTab = 'journal'"
      >{{ $t('misc.trading_journal') }}</button>
      <button
        :class="['tab-btn', { active: activeTab === 'events' }]"
        @click="activeTab = 'events'"
      >{{ $t('workflow.action_center') }}</button>
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <!-- ================================================================ -->
    <!-- Journal Tab -->
    <!-- ================================================================ -->
    <template v-if="activeTab === 'journal'">
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

      <div class="sort-tabs">
        <button :class="['s-tab', { active: sortBy === 'date' }]" @click="sortBy = 'date'">{{ $t('common.date') }}</button>
        <button :class="['s-tab', { active: sortBy === 'pnl' }]" @click="sortBy = 'pnl'">P&L</button>
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
    </template>

    <!-- ================================================================ -->
    <!-- Events Tab -->
    <!-- ================================================================ -->
    <template v-if="activeTab === 'events'">
      <div v-if="loading" class="status">加载中...</div>

      <!-- Event feed -->
      <div v-if="events.length > 0" class="event-feed">
        <div
          v-for="ev in sortedEvents"
          :key="ev.id"
          class="event-card border-info"
        >
          <div class="event-left">
            <span class="event-icon">&#9679;</span>
          </div>
          <div class="event-body">
            <div class="event-header">
              <span class="event-type-label">{{ ev.title }}</span>
              <span class="event-time">{{ formatEventTime(ev.time) }}</span>
            </div>
            <p class="event-message">{{ ev.detail }}</p>
            <div class="event-actions">
              <button class="evt-btn dismiss-btn" @click="dismissEvent(ev.id)">
                忽略
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else-if="!loading" class="empty-state">
        <p class="empty-icon">&#10003;</p>
        <p class="empty-text">{{ $t('workflow.no_recent_trades') }}</p>
        <p class="empty-sub">{{ $t('workflow.no_trades') }}</p>
      </div>
    </template>
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

/* -- Tab bar -- */
.tab-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
  flex-shrink: 0;
}

.tab-btn {
  padding: 5px 14px;
  background: var(--input, var(--color-bg-elevated));
  border: 1px solid var(--border, var(--color-border-strong));
  color: var(--muted, var(--color-text-tertiary));
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.tab-btn:first-child { border-radius: var(--radius-sm) 0 0 4px; }
.tab-btn:last-of-type { border-radius: 0 4px 4px 0; }
.tab-btn.active {
  background: var(--card, var(--color-bg-panel));
  border-color: var(--accent, var(--color-accent));
  color: var(--accent, var(--color-accent));
}

.refresh-btn {
  margin-left: auto;
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.status { display: flex; align-items: center; justify-content: center; padding: 20px; color: var(--muted); font-size: 13px; }

.empty-icon { font-size: 32px; color: var(--muted); margin: 0; }
.empty-text { font-size: 14px; color: var(--muted); margin: 0; }
.empty-sub { font-size: var(--font-xs); color: var(--muted); margin: 0; }

.sort-tabs { display: flex; gap: 4px; margin-bottom: 8px; }
.s-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.s-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }

.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 8px; }
.stat-card { padding: 10px; border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); text-align: center; }
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
  font-size: 10px; padding: 1px 6px; border-radius: var(--radius-sm); font-weight: 500;
}
.breakdown-chip.up { color: var(--color-up); background: rgba(220,38,38,0.1); }
.breakdown-chip.down { color: var(--color-down); background: rgba(22,163,74,0.1); }

/* -- Event feed -- */
.event-feed {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.event-card {
  display: flex;
  gap: 10px;
  padding: 10px;
  background: var(--card);
  border-radius: var(--radius-sm);
  border-left: 3px solid;
}

.border-info { border-left-color: var(--color-accent); }

.event-left {
  flex-shrink: 0;
  padding-top: 2px;
}

.event-icon {
  font-size: 14px;
  color: var(--color-accent);
}

.event-body {
  flex: 1;
  min-width: 0;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.event-type-label {
  font-size: var(--font-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
  color: var(--muted);
}

.event-time {
  font-size: var(--font-xs);
  color: var(--muted);
}

.event-message {
  font-size: 12px;
  color: var(--text);
  margin: 0 0 6px 0;
  line-height: 1.4;
}

.event-actions {
  display: flex;
  gap: 6px;
}

.evt-btn {
  padding: 3px 10px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
  border: none;
}
.evt-btn:hover { opacity: 0.8; }

.dismiss-btn {
  background: transparent;
  color: var(--muted);
  border: 1px solid var(--border);
}
</style>
