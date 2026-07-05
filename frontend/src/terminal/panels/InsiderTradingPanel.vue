<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useResearchStore } from '@/stores/research'
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

const trades = computed(() => store.research?.insider ?? [])

const netActivity = computed(() => {
  const list = trades.value
  if (!list || list.length === 0) return { label: 'Neutral', color: 'var(--color-text-tertiary)' }
  let buys = 0, sells = 0
  for (const t of list) {
    const shares = t.shares ?? 0
    if ((t.type ?? '').toLowerCase() === 'buy') buys += shares
    else sells += shares
  }
  if (buys > sells) return { label: 'Bullish', color: '#22c55e' }
  if (sells > buys) return { label: 'Bearish', color: '#ef4444' }
  return { label: 'Neutral', color: 'var(--color-text-tertiary)' }
})

watch(symbol, async (newVal) => {
  loadError.value = ''
  if (newVal) {
    try {
      await store.fetchStockResearch(newVal, ['insider'])
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
    await store.fetchStockResearch(symbol.value, ['insider'])
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  }
}

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
      <h3>{{ $t('research.insider') }} &mdash; {{ symbol.toUpperCase() }} {{ name }}</h3>
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
    <div v-else-if="trades.length > 0" class="panel-content">
      <!-- 净交易 Indicator -->
      <div class="activity-bar">
        <span class="activity-label">{{ $t('research.insider_net') }}</span>
        <span class="activity-indicator" :style="{ color: netActivity.color }">
          <span class="activity-dot" :style="{ background: netActivity.color }"></span>
          {{ netActivity.label }}
        </span>
      </div>

      <!-- Trades Table -->
      <table class="insider-table">
        <thead>
          <tr>
            <th>{{ $t('research.insider_name') }}</th>
            <th>{{ $t('research.insider_position') }}</th>
            <th>{{ $t('common.type') }}</th>
            <th>{{ $t('research.insider_shares') }}</th>
            <th>{{ $t('common.price') }}</th>
            <th>{{ $t('common.amount') }}</th>
            <th>{{ $t('common.date') }}</th>
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
      <p>输入代码后按 ↵ 查看内部交易</p>
    </div>
  </div>
</template>

<style scoped>
.panel { padding: 16px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg, var(--color-bg-panel)); }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.mock-banner { padding: 6px 10px; margin-bottom: 12px; border-radius: var(--radius-sm); background: var(--color-accent-soft); color: var(--color-accent); font-size: 12px; text-align: center; }
.panel-content { flex: 1; overflow-y: auto; }
.activity-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; padding: 10px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-md); background: var(--color-bg-elevated); font-size: 13px; }
.activity-label { color: var(--color-text-secondary); font-weight: 500; }
.activity-indicator { display: flex; align-items: center; gap: 6px; font-weight: 700; }
.activity-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.insider-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.insider-table th { text-align: left; padding: 6px 8px; color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.insider-table td { padding: 6px 8px; border-bottom: 1px solid var(--color-bg-elevated); }
.role-cell { color: var(--color-text-secondary); }
.type-badge { padding: 2px 10px; border-radius: var(--radius-lg); font-size: 11px; font-weight: 600; }
.type-badge.buy { background: var(--color-down); color: var(--color-down); }
.type-badge.sell { background: var(--color-up); color: var(--color-up); }
.num-cell { text-align: right; font-variant-numeric: tabular-nums; }
.date-cell { color: var(--color-text-secondary); }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }
.chart-fallback { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); }
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: rgba(239,68,68,0.1); color: #ef4444; font-size: 12px; }
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
