<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'

const { t } = useI18n()

const KEY_MAP: Record<string, string> = {
  symbol: 'common.symbol', name: 'common.name',
  price: 'common.price',
  sector: 'research.sector', market_cap: 'research.market_cap',
  total_shares: 'research.total_shares', float_shares: 'research.float_shares',
  list_date: 'research.list_date',
  pe_ratio: 'research.pe_ratio', pb_ratio: 'research.pb_ratio',
  roe: 'research.roe', roa: 'research.roa', debt_to_equity: 'research.debt_to_equity',
  net_margin: 'research.net_margin', revenue: 'research.revenue',
  net_income: 'research.net_profit', eps: 'quote.eps',
  total_assets: 'research.total_assets', total_equity: 'research.total_equity',
  total_debt: 'research.total_debt', free_cash_flow: 'research.free_cashflow',
  margin: 'research.margin', revenue_growth: 'research.revenue_growth',
  sentiment_score: 'research.sentiment_score',
  sentiment_label: 'research.sentiment_label',
  sentiment_confidence: 'research.sentiment_confidence',
}
function label(key: string): string { return KEY_MAP[key] ? t(KEY_MAP[key]) : key }

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const activeTab = ref('overview')

const tabs = [
  { id: 'overview', key: 'research.overview' },
  { id: 'financials', key: 'research.financials' },
  { id: 'sentiment', key: 'research.sentiment' },
  { id: 'peers', key: 'research.peer' },
  { id: 'estimates', key: 'research.analyst' },
  { id: 'insider', key: 'research.insider' },
]

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal)
}, { immediate: true })

// Subscribe to symbol context via link group
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

function refresh() {
  ctx.setGroupSymbol(pg.groupId, symbol.value)
  store.fetchStockResearch(symbol.value)
}
</script>

<template>
  <div class="research-panel">
    <div class="panel-header">
      <h3>{{ $t('research.title') }} &mdash; {{ symbol }} {{ name }}</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          v-model="symbol"
          :placeholder="$t('research.hint_enter_symbol')"
          @keyup.enter="refresh"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">
          {{ store.loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div class="tab-bar">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['tab-btn', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id"
      >{{ $t(tab.key) }}</button>
    </div>

    <div class="tab-content">
      <div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
      <template v-else>
      <!-- 概览 -->
      <div v-if="activeTab === 'overview'" class="tab-pane">
        <div v-if="store.research?.overview" class="kv-grid">
          <div v-for="(v, k) in store.research.overview" :key="k" class="kv-row">
            <span class="kv-key">{{ label(k) }}</span>
            <span class="kv-value">{{ v }}</span>
          </div>
        </div>
        <p v-else class="no-data">{{ $t('research.no_overview') }}</p>
      </div>

      <!-- 财务 -->
      <div v-if="activeTab === 'financials'" class="tab-pane">
        <div v-if="store.research?.financials" class="kv-grid">
          <div v-for="(v, k) in store.research.financials.data || {}" :key="k" class="kv-row">
            <span class="kv-key">{{ label(k) }}</span>
            <span class="kv-value">{{ typeof v === 'number' ? (v as number).toLocaleString() : v }}</span>
          </div>
        </div>
        <p v-else class="no-data">{{ $t('research.no_financials') }}</p>
      </div>

      <!-- 情绪 -->
      <div v-if="activeTab === 'sentiment'" class="tab-pane">
        <div v-if="store.research?.sentiment" class="kv-grid">
          <div class="kv-row"><span class="kv-key">得分</span><span class="kv-value">{{ store.research.sentiment.score }}</span></div>
          <div class="kv-row"><span class="kv-key">标签</span><span class="kv-value">{{ store.research.sentiment.label }}</span></div>
          <div class="kv-row"><span class="kv-key">置信度</span><span class="kv-value">{{ store.research.sentiment.confidence }}</span></div>
        </div>
        <p v-else class="no-data">{{ $t('research.no_sentiment') }}</p>
      </div>

      <!-- 同业 -->
      <div v-if="activeTab === 'peers'" class="tab-pane">
        <table v-if="store.research?.peers?.length" class="data-table">
          <thead><tr><th>{{ $t('quote.symbol') }}</th><th>{{ $t('quote.market_cap') }}</th><th>{{ $t('research.pe_ratio') }}</th><th>{{ $t('research.roe') }}</th></tr></thead>
          <tbody>
            <tr v-for="p in store.research.peers" :key="p.symbol">
              <td>{{ p.symbol }}</td><td>{{ p.market_cap?.toLocaleString() }}</td>
              <td>{{ p.pe_ratio }}</td><td>{{ p.roe }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">{{ $t('research.no_peer') }}</p>
      </div>

      <!-- 预测 -->
      <div v-if="activeTab === 'estimates'" class="tab-pane">
        <table v-if="store.research?.estimates?.length" class="data-table">
          <thead><tr><th>分析师</th><th>{{ $t('research.institution') }}</th><th>{{ $t('research.analyst_ratings') }}</th><th>目标价</th></tr></thead>
          <tbody>
            <tr v-for="e in store.research.estimates" :key="e.analyst">
              <td>{{ e.analyst }}</td><td>{{ e.firm }}</td>
              <td>{{ e.rating }}</td><td>{{ e.target_low }}-{{ e.target_high }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">{{ $t('research.no_analyst') }}</p>
      </div>

      <!-- 内部交易 -->
      <div v-if="activeTab === 'insider'" class="tab-pane">
        <table v-if="store.research?.insider?.length" class="data-table">
          <thead><tr><th>{{ $t('research.insider_name') }}</th><th>{{ $t('research.insider_position') }}</th><th>{{ $t('common.type') }}</th><th>{{ $t('research.insider_shares') }}</th><th>{{ $t('common.date') }}</th></tr></thead>
          <tbody>
            <tr v-for="t in store.research.insider" :key="t.name">
              <td>{{ t.name }}</td><td>{{ t.role }}</td>
              <td :class="{ buy: t.type === 'buy', sell: t.type === 'sell' }">{{ t.type }}</td>
              <td>{{ t.shares?.toLocaleString() }}</td><td>{{ t.date }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">{{ $t('research.no_insider') }}</p>
      </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.research-panel {
  padding: 16px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text-primary); background: var(--color-bg-panel);
}
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.tab-bar { display: flex; gap: 2px; margin-bottom: 12px; border-bottom: 1px solid var(--color-border-strong); overflow-x: auto; }
.tab-btn { padding: 6px 14px; border: none; background: none; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; border-bottom: 2px solid transparent; white-space: nowrap; }
.tab-btn.active { color: var(--color-text-primary); border-bottom-color: var(--color-accent); }
.tab-content { flex: 1; overflow-y: auto; }
.tab-pane { padding: 8px 0; }
.kv-grid { display: flex; flex-direction: column; gap: 6px; }
.kv-row { display: flex; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid var(--color-bg-elevated); }
.kv-key { color: var(--color-text-secondary); font-size: 12px; text-transform: capitalize; }
.kv-value { color: var(--color-text-primary); font-size: 13px; font-variant-numeric: tabular-nums; }
.data-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.data-table th { text-align: left; padding: 4px 8px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); }
.data-table td { padding: 4px 8px; border-bottom: 1px solid var(--color-bg-elevated); }
.buy { color: var(--color-down); } .sell { color: var(--color-up); }
.no-data { color: var(--color-text-tertiary); font-size: 13px; text-align: center; padding: 20px; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
</style>
