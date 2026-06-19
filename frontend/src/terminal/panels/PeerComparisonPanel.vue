<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useResearchStore } from '@/stores/research'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

const peers = computed(() => store.research?.peers ?? [])

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal, ['peers'])
}, { immediate: true })

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

function refresh() { store.fetchStockResearch(symbol.value, ['peers']) }

function formatMarketCap(v: number | undefined | null): string {
  if (v == null) return '--'
  if (v >= 1e12) return (v / 1e12).toFixed(2) + 'T'
  if (v >= 1e9) return (v / 1e9).toFixed(2) + 'B'
  if (v >= 1e6) return (v / 1e6).toFixed(2) + 'M'
  return v.toLocaleString()
}

function formatPct(v: number | undefined | null): string {
  if (v == null) return '--'
  return (v * 100).toFixed(1) + '%'
}

function formatRatio(v: number | undefined | null): string {
  if (v == null) return '--'
  return v.toFixed(2)
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>Peer Comparison — {{ symbol.toUpperCase() }}</h3>
      <div class="header-controls">
        <input class="symbol-input" v-model="symbol" placeholder="Symbol..." @keyup.enter="refresh" />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="store.isBridgeAvailable === false" class="mock-banner">
      Mock data — Python sidecar not connected
    </div>

    <div v-if="peers.length > 0" class="panel-content">
      <table class="peer-table">
        <thead>
          <tr>
            <th>Symbol</th>
            <th>Market Cap</th>
            <th>P/E</th>
            <th>Rev Growth</th>
            <th>Margin</th>
            <th>ROE</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in peers" :key="p.symbol ?? Math.random()">
            <td class="symbol-cell">{{ p.symbol }}</td>
            <td class="num-cell">{{ formatMarketCap(p.market_cap) }}</td>
            <td class="num-cell">{{ formatRatio(p.pe_ratio) }}</td>
            <td class="num-cell" :class="{ positive: p.revenue_growth > 0, negative: p.revenue_growth < 0 }">{{ formatPct(p.revenue_growth) }}</td>
            <td class="num-cell" :class="{ positive: p.margin > 0, negative: p.margin < 0 }">{{ formatPct(p.margin) }}</td>
            <td class="num-cell" :class="{ positive: p.roe > 0, negative: p.roe < 0 }">{{ formatPct(p.roe) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="empty-state">
      <p>Enter a symbol and press ↵ to view peer comparison</p>
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
.peer-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.peer-table th { text-align: left; padding: 6px 8px; color: #9ca3af; border-bottom: 1px solid #374151; font-weight: 500; white-space: nowrap; }
.peer-table td { padding: 6px 8px; border-bottom: 1px solid #1f2937; }
.symbol-cell { font-weight: 600; color: #60a5fa; }
.num-cell { text-align: right; font-variant-numeric: tabular-nums; }
.num-cell.positive { color: #22c55e; }
.num-cell.negative { color: #ef4444; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; font-size: 13px; }
</style>
