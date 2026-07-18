<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '000001')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const rawData = ref<any>(null)

const SOURCE = 'akshare'
const DATA_TYPE = 'margin'

interface MarginRow {
  date: string
  margin_balance: number
  short_balance: number
  margin_balance_day: number
  short_balance_day: number
  [key: string]: any
}

const rows = computed<MarginRow[]>(() => {
  if (!rawData.value) return []
  const data = rawData.value.data ?? rawData.value
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.items)) return data.items
  if (Array.isArray(data?.records)) return data.records
  return []
})

const latest = computed(() => rows.value[0] || null)

const columns = [
  { key: 'date', label: '日期', width: '100px' },
  { key: 'margin_balance', label: '融资余额(亿)', width: '110px', fmt: (v: number) => (v / 1e8).toFixed(2) },
  { key: 'short_balance', label: '融券余额(亿)', width: '110px', fmt: (v: number) => (v / 1e8).toFixed(2) },
  { key: 'margin_balance_day', label: '日融资买入(亿)', width: '120px', fmt: (v: number) => (v / 1e8).toFixed(2) },
  { key: 'short_balance_day', label: '日融券卖出(亿)', width: '120px', fmt: (v: number) => (v / 1e8).toFixed(2) },
]

function colValue(row: MarginRow, key: string) {
  const v = row[key] ?? row[key.toLowerCase()] ?? row[key.toUpperCase()]
  return v
}

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('margin:' + symbol.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) rawData.value = JSON.parse(result.data)
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
</script>

<template>
  <div class="margin-panel">
    <div class="panel-header">
      <h3>融资融券</h3>
      <div class="header-right">
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading && rows.length === 0" type="table" :rows="5" />

    <div v-else-if="error" class="status error">{{ error }}</div>
    <div v-else-if="!loading && rows.length === 0" class="status">暂无融资融券数据</div>

    <template v-else>
      <div v-if="latest" class="stats-row">
        <div class="stat-card">
          <span class="stat-label">融资余额</span>
          <span class="stat-value">{{ ((colValue(latest, 'margin_balance') || 0) / 1e8).toFixed(1) }}<small>亿</small></span>
        </div>
        <div class="stat-card">
          <span class="stat-label">融券余额</span>
          <span class="stat-value">{{ ((colValue(latest, 'short_balance') || 0) / 1e8).toFixed(1) }}<small>亿</small></span>
        </div>
        <div class="stat-card">
          <span class="stat-label">余额差值</span>
          <span class="stat-value">{{ (((colValue(latest, 'margin_balance') || 0) - (colValue(latest, 'short_balance') || 0)) / 1e8).toFixed(1) }}<small>亿</small></span>
        </div>
      </div>

      <div class="table-wrapper">
        <div class="table-header">
          <span v-for="col in columns" :key="col.key" :style="{ width: col.width }">{{ col.label }}</span>
        </div>
        <div class="table-body">
          <div v-for="row in rows.slice(0, 30)" :key="row.date" class="table-row">
            <span v-for="col in columns" :key="col.key" :style="{ width: col.width }">
              {{ col.fmt ? col.fmt(colValue(row, col.key) || 0) : colValue(row, col.key) }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.margin-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.header-right { display: flex; align-items: center; gap: 8px; }
.symbol-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: rgba(59,130,246,0.15);
  color: var(--color-accent);
  font-family: 'JetBrains Mono', monospace;
}
.refresh-btn {
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.status {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: var(--color-text-tertiary);
  font-size: 13px;
}
.status.error { color: var(--color-error); }
.stats-row {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
  flex-shrink: 0;
}
.stat-card {
  flex: 1;
  padding: 10px 14px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.stat-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.stat-value { font-size: 18px; font-weight: 700; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.stat-value small { font-size: 11px; font-weight: 400; color: var(--color-text-tertiary); margin-left: 2px; }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.table-row span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-variant-numeric: tabular-nums; }
</style>
