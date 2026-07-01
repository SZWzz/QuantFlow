<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const loadError = ref('')
const data = ref<any>(null)
const searchQuery = ref('')

const SOURCE = 'akshare'
const DATA_TYPE = 'bonds'

const columns = ['symbol', 'name', 'trade', 'changepercent', 'volume', 'amount', 'code', 'ticktime']
const colLabels: Record<string, string> = {
  symbol: '代码', name: '名称', trade: '最新价', changepercent: '涨跌幅',
  volume: '成交量', amount: '成交额', code: '正股代码', ticktime: '时间'
}
const numericCols = new Set(['trade', 'changepercent', 'volume', 'amount'])

const filteredData = computed(() => {
  const rows = data.value?.data ?? []
  if (!searchQuery.value) return rows
  const q = searchQuery.value.toLowerCase()
  return rows.filter((r: any) =>
    (r.symbol || '').toLowerCase().includes(q) ||
    (r.code || '').toLowerCase().includes(q) ||
    (r.name || '').toLowerCase().includes(q)
  )
})

async function loadData() {
  loading.value = true; loadError.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('bonds:' + symbol.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) loadError.value = result.error
    }
  } catch (e: any) { loadError.value = e.message || '加载失败' }
  finally { loading.value = false }
}

function fmt(v: any, col: string): string {
  if (v == null || v === '') return '-'
  if (col === 'changepercent') {
    const n = parseFloat(v)
    return (n >= 0 ? '+' : '') + n.toFixed(2) + '%'
  }
  if (col === 'volume') {
    const n = typeof v === 'number' ? v : parseFloat(v)
    if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
    if (n >= 1e4) return (n / 1e4).toFixed(1) + '万'
    return n.toLocaleString()
  }
  if (col === 'amount') {
    const n = typeof v === 'number' ? v : parseFloat(v)
    if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
    if (n >= 1e4) return (n / 1e4).toFixed(1) + '万'
    return n.toLocaleString()
  }
  return String(v)
}
function pctColor(v: any): string {
  const n = parseFloat(v)
  if (n > 0) return '#ef4444'
  if (n < 0) return '#22c55e'
  return 'inherit'
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
</script>

<template>
  <div class="panel-container">
    <div class="panel-header">
      <span class="title">可转债实时行情</span>
      <span class="symbol-badge">{{ symbol }} {{ name }}</span>
      <div class="header-actions">
        <input class="search-input" v-model="searchQuery" placeholder="搜索代码/名称" />
        <button class="btn-sm" @click="loadData">⟳ 刷新</button>
      </div>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="status">加载中...</div>
      <div v-else-if="loadError" class="status error">{{ loadError }}</div>
      <div v-else-if="!data || !data.success" class="status">{{ data?.error || '暂无数据' }}</div>

      <template v-else>
        <div class="info-row">共 {{ filteredData.length }} 只可转债</div>
        <div class="table-wrap">
          <table class="bond-table">
            <thead>
              <tr>
                <th v-for="col in columns" :key="col" class="th-{{ col }}">{{ colLabels[col] }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in filteredData" :key="row.symbol || row.code">
                <td v-for="col in columns" :key="col"
                  :class="['td', { 'td-code': col === 'symbol' || col === 'code', 'td-right': numericCols.has(col) }]"
                  :style="col === 'changepercent' ? { color: pctColor(row[col]) } : {}"
                >{{ fmt(row[col], col) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.panel-container{display:flex;flex-direction:column;height:100%;background:var(--color-bg-panel);color:var(--color-text-primary);font-size:13px}
.panel-header{display:flex;justify-content:space-between;align-items:center;padding:8px 12px;border-bottom:1px solid var(--color-border)}
.title{font-weight:500}
.header-actions{display:flex;gap:8px;align-items:center}
.search-input{width:130px;padding:2px 6px;border:1px solid var(--color-border-subtle);border-radius:4px;background:var(--color-bg-elevated);color:var(--color-text-primary);font-size:12px}
.btn-sm{padding:2px 8px;font-size:11px;border:1px solid var(--color-border);border-radius:4px;background:transparent;color:var(--color-text-secondary);cursor:pointer}
.btn-sm:hover{background:var(--color-bg-hover)}
.panel-body{flex:1;overflow:auto;padding:0 12px 12px}
.status{display:flex;align-items:center;justify-content:center;height:100%;color:var(--color-text-tertiary);font-size:13px}
.status.error{color:var(--color-error)}
.info-row{font-size:12px;color:var(--color-text-tertiary);padding:8px 0 4px}
.table-wrap{overflow-x:auto}
.bond-table{width:100%;border-collapse:collapse;font-size:11px;font-variant-numeric:tabular-nums}
.bond-table th{text-align:right;padding:4px 6px;color:var(--color-text-tertiary);font-weight:500;border-bottom:1px solid var(--color-border-subtle);white-space:nowrap;position:sticky;top:0;background:var(--color-bg-panel)}
.bond-table th:first-child{text-align:left}
.bond-table td{text-align:right;padding:3px 6px;border-bottom:1px solid var(--color-border-subtle)}
.bond-table tr:hover td{background:var(--color-bg-hover)}
.td-code{text-align:left!important;color:var(--color-text-secondary);font-family:monospace}
.td-right{text-align:right}
</style>
