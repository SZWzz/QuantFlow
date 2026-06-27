<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const loading = ref(false)
const error = ref('')
const data = ref<any>(null)

const SOURCE = 'sec'
const DATA_TYPE = 'financials'

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const result = await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

watch(symbol, loadData)
onMounted(loadData)
</script>

<template>
  <div class="panel-container">
    <div class="panel-header"><span class="symbol">{{ symbol }} {{ name }}</span><span class="badge">SEC财报</span></div>
    <div class="panel-body">
      <div v-if="loading" class="status">加载中...</div>
      <div v-if="error" class="status error">{{ error }}</div>
      <div v-if="!loading && !error && !data" class="status">选择标的查看数据</div>
      <pre v-if="data" class="data-view">{{ JSON.stringify(data, null, 2) }}</pre>
    </div>
  </div>
</template>

<style scoped>
.panel-container{display:flex;flex-direction:column;height:100%}
.panel-header{display:flex;justify-content:space-between;align-items:center;padding:8px 12px;border-bottom:1px solid var(--color-border-subtle)}
.symbol{font-weight:500}
.badge{font-size:11px;background:var(--color-primary);color:#fff;padding:2px 8px;border-radius:10px}
.panel-body{flex:1;overflow:auto;padding:12px}
.status{display:flex;align-items:center;justify-content:center;height:100%;color:var(--color-text-tertiary);font-size:13px}
.status.error{color:var(--color-error)}
.data-view{font-size:12px;font-family:monospace;white-space:pre-wrap;color:var(--color-text-secondary);margin:0}
</style>
