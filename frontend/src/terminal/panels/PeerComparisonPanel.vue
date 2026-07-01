<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useResearchStore } from '@/stores/research'
import { useStockName } from '@/lib/composables/useStockName'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const loadError = ref('')

const peers = computed(() => store.research?.peers ?? [])

watch(symbol, async (newVal) => {
  loadError.value = ''
  if (newVal) {
    try {
      await store.fetchStockResearch(newVal, ['peers'])
    } catch (e: any) {
      loadError.value = e?.message || String(e)
    }
  }
}, { immediate: true })

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

async function refresh() {
  loadError.value = ''
  try {
    await store.fetchStockResearch(symbol.value, ['peers'])
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  }
}

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
      <h3>{{ $t('research.peer') }} &mdash; {{ symbol.toUpperCase() }} {{ name }}</h3>
      <div class="header-controls">
        <input class="symbol-input" v-model="symbol" :placeholder="$t('research.hint_enter_symbol')" @keyup.enter="refresh" />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <div v-else-if="peers.length > 0" class="panel-content">
      <p class="peer-hint">{{ $t('research.peer_hint') }}</p>
      <table class="peer-table">
        <thead>
          <tr>
            <th>{{ $t('quote.symbol') }}</th>
            <th>{{ $t('common.name') }}</th>
            <th>{{ $t('quote.market_cap') }}</th>
            <th>{{ $t('research.pe_ratio') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in peers" :key="p.symbol ?? Math.random()">
            <td class="symbol-cell">{{ p.symbol }}</td>
            <td>{{ p.name }}</td>
            <td class="num-cell">{{ formatMarketCap(p.market_cap) }}</td>
            <td class="num-cell">{{ formatRatio(p.pe_ratio) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="empty-state">
      <p>输入代码后按 ↵ 查看同业对比</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg, var(--color-bg-panel)); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: 4px; background: var(--color-accent-soft); color: var(--color-accent); font-size: 12px; text-align: center; }
.peer-hint { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 8px; }
.panel-content { flex: 1; overflow-y: auto; }
.peer-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.peer-table th { text-align: left; padding: 6px 8px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.peer-table td { padding: 6px 8px; border-bottom: 1px solid var(--color-bg-elevated); }
.symbol-cell { font-weight: 600; color: var(--color-accent); }
.num-cell { text-align: right; font-variant-numeric: tabular-nums; }
.num-cell.positive { color: var(--color-down); }
.num-cell.negative { color: var(--color-up); }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: 4px; background: rgba(239,68,68,0.1); color: #ef4444; font-size: 12px; }
</style>
