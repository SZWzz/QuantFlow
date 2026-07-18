<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { useI18n } from 'vue-i18n'

use([BarChart, GridComponent, TooltipComponent, CanvasRenderer])

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()

// ── Tabs ──
type ViewTab = 'dcf' | 'forecast'
const activeView = ref<ViewTab>('dcf')

// ══════ DCF Valuation ══════
const dcfLoading = ref(false)
const dcfError = ref('')
const result = ref<any>(null)

const scenarios = computed(() => result.value?.scenarios || {})
const buySell = computed(() => result.value?.buy_sell || {})
const fcf = computed(() => result.value?.fcf || 0)
const maxPrice = computed(() => {
  const vals = Object.values(scenarios.value).map((s: any) => s.value_per_share || 0)
  return Math.max(...vals, buySell.value?.current_price || 0) * 1.15
})

const bsColor = computed(() => {
  const s = buySell.value?.suggestion
  return s === '买入' ? 'var(--color-down)' : s === '增持' ? '#3b82f6' : s === '减持' ? 'var(--color-up)' : '#f59e0b'
})

async function loadDCFData() {
  dcfLoading.value = true; dcfError.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetValuation) return
    const { data } = await fetchWithCache(
      `valuation:${symbol.value}`,
      async () => {
        const resp = await app.GetValuation(symbol.value)
        return resp?.data ? JSON.parse(resp.data) : resp
      },
    )
    result.value = data
  } catch (e: any) { dcfError.value = e.message || '加载失败' }
  finally { dcfLoading.value = false }
}

// ══════ Forecast ══════
let loadSeq = 0
const fcLoading = ref(false)
const fcError = ref('')
const fcResult = ref<any>(null)
const chartTheme = useChartTheme()

const forecastTable = computed(() => fcResult.value?.forecast_table || [])
const latestPeriod = computed(() => fcResult.value?.latest_period || '')
const periodType = computed(() => fcResult.value?.period_type || '')
const latestRev = computed(() => fcResult.value?.latest_rev || 0)
const latestProfit = computed(() => fcResult.value?.latest_profit || 0)
const annualRev = computed(() => fcResult.value?.annual_rev || 0)
const annualProfit = computed(() => fcResult.value?.annual_profit || 0)
const avgGrowth = computed(() => fcResult.value?.avg_growth ?? null)
const annualPeriods = computed(() => fcResult.value?.annual_periods || 0)

const isAnnualized = computed(() => periodType.value.startsWith('annualized_'))
const annualMargin = computed(() => {
  if (!annualRev.value || !annualProfit.value) return null
  return (annualProfit.value / annualRev.value * 100).toFixed(1)
})

const scenarioLabels: Record<string, string> = {
  '保守': t('research.forecast_scenario_conservative'),
  '基准': t('research.forecast_scenario_baseline'),
  '乐观': t('research.forecast_scenario_optimistic'),
}

const chartOption = computed(() => {
  const rows = forecastTable.value
  if (!rows.length) return {}
  const cats = rows.map((r: any) => scenarioLabels[r.scenario] || r.scenario)
  const baseVal = +(annualRev.value / 1e8).toFixed(2)
  const barColors = ['#6b7280', '#60a5fa', '#fbbf24', '#4ade80']

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: chartTheme.tooltipBg,
      borderColor: 'transparent',
      textStyle: { color: chartTheme.tooltipText, fontSize: 11 },
      formatter: (params: any[]) => {
        const scenario = params[0]?.axisValue || ''
        let html = `<div style="font-weight:600;margin-bottom:4px">${scenario}</div>`
        for (const p of params) {
          html += `<div style="display:flex;justify-content:space-between;gap:12px">
            <span>${p.marker} ${p.seriesName}</span>
            <span style="font-weight:600">${p.value}亿</span>
          </div>`
        }
        return html
      },
    },
    grid: { left: 32, right: 12, top: 8, bottom: 24 },
    xAxis: {
      type: 'category',
      data: cats,
      axisLabel: { color: chartTheme.axisColor, fontSize: 10 },
      axisLine: { lineStyle: { color: chartTheme.splitColor } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      name: '亿',
      nameTextStyle: { color: chartTheme.axisColor, fontSize: 10 },
      splitLine: { lineStyle: { color: chartTheme.splitColor } },
      axisLabel: { color: chartTheme.axisColor, fontSize: 10 },
    },
    series: [
      {
        name: '基准年',
        type: 'bar',
        barWidth: 14,
        barGap: '20%',
        data: cats.map(() => baseVal),
        itemStyle: { color: barColors[0], borderRadius: [2, 2, 0, 0] },
        label: {
          show: true,
          position: 'top',
          color: chartTheme.textColor,
          fontSize: 9,
          formatter: (p: any) => p.value + '亿',
        },
      },
      {
        name: 'Y1',
        type: 'bar',
        barWidth: 14,
        data: rows.map((r: any) => +(r.y1_rev / 1e8).toFixed(2)),
        itemStyle: { color: barColors[1], borderRadius: [2, 2, 0, 0] },
      },
      {
        name: 'Y2',
        type: 'bar',
        barWidth: 14,
        data: rows.map((r: any) => +(r.y2_rev / 1e8).toFixed(2)),
        itemStyle: { color: barColors[2], borderRadius: [2, 2, 0, 0] },
      },
    ],
  }
})

function scenarioLabel(scenario: string): string {
  return scenarioLabels[scenario] || scenario
}

function scenarioClass(scenario: string): string {
  if (scenario === '保守') return 'scenario-conservative'
  if (scenario === '基准') return 'scenario-baseline'
  if (scenario === '乐观') return 'scenario-optimistic'
  return ''
}

function calcMargin(profit: number, rev: number): string {
  if (!rev || !profit) return '--'
  return (profit / rev * 100).toFixed(1)
}

async function loadForecastData() {
  const seq = ++loadSeq
  fcLoading.value = true
  fcError.value = ''
  try {
    const { data } = await fetchWithCache<any>(`forecast:${symbol.value}`, () => (window as any).go?.main?.App?.GetForecast(symbol.value), 30 * 60 * 1000)
    if (seq !== loadSeq) return
    fcResult.value = data?.data ? JSON.parse(data.data) : data
  } catch (e: any) {
    fcError.value = e.message || t('common.panel_error')
  } finally {
    fcLoading.value = false
  }
}

const loading = computed(() => activeView.value === 'dcf' ? dcfLoading.value : fcLoading.value)

function loadData() {
  if (activeView.value === 'dcf') loadDCFData()
  else loadForecastData()
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) {
    symbol.value = newSym
    loadData()
  }
})
onMounted(loadDCFData)
</script>

<template>
  <div class="val-panel">
    <div class="panel-header">
      <h3>估值分析</h3>
      <!-- View tabs -->
      <div class="view-tabs">
        <button :class="['view-tab', { active: activeView === 'dcf' }]" @click="activeView = 'dcf'">DCF</button>
        <button :class="['view-tab', { active: activeView === 'forecast' }]" @click="activeView = 'forecast'; if (!fcResult) loadForecastData()">预测</button>
      </div>
      <div class="header-right">
        <span class="s">{{ symbol }} {{ name }}</span>
        <button class="r" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <!-- ── DCF ── -->
    <template v-if="activeView === 'dcf'">
      <SkeletonPanel v-if="dcfLoading && !result" type="card" :rows="2" />
      <div v-else-if="dcfError" class="st err">{{ dcfError }}</div>
      <div v-else-if="!result?.scenarios || !Object.keys(result.scenarios).length" class="st">{{ result?.error || '暂无估值数据' }}</div>
      <template v-else>
        <div class="fcf"><span class="lbl">自由现金流</span><span class="val">{{ fcf.toLocaleString() }}</span></div>
        <div v-for="s in ['保守','基准','乐观']" :key="s" class="row"><span class="sn">{{ s }}</span><div class="trk"><div class="fl" :style="{ width: ((scenarios[s]?.value_per_share||0)/maxPrice*100)+'%', background: s==='保守'?'#ef4444':s==='乐观'?'#22c55e':'#f59e0b' }"/><span class="p">{{ scenarios[s]?.value_per_share?.toFixed(2) || '--' }}</span></div></div>
        <div v-if="buySell.current_price" class="row"><span class="sn cp">当前</span><div class="trk"><div class="fl" :style="{ width: (buySell.current_price/maxPrice*100)+'%', background: 'var(--color-accent)' }"/><span class="p cpv">{{ buySell.current_price?.toFixed(2) }}</span></div></div>
        <div v-if="buySell.suggestion" class="bs"><span class="bsv" :style="{ color: bsColor }">{{ buySell.suggestion }}</span><span class="bsp">空间 {{ buySell.upside_pct>0?'+':'' }}{{ buySell.upside_pct }}%</span><span class="bsf">公允价值 {{ buySell.fair_value?.toFixed(2) }}</span></div>
      </template>
    </template>

    <!-- ── Forecast ── -->
    <template v-if="activeView === 'forecast'">
      <SkeletonPanel v-if="fcLoading && !fcResult" type="card" :rows="2" />
      <div v-else-if="fcError" class="st err">{{ fcError }}</div>
      <div v-else-if="!forecastTable.length" class="st">{{ fcResult?.error || t('research.no_forecast') }}</div>

      <template v-else>
        <!-- Baseline metrics -->
        <div class="metrics-bar">
          <div class="metric-item">
            <span class="metric-label">{{ t('research.forecast_latest_rev') }}</span>
            <span class="metric-value">{{ (annualRev / 1e8).toFixed(2) }}<span class="metric-unit">亿</span></span>
            <span class="metric-sub" v-if="latestPeriod">
              {{ t('research.forecast_scenario') }}: {{ latestPeriod }}
              <span v-if="isAnnualized" class="metric-tag">年化</span>
            </span>
          </div>
          <div class="metric-item">
            <span class="metric-label">{{ t('research.forecast_base_profit') }}</span>
            <span class="metric-value">{{ (annualProfit / 1e8).toFixed(2) }}<span class="metric-unit">亿</span></span>
            <span class="metric-sub">{{ annualMargin || '--' }}% {{ t('research.forecast_net_margin') }}</span>
          </div>
          <div class="metric-item" v-if="avgGrowth != null">
            <span class="metric-label">{{ t('research.forecast_base_growth') }}</span>
            <span class="metric-value" :class="avgGrowth >= 0 ? 'trend-up' : 'trend-down'">
              {{ avgGrowth >= 0 ? '+' : '' }}{{ avgGrowth }}<span class="metric-unit">%</span>
            </span>
            <span class="metric-sub">{{ t('research.forecast_annual_count', { count: annualPeriods }) }}</span>
          </div>
        </div>

        <!-- Actual revenue context -->
        <div class="context-bar" v-if="latestPeriod && isAnnualized">
          <span class="context-text">
            实际累计: {{ latestPeriod }} 营收 {{ (latestRev / 1e8).toFixed(2) }}亿 / 净利润 {{ (latestProfit / 1e8).toFixed(2) }}亿
          </span>
        </div>

        <!-- Chart -->
        <div class="chart-wrap">
          <VChart :option="chartOption" autoresize style="height:180px" />
        </div>

        <!-- Hint -->
        <div class="hint">{{ t('research.forecast_hint') }}</div>

        <!-- Forecast table -->
        <div class="forecast-table">
          <div class="table-header">
            <span class="th-cell th-scenario">{{ t('research.forecast_scenario') }}</span>
            <span class="th-cell th-growth">{{ t('research.forecast_growth') }}</span>
            <span class="th-cell th-number">Y1<br>{{ t('research.forecast_y1_rev') }}</span>
            <span class="th-cell th-number">Y2<br>{{ t('research.forecast_y2_rev') }}</span>
            <span class="th-cell th-number">Y1<br>{{ t('research.forecast_y1_profit') }}</span>
            <span class="th-cell th-number">Y2<br>{{ t('research.forecast_y2_profit') }}</span>
            <span class="th-cell th-number">Y1<br>{{ t('research.forecast_net_margin') }}</span>
            <span class="th-cell th-number">Y2<br>{{ t('research.forecast_net_margin') }}</span>
          </div>
          <div
            v-for="(row, i) in forecastTable"
            :key="i"
            class="table-row"
            :class="[scenarioClass(row.scenario)]"
          >
            <span class="td-cell th-scenario">
              <span class="scenario-badge" :class="scenarioClass(row.scenario)">{{ scenarioLabel(row.scenario) }}</span>
            </span>
            <span class="td-cell th-growth">{{ row.growth }}</span>
            <span class="td-cell th-number">{{ (row.y1_rev / 1e8).toFixed(2) }}</span>
            <span class="td-cell th-number">{{ (row.y2_rev / 1e8).toFixed(2) }}</span>
            <span class="td-cell th-number">{{ (row.y1_profit / 1e8).toFixed(2) }}</span>
            <span class="td-cell th-number">{{ (row.y2_profit / 1e8).toFixed(2) }}</span>
            <span class="td-cell th-number">{{ calcMargin(row.y1_profit, row.y1_rev) }}<span class="unit-pct">%</span></span>
            <span class="td-cell th-number">{{ calcMargin(row.y2_profit, row.y2_rev) }}<span class="unit-pct">%</span></span>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.val-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: auto; }

.header-right { display: flex; align-items: center; gap: 8px; margin-left: auto; }

/* View tabs */
.view-tabs { display: flex; gap: 0; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); overflow: hidden; }
.view-tab { padding: 3px 12px; border: none; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px; }
.view-tab + .view-tab { border-left: 1px solid var(--color-border-strong); }
.view-tab.active { color: var(--color-accent); background: rgba(59,130,246,0.1); }

.s { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(99,102,241,0.15); color: var(--color-accent); font-family: 'JetBrains Mono', monospace; }
.r { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.r:disabled { opacity: 0.5; cursor: not-allowed; }
.st { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.err { color: var(--color-error); }

/* ── DCF ── */
.fcf { display: flex; gap: 8px; align-items: baseline; margin-bottom: 10px; }
.lbl { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.val { font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
.row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.sn { width: 32px; font-size: 11px; color: var(--color-text-secondary); }
.cp { color: var(--color-accent); font-weight: 600; }
.trk { flex: 1; height: 22px; background: var(--color-bg-subtle); border-radius: var(--radius-sm); position: relative; overflow: hidden; }
.fl { height: 100%; border-radius: var(--radius-sm); opacity: 0.5; }
.p { position: absolute; right: 8px; top: 50%; transform: translateY(-50%); font-size: 11px; font-weight: 600; }
.cpv { color: var(--color-accent); }
.bs { display: flex; align-items: center; gap: 12px; margin-top: 12px; padding: 10px 14px; border: 1px solid var(--color-border); border-radius: var(--radius-lg); background: var(--color-bg-elevated); }
.bsv { font-size: 16px; font-weight: 700; }
.bsp { font-size: 12px; color: var(--color-text-secondary); }
.bsf { font-size: 12px; color: var(--color-text-tertiary); margin-left: auto; }

/* ── Forecast ── */
.metrics-bar { display: flex; gap: 24px; margin-bottom: 6px; flex-wrap: wrap; }
.metric-item { display: flex; flex-direction: column; gap: 2px; }
.metric-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; }
.metric-value { font-size: 18px; font-weight: 700; font-variant-numeric: tabular-nums; }
.metric-unit { font-size: 11px; font-weight: 400; color: var(--color-text-tertiary); margin-left: 2px; }
.metric-sub { font-size: 10px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; }
.metric-tag { font-size: 9px; padding: 0 4px; border-radius: 2px; background: rgba(245, 158, 11, 0.15); color: var(--color-accent); }
.trend-up { color: var(--color-up); }
.trend-down { color: var(--color-down); }

.context-bar { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 6px; padding: 4px 10px; background: rgba(245, 158, 11, 0.06); border-radius: var(--radius-sm); border-left: 2px solid var(--color-accent); }

.chart-wrap { flex-shrink: 0; margin-bottom: 8px; background: var(--color-bg-elevated); border-radius: var(--radius-md); padding: 4px; }

.hint { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 10px; padding: 6px 10px; background: rgba(255, 255, 255, 0.04); border-radius: var(--radius-sm); border-left: 2px solid var(--color-border-strong); }

.forecast-table { width: 100%; }
.table-header { display: flex; border-bottom: 2px solid var(--color-border-strong); font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; line-height: 1.3; }
.table-row { display: flex; border-bottom: 1px solid var(--color-border-subtle); font-size: 12px; font-variant-numeric: tabular-nums; transition: background 0.15s; }
.table-row:hover { background: var(--color-bg-elevated); }
.th-cell { flex: 1; padding: 4px 2px; text-align: right; }
.td-cell { flex: 1; padding: 6px 2px; text-align: right; }
.th-scenario, .td-cell.th-scenario { flex: 0 0 64px; text-align: center; }
.th-growth { flex: 0 0 60px; }
.td-cell.th-growth { flex: 0 0 60px; font-weight: 600; }
.th-number { flex: 0 0 80px; }
.td-cell.th-number { flex: 0 0 80px; font-weight: 600; display: flex; flex-direction: column; align-items: flex-end; justify-content: center; gap: 1px; }
.unit-pct { font-size: 10px; font-weight: 400; color: var(--color-text-tertiary); }

.scenario-badge { font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-sm); white-space: nowrap; }
.scenario-conservative .scenario-badge, .scenario-conservative.scenario-badge { background: rgba(245, 158, 11, 0.15); color: var(--color-accent); }
.scenario-baseline .scenario-badge, .scenario-baseline.scenario-badge { background: rgba(59, 130, 246, 0.15); color: var(--color-accent); }
.scenario-optimistic .scenario-badge, .scenario-optimistic.scenario-badge { background: rgba(34, 197, 94, 0.15); color: var(--color-down); }
.table-row.scenario-conservative { background: rgba(245, 158, 11, 0.03); }
.table-row.scenario-baseline { background: rgba(59, 130, 246, 0.03); }
.table-row.scenario-optimistic { background: rgba(34, 197, 94, 0.03); }
</style>
