<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()

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

onMounted(() => {
  store.fetchCongressTrades()
})

function set党派Filter(p: string) { partyFilter.value = p }
function set议院Filter(c: string) { chamberFilter.value = c }

function refresh() { store.fetchCongressTrades() }

function amountColor(amount: string): string {
  if (!amount) return '#e5e7eb'
  const top = ['$1M-$5M', '$5M+', '$500K-$1M']
  return top.some(t => amount.includes(t)) ? '#fbbf24' : '#e5e7eb'
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>Congress Trading</h3>
      <div class="header-controls">
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="store.isBridgeAvailable === false" class="mock-banner">
      Mock data — Python sidecar not connected
    </div>

    <!-- Filters -->
    <div class="filter-bar">
      <div class="filter-group">
        <span class="filter-label">党派</span>
        <div class="filter-buttons">
          <button
            v-for="p in parties" :key="p"
            :class="['filter-btn', { active: partyFilter === p }]"
            @click="set党派Filter(p)"
          >{{ p }}</button>
        </div>
      </div>
      <div class="filter-group">
        <span class="filter-label">议院</span>
        <div class="filter-buttons">
          <button
            v-for="c in chambers" :key="c"
            :class="['filter-btn', { active: chamberFilter === c }]"
            @click="set议院Filter(c)"
          >{{ c }}</button>
        </div>
      </div>
    </div>

    <!-- Summary -->
    <div class="summary-bar" v-if="store.congressTrades">
      <span>Showing {{ filteredTrades.length }} / {{ store.congressTrades.length }} trades</span>
      <span>{{ totalBuyVolume }}</span>
    </div>

    <!-- Trades Table -->
    <div v-if="store.congressTrades" class="panel-content">
      <table v-if="filteredTrades.length > 0" class="congress-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>议院</th>
            <th>党派</th>
            <th>Symbol</th>
            <th>Type</th>
            <th>Amount</th>
            <th>Date</th>
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
      <p v-else class="no-data">无匹配筛选条件的交易</p>
    </div>

    <div v-else class="empty-state">
      <p>加载国会议员交易...</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg, #111827); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.refresh-btn { padding: 4px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: 4px; background: #78350f; color: #fbbf24; font-size: 12px; text-align: center; }
.filter-bar { display: flex; gap: 16px; margin-bottom: 10px; flex-wrap: wrap; }
.filter-group { display: flex; align-items: center; gap: 6px; }
.filter-label { font-size: 11px; color: #9ca3af; font-weight: 500; text-transform: uppercase; }
.filter-buttons { display: flex; gap: 2px; }
.filter-btn { padding: 3px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #9ca3af; cursor: pointer; font-size: 11px; }
.filter-btn.active { background: #3b82f6; color: #fff; border-color: #3b82f6; }
.summary-bar { display: flex; justify-content: space-between; padding: 6px 10px; margin-bottom: 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; font-size: 11px; color: #9ca3af; }
.panel-content { flex: 1; overflow-y: auto; }
.congress-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.congress-table th { text-align: left; padding: 6px 8px; color: #9ca3af; border-bottom: 1px solid #374151; font-weight: 500; white-space: nowrap; }
.congress-table td { padding: 6px 8px; border-bottom: 1px solid #1f2937; }
.name-cell { font-weight: 500; }
.symbol-cell { font-weight: 600; color: #60a5fa; }
.party-badge { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
.party-badge.democrat { background: #1e3a5f; color: #93c5fd; }
.party-badge.republican { background: #5f1e1e; color: #fca5a5; }
.party-badge.independent { background: #374151; color: #d1d5db; }
.type-badge { padding: 2px 10px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.type-badge.buy { background: #14532d; color: #22c55e; }
.type-badge.sell { background: #7f1d1d; color: #ef4444; }
.amount-cell { font-variant-numeric: tabular-nums; font-weight: 500; }
.date-cell { color: #9ca3af; }
.no-data { color: #6b7280; font-size: 13px; text-align: center; padding: 20px; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; font-size: 13px; }
</style>
