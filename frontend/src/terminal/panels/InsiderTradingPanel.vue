<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

const trades = computed(() => store.research?.insider ?? [])

const netActivity = computed(() => {
  const list = trades.value
  if (!list || list.length === 0) return { label: 'Neutral', color: '#6b7280' }
  let buys = 0, sells = 0
  for (const t of list) {
    const shares = t.shares ?? 0
    if ((t.type ?? '').toLowerCase() === 'buy') buys += shares
    else sells += shares
  }
  if (buys > sells) return { label: 'Bullish', color: '#22c55e' }
  if (sells > buys) return { label: 'Bearish', color: '#ef4444' }
  return { label: 'Neutral', color: '#6b7280' }
})

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal, ['insider'])
}, { immediate: true })

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

function refresh() { store.fetchStockResearch(symbol.value, ['insider']) }

function formatShares(v: number | undefined | null): string {
  if (v == null) return '--'
  return v.toLocaleString()
}

function handleSymbolSubmit(e: Event) {
  const input = e.target as HTMLInputElement
  symbol.value = input.value.trim().toUpperCase()
  input.blur()
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>Insider Trading — {{ symbol.toUpperCase() }}</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          placeholder="Symbol..."
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="store.isBridgeAvailable === false" class="mock-banner">
      Mock data — Python sidecar not connected
    </div>

    <div v-if="trades.length > 0" class="panel-content">
      <!-- Net Activity Indicator -->
      <div class="activity-bar">
        <span class="activity-label">Net Activity</span>
        <span class="activity-indicator" :style="{ color: netActivity.color }">
          <span class="activity-dot" :style="{ background: netActivity.color }"></span>
          {{ netActivity.label }}
        </span>
      </div>

      <!-- Trades Table -->
      <table class="insider-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Role</th>
            <th>Type</th>
            <th>Shares</th>
            <th>Price</th>
            <th>Value</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(t, i) in trades" :key="t.name ?? i">
            <td>{{ t.name }}</td>
            <td class="role-cell">{{ t.role }}</td>
            <td>
              <span :class="['type-badge', (t.type ?? '').toLowerCase()]">{{ t.type }}</span>
            </td>
            <td class="num-cell">{{ formatShares(t.shares) }}</td>
            <td class="num-cell">{{ t.price != null ? '$' + Number(t.price).toFixed(2) : '--' }}</td>
            <td class="num-cell">{{ t.value != null ? '$' + Number(t.value).toLocaleString() : '--' }}</td>
            <td class="date-cell">{{ t.date }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="empty-state">
      <p>Enter a symbol and press ↵ to view insider trades</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg, #111827); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: 4px; background: #78350f; color: #fbbf24; font-size: 12px; text-align: center; }
.panel-content { flex: 1; overflow-y: auto; }
.activity-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; padding: 10px 12px; border: 1px solid #374151; border-radius: 6px; background: #1f2937; font-size: 13px; }
.activity-label { color: #9ca3af; font-weight: 500; }
.activity-indicator { display: flex; align-items: center; gap: 6px; font-weight: 700; }
.activity-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.insider-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.insider-table th { text-align: left; padding: 6px 8px; color: #9ca3af; border-bottom: 1px solid #374151; font-weight: 500; white-space: nowrap; }
.insider-table td { padding: 6px 8px; border-bottom: 1px solid #1f2937; }
.role-cell { color: #9ca3af; }
.type-badge { padding: 2px 10px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.type-badge.buy { background: #14532d; color: #22c55e; }
.type-badge.sell { background: #7f1d1d; color: #ef4444; }
.num-cell { text-align: right; font-variant-numeric: tabular-nums; }
.date-cell { color: #9ca3af; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; font-size: 13px; }
</style>
