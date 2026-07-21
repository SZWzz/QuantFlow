<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'; import { CanvasRenderer } from 'echarts/renderers'; import { BarChart, LineChart } from 'echarts/charts'; import { TooltipComponent, GridComponent, LegendComponent } from 'echarts/components'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

use([BarChart, LineChart, TooltipComponent, GridComponent, LegendComponent, CanvasRenderer])

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol ?? ctx.getGroupSymbol(pg.groupId) ?? '000001')
const { name } = useStockName(symbol)
const activeView = ref<'dcf' | 'forecast'>('dcf')
const { fetchWithCache } = usePanelCache()
const chartTheme = useChartTheme()
const loading = computed(() => activeView.value === 'dcf' ? dcfLoading.value : fcLoading.value)

// DCF
const result = ref<any>(null); const dcfLoading = ref(false); const dcfError = ref('')
const scenarios = computed(() => result.value?.scenarios || {})
const maxPrice = computed(() => { const vals = [scenarios.value['保守']?.value_per_share, scenarios.value['基准']?.value_per_share, scenarios.value['乐观']?.value_per_share, buySell.value.current_price].filter(v => v != null); return Math.max(...vals) || 1 })
const fcf = computed(() => result.value?.free_cash_flow || 0)
const buySell = computed(() => result.value?.buy_sell_suggestion || {})
const bsColor = computed(() => { if (!buySell.value.suggestion) return ''; return buySell.value.suggestion.includes('持有') ? 'var(--color-accent)' : buySell.value.suggestion.includes('买入') ? 'var(--color-down)' : 'var(--color-up)' })

async function loadDCFData() { dcfLoading.value = true; dcfError.value = ''; try { const app = (window as any).go?.main?.App; if (!app?.GetValuationDCF) return; const { data } = await fetchWithCache<any>(`valuation_dcf:${symbol.value}`, () => app.GetValuationDCF(symbol.value), 3600000); result.value = data || null } catch (e: any) { dcfError.value = e?.message || String(e) } finally { dcfLoading.value = false } }

// Forecast
const fcResult = ref<any>(null); const fcLoading = ref(false); const fcError = ref('')
const forecastTable = computed(() => { const r = fcResult.value; if (!r) return []; return [{ scenario: 'conservative', growth: (r.gr_low ?? -5) + '%', y1_rev: r.rev_base * (1 + (r.gr_low ?? -5) / 100), y2_rev: r.rev_base * (1 + (r.gr_low ?? -5) / 100) ** 2, y1_profit: r.profit_base * (1 + (r.gr_low ?? -5) / 100), y2_profit: r.profit_base * (1 + (r.gr_low ?? -5) / 100) ** 2 }, { scenario: 'baseline', growth: (r.gr_base ?? 10) + '%', y1_rev: r.rev_base * (1 + (r.gr_base ?? 10) / 100), y2_rev: r.rev_base * (1 + (r.gr_base ?? 10) / 100) ** 2, y1_profit: r.profit_base * (1 + (r.gr_base ?? 10) / 100), y2_profit: r.profit_base * (1 + (r.gr_base ?? 10) / 100) ** 2 }, { scenario: 'optimistic', growth: (r.gr_high ?? 20) + '%', y1_rev: r.rev_base * (1 + (r.gr_high ?? 20) / 100), y2_rev: r.rev_base * (1 + (r.gr_high ?? 20) / 100) ** 2, y1_profit: r.profit_base * (1 + (r.gr_high ?? 20) / 100), y2_profit: r.profit_base * (1 + (r.gr_high ?? 20) / 100) ** 2 }] })
const annualPeriods = computed(() => fcResult.value?.annual_periods ?? 0)
const annualRev = computed(() => fcResult.value?.annual_rev ?? 0)
const annualProfit = computed(() => fcResult.value?.annual_profit ?? 0)
const annualMargin = computed(() => { if (!annualRev.value) return 0; return (annualProfit.value / annualRev.value * 100).toFixed(1) })
const avgGrowth = computed(() => fcResult.value?.avg_growth ?? null)
const latestPeriod = computed(() => fcResult.value?.latest_period ?? '')
const isAnnualized = computed(() => fcResult.value?.is_annualized ?? false)
const latestRev = computed(() => fcResult.value?.latest_rev ?? 0)
const latestProfit = computed(() => fcResult.value?.latest_profit ?? 0)

function scenarioLabel(s: string): string { return s === 'conservative' ? '保守' : s === 'baseline' ? '基准' : '乐观' }
function scenarioClass(s: string): string { return `scenario-${s}` }
function calcMargin(profit: number, rev: number): string { return rev ? (profit / rev * 100).toFixed(1) : '--' }

const chartOption = computed(() => { const r = fcResult.value; if (!r) return {}; const pal = chartTheme.palette; return { tooltip: { trigger: 'axis' }, grid: { left: 50, right: 20, top: 10, bottom: 30 }, xAxis: { type: 'category', data: ['营收(亿)', '净利润(亿)'] }, yAxis: { type: 'value' }, series: [{ name: '保守', type: 'bar', data: [r.rev_low / 1e8, r.profit_low / 1e8], itemStyle: { color: pal[2] } }, { name: '基准', type: 'bar', data: [r.rev_base / 1e8, r.profit_base / 1e8], itemStyle: { color: pal[0] } }, { name: '乐观', type: 'bar', data: [r.rev_high / 1e8, r.profit_high / 1e8], itemStyle: { color: pal[1] } }] } })

async function loadForecastData() { fcLoading.value = true; fcError.value = ''; try { const app = (window as any).go?.main?.App; if (!app?.GetFinancialForecast) return; const { data } = await fetchWithCache<any>(`forecast:${symbol.value}`, () => app.GetFinancialForecast(symbol.value), 3600000); fcResult.value = data || null } catch (e: any) { fcError.value = e?.message || String(e) } finally { fcLoading.value = false } }

async function loadData() { if (activeView.value === 'dcf') loadDCFData(); else loadForecastData() }
watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (n) => { if (n && n !== symbol.value) { symbol.value = n; loadData() } })
onMounted(loadDCFData)

const viewTabs = [{ key: 'dcf', label: 'DCF' }, { key: 'forecast', label: '预测' }]
</script>

<template>
  <div class="val-panel">
    <PanelHeader title="估值分析" :tabs="viewTabs" :active-tab="activeView" @tab-change="(k: string) => { activeView = k as 'dcf' | 'forecast'; if (k === 'forecast' && !fcResult) loadForecastData() }">
      <template #controls>
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="btn btn-sm" @click="loadData" :disabled="loading">⟳</button>
      </template>
    </PanelHeader>

    <template v-if="activeView === 'dcf'">
      <LoadingState v-if="dcfLoading && !result" type="card" :rows="2" />
      <EmptyState v-else-if="dcfError" title="估值加载失败" :description="dcfError" />
      <EmptyState v-else-if="!result?.scenarios || !Object.keys(result.scenarios).length" :title="result?.error || '暂无估值数据'" />
      <template v-else>
        <div class="dcf-section">
          <div class="fcf"><span class="lbl">自由现金流</span><span class="val">{{ fcf.toLocaleString() }}</span></div>
          <div v-for="s in ['保守','基准','乐观']" :key="s" class="row"><span class="sn">{{ s }}</span><div class="trk"><div class="fl" :style="{ width: ((scenarios[s]?.value_per_share||0)/maxPrice*100)+'%', background: s==='保守'?chartTheme.palette[3]:s==='乐观'?chartTheme.palette[1]:chartTheme.palette[2] }"/><span class="p">{{ scenarios[s]?.value_per_share?.toFixed(2) || '--' }}</span></div></div>
          <div v-if="buySell.current_price" class="row"><span class="sn cp">当前</span><div class="trk"><div class="fl" :style="{ width: (buySell.current_price/maxPrice*100)+'%', background: 'var(--color-accent)' }"/><span class="p cpv">{{ buySell.current_price?.toFixed(2) }}</span></div></div>
          <div v-if="buySell.suggestion" class="bs"><span class="bsv" :style="{ color: bsColor }">{{ buySell.suggestion }}</span><span class="bsp">空间 {{ buySell.upside_pct>0?'+':'' }}{{ buySell.upside_pct }}%</span><span class="bsf">公允价值 {{ buySell.fair_value?.toFixed(2) }}</span></div>
        </div>
      </template>
    </template>

    <template v-if="activeView === 'forecast'">
      <LoadingState v-if="fcLoading && !fcResult" type="card" :rows="2" />
      <EmptyState v-else-if="fcError" title="预测加载失败" :description="fcError" />
      <EmptyState v-else-if="!forecastTable.length" :title="fcResult?.error || t('research.no_forecast')" />
      <template v-else>
        <div class="metrics-bar"><div class="metric-item"><span class="metric-label">{{ t('research.forecast_latest_rev') }}</span><span class="metric-value">{{ (annualRev / 1e8).toFixed(2) }}<span class="metric-unit">亿</span></span><span class="metric-sub" v-if="latestPeriod">{{ t('research.forecast_scenario') }}: {{ latestPeriod }}<span v-if="isAnnualized" class="metric-tag">年化</span></span></div><div class="metric-item"><span class="metric-label">{{ t('research.forecast_base_profit') }}</span><span class="metric-value">{{ (annualProfit / 1e8).toFixed(2) }}<span class="metric-unit">亿</span></span><span class="metric-sub">{{ annualMargin || '--' }}% {{ t('research.forecast_net_margin') }}</span></div><div class="metric-item" v-if="avgGrowth != null"><span class="metric-label">{{ t('research.forecast_base_growth') }}</span><span class="metric-value" :class="avgGrowth >= 0 ? 'trend-up' : 'trend-down'">{{ avgGrowth >= 0 ? '+' : '' }}{{ avgGrowth }}<span class="metric-unit">%</span></span><span class="metric-sub">{{ t('research.forecast_annual_count', { count: annualPeriods }) }}</span></div></div>
        <div class="context-bar" v-if="latestPeriod && isAnnualized"><span class="context-text">实际累计: {{ latestPeriod }} 营收 {{ (latestRev / 1e8).toFixed(2) }}亿 / 净利润 {{ (latestProfit / 1e8).toFixed(2) }}亿</span></div>
        <div class="chart-wrap"><VChart :option="chartOption" autoresize style="height:180px" /></div>
        <div class="hint">{{ t('research.forecast_hint') }}</div>
        <div class="forecast-table">
          <div class="table-header"><span class="th-cell th-scenario">{{ t('research.forecast_scenario') }}</span><span class="th-cell th-growth">{{ t('research.forecast_growth') }}</span><span class="th-cell th-number">Y1<br>{{ t('research.forecast_y1_rev') }}</span><span class="th-cell th-number">Y2<br>{{ t('research.forecast_y2_rev') }}</span><span class="th-cell th-number">Y1<br>{{ t('research.forecast_y1_profit') }}</span><span class="th-cell th-number">Y2<br>{{ t('research.forecast_y2_profit') }}</span><span class="th-cell th-number">Y1<br>{{ t('research.forecast_net_margin') }}</span><span class="th-cell th-number">Y2<br>{{ t('research.forecast_net_margin') }}</span></div>
          <div v-for="(row, i) in forecastTable" :key="i" class="table-row" :class="[scenarioClass(row.scenario)]"><span class="td-cell th-scenario"><span class="scenario-badge" :class="scenarioClass(row.scenario)">{{ scenarioLabel(row.scenario) }}</span></span><span class="td-cell th-growth">{{ row.growth }}</span><span class="td-cell th-number">{{ (row.y1_rev / 1e8).toFixed(2) }}</span><span class="td-cell th-number">{{ (row.y2_rev / 1e8).toFixed(2) }}</span><span class="td-cell th-number">{{ (row.y1_profit / 1e8).toFixed(2) }}</span><span class="td-cell th-number">{{ (row.y2_profit / 1e8).toFixed(2) }}</span><span class="td-cell th-number">{{ calcMargin(row.y1_profit, row.y1_rev) }}<span class="unit-pct">%</span></span><span class="td-cell th-number">{{ calcMargin(row.y2_profit, row.y2_rev) }}<span class="unit-pct">%</span></span></div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.val-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.symbol-badge { font-size: var(--font-xs); padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); background: var(--color-accent-soft); color: var(--color-accent); font-family: var(--font-mono); }
.dcf-section { flex: 1; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }
.fcf { display: flex; gap: var(--space-sm); align-items: baseline; margin-bottom: var(--space-md); }
.lbl { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.val { font-size: var(--font-xl); font-weight: 700; font-variant-numeric: tabular-nums; }
.row { display: flex; align-items: center; gap: var(--space-sm); margin-bottom: var(--space-sm); }
.sn { width: 32px; font-size: var(--font-xs); color: var(--color-text-secondary); }
.cp { color: var(--color-accent); font-weight: 600; }
.trk { flex: 1; height: 22px; background: var(--color-bg-subtle); border-radius: var(--radius-sm); position: relative; overflow: hidden; }
.fl { height: 100%; border-radius: var(--radius-sm); opacity: 0.5; }
.p { position: absolute; right: var(--space-sm); top: 50%; transform: translateY(-50%); font-size: var(--font-xs); font-weight: 600; }
.cpv { color: var(--color-accent); }
.bs { display: flex; align-items: center; gap: var(--space-md); margin-top: var(--space-md); padding: var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-lg); background: var(--color-bg-elevated); }
.bsv { font-size: var(--font-lg); font-weight: 700; }
.bsp { font-size: var(--font-xs); color: var(--color-text-secondary); }
.bsf { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-left: auto; }
.metrics-bar { display: flex; gap: var(--space-xl); padding: var(--space-sm) var(--panel-padding); flex-wrap: wrap; }
.metric-item { display: flex; flex-direction: column; gap: var(--space-xs); }
.metric-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; }
.metric-value { font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; }
.metric-unit { font-size: var(--font-xs); font-weight: 400; color: var(--color-text-tertiary); margin-left: var(--space-xs); }
.metric-sub { font-size: var(--font-xs); color: var(--color-text-tertiary); display: flex; align-items: center; gap: var(--space-xs); }
.metric-tag { font-size: var(--font-xs); padding: 0 var(--space-xs); border-radius: var(--radius-xs, 2px); background: var(--color-accent-soft); color: var(--color-accent); }
.trend-up { color: var(--color-up); }
.trend-down { color: var(--color-down); }
.context-bar { font-size: var(--font-xs); color: var(--color-text-tertiary); margin: 0 var(--panel-padding) var(--space-sm); padding: var(--space-xs) var(--space-md); background: var(--color-accent-soft); border-radius: var(--radius-sm); border-left: 2px solid var(--color-accent); }
.chart-wrap { flex-shrink: 0; margin: 0 var(--panel-padding) var(--space-sm); background: var(--color-bg-elevated); border-radius: var(--radius-md); padding: var(--space-xs); }
.hint { font-size: var(--font-xs); color: var(--color-text-tertiary); margin: 0 var(--panel-padding) var(--space-md); padding: var(--space-sm) var(--space-md); background: var(--color-bg-subtle); border-radius: var(--radius-sm); border-left: 2px solid var(--color-border-strong); }
.forecast-table { width: 100%; }
.table-header { display: flex; border-bottom: 2px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; line-height: 1.3; }
.table-row { display: flex; border-bottom: 1px solid var(--color-border-subtle); font-size: var(--font-xs); font-variant-numeric: tabular-nums; transition: background 0.15s; }
.table-row:hover { background: var(--color-bg-hover); }
.th-cell { flex: 1; padding: var(--space-xs) var(--space-xs); text-align: right; }
.td-cell { flex: 1; padding: var(--space-sm) var(--space-xs); text-align: right; }
.th-scenario, .td-cell.th-scenario { flex: 0 0 64px; text-align: center; }
.th-growth { flex: 0 0 60px; }
.td-cell.th-growth { flex: 0 0 60px; font-weight: 600; }
.th-number { flex: 0 0 80px; }
.td-cell.th-number { flex: 0 0 80px; font-weight: 600; display: flex; flex-direction: column; align-items: flex-end; justify-content: center; gap: 1px; }
.unit-pct { font-size: var(--font-xs); font-weight: 400; color: var(--color-text-tertiary); }
.scenario-badge { font-size: var(--font-xs); font-weight: 600; padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); white-space: nowrap; }
.scenario-conservative .scenario-badge, .scenario-conservative.scenario-badge { background: var(--color-accent-soft); color: var(--color-accent); }
.scenario-baseline .scenario-badge, .scenario-baseline.scenario-badge { background: var(--color-accent-soft); color: var(--color-accent); }
.scenario-optimistic .scenario-badge, .scenario-optimistic.scenario-badge { background: var(--color-down-soft); color: var(--color-down); }
.table-row.scenario-conservative { background: var(--color-accent-soft); }
.table-row.scenario-baseline { background: var(--color-bg-elevated); }
.table-row.scenario-optimistic { background: var(--color-down-soft); }
</style>
