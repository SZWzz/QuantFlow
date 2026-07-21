<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'

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

const headerTitle = computed(() => [symbol.value, name.value].filter(Boolean).join(' '))

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
  <div class="funds-panel">
    <PanelHeader :title="headerTitle" subtitle="公募基金" />

    <LoadingState v-if="loading" type="card" :rows="1" />
    <ErrorState v-else-if="error" :description="error" @retry="loadData" />
    <EmptyState v-else-if="!data || !data.success" :title="data?.error || '暂无数据'" />
    <EmptyState v-else-if="data.data?.length === 0" :title="`${name || symbol} 无基金持仓数据`" />

    <div v-else-if="data.data?.length === 1" class="panel-body">
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
    </div>

    <div v-else class="panel-body">
      <pre class="data-view">{{ JSON.stringify(data, null, 2) }}</pre>
    </div>
  </div>
</template>

<style scoped>
.funds-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }

.panel-body { flex: 1; overflow: auto; padding: var(--space-md) var(--panel-padding); }

.summary-card { border: 1px solid var(--color-border-subtle); border-radius: var(--radius-md); overflow: hidden; }
.summary-row { display: flex; justify-content: space-between; padding: var(--space-sm) var(--space-md); border-bottom: 1px solid var(--color-border-subtle); }
.summary-row:last-child { border-bottom: none; }
.summary-label { color: var(--color-text-tertiary); }
.summary-value { font-weight: 500; font-variant-numeric: tabular-nums; }
.summary-value.hl { color: var(--color-accent); font-size: var(--font-lg); }

.data-view { font-size: var(--font-xs); font-family: var(--font-mono); white-space: pre-wrap; color: var(--color-text-secondary); margin: 0; }
</style>
