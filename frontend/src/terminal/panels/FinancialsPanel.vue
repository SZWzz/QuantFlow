<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

const financials = computed(() => store.research?.financials)

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal, ['financials'])
}, { immediate: true })

function refresh() {
  ctx.setGroupSymbol(pg.groupId, symbol.value)
  store.fetchStockResearch(symbol.value, ['financials'])
}

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
      <h3>{{ $t('research.financials') }} &mdash; {{ symbol.toUpperCase() }}</h3>
      <div class="header-controls">
        <input class="symbol-input" v-model="symbol" :placeholder="$t('research.hint_enter_symbol')" @keyup.enter="refresh" />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <div v-else-if="financials" class="panel-content">
      <div class="card-grid">
        <!-- 利润表 -->
        <div class="card">
          <h4 class="card-title">{{ $t('research.income_stmt') }}</h4>
          <div class="card-row"><span>{{ $t('research.revenue') }}</span><span class="val">{{ formatNum(financials.data.revenue) }}</span></div>
          <div class="card-row"><span>{{ $t('research.net_profit') }}</span><span class="val">{{ formatNum(financials.data.net_income) }}</span></div>
          <div class="card-row"><span>{{ $t('quote.eps') }}</span><span class="val">{{ financials.data.eps?.toFixed(2) ?? '--' }}</span></div>
        </div>

        <!-- 资产负债表 -->
        <div class="card">
          <h4 class="card-title">{{ $t('research.balance_sheet') }}</h4>
          <div class="card-row"><span>{{ $t('research.total_assets') }}</span><span class="val">{{ formatNum(financials.data.total_assets) }}</span></div>
          <div class="card-row"><span>{{ $t('research.total_equity') }}</span><span class="val">{{ formatNum(financials.data.total_equity) }}</span></div>
          <div class="card-row"><span>{{ $t('research.total_liabilities') }}</span><span class="val">{{ formatNum(financials.data.total_debt) }}</span></div>
        </div>

        <!-- 现金流量表 -->
        <div class="card">
          <h4 class="card-title">{{ $t('research.cashflow_stmt') }}</h4>
          <div class="card-row"><span>{{ $t('research.free_cashflow') }}</span><span class="val">{{ formatNum(financials.data.free_cash_flow) }}</span></div>
          <div class="card-row"><span>{{ $t('quote.market_cap') }}</span><span class="val">{{ formatNum(financials.data.market_cap) }}</span></div>
        </div>

        <!-- 财务比率 -->
        <div class="card" v-if="financials.ratios && Object.keys(financials.ratios).length > 0">
          <h4 class="card-title">{{ $t('research.financial_ratios') }}</h4>
          <div class="card-row" v-for="(v, k) in financials.ratios" :key="k">
            <span>{{ $t('research.' + k) }}</span>
            <span class="val">{{ typeof v === 'number' ? (k.includes('margin') || k.includes('yield') || k.includes('rate') ? formatPct(v) : v.toFixed(2)) : v }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <p>输入代码后按 ↵ 查看财务数据</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text-primary); background: var(--color-bg-panel); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: 4px; background: #78350f; color: #fbbf24; font-size: 12px; text-align: center; }
.panel-content { flex: 1; overflow-y: auto; }
.card-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.card { padding: 12px; border: 1px solid var(--color-border-strong); border-radius: 6px; background: var(--color-bg-elevated); }
.card-title { margin: 0 0 8px 0; font-size: 12px; font-weight: 600; color: var(--color-text-secondary); text-transform: uppercase; letter-spacing: 0.5px; }
.card-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 12px; border-bottom: 1px solid var(--color-bg-elevated); }
.card-row:last-child { border-bottom: none; }
.card-row span { color: var(--color-text-secondary); }
.card-row .val { color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
</style>
