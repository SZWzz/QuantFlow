<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const loading = ref(false)
const error = ref('')
const rawData = ref<any>(null)

const SOURCE = 'sec'
const DATA_TYPE = 'financials'

interface FinRow { label: string; value: number | string }
const sections = computed(() => {
  if (!rawData.value) return []
  const data = rawData.value.data ?? rawData.value
  const result: { title: string; rows: FinRow[] }[] = []
  const items = Array.isArray(data) ? data : [data]
  for (const item of items) {
    if (!item || typeof item !== 'object') continue
    for (const [sectionKey, sectionVal] of Object.entries(item)) {
      if (typeof sectionVal !== 'object' || sectionVal === null) continue
      const rows: FinRow[] = []
      for (const [k, v] of Object.entries(sectionVal as Record<string, any>)) {
        if (typeof v === 'object' && v !== null) continue
        rows.push({ label: k.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase()), value: v })
      }
      if (rows.length > 0) result.push({ title: sectionKey.replace(/_/g, ' ').toUpperCase(), rows })
    }
  }
  return result
})

function fmtVal(v: number | string): string {
  if (typeof v === 'string') return v
  const abs = Math.abs(v)
  if (abs >= 1e12) return (v / 1e12).toFixed(2) + 'T'
  if (abs >= 1e9) return (v / 1e9).toFixed(2) + 'B'
  if (abs >= 1e6) return (v / 1e6).toFixed(2) + 'M'
  if (abs >= 1e3) return (v / 1e3).toFixed(1) + 'K'
  return v.toLocaleString()
}

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const result = await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
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
  <div class="sec-fin-panel">
    <div class="panel-header">
      <h3>SEC 财务报表</h3>
      <div class="header-right">
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>
    <SkeletonPanel v-if="loading && sections.length === 0" type="table" :rows="6" />
    <div v-else-if="error" class="status error">{{ error }}</div>
    <div v-else-if="!loading && sections.length === 0" class="status">暂无财务数据 — 输入美股代码查看 SEC XBRL 财务报表</div>
    <div v-else class="sections-scroll">
      <div v-for="section in sections" :key="section.title" class="fin-section">
        <h4 class="section-title">{{ section.title }}</h4>
        <div class="fin-table">
          <div v-for="row in section.rows" :key="row.label" class="fin-row">
            <span class="fin-label">{{ row.label }}</span>
            <span class="fin-value" :class="{ negative: typeof row.value === 'number' && row.value < 0 }">{{ fmtVal(row.value) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sec-fin-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, #e5e7eb); background: var(--color-bg-panel, #1a1a2e); overflow: hidden; }
.panel-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-right { display: flex; align-items: center; gap: 8px; }
.symbol-badge { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: rgba(30,144,255,0.15); color: #1e90ff; font-family: 'JetBrains Mono', monospace; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-error); }
.sections-scroll { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.fin-section { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: 8px; overflow: hidden; }
.section-title { margin: 0; padding: 6px 12px; font-size: 10px; font-weight: 600; color: var(--color-text-secondary); background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border-subtle); text-transform: uppercase; letter-spacing: 0.5px; }
.fin-table { padding: 2px 0; }
.fin-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 12px; border-bottom: 1px solid var(--color-border-subtle); }
.fin-row:last-child { border-bottom: none; }
.fin-row:hover { background: var(--color-bg-hover); }
.fin-label { font-size: 11px; color: var(--color-text-secondary); text-transform: capitalize; }
.fin-value { font-size: 12px; font-weight: 500; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.fin-value.negative { color: #f85149; }
</style>
