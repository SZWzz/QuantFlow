<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
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
const error = ref('')
const data = ref<any>(null)

const SOURCE = 'akshare'
const DATA_TYPE = 'funds'

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('funds:' + symbol.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) data.value = JSON.parse(result.data)
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)

function fmtShares(v: number | undefined): string {
  if (!v) return '--'
  if (v >= 1e8) return (v / 1e8).toFixed(2) + ' 亿'
  if (v >= 1e4) return (v / 1e4).toFixed(2) + ' 万'
  return v.toLocaleString()
}
function fmtValue(v: number | undefined): string {
  if (!v) return '--'
  if (v >= 1e8) return (v / 1e8).toFixed(2) + ' 亿元'
  return v.toLocaleString()
}
</script>

<template>
  <div class="panel-container">
    <div class="panel-header">
      <span class="symbol">{{ symbol }} {{ name }}</span>
      <span class="badge">公募基金</span>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="status">加载中...</div>
      <div v-else-if="error" class="status error">{{ error }}</div>
      <div v-else-if="!data || !data.success" class="status">{{ data?.error || '暂无数据' }}</div>

      <template v-else-if="data.data?.length === 0">
        <div class="status">{{ name || symbol }} 无基金持仓数据</div>
      </template>

      <template v-else-if="data.data?.length === 1">
        <div class="summary-card">
          <div class="summary-row">
            <span class="summary-label">报告期</span>
            <span class="summary-value">{{ data.data[0]['报告期'] || data.data[0]['date'] || '--' }}</span>
          </div>
          <div class="summary-row">
            <span class="summary-label">基金覆盖</span>
            <span class="summary-value hl">{{ data.data[0]['基金覆盖家数'] || 0 }} 家</span>
          </div>
          <div class="summary-row">
            <span class="summary-label">持股总数</span>
            <span class="summary-value">{{ fmtShares(data.data[0]['持股总数']) }}</span>
          </div>
          <div class="summary-row">
            <span class="summary-label">持股总市值</span>
            <span class="summary-value">{{ fmtValue(data.data[0]['持股总市值']) }}</span>
          </div>
        </div>
      </template>

      <template v-else>
        <pre class="data-view">{{ JSON.stringify(data, null, 2) }}</pre>
      </template>
    </div>
  </div>
</template>

<style scoped>
.panel-container{display:flex;flex-direction:column;height:100%;background:var(--color-bg-panel);color:var(--color-text-primary);font-size:13px}
.panel-header{display:flex;justify-content:space-between;align-items:center;padding:8px 12px;border-bottom:1px solid var(--color-border)}
.symbol{font-weight:500}
.badge{font-size:11px;background:var(--color-primary);color:var(--color-text-primary);padding:2px 8px;border-radius:10px}
.panel-body{flex:1;overflow:auto;padding:12px}
.status{display:flex;align-items:center;justify-content:center;height:100%;color:var(--color-text-tertiary);font-size:13px}
.status.error{color:var(--color-error)}

.summary-card{border:1px solid var(--color-border-subtle);border-radius:6px;overflow:hidden}
.summary-row{display:flex;justify-content:space-between;padding:10px 14px;border-bottom:1px solid var(--color-border-subtle)}
.summary-row:last-child{border-bottom:none}
.summary-label{color:var(--color-text-tertiary)}
.summary-value{font-weight:500;font-variant-numeric:tabular-nums}
.summary-value.hl{color:var(--color-primary);font-size:16px}

.data-view{font-size:12px;font-family:monospace;white-space:pre-wrap;color:var(--color-text-secondary);margin:0}
</style>
