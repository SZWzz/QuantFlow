<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const options = ref<any[]>([])

const SOURCE = 'akshare'
const DATA_TYPE = 'options'

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (app?.FetchData) {
      const { data: result } = await fetchWithCache<any>('options_data', () => app.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 5 * 60 * 1000)
      if (result?.data) {
        const parsed = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        if (parsed?.success === false) {
          error.value = parsed.error || '数据获取失败'
        } else {
          options.value = Array.isArray(parsed) ? parsed : (parsed?.data || [])
        }
      } else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

function fmt(v: any): string {
  if (v == null || v === '') return '-'
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (isNaN(n)) return String(v)
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  return n.toFixed(n % 1 === 0 ? 0 : 2)
}

onMounted(loadData)
</script>

<template>
  <div class="options-panel">
    <div class="panel-header">
      <span class="title">期权市场</span>
      <button class="btn-sm" @click="loadData">⟳ 刷新</button>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="state">加载中...</div>
      <div v-else-if="error" class="state error">{{ error }}</div>
      <div v-else-if="options.length === 0" class="state">暂无数据</div>

      <template v-else>
        <div class="table-wrap">
          <table class="opt-table">
            <thead>
              <tr>
                <th>合约代码</th>
                <th>合约简称</th>
                <th>标的</th>
                <th>类型</th>
                <th>行权价</th>
                <th>到期日</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in options" :key="row['合约编码'] || row['合约交易代码']">
                <td class="td-code">{{ row['合约交易代码'] || '-' }}</td>
                <td>{{ row['合约简称'] || '-' }}</td>
                <td>{{ row['标的券名称及代码'] || '-' }}</td>
                <td :class="row['类型'] === '认购' ? 'up' : row['类型'] === '认沽' ? 'down' : ''">{{ row['类型'] || '-' }}</td>
                <td>{{ row['行权价'] != null ? fmt(row['行权价']) : '-' }}</td>
                <td>{{ row['到期日'] || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.options-panel { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: 13px; }
.panel-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.title { font-weight: 600; font-size: 14px; }
.btn-sm { padding: 2px 8px; font-size: 11px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; }
.btn-sm:hover { background: var(--color-bg-hover); }
.panel-body { flex: 1; overflow: auto; padding: 12px; }
.state { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); font-size: 13px; }
.state.error { color: var(--color-error); }
.table-wrap { overflow-x: auto; }
.opt-table { width: 100%; border-collapse: collapse; font-size: 12px; font-variant-numeric: tabular-nums; }
.opt-table th { text-align: right; padding: 4px 8px; color: var(--color-text-tertiary); font-weight: 500; border-bottom: 1px solid var(--color-border-subtle); white-space: nowrap; }
.opt-table th:first-child { text-align: left; }
.opt-table td { text-align: right; padding: 4px 8px; border-bottom: 1px solid var(--color-border-subtle); }
.opt-table tr:hover td { background: var(--color-bg-hover); }
.td-code { text-align: left !important; color: var(--color-text-secondary); font-family: monospace; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }
</style>
