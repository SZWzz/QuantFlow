<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

// ══════ Shared ══════
type Market = 'CN' | 'US'
const market = ref<Market>('CN')

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || (market.value === 'CN' ? '600519' : 'AAPL'))
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()

// ══════ CN (A-share) financials ══════
interface FinPeriod {
  report_date: string
  [key: string]: string
}

interface FinStatements {
  income: FinPeriod[]
  balance: FinPeriod[]
  cashflow: FinPeriod[]
}

let loadSeq = 0
const cnLoading = ref(false)
const cnError = ref('')
const statements = ref<FinStatements | null>(null)
const activeTab = ref<'income' | 'balance' | 'cashflow'>('income')

const tabs = [
  { key: 'income', label: '利润表' },
  { key: 'balance', label: '资产负债表' },
  { key: 'cashflow', label: '现金流量表' },
] as const

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

function smartFormat(val: string): string {
  const n = parseFloat(val)
  if (isNaN(n)) return val
  const abs = Math.abs(n)
  if (abs >= 1e12) return (n / 1e12).toFixed(2) + '万亿'
  if (abs >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toLocaleString('zh-CN')
}

function getItemValue(data: any[], period: string, item: string): string {
  const periodData = data.find(d => d.report_date === period)
  if (!periodData) return ''
  // New format: items array with {item, value} pairs
  if (Array.isArray(periodData.items)) {
    const found = periodData.items.find((it: any) => it.item === item)
    return found?.value ?? ''
  }
  // Old format: flat key-value (backward compat)
  return periodData[item] ?? ''
}

async function loadCNData() {
  if (!symbol.value) return
  const seq = ++loadSeq
  cnLoading.value = true
  cnError.value = ''
  try {
    const { data: res } = await fetchWithCache<any>(`financials:${symbol.value}`, () => (window as any).go?.main?.App?.GetFinancialStatements(symbol.value), 10 * 60 * 1000)
    if (seq !== loadSeq) return
    statements.value = {
      income: res.income || [],
      balance: res.balance || [],
      cashflow: res.cashflow || [],
    }
  } catch (e: any) {
    cnError.value = e?.message || String(e)
  } finally {
    cnLoading.value = false
  }
}

const activeData = computed(() => {
  if (!statements.value) return { periods: [] as string[], items: [] as string[], data: [] as FinPeriod[] }
  const data = statements.value[activeTab.value]
  if (!data || data.length === 0) return { periods: [], items: [], data: [] }
  const periods = data.map(p => p.report_date)
  // Build ordered item list from the first period's items array (all periods share same items)
  const itemList: string[] = []
  const firstItems = data[0]?.items
  if (Array.isArray(firstItems)) {
    for (const it of firstItems) {
      if (it.item) itemList.push(it.item)
    }
  } else {
    // Fallback: old format with flat keys (for backward compat during transition)
    const seen = new Set<string>()
    for (const p of data) {
      for (const k of Object.keys(p)) {
        if (k === 'report_date' || k === 'items') continue
        if (!seen.has(k)) { seen.add(k); itemList.push(k) }
      }
    }
  }
  return { periods, items: itemList, data }
})

// ══════ US (SEC) financials ══════
const usLoading = ref(false)
const usError = ref('')
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
  if (abs >= 1e12) return (v / 1e12).toFixed(2) + '万亿'
  if (abs >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (v / 1e4).toFixed(2) + '万'
  return v.toLocaleString()
}

async function loadUSData() {
  usLoading.value = true; usError.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('sec_financials:' + symbol.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) rawData.value = JSON.parse(result.data)
      else if (result?.error) usError.value = result.error
    }
  } catch (e: any) { usError.value = e.message || '加载失败' }
  finally { usLoading.value = false }
}

const loading = computed(() => market.value === 'CN' ? cnLoading.value : usLoading.value)

function loadData() {
  if (market.value === 'CN') loadCNData()
  else loadUSData()
}

function onMarketChange(newMarket: Market) {
  market.value = newMarket
  // Use a sensible default symbol for the market
  if (newMarket === 'CN' && symbol.value === 'AAPL') symbol.value = '600519'
  if (newMarket === 'US' && symbol.value === '600519') symbol.value = 'AAPL'
  loadData()
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadCNData)
</script>

<template>
  <div class="fin-panel">
    <div class="panel-header">
      <h3>财务报表</h3>
      <!-- Market selector -->
      <div class="market-selector">
        <button :class="['market-tab', { active: market === 'CN' }]" @click="onMarketChange('CN')">A股</button>
        <button :class="['market-tab', { active: market === 'US' }]" @click="onMarketChange('US'); if (!rawData && !usLoading) loadUSData()">美股</button>
      </div>
      <div class="header-right">
        <button v-if="addToWfControl" class="wf-btn" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <!-- ── CN content ── -->
    <template v-if="market === 'CN'">
      <SkeletonPanel v-if="cnLoading && !statements" type="table" :rows="5" />
      <div v-else-if="cnError" class="status error">{{ cnError }}</div>
      <div v-else-if="!cnLoading && !statements?.income.length && !statements?.balance.length" class="status">暂无财务数据 — 输入 A 股代码查看</div>

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
                >{{ smartFormat(getItemValue(activeData.data, p, item)) }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </template>

    <!-- ── US content ── -->
    <template v-if="market === 'US'">
      <SkeletonPanel v-if="usLoading && sections.length === 0" type="table" :rows="6" />
      <div v-else-if="usError" class="status error">{{ usError }}</div>
      <div v-else-if="!usLoading && sections.length === 0" class="status">暂无财务数据 — 输入美股代码查看 SEC XBRL 财务报表</div>
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
    </template>
  </div>
</template>

<style scoped>
.fin-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text,var(--color-border)); background: var(--color-bg-panel,var(--color-bg-panel)); overflow: hidden; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; flex-shrink: 0; gap: 8px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; white-space: nowrap; }
.header-right { display: flex; align-items: center; gap: 8px; }

/* Market selector */
.market-selector { display: flex; gap: 0; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); overflow: hidden; }
.market-tab { padding: 2px 10px; border: none; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px; font-weight: 500; }
.market-tab + .market-tab { border-left: 1px solid var(--color-border-strong); }
.market-tab.active { color: var(--color-accent); background: rgba(59,130,246,0.1); }

.symbol-badge { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(59,130,246,0.15); color: var(--color-accent); font-family: monospace; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-error); }

/* CN tabs */
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

/* US sections */
.sections-scroll { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.fin-section { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); overflow: hidden; }
.section-title { margin: 0; padding: 6px 12px; font-size: 10px; font-weight: 600; color: var(--color-text-secondary); background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border-subtle); text-transform: uppercase; letter-spacing: 0.5px; }
.fin-table { padding: 2px 0; }
.fin-row { display: flex; justify-content: space-between; align-items: center; padding: 4px 12px; border-bottom: 1px solid var(--color-border-subtle); }
.fin-row:last-child { border-bottom: none; }
.fin-row:hover { background: var(--color-bg-hover); }
.fin-label { font-size: 11px; color: var(--color-text-secondary); text-transform: capitalize; }
.fin-value { font-size: 12px; font-weight: 500; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.fin-value.negative { color: var(--color-up); }

.wf-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  line-height: 1;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.wf-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(88, 166, 255, 0.1);
}
</style>
