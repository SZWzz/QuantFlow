<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const activeTab = ref<'income' | 'balance' | 'cashflow'>('income')
const loading = ref(false)
const error = ref('')
const data = ref<Record<string, any>>({})

async function loadFinancials() {
  loading.value = true
  error.value = ''
  try {
    const go = (window as any).go
    if (go?.main?.App?.FetchData) {
      const result = await go.main.App.FetchData('akshare', 'financials', [symbol.value], '', '', {})
      if (result?.data) {
        data.value = JSON.parse(result.data)
      } else if (result?.error) {
        error.value = result.error
      }
    } else {
      data.value = {}
    }
  } catch (e: any) {
    error.value = e.message || 'Failed to load financials'
  } finally {
    loading.value = false
  }
}

watch(symbol, loadFinancials)
onMounted(loadFinancials)
</script>

<template>
  <div class="financials-panel">
    <div class="panel-header">
      <span class="symbol">{{ symbol }} {{ name }}</span>
      <div class="tabs">
        <button :class="{ active: activeTab === 'income' }" @click="activeTab = 'income'">利润表</button>
        <button :class="{ active: activeTab === 'balance' }" @click="activeTab = 'balance'">资产负债表</button>
        <button :class="{ active: activeTab === 'cashflow' }" @click="activeTab = 'cashflow'">现金流量表</button>
      </div>
    </div>
    <div class="panel-body">
      <div v-if="loading" class="status">加载中...</div>
      <div v-else-if="error" class="status error">{{ error }}</div>
      <div v-else-if="!Object.keys(data).length" class="status">选择标的查看财务数据</div>
      <pre v-else class="data-view">{{ JSON.stringify(data, null, 2) }}</pre>
    </div>
  </div>
</template>

<style scoped>
.financials-panel { display: flex; flex-direction: column; height: 100%; }
.panel-header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid var(--color-border-subtle); }
.symbol { font-weight: 500; }
.tabs { display: flex; gap: 4px; }
.tabs button { padding: 4px 10px; border: 1px solid var(--color-border-subtle); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; }
.tabs button.active { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.panel-body { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); padding: 16px; }
.status { font-size: 13px; }
.error { color: var(--color-error, #e74c3c); }
.data-view { width: 100%; height: 100%; overflow: auto; font-size: 12px; font-family: monospace; white-space: pre-wrap; padding: 8px; color: var(--color-text-primary); background: var(--color-bg-subtle); border-radius: 4px; }
</style>
