<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed, nextTick } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, LoadingState, EmptyState, ErrorState } from '@/terminal/components/panel'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import * as echarts from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([BarChart, LineChart, GridComponent, TooltipComponent, CanvasRenderer])

// ══════ Shared ══════
type Market = 'CN' | 'HK' | 'US'
function detectMarket(sym: string): Market {
  if (/^\d{6}$/.test(sym)) return 'CN'
  if (/^\d{1,5}$/.test(sym)) return 'HK'
  return 'US'
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const chartTheme = useChartTheme()

const market = computed<Market>(() => detectMarket(symbol.value))

// ══════ CN (A-share) financials ══════
interface FinPeriod {
  report_date: string
  items?: { item: string; value: number }[]
  [key: string]: any
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
const showGrowth = ref(true)
const reportType = ref<'annual' | 'quarterly'>('annual')
const chartContainer = ref<HTMLElement | null>(null)
let chartInstance: echarts.ECharts | null = null

const tabs = [
  { key: 'income' as const, label: '利润表' },
  { key: 'balance' as const, label: '资产负债表' },
  { key: 'cashflow' as const, label: '现金流量表' },
]

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

const marketBadge = computed(() => market.value === 'CN' ? 'A股' : market.value === 'HK' ? '港股' : '美股')
const headerSubtitle = computed(() => `${symbol.value} ${name.value || ''}`.trim())

const headerControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', title: '刷新', action: loadData, loading: loading.value })
  return list
})

function parseNum(v: any): number {
  const n = typeof v === 'string' ? parseFloat(v) : v
  return (typeof n === 'number' && !isNaN(n)) ? n : 0
}

function smartFormat(val: any): string {
  const n = parseNum(val)
  if (n === 0 && val !== 0 && String(val) !== '0') return String(val ?? '')
  const abs = Math.abs(n)
  if (abs >= 1e12) return (n / 1e12).toFixed(2) + '万亿'
  if (abs >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (n / 1e4).toFixed(2) + '万'
  return n.toLocaleString('zh-CN')
}

function getItemValue(data: any[], period: string, item: string): number | string {
  const periodData = data.find(d => d.report_date === period)
  if (!periodData) return ''
  if (Array.isArray(periodData.items)) {
    const found = periodData.items.find((it: any) => it.item === item)
    return found?.value ?? ''
  }
  return periodData[item] ?? ''
}

function getYoY(data: any[], period: string, item: string): { pct: number | null; trend: 'up' | 'down' | 'flat' } {
  const periods = data.map(p => p.report_date).sort()
  const idx = periods.indexOf(period)
  if (idx <= 0) return { pct: null, trend: 'flat' }
  const prev = periods[idx - 1]
  const cur = parseNum(getItemValue(data, period, item))
  const prv = parseNum(getItemValue(data, prev, item))
  if (cur === 0 || prv === 0) return { pct: null, trend: 'flat' }
  const pct = ((cur - prv) / Math.abs(prv)) * 100
  return { pct, trend: pct > 0.5 ? 'up' : pct < -0.5 ? 'down' : 'flat' }
}

function isSubtotalRow(item: string): boolean {
  return item.endsWith('合计') || item.endsWith('净额') || item.endsWith('余额')
    || item === '净利润' || item === '营业利润' || item === '利润总额'
    || item === '资产总计' || item === '负债合计' || item === '所有者权益合计'
}

function isHighlightRow(item: string): boolean {
  return item.includes('合计') || item.includes('总计') || item === '净利润' || item === '营业利润'
    || item === '利润总额' || item === '营业收入' || item === '资产总计'
    || item === '负债合计' || item === '所有者权益合计'
    || item.includes('现金流量净额') || item === '期末现金及现金等价物余额'
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
  const stmts = statements.value
  if (!stmts) return { periods: [] as string[], items: [] as string[], data: [] as FinPeriod[] }
  const data = stmts[activeTab.value]
  if (!data || data.length === 0) return { periods: [], items: [], data: [] }
  const sorted = [...data]
    .filter(p => reportType.value === 'annual' ? p.report_date.endsWith('-12-31') : !p.report_date.endsWith('-12-31'))
    .sort((a, b) => a.report_date.localeCompare(b.report_date))
  const periods = sorted.map(p => p.report_date)
  const itemList: string[] = []
  const firstItems = sorted[0]?.items
  if (Array.isArray(firstItems)) {
    for (const it of firstItems) {
      if (it.item) itemList.push(it.item)
    }
  } else {
    const seen = new Set<string>()
    for (const p of sorted) {
      for (const k of Object.keys(p)) {
        if (k === 'report_date' || k === 'items') continue
        if (!seen.has(k)) { seen.add(k); itemList.push(k) }
      }
    }
  }
  return { periods, items: itemList.filter(item => sorted.some(p => {
    const v = getItemValue(sorted, p.report_date, item)
    return parseNum(v) !== 0
  })), data: sorted }
})

// KPI summary from latest period
const kpiSummary = computed(() => {
  if (!activeData.value.data.length || !activeData.value.items.length) return []
  const sortedData = activeData.value.data
  const latest = sortedData[sortedData.length - 1]
  const items = activeData.value.items
  const kpiKeys: Record<string, string[]> = {
    income: ['营业收入', '营业利润', '利润总额', '净利润'],
    balance: ['资产总计', '负债合计', '所有者权益合计'],
    cashflow: ['经营活动产生的现金流量净额', '投资活动产生的现金流量净额', '筹资活动产生的现金流量净额', '期末现金及现金等价物余额'],
  }
  const keys = kpiKeys[activeTab.value] || []
  return keys.filter(k => items.includes(k)).map(k => {
    const val = getItemValue(sortedData, latest.report_date, k)
    const yoy = getYoY(sortedData, latest.report_date, k)
    return { item: k, value: val, yoy }
  })
})

// Computed financial ratios
const ratios = computed(() => {
  const data = activeData.value.data
  if (!data.length) return []
  const latest = data[data.length - 1]
  const items = activeData.value.items

  function val(name: string): number {
    return parseNum(getItemValue(data, latest.report_date, name))
  }

  const result: { label: string; value: string; desc: string }[] = []

  if (activeTab.value === 'income') {
    const revenue = val('营业收入')
    const cost = val('营业成本')
    const profit = val('净利润')
    if (revenue > 0 && cost > 0) {
      result.push({ label: '毛利率', value: (((revenue - cost) / revenue) * 100).toFixed(1) + '%', desc: '(收入-成本)/收入' })
    }
    if (revenue > 0 && profit !== 0) {
      result.push({ label: '净利率', value: ((profit / revenue) * 100).toFixed(1) + '%', desc: '净利润/收入' })
    }
  }

  if (activeTab.value === 'balance') {
    const assets = val('资产总计')
    const debt = val('负债合计')
    const equity = val('所有者权益合计')
    if (assets > 0 && debt > 0) {
      result.push({ label: '资产负债率', value: ((debt / assets) * 100).toFixed(1) + '%', desc: '负债/总资产' })
    }
    if (equity > 0 && assets > 0) {
      result.push({ label: '权益乘数', value: (assets / equity).toFixed(2), desc: '总资产/净资产' })
    }
  }

  return result
})

// Trend chart
const trendMetrics = computed(() => {
  const data = activeData.value.data
  if (!data.length) return { series: [] as string[], chartData: [] as any[] }
  const items = activeData.value.items
  const map: Record<string, string[]> = {
    income: ['营业收入', '净利润'],
    balance: ['资产总计', '负债合计', '所有者权益合计'],
    cashflow: ['经营活动产生的现金流量净额'],
  }
  const keys = (map[activeTab.value] || []).filter(k => items.includes(k))
  const chartData = data.map(p => {
    const row: any = { date: p.report_date.slice(0, 7) }
    for (const k of keys) row[k] = parseNum(getItemValue(data, p.report_date, k))
    return row
  })
  return { series: keys, chartData }
})

// ECharts option as computed so it redraws on theme change (P6)
const chartOption = computed<echarts.EChartsCoreOption | null>(() => {
  const { series, chartData } = trendMetrics.value
  if (!series.length || !chartData.length) return null
  const textColor = chartTheme.axisColor
  const borderColor = chartTheme.splitColor
  return {
    grid: { left: 10, right: 20, top: 20, bottom: 30 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category', data: chartData.map(d => d.date),
      axisLabel: { fontSize: 10, color: textColor }, axisLine: { lineStyle: { color: borderColor } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 10, color: textColor, formatter: (v: number) => smartFormat(v) },
      splitLine: { lineStyle: { color: chartTheme.gridColor, type: 'dashed' } },
    },
    series: series.map((name, i) => ({
      name,
      type: i === 0 ? 'bar' : 'line',
      data: chartData.map(d => d[name] || 0),
      itemStyle: { color: i === 0 ? chartTheme.palette[0] : chartTheme.palette[1] },
      lineStyle: { width: 2 },
      symbol: 'circle', symbolSize: 4,
      smooth: true,
    })),
  }
})

function buildChart() {
  if (!chartContainer.value) return
  if (!chartInstance) {
    chartInstance = echarts.init(chartContainer.value)
  }
  if (!chartOption.value) return
  chartInstance.setOption(chartOption.value, true)
}

watch([activeTab, trendMetrics, chartOption], () => nextTick(buildChart))
watch(() => statements.value, () => nextTick(buildChart))

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

// ══════ HK financials ══════
const hkLoading = ref(false)
const hkError = ref('')

async function loadHKData() {
  if (!symbol.value) return
  hkLoading.value = true
  hkError.value = ''
  try {
    const { data: res } = await fetchWithCache<any>(`hk_financials:${symbol.value}`, () => (window as any).go?.main?.App?.GetHKFinancialStatements(symbol.value), 10 * 60 * 1000)
    statements.value = {
      income: res.income || [],
      balance: res.balance || [],
      cashflow: res.cashflow || [],
    }
  } catch (e: any) {
    hkError.value = e?.message || String(e)
  } finally {
    hkLoading.value = false
  }
}

const loading = computed(() => {
  if (market.value === 'CN') return cnLoading.value
  if (market.value === 'HK') return hkLoading.value
  return usLoading.value
})

const activeError = computed(() => {
  if (market.value === 'CN') return cnError.value
  if (market.value === 'HK') return hkError.value
  return usError.value
})

function onTabChange(key: string) {
  activeTab.value = key as typeof activeTab.value
}

function loadData() {
  if (market.value === 'CN') loadCNData()
  else if (market.value === 'HK') loadHKData()
  else loadUSData()
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
onUnmounted(() => { chartInstance?.dispose(); chartInstance = null })
</script>

<template>
  <div class="fin-panel">
    <!-- ═══ Header (P1) ═══ -->
    <PanelHeader
      title="财务报表"
      :subtitle="headerSubtitle"
      :tabs="tabs"
      :active-tab="activeTab"
      :controls="headerControls"
      @tab-change="onTabChange"
    >
      <template #controls>
        <span class="market-badge">{{ marketBadge }}</span>
      </template>
    </PanelHeader>

    <!-- ═══ CN/HK: A-Share + HK Content ═══ -->
    <template v-if="market !== 'US'">
      <LoadingState v-if="loading && !statements" type="table" :rows="8" />
      <ErrorState v-else-if="cnError || hkError" :description="cnError || hkError" @retry="loadData" />
      <EmptyState
        v-else-if="!loading && !statements?.income.length && !statements?.balance.length"
        icon="search"
        title="暂无财务数据"
        description="输入股票代码查看"
      />

      <template v-else>
        <!-- Secondary period bar (P1 tokenized): report-type toggle + 同比 checkbox -->
        <div class="period-bar">
          <div class="report-toggle">
            <button :class="{ active: reportType === 'annual' }" @click="reportType = 'annual'">年报</button>
            <button :class="{ active: reportType === 'quarterly' }" @click="reportType = 'quarterly'">季报</button>
          </div>
          <label class="growth-toggle" title="显示同比变化">
            <input type="checkbox" v-model="showGrowth" />
            <span class="toggle-label">同比</span>
          </label>
        </div>

        <div class="body-scroll">
          <!-- Trend chart -->
          <div v-if="trendMetrics.series.length" class="trend-section">
            <div ref="chartContainer" class="trend-chart" />
          </div>

          <!-- KPI cards + ratios -->
          <div class="kpi-section">
            <div v-if="kpiSummary.length" class="kpi-row">
              <div v-for="kpi in kpiSummary" :key="kpi.item" class="kpi-card" :class="{ highlight: isHighlightRow(kpi.item) }">
                <div class="kpi-label">{{ kpi.item }}</div>
                <div class="kpi-value">
                  {{ smartFormat(kpi.value) }}
                  <span v-if="showGrowth && kpi.yoy.pct !== null" class="kpi-yoy-inline" :class="kpi.yoy.trend">
                    {{ kpi.yoy.trend === 'up' ? '↑' : kpi.yoy.trend === 'down' ? '↓' : '' }}{{ Math.abs(kpi.yoy.pct).toFixed(1) }}%
                  </span>
                </div>
              </div>
            </div>

            <!-- Ratios -->
            <div v-if="ratios.length" class="ratio-row">
              <div v-for="r in ratios" :key="r.label" class="ratio-card" :title="r.desc">
                <div class="ratio-label">{{ r.label }}</div>
                <div class="ratio-value">{{ r.value }}</div>
              </div>
            </div>
          </div>

          <!-- Statement table — 自绘保留：动态列(每期一列) + subtotal/highlight 行级着色 + 单元格内嵌 YoY 副值，PanelTable 表达不了 -->
          <div class="table-container">
            <div class="table-inner">
              <div class="t-head">
                <div class="t-row">
                  <div class="t-cell t-h t-label">科目</div>
                  <div v-for="p in activeData.periods" :key="p" class="t-cell t-h t-period">
                    <div class="period-label">{{ p.slice(0, 7) }}</div>
                  </div>
                </div>
              </div>
              <div class="t-body">
                <div
                  v-for="item in activeData.items"
                  :key="item"
                  class="t-row"
                  :class="{ 't-subtotal': isSubtotalRow(item), 't-highlight': isHighlightRow(item) }"
                >
                  <div class="t-cell t-label">{{ item }}</div>
                  <div v-for="p in activeData.periods" :key="p" class="t-cell t-val">
                    <div class="val-row">
                      <span class="val-main">{{ smartFormat(getItemValue(activeData.data, p, item)) }}</span>
                      <span v-if="showGrowth" class="val-yoy" :class="getYoY(activeData.data, p, item).trend">
                        <template v-if="getYoY(activeData.data, p, item).pct !== null">
                          {{ getYoY(activeData.data, p, item).trend === 'up' ? '+' : '' }}{{ (getYoY(activeData.data, p, item).pct!).toFixed(1) }}%
                        </template>
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </template>

    <!-- ═══ US: SEC Content ═══ -->
    <template v-if="market === 'US'">
      <LoadingState v-if="usLoading && sections.length === 0" type="table" :rows="6" />
      <ErrorState v-else-if="usError" :description="usError" @retry="loadData" />
      <EmptyState
        v-else-if="!usLoading && sections.length === 0"
        icon="search"
        title="暂无财务数据"
        description="输入美股代码查看 SEC XBRL 财务报表"
      />
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
.fin-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: var(--color-text-primary);
}
.body-scroll {
  flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: var(--space-sm); min-height: 0;
}

/* Header badge (P1) — placed in #controls slot */
.market-badge {
  font-size: var(--font-xs);
  padding: var(--space-xs) var(--space-sm);
  border-radius: var(--radius-sm);
  background: var(--color-accent-soft);
  color: var(--color-accent);
  font-weight: 500;
}

/* Secondary period bar (P1 tokenized) */
.period-bar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-xs) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}
.growth-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast);
}
.growth-toggle:hover { color: var(--color-text-primary); }
.growth-toggle input { accent-color: var(--color-accent); }
.toggle-label { user-select: none; }

.report-toggle {
  display: flex;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.report-toggle button {
  padding: var(--space-xs) var(--space-sm);
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
  font-weight: 500;
  transition: all var(--transition-fast);
}
.report-toggle button + button { border-left: 1px solid var(--color-border-strong); }
.report-toggle button.active {
  color: var(--color-accent);
  background: var(--color-accent-soft);
}

/* Trend chart */
.trend-section {
  flex-shrink: 0;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  padding: var(--space-sm) var(--space-xs) 0 var(--space-xs);
}
.trend-chart { width: 100%; height: 140px; }

/* KPI Section */
.kpi-section { flex-shrink: 0; display: flex; flex-direction: column; gap: var(--space-sm); }
.kpi-row { display: flex; gap: var(--space-sm); flex-wrap: wrap; }
.kpi-card {
  flex: 0 0 170px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  padding: var(--space-sm) var(--space-md);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.kpi-card.highlight {
  border-color: var(--color-accent-soft);
  background: linear-gradient(135deg, var(--color-accent-soft), transparent);
}
.kpi-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.kpi-value {
  font-size: var(--font-xl);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.3px;
  display: flex;
  align-items: baseline;
  gap: var(--space-sm);
}
.kpi-yoy-inline { font-size: var(--font-xs); font-weight: 500; }
.kpi-yoy-inline.up { color: var(--color-down); }
.kpi-yoy-inline.down { color: var(--color-up); }
.kpi-yoy-inline.flat { color: var(--color-text-tertiary); }

/* Ratio cards */
.ratio-row { display: flex; gap: var(--space-sm); flex-wrap: wrap; }
.ratio-card {
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  padding: var(--space-xs) var(--space-md);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ratio-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.ratio-value {
  font-size: var(--font-base);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-accent);
}

/* Table — 自绘保留（动态列 + 行级 subtotal/highlight + 单元格内嵌 YoY），全部 token 化 */
.table-container { flex-shrink: 0; }
.table-inner {
  display: flex;
  flex-direction: column;
  min-width: max-content;
  font-size: var(--font-xs);
}
.t-head {
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--color-bg-panel);
  box-shadow: var(--shadow-sm);
}
.t-row {
  display: flex;
  border-bottom: 1px solid var(--color-border-subtle);
  transition: background var(--transition-fast);
}
.t-row:hover { background: var(--color-bg-elevated); }
.t-cell {
  padding: 5px var(--space-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}
.t-h {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  font-weight: 600;
  padding: 7px var(--space-sm);
  letter-spacing: 0.3px;
}
.t-label {
  min-width: 155px;
  max-width: 155px;
  text-align: left;
  border-right: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}
.t-period { min-width: 110px; text-align: right; }
.t-val { min-width: 135px; text-align: right; }
.period-label { font-weight: 600; }
.t-subtotal { background: var(--color-bg-subtle); }
.t-subtotal .t-label { font-weight: 600; color: var(--color-text-secondary); }
.t-highlight { background: var(--color-accent-soft); }
.t-highlight .t-label {
  font-weight: 700;
  color: var(--color-accent);
  font-size: var(--font-sm);
}
.t-highlight .val-main { font-weight: 700; }
.val-row {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 1px;
}
.val-main { font-size: var(--font-xs); }
.val-yoy { font-size: var(--font-xs); font-weight: 500; line-height: 1; }
.val-yoy.up { color: var(--color-down); }
.val-yoy.down { color: var(--color-up); }
.val-yoy.flat { color: var(--color-text-tertiary); }

/* US Sections */
.sections-scroll {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}
.fin-section {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.section-title {
  margin: 0;
  padding: var(--space-sm) var(--space-md);
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border-subtle);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.fin-table { padding: 2px 0; }
.fin-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 5px var(--space-md);
  border-bottom: 1px solid var(--color-border-subtle);
}
.fin-row:last-child { border-bottom: none; }
.fin-row:hover { background: var(--color-bg-hover); }
.fin-label { font-size: var(--font-xs); color: var(--color-text-secondary); text-transform: capitalize; }
.fin-value {
  font-size: var(--font-xs);
  font-weight: 500;
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}
.fin-value.negative { color: var(--color-up); }
</style>
