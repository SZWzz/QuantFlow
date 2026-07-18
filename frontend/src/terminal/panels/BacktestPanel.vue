<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { CandlestickChart, BarChart, LineChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, DataZoomComponent, MarkPointComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { PanelHeader, PanelTable, PanelCard, EmptyState, LoadingState } from '@/terminal/components/panel'
import { confirmDialog, alertDialog } from '@/lib/wails'
import { useI18n } from 'vue-i18n'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { buildKlineOption } from '@/lib/buildChartOption'
import type { TradeSignal, KlineDataItem } from '@/lib/buildChartOption'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { createIndicatorCache } from '@/lib/composables/useIndicators'

use([CanvasRenderer, CandlestickChart, BarChart, LineChart, TooltipComponent, GridComponent, DataZoomComponent, MarkPointComponent])

interface BacktestSummary {
  id: number
  run_id: string
  workflow_name: string
  strategy_name: string
  symbol: string
  engine_type: string
  total_return: number
  cagr: number
  max_drawdown: number
  sharpe_ratio: number
  sortino_ratio: number
  calmar_ratio: number
  win_rate: number
  profit_factor: number
  total_trades: number
  backtest_start: string
  backtest_end: string
  started_at: string
  finished_at: string
  created_at: string
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const { t } = useI18n()
const chartTheme = useChartTheme()
const indicatorCache = createIndicatorCache()

const listControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', title: t('common.refresh'), action: loadList, loading: loading })
  list.push({ icon: 'delete', title: '清空全部记录', action: deleteAllRecords })
  return list
})

const detailControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ label: '← 返回列表', action: goBack })
  list.push({ label: t('common.delete'), action: deleteDetail })
  return list
})

type ViewMode = 'list' | 'detail'
const view = ref<ViewMode>('list')
const detailId = ref<number | null>(null)
const deleted = ref(false)

watch(() => props.params?.storeId, (id) => {
  if (id) { detailId.value = id; view.value = 'detail'; loadStoredResult(id) }
}, { immediate: true })

function goBack() { view.value = 'list'; detailId.value = null; loadList() }

const items = ref<BacktestSummary[]>([])
const loading = ref(false)

const columns = computed(() => [
  { key: 'backtest_start', label: '回测开始', width: 90, formatter: (v: string) => v?.slice(0, 10) || '-', align: 'left' as const },
  { key: 'backtest_end', label: '回测结束', width: 90, formatter: (v: string) => v?.slice(0, 10) || '-', align: 'left' as const },
  { key: 'workflow_name', label: '工作流', flex: 1 },
  { key: 'strategy_name', label: '策略', width: 90, formatter: (v: string) => v || '-' },
  { key: 'symbol', label: '标的', width: 70, formatter: (v: string) => v || '-' },
  { key: 'total_return', label: '收益率', width: 80, align: 'right' as const, format: 'percent' as const, colorize: true },
  { key: 'sharpe_ratio', label: 'Sharpe', width: 70, align: 'right' as const, format: 'number' as const },
  { key: 'total_trades', label: '交易', width: 50, align: 'right' as const },
])

async function loadList() {
  loading.value = true
  try {
    const res = await (window as any).go.main.App.ListBacktestHistory(100, 0)
    items.value = res || []
  } catch (e) { console.error('ListBacktestHistory failed:', e) }
  finally { loading.value = false }
}

function openRow(row: any) {
  selectedRow = row
  detailId.value = row.id
  view.value = 'detail'
  loadStoredResult(row.id)
}

async function deleteAllRecords() {
  if (!items.value.length) return
  if (!(await confirmDialog(`确定删除全部 ${items.value.length} 条回测记录？此操作不可撤销。`))) return
  try {
    await (window as any).go.main.App.ClearBacktestResults()
    await loadList()
  } catch (e: any) { await alertDialog(`删除失败: ${e.message || e}`) }
}

async function deleteSingleList(id: number) {
  if (!(await confirmDialog('确定删除此回测记录？'))) return
  try {
    await (window as any).go.main.App.DeleteBacktestResult(id)
    await loadList()
  } catch (e: any) { await alertDialog(`删除失败: ${e.message || e}`) }
}

const storedLoading = ref(false)
const storedData = ref<any>(null)

async function loadStoredResult(id: number) {
  storedLoading.value = true; storedData.value = null; deleted.value = false
  try {
    const res = await (window as any).go.main.App.GetStoredBacktestResult(id)
    if (!res) return
    storedData.value = {
      equity_curve: safeParseJSON(res.equity_curve, []).map((p: any) => p.equity ?? p),
      trades: safeParseJSON(res.trades_json, []),
      ohlcv: safeParseJSON(res.ohlcv_data, []),
      metrics: { total_return: res.total_return, cagr: res.cagr, max_drawdown: res.max_drawdown, sharpe_ratio: res.sharpe_ratio, sortino_ratio: res.sortino_ratio, calmar_ratio: res.calmar_ratio, win_rate: res.win_rate, profit_factor: res.profit_factor, total_trades: res.total_trades },
      workflow_name: res.workflow_name, strategy_name: res.strategy_name, symbol: res.symbol,
      backtest_start: res.backtest_start, backtest_end: res.backtest_end,
    }
  } catch (e) { console.error('GetStoredBacktestResult failed:', e) }
  finally { storedLoading.value = false }
}

function safeParseJSON(s: string | undefined, fallback: any): any {
  if (!s) return fallback
  try { return JSON.parse(s) } catch { return fallback }
}

async function deleteDetail() {
  if (!detailId.value) return
  if (!(await confirmDialog('确定删除此回测记录？'))) return
  try {
    await (window as any).go.main.App.DeleteBacktestResult(detailId.value)
    goBack()
  } catch (e) { await alertDialog('删除失败，请重试') }
}

const klineData = computed<KlineDataItem[]>(() => {
  const bars = storedData.value?.ohlcv
  if (!bars || bars.length === 0) return []
  return bars.map((b: any) => ({ date: b.Date || b.date || '', open: b.Open ?? b.open ?? 0, high: b.High ?? b.high ?? 0, low: b.Low ?? b.low ?? 0, close: b.Close ?? b.close ?? 0, volume: b.Volume ?? b.volume ?? 0 }))
})

function findCandleIndex(dateStr: string): number { return klineData.value.findIndex(d => d.date === dateStr) }

const tradeSignals = computed<TradeSignal[]>(() => {
  const ts = storedData.value?.trades
  if (!ts || klineData.value.length === 0) return []
  const signals: TradeSignal[] = []
  for (const t of ts) {
    const dataIndex = t.date ? findCandleIndex(t.date) : -1
    if (dataIndex < 0 && t.price) signals.push({ dataIndex: -1, direction: t.direction || t.side, price: t.price, label: t.direction === 'buy' ? 'B' : 'S' })
    else if (dataIndex >= 0) signals.push({ dataIndex, direction: t.direction || t.side, price: t.price ?? klineData.value[dataIndex].close, label: t.direction === 'buy' ? 'B' : 'S' })
  }
  return signals
})

const chartOption = computed<ECBasicOption>(() => {
  const kd = klineData.value
  if (kd.length === 0) return {} as ECBasicOption
  return buildKlineOption(
    kd,
    'ma',
    'volume',
    chartTheme,
    indicatorCache,
    storedData.value?.symbol || '',
    '1d',
    undefined,
    undefined,
    tradeSignals.value.filter(s => s.dataIndex >= 0),
  )
})

const equityOption = computed<ECBasicOption>(() => {
  const eq = storedData.value?.equity_curve
  if (!eq || eq.length < 2) return {} as ECBasicOption
  const upColor = eq[eq.length - 1] >= eq[0] ? chartTheme.upColor : chartTheme.downColor
  return { backgroundColor: 'transparent', animation: false, grid: { left: 54, right: 16, top: 10, bottom: 20 }, xAxis: { type: 'category', show: false, axisLine: { lineStyle: { color: chartTheme.axisColor } } }, yAxis: { type: 'value', scale: true, axisLabel: { color: chartTheme.axisColor, fontSize: 10, formatter: (v: number) => (v / 10000).toFixed(2) + '万' }, splitLine: { lineStyle: { color: chartTheme.gridColor } }, min: Math.min(...eq), max: Math.max(...eq) }, series: [{ type: 'line', data: eq, smooth: true, symbol: 'none', lineStyle: { color: upColor, width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: upColor + '40' }, { offset: 1, color: 'rgba(0,0,0,0)' }] } } }], tooltip: { trigger: 'axis', formatter: (ps: any[]) => ps?.[0] ? `<div style="font-size:12px">净值: ${Number(ps[0].value).toFixed(2)}</div>` : '' } } as ECBasicOption
})

const metricCards = computed(() => {
  const m = storedData.value?.metrics; if (!m) return []
  return [{ label: '总收益率', value: m.total_return, format: 'percent' as const, color: true }, { label: '年化收益率', value: m.cagr, format: 'percent' as const, color: true }, { label: '最大回撤', value: m.max_drawdown, format: 'percent' as const, color: true }, { label: '夏普比率', value: m.sharpe_ratio, format: 'number' as const }, { label: '索提诺比率', value: m.sortino_ratio, format: 'number' as const }, { label: '卡玛比率', value: m.calmar_ratio, format: 'number' as const }, { label: '胜率', value: m.win_rate, format: 'percent' as const, color: true }, { label: '盈亏比', value: m.profit_factor >= 999998 ? Infinity : m.profit_factor, format: 'number' as const }, { label: '总交易次数', value: m.total_trades, format: 'number' as const }]
})

const tradeColumns = [
  { key: 'date', label: '日期', width: 100 }, { key: 'symbol', label: '标的', width: 70 },
  { key: 'side', label: '方向', width: 50, formatter: (v: string) => v === 'buy' ? '买入' : v === 'sell' ? '卖出' : v ?? '-' },
  { key: 'quantity', label: '数量', width: 60, align: 'right' as const },
  { key: 'price', label: '价格', width: 70, align: 'right' as const, format: 'price' as const },
  { key: 'pnl', label: '盈亏', width: 80, align: 'right' as const, format: 'price' as const, colorize: true },
]

let selectedRow: any = null

async function onKeyDown(e: KeyboardEvent) {
  if (!selectedRow) return
  if (e.key === 'Delete' || e.key === 'Backspace') {
    e.preventDefault()
    if (await confirmDialog('确定删除此回测记录？')) {
      try {
        await (window as any).go.main.App.DeleteBacktestResult(selectedRow.id)
        await loadList()
      } catch (err: any) {
        await alertDialog('删除失败: ' + (err?.message || err))
      }
    }
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    openRow(selectedRow)
  }
}

function selectRow(row: any) { selectedRow = row }

onMounted(() => {
  if (!props.params?.storeId) loadList()
  document.addEventListener('keydown', onKeyDown)
})
onUnmounted(() => { document.removeEventListener('keydown', onKeyDown) })
</script>

<template>
  <div class="backtest-panel" data-testid="backtest-panel">
    <template v-if="view === 'list'">
      <PanelHeader
        title="回测历史"
        :subtitle="`${items.length} 条记录`"
        :controls="listControls"
      />
      <div class="panel-body">
        <LoadingState v-if="loading && items.length === 0" type="table" :rows="5" :cols="7" />
        <EmptyState
          v-else-if="items.length === 0 && !loading"
          icon="inbox"
          title="暂无回测记录"
          description="在 Workflow Editor 中运行回测工作流后，结果将在此显示。"
        />
        <template v-else>
          <PanelTable
            :columns="columns"
            :data="items"
            :loading="loading"
            clickable
            rowTestId="backtest-row"
            @rowClick="openRow"
          >
            <template #action="{ row }">
              <button class="btn-icon-sm" title="删除" @click.stop="deleteSingleList(row.id)">✕</button>
            </template>
          </PanelTable>
        </template>
      </div>
    </template>

    <template v-else>
      <PanelHeader
        :title="storedData?.strategy_name || '回测详情'"
        :subtitle="storedData ? `${storedData.symbol} ｜ ${storedData.backtest_start?.slice(0,10) || '?'} → ${storedData.backtest_end?.slice(0,10) || '?'}` : ''"
        :controls="detailControls"
      />
      <div class="panel-body scrollable">
        <LoadingState v-if="storedLoading" type="chart" />
        <EmptyState v-else-if="deleted" icon="inbox" title="已删除" description="此回测记录已被删除。" />
        <EmptyState v-else-if="!storedData" icon="chart" title="暂无回测结果" description="历史回测数据加载失败或数据不完整。" />
        <template v-else>
          <div v-if="klineData.length > 0" class="chart-section">
            <div class="section-label">K 线 + 买卖点</div>
            <VChart :option="chartOption" autoresize class="kline-chart" />
          </div>
          <div v-if="storedData.equity_curve?.length >= 2" class="chart-section">
            <div class="section-label">净值曲线</div>
            <VChart :option="equityOption" autoresize class="equity-chart" />
          </div>
          <div v-if="metricCards.length" class="section">
            <div class="section-label">回测指标</div>
            <div class="metrics-grid" data-testid="backtest-metrics">
              <PanelCard v-for="m in metricCards" :key="m.label" :title="m.label" :value="m.value" :format="m.format" />
            </div>
          </div>
          <div v-if="storedData.trades?.length" class="section">
            <div class="section-label">交易记录 ({{ storedData.trades.length }})</div>
            <PanelTable :columns="tradeColumns" :data="storedData.trades.slice(0, 50)" :striped="true" />
          </div>
        </template>
      </div>
    </template>
  </div>
</template>

<style scoped>
.backtest-panel { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-panel); position: relative; }
.panel-body { flex: 1; overflow: hidden; display: flex; flex-direction: column; padding: var(--space-sm); }
.panel-body.scrollable { overflow-y: auto; }
.chart-section { margin-bottom: var(--space-lg); }
.section-label { font-size: var(--font-sm); font-weight: 600; color: var(--color-text-secondary); margin-bottom: var(--space-sm); text-transform: uppercase; letter-spacing: 0.5px; }
.section { margin-bottom: var(--space-lg); }
.kline-chart { height: 360px; width: 100%; }
.equity-chart { height: 200px; width: 100%; }
.metrics-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(200px, 100%), 1fr)); gap: var(--card-gap); }
.btn-icon-sm { background: none; border: none; cursor: pointer; padding: 2px 6px; font-size: var(--font-xs); color: var(--color-text-tertiary); border-radius: var(--radius-sm); transition: all var(--transition-fast); }
.btn-icon-sm:hover { color: var(--color-down); background: var(--color-down-soft); }
</style>
