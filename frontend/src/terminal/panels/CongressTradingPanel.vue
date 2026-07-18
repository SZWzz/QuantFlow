<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()

const loadError = ref('')

const partyFilter = ref('All')
const chamberFilter = ref('All')

const parties = ['All', 'Democrat', 'Republican', 'Independent']
const chambers = ['All', 'House', 'Senate']

const filteredTrades = computed(() => {
  const trades = store.congressTrades ?? []
  return trades.filter((t: any) => {
    if (partyFilter.value !== 'All' && t.party !== partyFilter.value) return false
    if (chamberFilter.value !== 'All' && t.chamber !== chamberFilter.value) return false
    return true
  })
})

const totalBuyVolume = computed(() => {
  const isBuy = (t: any) => (t.type ?? '').toLowerCase() === 'buy'
  const buys = filteredTrades.value.filter(isBuy)
  const sells = filteredTrades.value.filter((t: any) => (t.type ?? '').toLowerCase() === 'sell')
  return `${buys.length} Buys / ${sells.length} Sells`
})

onMounted(async () => {
  loadError.value = ''
  try {
    await store.fetchCongressTrades()
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  }
})

function setPartyFilter(p: string) { partyFilter.value = p }
function setChamberFilter(c: string) { chamberFilter.value = c }

async function refresh() {
  loadError.value = ''
  try {
    await store.fetchCongressTrades()
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  }
}

function amountColor(amount: string): string {
  if (!amount) return 'var(--color-text-tertiary)'
  const top = ['$1M-$5M', '$5M+', '$500K-$1M']
  return top.some(t => amount.includes(t)) ? 'var(--chart-3)' : 'var(--color-text-primary)'
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>{{ $t('research.congress') }}</h3>
      <div class="header-controls">
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filter-bar">
      <div class="filter-group">
        <span class="filter-label">{{ $t('research.party') }}</span>
        <div class="filter-buttons">
          <button
            v-for="p in parties" :key="p"
            :class="['filter-btn', { active: partyFilter === p }]"
            @click="setPartyFilter(p)"
          >{{ p === 'All' ? $t('common.all') : p }}</button>
        </div>
      </div>
      <div class="filter-group">
        <span class="filter-label">{{ $t('research.chamber') }}</span>
        <div class="filter-buttons">
          <button
            v-for="c in chambers" :key="c"
            :class="['filter-btn', { active: chamberFilter === c }]"
            @click="setChamberFilter(c)"
          >{{ c === 'All' ? $t('common.all') : c }}</button>
        </div>
      </div>
    </div>

    <!-- Summary -->
    <div class="summary-bar" v-if="store.congressTrades">
      <span>Showing {{ filteredTrades.length }} / {{ store.congressTrades.length }} trades</span>
      <span>{{ totalBuyVolume }}</span>
    </div>

    <!-- Trades Table -->
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <div v-else-if="store.congressTrades" class="panel-content">
      <table v-if="filteredTrades.length > 0" class="congress-table">
        <thead>
          <tr>
            <th>{{ $t('common.name') }}</th>
            <th>{{ $t('research.chamber') }}</th>
            <th>{{ $t('research.party') }}</th>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('common.type') }}</th>
            <th>{{ $t('common.amount') }}</th>
            <th>{{ $t('common.date') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(t, i) in filteredTrades" :key="t.name + t.symbol + t.date + i">
            <td class="name-cell">{{ t.name }}</td>
            <td>{{ t.chamber }}</td>
            <td>
              <span :class="['party-badge', (t.party ?? '').toLowerCase()]">{{ t.party }}</span>
            </td>
            <td class="symbol-cell">{{ t.symbol }}</td>
            <td>
              <span :class="['type-badge', (t.type ?? '').toLowerCase()]">{{ t.type }}</span>
            </td>
            <td class="amount-cell" :style="{ color: amountColor(t.amount) }">{{ t.amount }}</td>
            <td class="date-cell">{{ t.date }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="no-data">{{ $t('research.congress_no_match') }}</p>
    </div>

    <div v-else class="empty-state">
      <p>{{ $t('common.no_data') }}</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text-primary); background: var(--color-bg-panel); }

.header-controls { display: flex; gap: 8px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: var(--radius-sm); background: var(--color-accent-soft); color: var(--color-accent); font-size: 12px; text-align: center; }
.filter-bar { display: flex; gap: 16px; margin-bottom: 10px; flex-wrap: wrap; }
.filter-group { display: flex; align-items: center; gap: 6px; }
.filter-label { font-size: 11px; color: var(--color-text-primary); font-weight: 500; text-transform: uppercase; }
.filter-buttons { display: flex; gap: 2px; }
.filter-btn { padding: 3px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-secondary); cursor: pointer; font-size: 11px; }
.filter-btn.active { background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent); }
.summary-bar { display: flex; justify-content: space-between; padding: 6px 10px; margin-bottom: 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); font-size: 11px; color: var(--color-text-secondary); }
.panel-content { flex: 1; overflow-y: auto; }
.congress-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.congress-table th { text-align: left; padding: 6px 8px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.congress-table td { padding: 6px 8px; border-bottom: 1px solid var(--color-bg-elevated); color: var(--color-text-primary); }
.name-cell { font-weight: 500; color: var(--color-text-primary); }
.symbol-cell { font-weight: 600; color: var(--color-accent); }
.party-badge { padding: 2px 8px; border-radius: var(--radius-sm); font-size: 11px; font-weight: 600; }
.party-badge.democrat { background: var(--color-accent-soft); color: var(--color-accent); }
.party-badge.republican { background: var(--color-up-bg, rgba(239,68,68,0.08)); color: var(--color-up); }
.party-badge.independent { background: var(--color-border-subtle); color: var(--color-text-secondary); }
.type-badge { padding: 2px 10px; border-radius: var(--radius-lg); font-size: 11px; font-weight: 600; }
.type-badge.buy { background: var(--color-down-bg, rgba(34,197,94,0.08)); color: var(--color-down); }
.type-badge.sell { background: var(--color-up-bg, rgba(239,68,68,0.08)); color: var(--color-up); }
.amount-cell { font-variant-numeric: tabular-nums; font-weight: 500; color: var(--color-text-primary); }
.date-cell { color: var(--color-text-primary); }
.no-data { color: var(--color-text-tertiary); font-size: 13px; text-align: center; padding: 20px; }

.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
</style>
