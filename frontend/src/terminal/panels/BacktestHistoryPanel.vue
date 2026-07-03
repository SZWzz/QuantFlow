<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'

interface BacktestSummary {
  id: number
  run_id: string
  workflow_name: string
  strategy_name: string
  symbol: string
  engine_type: string
  total_return: number
  cagr: number
  max_drawdown: number
  sharpe_ratio: number
  sortino_ratio: number
  calmar_ratio: number
  win_rate: number
  profit_factor: number
  total_trades: number
  started_at: string
  finished_at: string
  created_at: string
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()

const items = ref<BacktestSummary[]>([])
const loading = ref(false)
const selectedIds = ref<Set<number>>(new Set())
const sortField = ref<'finished_at' | 'total_return' | 'sharpe_ratio'>('finished_at')
const sortAsc = ref(false)

async function loadData() {
  loading.value = true
  try {
    const res = await (window as any).go.main.App.ListBacktestHistory(100, 0)
    items.value = res || []
  } catch (e) {
    console.error('ListBacktestHistory failed:', e)
  } finally {
    loading.value = false
  }
}

function sortedItems(): BacktestSummary[] {
  const sorted = [...items.value]
  sorted.sort((a, b) => {
    let va: any, vb: any
    if (sortField.value === 'finished_at') {
      va = a.finished_at
      vb = b.finished_at
    } else if (sortField.value === 'total_return') {
      va = a.total_return
      vb = b.total_return
    } else {
      va = a.sharpe_ratio
      vb = b.sharpe_ratio
    }
    if (sortAsc.value) return va > vb ? 1 : -1
    return va < vb ? 1 : -1
  })
  return sorted
}

function toggleSort(field: typeof sortField.value) {
  if (sortField.value === field) sortAsc.value = !sortAsc.value
  else { sortField.value = field; sortAsc.value = false }
}

function toggleSelect(id: number) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

function toggleSelectAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  selectedIds.value = checked ? new Set(items.value.map(i => i.id)) : new Set()
}

function openDetail(id: number) {
  terminal.openPanel('backtest-result', { storeId: id })
}

async function deleteSelected() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  if (!confirm(`确定删除选中的 ${ids.length} 条回测记录？`)) return
  for (const id of ids) {
    try {
      await (window as any).go.main.App.DeleteBacktestResult(id)
    } catch (e) {
      console.error('DeleteBacktestResult failed:', e)
    }
  }
  selectedIds.value = new Set()
  await loadData()
}

async function deleteSingle(id: number) {
  if (!confirm('确定删除此回测记录？')) return
  try {
    await (window as any).go.main.App.DeleteBacktestResult(id)
    selectedIds.value = new Set()
    await loadData()
  } catch (e) {
    console.error('DeleteBacktestResult failed:', e)
  }
}

function fmt(v: number | undefined | null, decimals = 2): string {
  if (v == null || isNaN(v)) return '-'
  return v.toFixed(decimals)
}
function pct(v: number | undefined | null): string {
  if (v == null || isNaN(v)) return '-'
  return (v * 100).toFixed(2) + '%'
}

onMounted(loadData)
</script>

<template>
  <div class="backtest-history-panel">
    <div class="panel-toolbar">
      <span class="panel-title">回测历史 ({{ items.length }})</span>
      <div class="toolbar-actions">
        <button v-if="selectedIds.size > 0" class="btn btn-danger btn-sm" @click="deleteSelected">
          删除选中 ({{ selectedIds.size }})
        </button>
        <button class="btn btn-sm" @click="loadData" :disabled="loading">
          {{ loading ? '加载中...' : '刷新' }}
        </button>
      </div>
    </div>

    <div v-if="loading && items.length === 0" class="loading">加载中...</div>
    <div v-else-if="items.length === 0" class="empty">暂无回测记录</div>
    <table v-else class="history-table">
      <thead>
        <tr>
          <th class="col-check"><input type="checkbox" @change="toggleSelectAll" /></th>
          <th class="col-date sortable" @click="toggleSort('finished_at')">
            日期 {{ sortField === 'finished_at' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-wf">工作流</th>
          <th class="col-strategy">策略</th>
          <th class="col-symbol">标的</th>
          <th class="col-return sortable" @click="toggleSort('total_return')">
            收益率 {{ sortField === 'total_return' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-sharpe sortable" @click="toggleSort('sharpe_ratio')">
            Sharpe {{ sortField === 'sharpe_ratio' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-trades">交易</th>
          <th class="col-actions">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in sortedItems()" :key="item.id"
            :class="{ selected: selectedIds.has(item.id) }"
            @click="openDetail(item.id)">
          <td class="col-check" @click.stop>
            <input type="checkbox" :checked="selectedIds.has(item.id)" @change="toggleSelect(item.id)" />
          </td>
          <td>{{ item.finished_at?.slice(0, 10) }}</td>
          <td>{{ item.workflow_name }}</td>
          <td>{{ item.strategy_name }}</td>
          <td>{{ item.symbol }}</td>
          <td :class="(item.total_return ?? 0) >= 0 ? 'positive' : 'negative'">{{ pct(item.total_return) }}</td>
          <td>{{ fmt(item.sharpe_ratio) }}</td>
          <td>{{ item.total_trades }}</td>
          <td class="col-actions" @click.stop>
            <button class="btn-icon" title="删除" @click="deleteSingle(item.id)">🗑</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.backtest-history-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 8px;
  overflow-y: auto;
}
.panel-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.toolbar-actions { display: flex; gap: 4px; }
.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.history-table th {
  text-align: left;
  padding: 6px 4px;
  border-bottom: 1px solid var(--border-color, #334);
  font-weight: 600;
  white-space: nowrap;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--accent, #3b82f6); }
.history-table td {
  padding: 4px;
  border-bottom: 1px solid var(--border-color, #2a2a3a);
  cursor: pointer;
}
.history-table tr:hover { background: var(--hover-bg, rgba(59,130,246,0.08)); }
.history-table tr.selected { background: var(--selected-bg, rgba(59,130,246,0.15)); }
.col-check { width: 24px; }
.col-date { width: 90px; }
.col-wf { min-width: 100px; }
.col-strategy { min-width: 80px; }
.col-symbol { width: 70px; }
.col-return { width: 80px; text-align: right; }
.col-sharpe { width: 70px; text-align: right; }
.col-trades { width: 50px; text-align: right; }
.col-actions { width: 40px; text-align: center; }
.positive { color: #ef4444; }
.negative { color: #22c55e; }
.loading, .empty { padding: 20px; text-align: center; color: #888; }
.btn-sm { padding: 2px 8px; font-size: 11px; cursor: pointer; }
.btn-danger { background: #dc2626; color: #fff; border: none; border-radius: 3px; }
.btn-icon { background: none; border: none; cursor: pointer; padding: 2px 4px; font-size: 13px; }
</style>
