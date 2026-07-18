<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const { fetchWithCache } = usePanelCache()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const loading = ref(false)
const error = ref('')
const data = ref<any>(null)
const searchQuery = ref('')

const SOURCE = 'akshare'
const DATA_TYPE = 'futures'

const columns = ['代码', '名称', '最新价', '涨跌幅', '涨跌额', '今开', '最高', '最低', '昨结', '成交量', '持仓量']
const keys = ['代码', '名称', '最新价', '涨跌幅', '涨跌额', '今开', '最高', '最低', '昨结', '成交量', '持仓量']
const numericKeys = new Set(['最新价', '涨跌幅', '涨跌额', '今开', '最高', '最低', '昨结', '成交量', '持仓量'])

const filteredData = computed(() => {
  const rows = data.value?.data ?? []
  if (!searchQuery.value) return rows
  const q = searchQuery.value.toLowerCase()
  return rows.filter((r: any) =>
    (String(r['代码'] || '')).toLowerCase().includes(q) ||
    (String(r['名称'] || '')).toLowerCase().includes(q)
  )
})

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache<any>('futures_data', () => w.go.main.App.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 15 * 60 * 1000)
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

function pctColor(v: any): string {
  const n = parseFloat(v)
  if (n > 0) return '#ef4444'
  if (n < 0) return '#22c55e'
  return 'inherit'
}
function fmt(v: any, key: string): string {
  if (v == null || v === '') return '-'
  if (key === '涨跌幅') {
    const n = parseFloat(v)
    return (n >= 0 ? '+' : '') + n.toFixed(2) + '%'
  }
  if (key === '成交量' || key === '持仓量') {
    const n = typeof v === 'number' ? v : parseFloat(v)
    if (n >= 1e8) return (n / 1e8).toFixed(2) + '亿'
    if (n >= 1e4) return (n / 1e4).toFixed(1) + '万'
    return n.toLocaleString()
  }
  if (key === '最新价' || key === '涨跌额' || key === '今开' || key === '最高' || key === '最低' || key === '昨结') {
    const n = parseFloat(v)
    return n.toFixed(2)
  }
  return String(v)
}

onMounted(loadData)
</script>

<template>
  <div class="panel-container">
    <div class="panel-header">
      <span class="title">全球期货实时行情</span>
      <div class="header-actions">
        <input class="search-input" v-model="searchQuery" placeholder="搜索代码/名称" />
        <button class="btn-sm" @click="loadData">⟳ 刷新</button>
      </div>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="status">加载中...</div>
      <div v-else-if="error" class="status error">{{ error }}</div>
      <div v-else-if="!data || !data.success" class="status">{{ data?.error || '暂无数据' }}</div>
      <template v-else>
        <div class="info-row">共 {{ filteredData.length }} 个合约</div>
        <div class="table-wrap">
          <table class="futures-table">
            <thead>
              <tr>
                <th v-for="col in columns" :key="col">{{ col }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, idx) in filteredData" :key="row['代码'] || idx">
                <td v-for="key in keys" :key="key"
                  :class="['td', { 'td-code': key === '代码', 'td-right': numericKeys.has(key) }]"
                  :style="key === '涨跌幅' ? { color: pctColor(row[key]) } : {}"
                >{{ fmt(row[key], key) }}</td>
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

.title{font-weight:500}
.header-actions{display:flex;gap:8px;align-items:center}
.search-input{width:130px;padding:2px 6px;border:1px solid var(--color-border-subtle);border-radius: var(--radius-sm);background:var(--color-bg-elevated);color:var(--color-text-primary);font-size:12px}
.btn-sm{padding:2px 8px;font-size:11px;border:1px solid var(--color-border);border-radius: var(--radius-sm);background:transparent;color:var(--color-text-secondary);cursor:pointer}
.btn-sm:hover{background:var(--color-bg-hover)}
.panel-body{flex:1;overflow:auto;padding:0 12px 12px}
.status{display:flex;align-items:center;justify-content:center;height:100%;color:var(--color-text-tertiary);font-size:13px}
.status.error{color:var(--color-danger)}
.info-row{font-size:12px;color:var(--color-text-tertiary);padding:8px 0 4px}
.table-wrap{overflow-x:auto}
.futures-table{width:100%;border-collapse:collapse;font-size:11px;font-variant-numeric:tabular-nums}
.futures-table th{text-align:right;padding:4px 6px;color:var(--color-text-tertiary);font-weight:500;border-bottom:1px solid var(--color-border-subtle);white-space:nowrap;position:sticky;top:0;background:var(--color-bg-panel)}
.futures-table th:first-child{text-align:left}
.futures-table td{text-align:right;padding:3px 6px;border-bottom:1px solid var(--color-border-subtle);white-space:nowrap}
.futures-table tr:hover td{background:var(--color-bg-hover)}
.td-code{text-align:left!important;color:var(--color-text-secondary);font-family:monospace}
.td-right{text-align:right}
</style>
