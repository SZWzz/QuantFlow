<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed, nextTick } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import * as echarts from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([BarChart, LineChart, GridComponent, TooltipComponent, CanvasRenderer])

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
const chartContainer = ref<HTMLElement | null>(null)
let chartInstance: echarts.ECharts | null = null

const tabs = [
  { key: 'income' as const, label: '利润表' },
  { key: 'balance' as const, label: '资产负债表' },
  { key: 'cashflow' as const, label: '现金流量表' },
]

const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)

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
  if (!statements.value) return { periods: [] as string[], items: [] as string[], data: [] as FinPeriod[] }
  const data = statements.value[activeTab.value]
  if (!data || data.length === 0) return { periods: [], items: [], data: [] }
  const sorted = [...data].sort((a, b) => a.report_date.localeCompare(b.report_date))
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

function buildChart() {
  if (!chartContainer.value) return
  if (!chartInstance) {
    chartInstance = echarts.init(chartContainer.value)
  }
  const { series, chartData } = trendMetrics.value
  if (!series.length || !chartData.length) return

  const isDark = document.documentElement.classList.contains('dark')
  const textColor = isDark ? '#8b949e' : '#666'
  const borderColor = isDark ? '#30363d' : '#e5e7eb'

  chartInstance.setOption({
    grid: { left: 10, right: 20, top: 20, bottom: 30 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category', data: chartData.map(d => d.date),
      axisLabel: { fontSize: 10, color: textColor }, axisLine: { lineStyle: { color: borderColor } },
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 10, color: textColor, formatter: (v: number) => smartFormat(v) },
      splitLine: { lineStyle: { color: borderColor, type: 'dashed' } },
    },
    series: series.map((name, i) => ({
      name,
      type: i === 0 ? 'bar' : 'line',
      data: chartData.map(d => d[name] || 0),
      itemStyle: { color: i === 0 ? '#58a6ff' : '#f0883e' },
      lineStyle: { width: 2 },
      symbol: 'circle', symbolSize: 4,
      smooth: true,
    })),
  }, true)
}

watch([activeTab, trendMetrics], () => nextTick(buildChart))
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

const loading = computed(() => market.value === 'CN' ? cnLoading.value : usLoading.value)

function loadData() {
  if (market.value === 'CN') loadCNData()
  else loadUSData()
}

function onMarketChange(newMarket: Market) {
  market.value = newMarket
  if (newMarket === 'CN' && symbol.value === 'AAPL') symbol.value = '600519'
  if (newMarket === 'US' && symbol.value === '600519') symbol.value = 'AAPL'
  loadData()
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadCNData)
onUnmounted(() => { chartInstance?.dispose(); chartInstance = null })
</script>

<template>
  <div class="fin-panel">
    <!-- ═══ Header ═══ -->
    <div class="panel-header">
      <div class="header-left">
        <h3>财务报表</h3>
        <div class="market-selector">
          <button :class="['market-tab', { active: market === 'CN' }]" @click="onMarketChange('CN')">A股</button>
          <button :class="['market-tab', { active: market === 'US' }]" @click="onMarketChange('US'); if (!rawData && !usLoading) loadUSData()">美股</button>
        </div>
      </div>
      <div class="header-right">
        <button v-if="addToWfControl" class="wf-btn" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <!-- ═══ CN: A-Share Content ═══ -->
    <template v-if="market === 'CN'">
      <SkeletonPanel v-if="cnLoading && !statements" type="table" :rows="8" />
      <div v-else-if="cnError" class="status error">
        <span v-html="getIcon('warning')" />
        <span>{{ cnError }}</span>
        <button class="retry-btn" @click="loadCNData">重试</button>
      </div>
      <div v-else-if="!cnLoading && !statements?.income.length && !statements?.balance.length" class="status">
        <span v-html="getIcon('search')" />
        <span>暂无财务数据 — 输入 A 股代码查看</span>
      </div>

      <template v-else>
        <!-- Tab bar + toggle -->
        <div class="tab-row">
          <div class="tab-bar">
            <button v-for="t in tabs" :key="t.key" class="tab-btn" :class="{ active: activeTab === t.key }" @click="activeTab = t.key">{{ t.label }}</button>
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

          <!-- Statement table -->
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
.fin-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text-primary); background: var(--color-bg-panel);
  overflow: hidden; gap: 6px;
}
.body-scroll {
  flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 8px; min-height: 0;
}

/* Header */
.panel-header {
  display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; gap: 8px;
}
.header-left { display: flex; align-items: center; gap: 10px; }
.panel-header h3 { margin: 0; font-size: 15px; font-weight: 700; letter-spacing: -0.2px; }
.header-right { display: flex; align-items: center; gap: 8px; }
.market-selector { display: flex; border: 1px solid var(--color-border-strong); border-radius: 6px; overflow: hidden; }
.market-tab { padding: 3px 12px; border: none; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 12px; font-weight: 500; transition: all .15s; }
.market-tab + .market-tab { border-left: 1px solid var(--color-border-strong); }
.market-tab.active { color: var(--color-accent); background: rgba(88,166,255,0.1); }
.symbol-badge { font-size: 11px; padding: 3px 10px; border-radius: 6px; background: rgba(88,166,255,0.1); color: var(--color-accent); font-family: 'SF Mono', monospace; font-weight: 500; }
.refresh-btn { width: 28px; height: 28px; display: flex; align-items: center; justify-content: center; border: 1px solid var(--color-border-strong); border-radius: 6px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 14px; }
.refresh-btn:disabled { opacity: 0.4; cursor: default; }
.status { display: flex; align-items: center; justify-content: center; gap: 8px; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-error); }
.retry-btn { padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: transparent; color: var(--color-accent); cursor: pointer; font-size: 11px; }

/* Tab Row */
.tab-row { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--color-border-strong); flex-shrink: 0; }
.tab-bar { display: flex; gap: 0; }
.tab-btn { padding: 7px 18px; border: none; border-bottom: 2px solid transparent; background: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 13px; font-weight: 500; transition: all .15s; }
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn.active { color: var(--color-accent); border-bottom-color: var(--color-accent); }
.growth-toggle { display: flex; align-items: center; gap: 4px; padding: 4px 8px; margin-bottom: 4px; cursor: pointer; font-size: 11px; color: var(--color-text-tertiary); border-radius: 4px; }
.growth-toggle:hover { color: var(--color-text-primary); }
.growth-toggle input { accent-color: var(--color-accent); }
.toggle-label { user-select: none; }

/* Trend chart */
.trend-section {
  flex-shrink: 0; background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle); border-radius: 8px;
  padding: 8px 4px 0 4px;
}
.trend-chart { width: 100%; height: 140px; }

/* KPI Section */
.kpi-section { flex-shrink: 0; display: flex; flex-direction: column; gap: 8px; }
.kpi-row { display: flex; gap: 8px; flex-wrap: wrap; }
.kpi-card {
  flex: 0 0 170px;
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle);
  border-radius: 8px; padding: 8px 12px; display: flex; flex-direction: column; gap: 2px;
}
.kpi-card.highlight {
  border-color: var(--color-accent-soft);
  background: linear-gradient(135deg, rgba(88,166,255,0.06), rgba(88,166,255,0.02));
}
.kpi-label { font-size: 11px; color: var(--color-text-tertiary); }
.kpi-value { font-size: 17px; font-weight: 700; font-variant-numeric: tabular-nums; letter-spacing: -0.3px; display: flex; align-items: baseline; gap: 6px; }
.kpi-yoy-inline { font-size: 12px; font-weight: 500; }
.kpi-yoy-inline.up { color: var(--color-down); }
.kpi-yoy-inline.down { color: var(--color-up); }
.kpi-yoy-inline.flat { color: var(--color-text-tertiary); }

/* Ratio cards */
.ratio-row { display: flex; gap: 8px; flex-wrap: wrap; }
.ratio-card {
  background: var(--color-bg-subtle); border: 1px solid var(--color-border-subtle);
  border-radius: 6px; padding: 6px 12px; display: flex; flex-direction: column; gap: 2px;
}
.ratio-label { font-size: 10px; color: var(--color-text-tertiary); }
.ratio-value { font-size: 14px; font-weight: 600; font-variant-numeric: tabular-nums; color: var(--color-accent); }

/* Table */
.table-container { flex-shrink: 0; }
.table-inner { display: flex; flex-direction: column; min-width: max-content; font-size: 12px; }
.t-head { flex-shrink: 0; position: sticky; top: 0; z-index: 1; background: var(--color-bg-panel); box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
.t-row { display: flex; border-bottom: 1px solid var(--color-border-subtle); transition: background .1s; }
.t-row:hover { background: var(--color-bg-elevated); }
.t-cell { padding: 5px 10px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-variant-numeric: tabular-nums; }
.t-h { font-size: 10px; color: var(--color-text-tertiary); font-weight: 600; padding: 7px 10px; letter-spacing: 0.3px; }
.t-label { min-width: 155px; max-width: 155px; text-align: left; border-right: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.t-period { min-width: 110px; text-align: right; }
.t-val { min-width: 135px; text-align: right; }
.period-label { font-weight: 600; }
.t-subtotal { background: rgba(96,165,250,0.03); }
.t-subtotal .t-label { font-weight: 600; color: var(--color-text-secondary); }
.t-highlight { background: rgba(88,166,255,0.06); }
.t-highlight .t-label { font-weight: 700; color: var(--color-accent); font-size: 12.5px; }
.t-highlight .val-main { font-weight: 700; }
.val-row { display: flex; flex-direction: column; align-items: flex-end; gap: 1px; }
.val-main { font-size: 12px; }
.val-yoy { font-size: 10px; font-weight: 500; line-height: 1; }
.val-yoy.up { color: var(--color-down); }
.val-yoy.down { color: var(--color-up); }
.val-yoy.flat { color: var(--color-text-tertiary); }

/* US Sections */
.sections-scroll { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 12px; }
.fin-section { background: var(--color-bg-elevated); border: 1px solid var(--color-border-subtle); border-radius: 8px; overflow: hidden; }
.section-title { margin: 0; padding: 8px 14px; font-size: 11px; font-weight: 600; color: var(--color-text-secondary); background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border-subtle); text-transform: uppercase; letter-spacing: 0.5px; }
.fin-table { padding: 2px 0; }
.fin-row { display: flex; justify-content: space-between; align-items: center; padding: 5px 14px; border-bottom: 1px solid var(--color-border-subtle); }
.fin-row:last-child { border-bottom: none; }
.fin-row:hover { background: var(--color-bg-hover); }
.fin-label { font-size: 11px; color: var(--color-text-secondary); text-transform: capitalize; }
.fin-value { font-size: 12px; font-weight: 500; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.fin-value.negative { color: var(--color-up); }
.wf-btn { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1px solid var(--color-border-strong); border-radius: 6px; background: var(--color-bg-elevated); color: var(--color-text-secondary); font-size: 16px; font-weight: 600; cursor: pointer; line-height: 1; transition: all var(--transition-fast); flex-shrink: 0; }
.wf-btn:hover { border-color: var(--color-accent); color: var(--color-accent); background: rgba(88,166,255,0.1); }
</style>
