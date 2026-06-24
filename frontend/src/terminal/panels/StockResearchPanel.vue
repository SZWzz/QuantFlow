<script setup lang="ts">
import { ref, watch } from 'vue'
import { useResearchStore } from '@/stores/research'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const activeTab = ref('overview')

const tabs = [
  { id: 'overview', label: '概览' },
  { id: 'financials', label: '财务' },
  { id: 'sentiment', label: '情绪' },
  { id: 'peers', label: '同业' },
  { id: 'estimates', label: '预测' },
  { id: 'insider', label: '内部交易' },
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
      <h3>个股研究</h3>
      <div class="header-controls">
        <input
          class="symbol-input"
          v-model="symbol"
          placeholder="代码..."
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
      >{{ tab.label }}</button>
    </div>

    <div class="tab-content">
      <!-- 概览 -->
      <div v-if="activeTab === 'overview'" class="tab-pane">
        <div v-if="store.research?.overview" class="kv-grid">
          <div v-for="(v, k) in store.research.overview" :key="k" class="kv-row">
            <span class="kv-key">{{ k }}</span>
            <span class="kv-value">{{ v }}</span>
          </div>
        </div>
        <p v-else class="no-data">暂无概览数据</p>
      </div>

      <!-- 财务 -->
      <div v-if="activeTab === 'financials'" class="tab-pane">
        <div v-if="store.research?.financials" class="kv-grid">
          <div v-for="(v, k) in store.research.financials.data || {}" :key="k" class="kv-row">
            <span class="kv-key">{{ k }}</span>
            <span class="kv-value">{{ typeof v === 'number' ? (v as number).toLocaleString() : v }}</span>
          </div>
        </div>
        <p v-else class="no-data">暂无财务数据</p>
      </div>

      <!-- 情绪 -->
      <div v-if="activeTab === 'sentiment'" class="tab-pane">
        <div v-if="store.research?.sentiment" class="kv-grid">
          <div class="kv-row"><span class="kv-key">得分</span><span class="kv-value">{{ store.research.sentiment.score }}</span></div>
          <div class="kv-row"><span class="kv-key">标签</span><span class="kv-value">{{ store.research.sentiment.label }}</span></div>
          <div class="kv-row"><span class="kv-key">置信度</span><span class="kv-value">{{ store.research.sentiment.confidence }}</span></div>
        </div>
        <p v-else class="no-data">暂无情绪数据</p>
      </div>

      <!-- 同业 -->
      <div v-if="activeTab === 'peers'" class="tab-pane">
        <table v-if="store.research?.peers?.length" class="data-table">
          <thead><tr><th>Symbol</th><th>市值</th><th>P/E</th><th>ROE</th></tr></thead>
          <tbody>
            <tr v-for="p in store.research.peers" :key="p.symbol">
              <td>{{ p.symbol }}</td><td>{{ p.market_cap?.toLocaleString() }}</td>
              <td>{{ p.pe_ratio }}</td><td>{{ p.roe }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">暂无同业数据</p>
      </div>

      <!-- 预测 -->
      <div v-if="activeTab === 'estimates'" class="tab-pane">
        <table v-if="store.research?.estimates?.length" class="data-table">
          <thead><tr><th>分析师</th><th>机构</th><th>评级</th><th>目标价</th></tr></thead>
          <tbody>
            <tr v-for="e in store.research.estimates" :key="e.analyst">
              <td>{{ e.analyst }}</td><td>{{ e.firm }}</td>
              <td>{{ e.rating }}</td><td>{{ e.target_low }}-{{ e.target_high }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">暂无分析师预测</p>
      </div>

      <!-- 内部交易 -->
      <div v-if="activeTab === 'insider'" class="tab-pane">
        <table v-if="store.research?.insider?.length" class="data-table">
          <thead><tr><th>姓名</th><th>职位</th><th>类型</th><th>股数</th><th>日期</th></tr></thead>
          <tbody>
            <tr v-for="t in store.research.insider" :key="t.name">
              <td>{{ t.name }}</td><td>{{ t.role }}</td>
              <td :class="{ buy: t.type === 'buy', sell: t.type === 'sell' }">{{ t.type }}</td>
              <td>{{ t.shares?.toLocaleString() }}</td><td>{{ t.date }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="no-data">暂无内部交易</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.research-panel {
  padding: 16px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, #e5e7eb); background: var(--color-bg, #111827);
}
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-controls { display: flex; gap: 8px; }
.symbol-input { width: 100px; padding: 4px 8px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; font-size: 13px; }
.refresh-btn { padding: 4px 10px; border: 1px solid #374151; border-radius: 4px; background: #1f2937; color: #e5e7eb; cursor: pointer; font-size: 13px; }
.tab-bar { display: flex; gap: 2px; margin-bottom: 12px; border-bottom: 1px solid #374151; overflow-x: auto; }
.tab-btn { padding: 6px 14px; border: none; background: none; color: #9ca3af; cursor: pointer; font-size: 12px; border-bottom: 2px solid transparent; white-space: nowrap; }
.tab-btn.active { color: #e5e7eb; border-bottom-color: #3b82f6; }
.tab-content { flex: 1; overflow-y: auto; }
.tab-pane { padding: 8px 0; }
.kv-grid { display: flex; flex-direction: column; gap: 6px; }
.kv-row { display: flex; justify-content: space-between; padding: 4px 0; border-bottom: 1px solid #1f2937; }
.kv-key { color: #9ca3af; font-size: 12px; text-transform: capitalize; }
.kv-value { font-size: 13px; font-variant-numeric: tabular-nums; }
.data-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.data-table th { text-align: left; padding: 4px 8px; color: #9ca3af; border-bottom: 1px solid #374151; }
.data-table td { padding: 4px 8px; border-bottom: 1px solid #1f2937; }
.buy { color: #22c55e; } .sell { color: #ef4444; }
.no-data { color: #6b7280; font-size: 13px; text-align: center; padding: 20px; }
</style>
