<script setup lang="ts">
import { computed, shallowRef, ref, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { CandlestickChart, BarChart, LineChart } from 'echarts/charts'
import { TooltipComponent, GridComponent, DataZoomComponent, MarkPointComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import type { ECBasicOption } from 'echarts/types/dist/shared'
import { useWorkflowStore } from '@/stores/workflow'
import type { TradeSignal, KlineDataItem } from '@/lib/buildChartOption'

use([CanvasRenderer, CandlestickChart, BarChart, LineChart, TooltipComponent, GridComponent, DataZoomComponent, MarkPointComponent])

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const workflow = useWorkflowStore()

/* ── dual-mode: runtime (workflow.nodeOutputs) vs history (storeId) ── */

const storeId = computed(() => props.params?.storeId as number | undefined)
const storedLoading = ref(false)
const storedData = ref<any>(null) // transformed to match btOutput shape

function findBacktestOutput(): any {
  for (const [, outputs] of workflow.nodeOutputs) {
    if (outputs && outputs.equity_curve) return outputs
  }
  return null
}

function findDataLoaderOhlcv(): any[] | null {
  for (const edge of workflow.edges) {
    if (edge.targetHandle === 'ohlcv_data') {
      const srcOutputs = workflow.nodeOutputs.get(edge.source)
      if (srcOutputs?.ohlcv && Array.isArray(srcOutputs.ohlcv)) {
        return srcOutputs.ohlcv
      }
    }
  }
  return null
}

async function loadStoredResult(id: number) {
  storedLoading.value = true
  try {
    const res = await (window as any).go.main.App.GetStoredBacktestResult(id)
    if (!res) return
    storedData.value = {
      equity_curve: safeParseJSON(res.equity_curve, []).map((p: any) => p.equity ?? p),
      trades: safeParseJSON(res.trades_json, []),
      ohlcv: safeParseJSON(res.ohlcv_data, []),
      metrics: {
        total_return: res.total_return,
        cagr: res.cagr,
        max_drawdown: res.max_drawdown,
        sharpe_ratio: res.sharpe_ratio,
        sortino_ratio: res.sortino_ratio,
        calmar_ratio: res.calmar_ratio,
        win_rate: res.win_rate,
        profit_factor: res.profit_factor,
        annual_volatility: res.annual_volatility ?? 0,
        total_trades: res.total_trades,
      },
    }
  } catch (e) {
    console.error('GetStoredBacktestResult failed:', e)
  } finally {
    storedLoading.value = false
  }
}

function safeParseJSON(s: string | undefined, fallback: any): any {
  if (!s) return fallback
  try { return JSON.parse(s) } catch { return fallback }
}

watch(storeId, (id) => {
  if (id) { storedData.value = null; loadStoredResult(id) }
}, { immediate: true })

/* ── reactive state ── */

const btOutput = computed(() => (storeId.value ? storedData.value : findBacktestOutput()))
const ohlcvBars = computed(() => (storeId.value ? storedData.value?.ohlcv ?? null : findDataLoaderOhlcv()))
const trades = computed<any[] | null>(() => btOutput.value?.trades ?? null)
const btMetrics = computed<any>(() => btOutput.value?.metrics ?? null)
const equityCurve = computed<number[] | null>(() => btOutput.value?.equity_curve ?? null)

/* ── K-line data conversion ── */

const klineData = computed<KlineDataItem[]>(() => {
  const bars = ohlcvBars.value
  if (!bars || bars.length === 0) return []
  return bars.map((b: any) => ({
    date: b.Date || b.date || '',
    open: b.Open ?? b.open ?? 0,
    high: b.High ?? b.high ?? 0,
    low: b.Low ?? b.low ?? 0,
    close: b.Close ?? b.close ?? 0,
    volume: b.Volume ?? b.volume ?? 0,
  }))
})

/* ── trade signals → markPoint ── */

function findCandleIndex(dateStr: string): number {
  return klineData.value.findIndex(d => d.date === dateStr)
}

const tradeSignals = computed<TradeSignal[]>(() => {
  const ts = trades.value
  if (!ts || klineData.value.length === 0) return []
  const signals: TradeSignal[] = []
  for (const t of ts) {
    const dataIndex = t.date ? findCandleIndex(t.date) : -1
    if (dataIndex < 0 && t.price) {
      signals.push({
        dataIndex: -1,
        direction: t.direction || t.side,
        price: t.price,
        label: t.direction === 'buy' ? 'B' : 'S',
      })
    } else if (dataIndex >= 0) {
      signals.push({
        dataIndex,
        direction: t.direction || t.side,
        price: t.price ?? klineData.value[dataIndex].close,
        label: t.direction === 'buy' ? 'B' : 'S',
      })
    }
  }
  return signals
})

/* ── ECharts option ── */

const chartOption = shallowRef<ECBasicOption>({})

function buildOption(): ECBasicOption {
  const kd = klineData.value
  if (kd.length === 0) return {} as ECBasicOption

  const dates = kd.map(d => d.date)
  const kdata = kd.map(d => [d.open, d.close, d.low, d.high])
  const close = kd.map(d => d.close)
  const upCol = '#ef5350'
  const downCol = '#26a69a'

  const volData = kd.map(d => ({
    value: d.volume / 10000,
    itemStyle: { color: d.close >= d.open ? upCol : downCol },
  }))

  const signals = tradeSignals.value
  const hasValidSignals = signals.some(s => s.dataIndex >= 0)

  const series: any[] = [
    {
      type: 'candlestick',
      data: kdata,
      itemStyle: { color: upCol, color0: downCol, borderColor: upCol, borderColor0: downCol },
      markPoint: hasValidSignals ? {
        silent: true,
        symbolSize: 28,
        data: signals.filter(s => s.dataIndex >= 0).map(s => ({
          coord: [s.dataIndex, s.price],
          itemStyle: { color: s.direction === 'buy' ? '#f85149' : '#3fb950' },
          symbol: 'pin',
          symbolRotate: s.direction === 'buy' ? 180 : 0,
          label: { formatter: s.direction === 'buy' ? 'B' : 'S', color: '#fff', fontSize: 10, fontWeight: 'bold' as const },
        })),
      } : undefined,
    },
    {
      type: 'line', name: 'SMA5', data: sma(close, 5),
      symbol: 'none', lineStyle: { width: 1, color: '#f59e0b' },
    },
    {
      type: 'line', name: 'SMA20', data: sma(close, 20),
      symbol: 'none', lineStyle: { width: 1, color: '#8b5cf6' },
    },
    { type: 'bar', name: '成交量', data: volData, xAxisIndex: 1, yAxisIndex: 1 },
  ]

  const totalPoints = kd.length
  const windowSize = Math.min(totalPoints, 250)
  const startPct = totalPoints > windowSize ? ((totalPoints - windowSize) / totalPoints * 100) : 0

  return {
    backgroundColor: 'transparent',
    animation: false,
    grid: [
      { left: 54, right: 10, top: 8, height: '52%' },
      { left: 54, right: 10, top: '68%', height: '26%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#2a2a3a' } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: '#2a2a3a' } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: '#8b8ba0', fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { type: 'value', gridIndex: 1, scale: true, axisLabel: { color: '#8b8ba0', fontSize: 10, formatter: (v: number) => v.toFixed(0) }, splitLine: { show: false } },
    ],
    series,
    tooltip: {
      trigger: 'axis',
      formatter: (ps: any[]) => {
        if (!ps?.length) return ''
        const lines: string[] = [`<div style="font-size:12px">${ps[0].name || ''}</div>`]
        for (const p of ps) {
          if (p.seriesType === 'candlestick' && Array.isArray(p.data)) {
            lines.push(`<div style="margin-top:4px">开: ${p.data[0].toFixed(2)}</div>`)
            lines.push(`<div>收: ${p.data[1].toFixed(2)}</div>`)
            lines.push(`<div>低: ${p.data[2].toFixed(2)}</div>`)
            lines.push(`<div>高: ${p.data[3].toFixed(2)}</div>`)
          } else if (p.seriesType === 'bar') {
            const raw = kd[p.dataIndex]?.volume ?? 0
            lines.push(`<div>成交量: ${(raw / 10000).toFixed(1)}万</div>`)
          } else {
            lines.push(`<div>${p.seriesName}: ${p.value?.toFixed(2) ?? ''}</div>`)
          }
        }
        return lines.join('')
      },
    },
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: startPct, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], start: startPct, end: 100, bottom: 0, height: 18 },
    ],
  } as ECBasicOption
}

/* ── equity curve chart option ── */

const equityOption = computed<ECBasicOption>(() => {
  const eq = equityCurve.value
  if (!eq || eq.length < 2) return {} as ECBasicOption
  const min = Math.min(...eq)
  const max = Math.max(...eq)
  const range = (max - min) || 1
  const upColor = eq[eq.length - 1] >= eq[0] ? '#ef5350' : '#26a69a'

  return {
    backgroundColor: 'transparent',
    animation: false,
    grid: { left: 54, right: 16, top: 10, bottom: 20 },
    xAxis: { type: 'category', show: false, axisLine: { lineStyle: { color: '#2a2a3a' } } },
    yAxis: {
      type: 'value', scale: true,
      axisLabel: { color: '#8b8ba0', fontSize: 10, formatter: (v: number) => (v / 10000).toFixed(2) + '万' },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
      min, max,
    },
    series: [{
      type: 'line',
      data: eq,
      smooth: true,
      symbol: 'none',
      lineStyle: { color: upColor, width: 2 },
      areaStyle: {
        color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
          { offset: 0, color: upColor + '40' },
          { offset: 1, color: 'rgba(0,0,0,0)' },
        ]},
      },
    }],
    tooltip: {
      trigger: 'axis',
      formatter: (ps: any[]) => {
        if (!ps?.length) return ''
        return `<div style="font-size:12px">净值: ${Number(ps[0].value).toFixed(2)}</div>`
      },
    },
  } as ECBasicOption
})

// watch kline data and rebuild
watch(() => klineData.value.length, () => {
  chartOption.value = buildOption()
}, { immediate: true })

/* ── metrics helpers ── */

function gm(v: number | undefined | null): string {
  if (v == null) return '-'
  return (v * 100).toFixed(2) + '%'
}
function gn(v: number | undefined | null): string {
  if (v == null) return '-'
  return v.toFixed(2)
}

const metricsList = computed(() => {
  const m = btMetrics.value
  if (!m) return []
  return [
    { label: '总收益率', value: gm(m.total_return ?? m.TotalReturn) },
    { label: '年化收益率', value: gm(m.cagr ?? m.CAGR) },
    { label: '最大回撤', value: gm(m.max_drawdown ?? m.MaxDrawdown) },
    { label: '夏普比率', value: gn(m.sharpe_ratio ?? m.SharpeRatio) },
    { label: '索提诺比率', value: gn(m.sortino_ratio ?? m.SortinoRatio) },
    { label: '卡玛比率', value: gn(m.calmar_ratio ?? m.CalmarRatio) },
    { label: '胜率', value: gm(m.win_rate ?? m.WinRate) },
    { label: '盈亏比', value: gn(m.profit_factor ?? m.ProfitFactor) },
    { label: '年化波动率', value: gm(m.annual_volatility ?? m.AnnualVolatility) },
    { label: '总交易次数', value: String(m.total_trades ?? m.TotalTrades ?? '-') },
  ]
})

/* ── SMA helper (no external cache) ── */
function sma(data: number[], period: number): number[] {
  const r: number[] = []
  for (let i = 0; i < data.length; i++) {
    const start = Math.max(0, i - period + 1)
    let sum = 0
    for (let j = start; j <= i; j++) sum += data[j]
    r.push(sum / (i - start + 1))
  }
  return r
}
</script>

<template>
  <div class="bt-panel">
    <!-- loading state (history mode) -->
    <div v-if="storedLoading" class="empty-state">
      <span class="empty-text">加载中...</span>
    </div>

    <!-- empty state -->
    <div v-else-if="klineData.length === 0 && !btOutput" class="empty-state">
      <span class="empty-icon">📊</span>
      <span class="empty-text">暂无回测结果</span>
      <span class="empty-desc" v-if="!storeId">在 Workflow Editor 中运行回测工作流后，结果将在此显示。</span>
      <span class="empty-desc" v-else>历史回测数据加载失败或数据不完整。</span>
    </div>

    <template v-else>
      <!-- K-line chart with buy/sell markers -->
      <div v-if="klineData.length > 0" class="section">
        <div class="section-title">K线 + 买卖点</div>
        <VChart :option="chartOption" autoresize style="height:360px; width:100%" />
      </div>

      <!-- equity curve -->
      <div v-if="equityCurve && equityCurve.length >= 2" class="section">
        <div class="section-title">净值曲线</div>
        <VChart :option="equityOption" autoresize style="height:200px; width:100%" />
      </div>

      <!-- metrics grid -->
      <div v-if="btMetrics" class="section">
        <div class="section-title">回测指标</div>
        <div class="metrics-grid">
          <div v-for="item in metricsList" :key="item.label" class="metric-item">
            <span class="metric-label">{{ item.label }}</span>
            <span class="metric-value">{{ item.value }}</span>
          </div>
        </div>
      </div>

      <!-- trades table -->
      <div v-if="trades && trades.length > 0" class="section">
        <div class="section-title">交易记录 ({{ trades.length }})</div>
        <div class="trades-table">
          <div class="trade-row trade-header">
            <span class="td-date">日期</span>
            <span class="td-symbol">标的</span>
            <span class="td-dir">方向</span>
            <span class="td-qty">数量</span>
            <span class="td-price">价格</span>
            <span class="td-pnl">盈亏</span>
          </div>
          <div v-for="(t, i) in trades.slice(0, 50)" :key="i" class="trade-row">
            <span class="td-date">{{ t.date || '-' }}</span>
            <span class="td-symbol">{{ t.symbol || '-' }}</span>
            <span class="td-dir" :class="(t.direction || t.side) === 'buy' ? 'buy' : 'sell'">{{ t.direction || t.side || '-' }}</span>
            <span class="td-qty">{{ t.quantity ?? t.qty ?? '-' }}</span>
            <span class="td-price">{{ (t.price ?? '-').toFixed ? t.price.toFixed(2) : t.price ?? '-' }}</span>
            <span class="td-pnl" :class="(t.pnl ?? t.profit ?? 0) > 0 ? 'profit' : (t.pnl ?? t.profit ?? 0) < 0 ? 'loss' : ''">{{ t.pnl != null ? t.pnl.toFixed(2) : t.profit != null ? t.profit.toFixed(2) : '-' }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.bt-panel { padding: 12px; height: 100%; overflow-y: auto; background: var(--color-bg-panel); }
.section { margin-bottom: 16px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px; }
.metrics-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.metric-item { display: flex; justify-content: space-between; padding: 4px 8px; background: var(--color-bg-input); border-radius: var(--radius-sm); }
.metric-label { font-size: 11px; color: var(--color-text-tertiary); }
.metric-value { font-size: 11px; font-weight: 600; color: var(--color-text-primary); font-family: monospace; }
.trades-table { display: flex; flex-direction: column; gap: 1px; }
.trade-row { display: flex; gap: 4px; padding: 3px 6px; font-size: 10px; font-family: monospace; background: var(--color-bg-input); border-radius: 2px; }
.trade-header { font-weight: 600; color: var(--color-text-tertiary); background: transparent; }
.td-date { width: 80px; flex-shrink: 0; }
.td-symbol { width: 60px; flex-shrink: 0; }
.td-dir { width: 40px; flex-shrink: 0; }
.td-qty { width: 50px; flex-shrink: 0; text-align: right; }
.td-price { width: 60px; flex-shrink: 0; text-align: right; }
.td-pnl { width: 70px; flex-shrink: 0; text-align: right; }
.buy { color: #f85149; }
.sell { color: #3fb950; }
.profit { color: #3fb950; }
.loss { color: #f85149; }
.empty-state { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; gap: 8px; }
.empty-icon { font-size: 32px; }
.empty-text { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.empty-desc { font-size: 11px; color: var(--color-text-tertiary); text-align: center; max-width: 280px; }
</style>
