<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useTerminalStore } from '@/stores/terminal'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const terminal = useTerminalStore()
const symbol = ref(props.params?.symbol || terminal.activeSymbol || 'AAPL')

// Subscribe to symbol context
watch(() => terminal.activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

const financials = computed(() => store.research?.financials)

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal, ['financials'])
}, { immediate: true })

function refresh() { store.fetchStockResearch(symbol.value, ['financials']) }

function formatNum(v: number | undefined | null): string {
  if (v == null) return '--'
  if (Math.abs(v) >= 1e12) return (v / 1e12).toFixed(2) + 'T'
  if (Math.abs(v) >= 1e9) return (v / 1e9).toFixed(2) + 'B'
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(2) + 'M'
  if (Math.abs(v) >= 1e3) return (v / 1e3).toFixed(2) + 'K'
  return v.toLocaleString(undefined, { maximumFractionDigits: 2 })
}

function formatPct(v: number | undefined | null): string {
  if (v == null) return '--'
  return (v * 100).toFixed(2) + '%'
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>Financials — {{ symbol.toUpperCase() }}</h3>
      <div class="header-controls">
        <input class="symbol-input" v-model="symbol" placeholder="Symbol..." @keyup.enter="refresh" />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="store.isBridgeAvailable === false" class="mock-banner">
      Mock data — Python sidecar not connected
    </div>

    <div v-if="financials" class="panel-content">
      <div class="card-grid">
        <!-- Income Statement -->
        <div class="card">
          <h4 class="card-title">Income Statement</h4>
          <div class="card-row"><span>Revenue</span><span class="val">{{ formatNum(financials.data.revenue) }}</span></div>
          <div class="card-row"><span>Net Income</span><span class="val">{{ formatNum(financials.data.net_income) }}</span></div>
          <div class="card-row"><span>EPS</span><span class="val">{{ financials.data.eps?.toFixed(2) ?? '--' }}</span></div>
        </div>

        <!-- Balance Sheet -->
        <div class="card">
          <h4 class="card-title">Balance Sheet</h4>
          <div class="card-row"><span>Total Assets</span><span class="val">{{ formatNum(financials.data.total_assets) }}</span></div>
          <div class="card-row"><span>Total Equity</span><span class="val">{{ formatNum(financials.data.total_equity) }}</span></div>
          <div class="card-row"><span>Total Debt</span><span class="val">{{ formatNum(financials.data.total_debt) }}</span></div>
        </div>

        <!-- Cash Flow -->
        <div class="card">
          <h4 class="card-title">Cash Flow</h4>
          <div class="card-row"><span>Free Cash Flow</span><span class="val">{{ formatNum(financials.data.free_cash_flow) }}</span></div>
          <div class="card-row"><span>Market Cap</span><span class="val">{{ formatNum(financials.data.market_cap) }}</span></div>
        </div>

        <!-- Ratios -->
        <div class="card" v-if="financials.ratios && Object.keys(financials.ratios).length > 0">
          <h4 class="card-title">Ratios</h4>
          <div class="card-row" v-for="(v, k) in financials.ratios" :key="k">
            <span>{{ k.replace(/_/g, ' ') }}</span>
            <span class="val">{{ typeof v === 'number' ? (k.includes('margin') || k.includes('yield') || k.includes('rate') ? formatPct(v) : v.toFixed(2)) : v }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>Enter a symbol and press ↵ to view financials</p>
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
.card-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.card { padding: 12px; border: 1px solid #374151; border-radius: 6px; background: #1f2937; }
.card-title { margin: 0 0 8px 0; font-size: 12px; font-weight: 600; color: #9ca3af; text-transform: uppercase; letter-spacing: 0.5px; }
.card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 12px; border-bottom: 1px solid #1f2937; }
.card-row:last-child { border-bottom: none; }
.card-row span { color: #9ca3af; }
.card-row .val { color: #e5e7eb; font-variant-numeric: tabular-nums; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; font-size: 13px; }
</style>
