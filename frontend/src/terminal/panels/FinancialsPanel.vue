<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

interface FinPeriod {
  report_date: string
  [key: string]: string
}

interface FinStatements {
  income: FinPeriod[]
  balance: FinPeriod[]
  cashflow: FinPeriod[]
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
let loadSeq = 0
const loading = ref(false)
const error = ref('')
const statements = ref<FinStatements | null>(null)
const activeTab = ref<'income' | 'balance' | 'cashflow'>('income')

const tabs = [
  { key: 'income', label: '利润表' },
  { key: 'balance', label: '资产负债表' },
  { key: 'cashflow', label: '现金流量表' },
] as const

function smartFormat(val: string): string {
  const n = parseFloat(val)
  if (isNaN(n)) return val
  const abs = Math.abs(n)
  if (abs >= 1e12) return (n / 1e12).toFixed(2) + '万亿'
  if (abs >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toLocaleString('zh-CN')
}

async function loadData() {
  if (!symbol.value) return
  const seq = ++loadSeq
  loading.value = true
  error.value = ''
  try {
    const { data: res } = await fetchWithCache<any>(`financials:${symbol.value}`, () => (window as any).go?.main?.App?.GetFinancialStatements(symbol.value), 10 * 60 * 1000)
    if (seq !== loadSeq) return
    statements.value = {
      income: res.income || [],
      balance: res.balance || [],
      cashflow: res.cashflow || [],
    }
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

const activeData = computed(() => {
  if (!statements.value) return { periods: [] as string[], items: [] as string[], data: [] as FinPeriod[] }
  const data = statements.value[activeTab.value]
  if (!data || data.length === 0) return { periods: [], items: [], data: [] }
  const periods = data.map(p => p.report_date)
  const items: string[] = []
  const seen = new Set<string>()
  for (const p of data) {
    for (const k of Object.keys(p)) {
      if (k === 'report_date') continue
      if (!seen.has(k)) { seen.add(k); items.push(k) }
    }
  }
  return { periods, items, data }
})

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
</script>

<template>
  <div class="fin-panel">
    <div class="panel-header">
      <h3>财务报表</h3>
      <div class="header-right">
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading && !statements" type="table" :rows="5" />
    <div v-else-if="error" class="status error">{{ error }}</div>
    <div v-else-if="!loading && !statements?.income.length && !statements?.balance.length" class="status">暂无财务数据 — 输入 A 股代码查看</div>

    <template v-else>
      <div class="tab-bar">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="tab-btn"
          :class="{ active: activeTab === t.key }"
          @click="activeTab = t.key"
        >{{ t.label }}</button>
      </div>

      <div class="table-container">
        <div class="table-inner">
          <div class="t-head">
            <div class="t-row">
              <div class="t-cell t-h t-label">科目</div>
              <div
                v-for="p in activeData.periods"
                :key="p"
                class="t-cell t-h t-period"
              >{{ p.slice(0, 7) }}</div>
            </div>
          </div>
          <div class="t-body">
            <div
              v-for="item in activeData.items"
              :key="item"
              class="t-row"
              :class="{ 't-section': item.endsWith('合计') || item.endsWith('净额') }"
            >
              <div class="t-cell t-label">{{ item }}</div>
              <div
                v-for="p in activeData.periods"
                :key="p"
                class="t-cell t-val"
              >{{ smartFormat(activeData.data.find(d => d.report_date === p)?.[item] || '') }}</div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.fin-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text,var(--color-border)); background: var(--color-bg-panel,var(--color-bg-panel)); overflow: hidden; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-right { display: flex; align-items: center; gap: 8px; }
.symbol-badge { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(59,130,246,0.15); color: var(--color-accent); font-family: monospace; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-error); }

.tab-bar { display: flex; gap: 0; margin-bottom: 8px; border-bottom: 1px solid var(--color-border-strong); flex-shrink: 0; }
.tab-btn { padding: 6px 16px; border: none; border-bottom: 2px solid transparent; background: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; }
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn.active { color: var(--color-accent); border-bottom-color: var(--color-accent); }

.table-container { flex: 1; overflow: auto; min-height: 0; }
.table-inner { display: flex; flex-direction: column; min-width: max-content; font-size: 12px; }
.t-head { flex-shrink: 0; position: sticky; top: 0; z-index: 1; background: var(--color-bg-panel,var(--color-bg-panel)); }
.t-row { display: flex; border-bottom: 1px solid var(--color-border-subtle); }
.t-row:hover { background: var(--color-bg-elevated); }
.t-cell { padding: 4px 8px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-variant-numeric: tabular-nums; }
.t-h { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; font-weight: 600; padding: 6px 8px; }
.t-label { min-width: 140px; max-width: 140px; text-align: left; border-right: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.t-period { min-width: 100px; text-align: right; }
.t-val { min-width: 100px; text-align: right; }
.t-section { background: rgba(96,165,250,0.04); }
.t-section .t-label { font-weight: 600; color: var(--color-accent); }
</style>
