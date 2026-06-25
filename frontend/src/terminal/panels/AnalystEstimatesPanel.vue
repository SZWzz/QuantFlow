<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')

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
  if (!c) return '#6b7280'
  if (c.label === '买入') return '#22c55e'
  if (c.label === '卖出') return '#ef4444'
  return '#eab308'
})

watch(symbol, (newVal) => {
  if (newVal) store.fetchStockResearch(newVal, ['estimates'])
}, { immediate: true })

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
  }
})

function refresh() { store.fetchStockResearch(symbol.value, ['estimates']) }

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
      <h3>分析师 Estimates — {{ symbol.toUpperCase() }}</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          :value="symbol"
          placeholder="代码..."
          @keyup.enter="handleSymbolSubmit"
        />
        <button class="refresh-btn" @click="refresh" :disabled="store.loading">{{ store.loading ? '...' : '⟳' }}</button>
      </div>
    </div>

    <div v-if="estimates.length > 0" class="panel-content">
      <!-- Consensus Badge -->
      <div class="consensus-bar" v-if="consensus">
        <div class="consensus-badge" :style="{ background: consensusColor, color: '#111827' }">
          {{ consensus.label }} — {{ ((consensus.buy / consensus.total) * 100).toFixed(0) }}% 买方占比
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
            <th>机构</th>
            <th>评级</th>
            <th>目标价(低)</th>
            <th>目标价(高)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(e, i) in estimates" :key="e.analyst ?? i">
            <td>{{ e.analyst }}</td>
            <td class="firm-cell">{{ e.firm }}</td>
            <td>
              <span class="rating-pill" :style="{ background: ratingColor(e.rating), color: '#111827' }">{{ e.rating }}</span>
            </td>
            <td class="num-cell">{{ e.target_low ?? '--' }}</td>
            <td class="num-cell">{{ e.target_high ?? '--' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="empty-state">
      <p>输入代码后按 ↵ 查看分析师预测</p>
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
.consensus-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; padding: 10px 12px; border: 1px solid #374151; border-radius: 6px; background: #1f2937; }
.consensus-badge { padding: 4px 14px; border-radius: 4px; font-size: 13px; font-weight: 700; white-space: nowrap; }
.consensus-breakdown { display: flex; gap: 8px; }
.badge { padding: 2px 10px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.badge-buy { background: #14532d; color: #22c55e; }
.badge-hold { background: #713f12; color: #eab308; }
.badge-sell { background: #7f1d1d; color: #ef4444; }
.estimates-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.estimates-table th { text-align: left; padding: 6px 8px; color: #9ca3af; border-bottom: 1px solid #374151; font-weight: 500; white-space: nowrap; }
.estimates-table td { padding: 6px 8px; border-bottom: 1px solid #1f2937; }
.firm-cell { color: #9ca3af; }
.rating-pill { padding: 2px 10px; border-radius: 10px; font-size: 11px; font-weight: 600; }
.num-cell { font-variant-numeric: tabular-nums; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: #6b7280; font-size: 13px; }
</style>
