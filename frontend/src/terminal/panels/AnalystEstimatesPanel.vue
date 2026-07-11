<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const loadError = ref('')

const estimates = computed(() => store.research?.estimates ?? [])

const consensus = computed(() => {
  const list = estimates.value
  if (!list || list.length === 0) return null
  let buy = 0, hold = 0, sell = 0
  for (const e of list) {
    const r = (e.rating ?? '').toLowerCase()
    if (r.includes('buy') || r.includes('overweight') || r.includes('outperform')) buy++
    else if (r.includes('sell') || r.includes('underweight') || r.includes('underperform')) sell++
    else hold++
  }
  const total = buy + hold + sell
  const label = buy > sell && buy > hold ? '买入' : sell > buy && sell > hold ? '卖出' : '持有'
  return { buy, hold, sell, total, label }
})

const consensusColor = computed(() => {
  const c = consensus.value
  if (!c) return 'var(--color-text-tertiary)'
  if (c.label === '买入') return '#22c55e'
  if (c.label === '卖出') return '#ef4444'
  return '#eab308'
})

watch(symbol, async (newVal) => {
  if (newVal) {
    loadError.value = ''
    try {
      await store.fetchStockResearch(newVal, ['estimates'])
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

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

async function refresh() {
  loadError.value = ''
  try {
    await store.fetchStockResearch(symbol.value, ['estimates'])
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  }
}

function ratingColor(rating: string): string {
  const r = (rating ?? '').toLowerCase()
  if (r.includes('buy') || r.includes('overweight') || r.includes('outperform')) return '#22c55e'
  if (r.includes('sell') || r.includes('underweight') || r.includes('underperform')) return '#ef4444'
  return '#eab308'
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
      <h3>{{ $t('research.analyst') }} &mdash; {{ symbol.toUpperCase() }} {{ name }}</h3>
      <div class="header-controls">
        <button v-if="addToWfControl" class="wf-btn" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <input
          class="symbol-input"
          :value="symbol"
          :placeholder="$t('research.hint_enter_symbol')"
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-if="store.loading" class="chart-fallback">{{ $t('common.loading') }}</div>
    <div v-else-if="estimates.length > 0" class="panel-content">
      <!-- Consensus Badge -->
      <div class="consensus-bar" v-if="consensus">
        <div class="consensus-badge" :style="{ background: consensusColor, color: 'var(--color-bg-panel)' }">
          {{ consensus.label }} &mdash; {{ ((consensus.buy / consensus.total) * 100).toFixed(0) }}% 买方占比
        </div>
        <div class="consensus-breakdown">
          <span class="badge badge-buy">{{ consensus.buy }} 买入</span>
          <span class="badge badge-hold">{{ consensus.hold }} 持有</span>
          <span class="badge badge-sell">{{ consensus.sell }} 卖出</span>
        </div>
      </div>

      <!-- Estimates Table -->
      <table class="estimates-table">
        <thead>
          <tr>
            <th>分析师</th>
            <th>{{ $t('research.institution') }}</th>
            <th>{{ $t('research.analyst_ratings') }}</th>
            <th>{{ $t('research.target_low') }}</th>
            <th>{{ $t('research.target_high') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(e, i) in estimates" :key="e.analyst ?? i">
            <td>{{ e.analyst }}</td>
            <td class="firm-cell">{{ e.firm }}</td>
            <td>
              <span class="rating-pill" :style="{ background: ratingColor(e.rating), color: 'var(--color-bg-panel)' }">{{ e.rating }}</span>
            </td>
            <td class="num-cell">{{ e.target_low ?? '--' }}</td>
            <td class="num-cell">{{ e.target_high ?? '--' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="!store.loading && !loadError" class="empty-state">
      <p>输入代码后按 ↵ 查看分析师预测</p>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 12px; margin-bottom: 10px; color: var(--color-up); background: var(--color-up-soft); border: 1px solid var(--color-up-glow); border-radius: var(--radius-md); font-size: 12px; }
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg, var(--color-bg-panel)); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: var(--radius-sm); background: var(--color-accent-soft); color: var(--color-accent); font-size: 12px; text-align: center; }
.panel-content { flex: 1; overflow-y: auto; }
.consensus-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; padding: 10px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-md); background: var(--color-bg-elevated); }
.consensus-badge { padding: 4px 14px; border-radius: var(--radius-sm); font-size: 13px; font-weight: 700; white-space: nowrap; }
.consensus-breakdown { display: flex; gap: 8px; }
.badge { padding: 2px 10px; border-radius: var(--radius-lg); font-size: 11px; font-weight: 600; }
.badge-buy { background: var(--color-down); color: var(--color-down); }
.badge-hold { background: var(--color-accent); color: var(--color-accent); }
.badge-sell { background: var(--color-up); color: var(--color-up); }
.estimates-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.estimates-table th { text-align: left; padding: 6px 8px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.estimates-table td { padding: 6px 8px; border-bottom: 1px solid var(--color-bg-elevated); }
.firm-cell { color: var(--color-text-secondary); }
.rating-pill { padding: 2px 10px; border-radius: var(--radius-lg); font-size: 11px; font-weight: 600; }
.num-cell { font-variant-numeric: tabular-nums; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
.wf-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  line-height: 1;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.wf-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(88, 166, 255, 0.1);
}
</style>
