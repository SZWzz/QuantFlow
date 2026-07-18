<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import VChart from 'vue-echarts'
import { PanelTabs } from '@/terminal/components/panel'
import 'echarts'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const audit = ref<any>(null)
const analysis = ref<any>(null)
const activeTab = ref<'audit' | 'delist'>('audit')
const showBreakdown = ref(true)
const showTrend = ref(false)
const showHistory = ref(false)
const reportType = ref<'annual' | 'quarterly'>('annual')
const delisting = ref<Record<string, any> | null>(null)
const delistingLoading = ref(false)
const delistingError = ref('')
const chartTheme = useChartTheme()

const findings = computed(() => audit.value?.findings || [])
const riskScore = computed(() => audit.value?.risk_score ?? 0)
const riskGrade = computed(() => audit.value?.risk_grade ?? '--')
const highCount = computed(() => audit.value?.high_count ?? 0)
const mediumCount = computed(() => audit.value?.medium_count ?? 0)

const healthScore = computed(() => analysis.value?.health_score ?? null)
const healthGrade = computed(() => analysis.value?.health_grade ?? '--')
const breakdown = computed(() => analysis.value?.score_breakdown || [])
const periods = computed(() => analysis.value?.periods || [])
const metrics = computed(() => analysis.value?.metrics || {})
const anomalyFlags = computed(() => analysis.value?.anomaly_flags || [])

function riskColor(grade: string): string {
  if (grade.includes('高')) return 'var(--color-up)'
  if (grade.includes('中')) return '#f59e0b'
  return 'var(--color-down)'
}

function healthColor(score: number | null): string {
  if (score === null) return 'var(--color-text-tertiary)'
  if (score >= 80) return 'var(--color-down)'
  if (score >= 60) return 'var(--color-accent)'
  if (score >= 40) return '#f59e0b'
  return 'var(--color-up)'
}

function levelIcon(level: string): string {
  return level === 'high' ? '🔴' : level === 'medium' ? '🟡' : '🟢'
}

function toNum(v: any): number {
  if (v == null) return NaN
  if (typeof v === 'number') return v
  const n = parseFloat(String(v).replace(/[^0-9.\-]/g, ''))
  return isNaN(n) ? NaN : n
}

function formatPct(v: any): string {
  const n = toNum(v)
  if (isNaN(n)) return '--'
  return n.toFixed(1) + '%'
}

function formatChange(v: any): string {
  const n = toNum(v)
  if (isNaN(n)) return '--'
  return (n >= 0 ? '+' : '') + n.toFixed(1) + '%'
}

function formatNum(v: any, unit: string = ''): string {
  const n = toNum(v)
  if (isNaN(n)) return '--'
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿' + unit
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(1) + '万' + unit
  return n.toFixed(2) + unit
}

async function loadData() {
  loading.value = true
  error.value = ''
  const app = (window as any).go?.main?.App
  if (!app?.GetAuditFindings) { loading.value = false; return }

  try {
    const { data: auditData } = await fetchWithCache(
      `audit:${symbol.value}`,
      async () => {
        const resp = await app.GetAuditFindings(symbol.value)
        return resp?.data ? JSON.parse(resp.data) : resp
      },
    )
    audit.value = auditData
  } catch (e: any) {
    error.value = e.message || '审计数据加载失败'
  }

  try {
    if (app.GetFinancialAnalysis) {
      const { data: analysisData } = await fetchWithCache(
        `analysis:${symbol.value}`,
        async () => {
          const resp = await app.GetFinancialAnalysis(symbol.value)
          return resp?.data ? JSON.parse(resp.data) : resp
        },
        10 * 60 * 1000,
      )
      analysis.value = analysisData
    }
  } catch {
    // analysis is optional
  }

  loading.value = false
}

async function loadDelistingRisk() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDelistingRisk) return
  delisting.value = null
  delistingError.value = ''
  delistingLoading.value = true
  try {
    const { data: delistData } = await fetchWithCache<Record<string, any>>(
      `delist:${symbol.value}`,
      () => app.GetDelistingRisk(symbol.value),
      10 * 60 * 1000,
    )
    delisting.value = delistData
  } catch (e: any) {
    delistingError.value = e.message || '退市风险数据加载失败'
  }
  delistingLoading.value = false
}

const latestPeriodPeriod = computed(() => latestPeriod.value?.period || '')

const allFindings = computed(() => {
  const items: any[] = []
  for (const f of findings.value) {
    items.push({ ...f, source: 'audit' })
  }
  const lp = latestPeriodPeriod.value
  for (const f of anomalyFlags.value) {
    items.push({
      metric: f.type,
      level: f.level,
      value: f.count != null ? `持续${f.count}期` : '',
      threshold: '',
      detail: f.detail,
      source: 'analysis',
      period: f.period,
      isLatest: f.period === lp,
    })
  }
  return items
})

const latestFindings = computed(() => allFindings.value.filter(f => f.source === 'audit' || f.isLatest))
const trendFindings = computed(() => allFindings.value.filter(f => f.source !== 'audit' && !f.isLatest))

function growthRate(): number | null {
  const ps = filteredPeriods.value
  if (ps.length < 2) return null
  const latest = ps[0]
  if (!latest?.period || !latest.revenue) return null
  // Match same period type (month-day) across years → 同比
  const latestMD = latest.period.slice(5) // "2026-03-31" → "03-31"
  for (let i = 1; i < ps.length; i++) {
    const p = ps[i]
    if (!p?.period || !p.revenue) continue
    if (p.period.slice(5) === latestMD) {
      return ((latest.revenue - p.revenue) / p.revenue) * 100
    }
  }
  return null
}

function fmtAxis(v: any): string {
  const n = parseFloat(v)
  if (isNaN(n)) return '--'
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(1) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(0) + '万'
  return n.toFixed(0)
}

const latestPeriod = computed(() => {
  const p = filteredPeriods.value[0]
  if (!p) return {}
  const rev = +(p.revenue || 0)
  const profit = +(p.net_profit || 0)
  return {
    ...p,
    profit_margin: p.profit_margin ?? (rev ? (profit / rev) * 100 : undefined),
  }
})

const filteredPeriods = computed(() => {
  const ps = periods.value
  if (!ps || ps.length === 0) return []
  if (reportType.value === 'annual') {
    return ps.filter((p: any) => p.period && p.period.endsWith('12-31'))
  }
  return ps.filter((p: any) => p.period && !p.period.endsWith('12-31'))
})

const chartOption = computed(() => {
  const ps = filteredPeriods.value
  if (!ps || ps.length < 2) return {}
  const dates = ps.map((p: any) => (p.period || '').slice(0, 7)).reverse()
  const revenue = ps.map((p: any) => +(p.revenue || 0)).reverse()
  const profit = ps.map((p: any) => +(p.net_profit || 0)).reverse()
  const roe = ps.map((p: any) => +((p.roe || 0) * 1)).reverse()
  const gross = ps.map((p: any) => +((p.gross_margin || 0) * 1)).reverse()
  const debt = ps.map((p: any) => +((p.debt_ratio || 0) * 1)).reverse()
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartTheme.tooltipBg,
      borderColor: 'transparent',
      textStyle: { color: chartTheme.tooltipText, fontSize: 11 },
    },
    legend: {
      data: ['营收', '净利润', 'ROE', '毛利率', '负债率'],
      textStyle: { color: chartTheme.axisColor, fontSize: 10 },
      top: 0,
      right: 0,
    },
    grid: { left: 50, right: 50, top: 24, bottom: 24 },
    xAxis: {
      type: 'category',
      data: dates,
      axisLabel: { color: chartTheme.axisColor, fontSize: 10 },
      axisLine: { lineStyle: { color: chartTheme.splitColor } },
      axisTick: { show: false },
    },
    yAxis: [
      {
        type: 'value',
        name: '营收/净利润',
        nameTextStyle: { color: chartTheme.axisColor, fontSize: 10 },
        splitLine: { lineStyle: { color: chartTheme.splitColor } },
        axisLabel: { color: chartTheme.axisColor, fontSize: 9, formatter: (v: number) => fmtAxis(v) },
      },
      {
        type: 'value',
        name: '%',
        min: -100,
        max: 100,
        nameTextStyle: { color: chartTheme.axisColor, fontSize: 10 },
        splitLine: { show: false },
        axisLabel: { color: chartTheme.axisColor, fontSize: 9, formatter: '{value}%' },
      },
    ],
    series: [
      {
        name: '营收', type: 'line', data: revenue, smooth: true,
        yAxisIndex: 0, lineStyle: { color: '#3b82f6', width: 2 },
        itemStyle: { color: '#3b82f6' }, showSymbol: false,
      },
      {
        name: '净利润', type: 'line', data: profit, smooth: true,
        yAxisIndex: 0, lineStyle: { color: '#22c55e', width: 2 },
        itemStyle: { color: '#22c55e' }, showSymbol: false,
      },
      {
        name: 'ROE', type: 'line', data: roe, smooth: true,
        yAxisIndex: 1, lineStyle: { color: '#f59e0b', width: 2 },
        itemStyle: { color: '#f59e0b' }, showSymbol: false,
      },
      {
        name: '毛利率', type: 'line', data: gross, smooth: true,
        yAxisIndex: 1, lineStyle: { color: '#8b5cf6', width: 2 },
        itemStyle: { color: '#8b5cf6' }, showSymbol: false,
      },
      {
        name: '负债率', type: 'line', data: debt, smooth: true,
        yAxisIndex: 1, lineStyle: { color: '#ef4444', width: 2 },
        itemStyle: { color: '#ef4444' }, showSymbol: false,
      },
    ],
  }
})

watch(symbol, () => { loadData(); loadDelistingRisk() })
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (n) => {
  if (n && n !== symbol.value) { symbol.value = n; loadData(); loadDelistingRisk() }
})
watch(activeTab, (tab) => {
  if (tab === 'delist') loadDelistingRisk()
})
onMounted(() => {
  loadData()
  loadDelistingRisk()
})
</script>

<template>
  <div class="ap">
    <div class="h">
      <h3>财务审计</h3>
      <div class="hr">
        <span class="s">{{ symbol }} {{ name }}</span>
        <button class="r" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <!-- Tab bar -->
    <PanelTabs
      variant="pill"
      :tabs="[{ key: 'audit', label: '审计异常' }, { key: 'delist', label: '退市风险' }]"
      :active="activeTab"
      @change="(k: string) => activeTab = k as 'audit' | 'delist'"
    />

    <template v-if="activeTab === 'audit'">
    <SkeletonPanel v-if="loading && !audit" type="card" :rows="3" />
    <div v-else-if="error && !audit" class="st err">{{ error }}</div>
    <template v-else>
      <!-- Risk Gauges -->
      <div class="gauges">
        <div class="gauge-card">
          <div class="gauge-label">风险等级</div>
          <div class="gauge-row">
            <div class="gauge-bar">
              <div class="gauge-fill" :style="{ width: Math.min(riskScore * 8, 100) + '%', background: riskColor(riskGrade) }" />
            </div>
            <span class="gauge-val" :style="{ color: riskColor(riskGrade) }">{{ riskGrade }}</span>
          </div>
          <div class="gauge-meta">
            <span>评分 {{ riskScore }}</span>
            <span v-if="highCount">高危 {{ highCount }} 项</span>
            <span v-if="mediumCount">中危 {{ mediumCount }} 项</span>
          </div>
        </div>
        <div v-if="healthScore !== null" class="gauge-card">
          <div class="gauge-label">财务健康</div>
          <div class="gauge-row">
            <div class="gauge-bar">
              <div class="gauge-fill" :style="{ width: healthScore + '%', background: healthColor(healthScore) }" />
            </div>
            <span class="gauge-val" :style="{ color: healthColor(healthScore) }">{{ healthGrade }}</span>
          </div>
          <div class="gauge-meta">
            <span>评分 {{ healthScore }}/100</span>
            <span v-if="breakdown.length">明细 {{ breakdown.length }} 项</span>
          </div>
        </div>
      </div>

      <!-- KPI Cards -->
      <div class="kpis">
        <div class="kpi">
          <span class="kpi-label">ROE</span>
          <span class="kpi-val" :style="{ color: (latestPeriod.roe ?? 0) > 8 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatPct(latestPeriod.roe) }}</span>
        </div>
        <div class="kpi">
          <span class="kpi-label">负债率</span>
          <span class="kpi-val" :style="{ color: (latestPeriod.debt_ratio ?? 100) < 60 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatPct(latestPeriod.debt_ratio) }}</span>
        </div>
        <div class="kpi">
          <span class="kpi-label">净利率</span>
          <span class="kpi-val" :style="{ color: (latestPeriod.profit_margin ?? 0) > 10 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatPct(latestPeriod.profit_margin) }}</span>
        </div>
        <div class="kpi">
          <span class="kpi-label">毛利率</span>
          <span class="kpi-val" :style="{ color: 'var(--color-accent)' }">{{ formatPct(latestPeriod.gross_margin) }}</span>
        </div>
        <div class="kpi">
          <span class="kpi-label">营收增长</span>
          <span class="kpi-val" :style="{ color: (growthRate() ?? 0) > 0 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatChange(growthRate()) }}</span>
        </div>
        <div class="kpi">
          <span class="kpi-label">商誉/净资产</span>
          <span class="kpi-val" :style="{ color: '#f59e0b' }">{{ findings.find((f: any) => f.metric.includes('商誉'))?.value || '--' }}</span>
        </div>
      </div>

      <!-- Report Type Toggle -->
      <div v-if="periods.length >= 2" class="report-toggle">
        <button :class="{ active: reportType === 'annual' }" @click="reportType = 'annual'">年报</button>
        <button :class="{ active: reportType === 'quarterly' }" @click="reportType = 'quarterly'">季报</button>
      </div>

      <!-- ECharts Trend -->
      <div v-if="filteredPeriods.length >= 2" class="section chart-section">
        <VChart :option="chartOption" autoresize style="height: 200px" />
      </div>
      <div v-else-if="periods.length >= 2 && filteredPeriods.length < 2" class="section chart-section" style="text-align:center;padding:20px;color:var(--color-text-tertiary);font-size:12px">
        暂无足够{{ reportType === 'annual' ? '年报' : '季报' }}数据
      </div>

      <!-- Score Breakdown (collapsible) -->
      <div class="section">
        <div class="section-h" @click="showBreakdown = !showBreakdown">
          <span class="section-title">评分明细</span>
          <span class="section-toggle">{{ showBreakdown ? '收起' : '展开' }}</span>
        </div>
        <div v-if="showBreakdown && breakdown.length" class="breakdown-list">
          <div v-for="(b, i) in breakdown" :key="i" class="br-item">
            <span class="br-name">{{ b.item }}</span>
            <span class="br-effect" :style="{ color: (b.effect || 0) >= 0 ? 'var(--color-down)' : 'var(--color-up)' }">{{ (b.effect || 0) >= 0 ? '+' : '' }}{{ b.effect }}</span>
            <span class="br-detail">{{ b.detail }}</span>
          </div>
          <div class="br-total">
            <span class="br-name">总分</span>
            <span class="br-effect" :style="{ color: healthColor(healthScore) }">{{ healthScore }}/100</span>
            <span class="br-detail">{{ healthGrade }}</span>
          </div>
        </div>
        <div v-else-if="!breakdown.length && !loading" class="section-empty">暂无评分明细</div>
      </div>

      <!-- Findings: Latest -->
      <div class="section">
        <div class="section-h">
          <span class="section-title">异常发现</span>
          <span class="section-count">{{ latestFindings.length + trendFindings.length }}</span>
        </div>
        <div v-if="!latestFindings.length && !trendFindings.length && !loading" class="section-empty">暂无异常发现</div>
        <div v-for="(f, i) in latestFindings" :key="'l'+i" class="finding" :class="f.level">
          <span class="finding-icon">{{ levelIcon(f.level) }}</span>
          <div class="finding-body">
            <div class="finding-head">
              <span class="finding-metric">{{ f.metric }}</span>
              <span v-if="f.value" class="finding-value" :class="f.level">{{ f.value }}</span>
            </div>
            <div class="finding-detail">{{ f.detail }}</div>
            <div v-if="f.threshold" class="finding-threshold">{{ f.threshold }}</div>
          </div>
        </div>
        <!-- Trend Findings (collapsible) -->
        <div v-if="trendFindings.length" class="trend-section">
          <div class="section-h trend-h" @click="showTrend = !showTrend">
            <span class="section-title">趋势发现 ({{ trendFindings.length }})</span>
            <span class="section-toggle">{{ showTrend ? '收起' : '展开' }}</span>
          </div>
          <div v-if="showTrend">
            <div v-for="(f, i) in trendFindings" :key="'t'+i" class="finding" :class="f.level">
              <span class="finding-icon">{{ levelIcon(f.level) }}</span>
              <div class="finding-body">
                <div class="finding-head">
                  <span class="finding-metric">{{ f.metric }}</span>
                  <span v-if="f.value" class="finding-value" :class="f.level">{{ f.value }}</span>
                </div>
                <div class="finding-detail">{{ f.detail }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Financial History (collapsible) -->
      <div class="section">
        <div class="section-h" @click="showHistory = !showHistory">
          <span class="section-title">财务历史 ({{ filteredPeriods.length }} 期 {{ reportType === 'annual' ? '年报' : '季报' }})</span>
          <span class="section-toggle">{{ showHistory ? '收起' : '展开' }}</span>
        </div>
        <div v-if="showHistory && filteredPeriods.length" class="hist-table-wrap">
          <table class="hist-table">
            <thead>
              <tr>
                <th>报告期</th>
                <th class="num">营收</th>
                <th class="num">净利润</th>
                <th class="num">ROE</th>
                <th class="num">负债率</th>
                <th class="num">毛利率</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(p, i) in filteredPeriods" :key="i">
                <td class="period">{{ p.period }}</td>
                <td class="num">{{ formatNum(p.revenue) }}</td>
                <td class="num">{{ formatNum(p.net_profit) }}</td>
                <td class="num" :style="{ color: (p.roe ?? 0) > 0 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatPct(p.roe) }}</td>
                <td class="num" :style="{ color: (p.debt_ratio ?? 0) < 60 ? 'var(--color-down)' : 'var(--color-up)' }">{{ formatPct(p.debt_ratio) }}</td>
                <td class="num">{{ formatPct(p.gross_margin) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else-if="!periods.length && !loading" class="section-empty">暂无财务数据</div>
      </div>
    </template>
    </template>

    <template v-if="activeTab === 'delist'">
      <SkeletonPanel v-if="delistingLoading && !delisting" type="card" :rows="4" />
      <div v-else-if="delistingError && !delisting" class="st err">{{ delistingError }}</div>
      <template v-else-if="delisting">
        <!-- Overall Risk Badge -->
        <div class="dr-overall">
          <div class="dr-badge" :class="'dr-' + delisting.overall_risk">
            <span class="dr-badge-label">{{ delisting.overall_risk === 'high' ? '高风险' : delisting.overall_risk === 'medium' ? '中风险' : '低风险' }}</span>
            <span class="dr-board">{{ delisting.market }} · {{ delisting.board }}</span>
            <span v-if="delisting.is_st" class="st-tag">ST</span>
          </div>
          <p class="dr-summary">{{ delisting.summary }}</p>
        </div>

        <!-- Category Cards -->
        <div v-for="cat in delisting.categories" :key="cat.name" class="dr-category">
          <div class="dr-cat-h">
            <span class="dr-cat-dot" :class="'dot-' + cat.level"></span>
            <span class="dr-cat-name">{{ cat.name }}</span>
          </div>
          <div class="dr-items">
            <div v-for="item in cat.items" :key="item.indicator" class="dr-item" :class="'dr-item-' + item.status">
              <div class="dr-item-left">
                <span class="dr-dot" :class="'dot-' + (item.status === 'danger' ? 'red' : item.status === 'warn' ? 'yellow' : 'green')"></span>
                <span class="dr-indicator">{{ item.indicator }}</span>
              </div>
              <div class="dr-item-right">
                <span class="dr-current">{{ item.current }}</span>
                <span class="dr-threshold">阈值: {{ item.threshold }}</span>
              </div>
              <div v-if="item.detail" class="dr-detail">{{ item.detail }}</div>
            </div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.ap { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow-y: auto; gap: 10px; }
.h { display: flex; justify-content: space-between; align-items: center; flex-shrink: 0; }
.h h3 { margin: 0; font-size: 14px; font-weight: 600; }
.hr { display: flex; align-items: center; gap: 8px; }
.s { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: rgba(239,68,68,0.15); color: var(--color-up); font-family: 'JetBrains Mono', monospace; }
.r { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.r:disabled { opacity: 0.5; cursor: not-allowed; }
.st { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.err { color: var(--color-danger); }

/* Gauges */
.gauges { display: flex; gap: 10px; flex-shrink: 0; }
.gauge-card { flex: 1; padding: 10px 12px; border-radius: var(--radius-lg); background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); }
.gauge-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 6px; }
.gauge-row { display: flex; align-items: center; gap: 10px; }
.gauge-bar { flex: 1; height: 8px; border-radius: var(--radius-sm); background: var(--color-border-strong); overflow: hidden; }
.gauge-fill { height: 100%; border-radius: var(--radius-sm); transition: width 0.5s; }
.gauge-val { font-size: 14px; font-weight: 700; white-space: nowrap; }
.gauge-meta { display: flex; gap: 12px; margin-top: 4px; font-size: 10px; color: var(--color-text-tertiary); }

/* KPI Cards */
.kpis { display: flex; gap: 6px; flex-wrap: wrap; flex-shrink: 0; }
.kpi { flex: 1; min-width: 80px; padding: 8px 10px; border-radius: var(--radius-md); background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong); display: flex; flex-direction: column; gap: 2px; }
.kpi-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; }
.kpi-val { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; }

/* Sections */
.section { flex-shrink: 0; }
.section-h { display: flex; justify-content: space-between; align-items: center; cursor: pointer; padding: 4px 0; }
.section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); }
.section-toggle { font-size: 10px; color: var(--color-text-tertiary); }
.section-count { font-size: 10px; padding: 1px 6px; border-radius: var(--radius-lg); background: var(--color-border-strong); color: var(--color-text-tertiary); }
.section-empty { font-size: 11px; color: var(--color-text-tertiary); padding: 8px 0; text-align: center; }

/* Score Breakdown */
.breakdown-list { padding: 4px 0; }
.br-item { display: flex; align-items: center; gap: 8px; padding: 3px 0; font-size: 11px; }
.br-name { flex: 0 0 80px; font-weight: 500; }
.br-effect { flex: 0 0 40px; text-align: right; font-weight: 700; font-variant-numeric: tabular-nums; }
.br-detail { flex: 1; color: var(--color-text-tertiary); font-size: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.br-total { display: flex; align-items: center; gap: 8px; padding: 4px 0; margin-top: 2px; border-top: 1px solid var(--color-border-strong); font-size: 12px; font-weight: 600; }

/* Report Type Toggle */
.report-toggle { display: flex; gap: 4px; flex-shrink: 0; }
.report-toggle button {
  padding: 2px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-tertiary); font-size: 11px; cursor: pointer;
}
.report-toggle button.active { background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent); }

/* Chart */
.chart-section { padding: 4px 0; }

/* Trend */
.trend-section { margin-top: 2px; padding-top: 4px; border-top: 1px dashed var(--color-border-subtle); }
.trend-h { cursor: pointer; }

/* Findings */
.finding { display: flex; gap: 8px; padding: 8px; border-radius: var(--radius-md); margin-bottom: 3px; align-items: flex-start; }
.finding.high { background: rgba(239,68,68,0.08); border-left: 2px solid var(--color-up); }
.finding.medium { background: rgba(245,158,11,0.06); border-left: 2px solid var(--color-accent); }
.finding.low { background: rgba(34,197,94,0.04); border-left: 2px solid var(--color-down); }
.finding-icon { font-size: 12px; flex-shrink: 0; margin-top: 1px; }
.finding-body { flex: 1; min-width: 0; }
.finding-head { display: flex; justify-content: space-between; align-items: center; }
.finding-metric { font-size: 12px; font-weight: 600; }
.finding-value { font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; padding: 1px 6px; border-radius: var(--radius-sm); }
.finding-value.high { color: var(--color-up); background: rgba(239,68,68,0.1); }
.finding-value.medium { color: var(--color-accent); background: rgba(245,158,11,0.1); }
.finding-value.low { color: var(--color-down); background: rgba(34,197,94,0.1); }
.finding-detail { font-size: 10px; color: var(--color-text-tertiary); margin-top: 2px; }
.finding-threshold { font-size: 10px; color: var(--color-text-tertiary); margin-top: 1px; font-family: 'JetBrains Mono', monospace; opacity: 0.7; }

/* History Table */
.hist-table-wrap { overflow-x: auto; }
.hist-table { width: 100%; border-collapse: collapse; font-size: 10px; font-variant-numeric: tabular-nums; }
.hist-table th { text-align: left; padding: 4px 6px; border-bottom: 1px solid var(--color-border-strong); color: var(--color-text-tertiary); font-weight: 500; }
.hist-table th.num { text-align: right; }
.hist-table td { padding: 3px 6px; border-bottom: 1px solid var(--color-border-strong); }
.hist-table td.num { text-align: right; }
.hist-table td.period { font-family: 'JetBrains Mono', monospace; color: var(--color-text-secondary); }
.hist-table tr:hover td { background: rgba(255,255,255,0.02); }
.hist-table tr:last-child td { border-bottom: none; }

/* Overall risk */
.dr-overall { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.dr-badge { display: flex; align-items: center; gap: 8px; padding: 10px 16px; border-radius: var(--radius-lg); font-size: 13px; }
.dr-badge.dr-high { background: rgba(239,68,68,0.15); border: 1px solid rgba(239,68,68,0.3); }
.dr-badge.dr-medium { background: rgba(234,179,8,0.15); border: 1px solid rgba(234,179,8,0.3); }
.dr-badge.dr-low { background: rgba(34,197,94,0.15); border: 1px solid rgba(34,197,94,0.3); }
.dr-badge-label { font-weight: 600; font-size: 15px; }
.dr-board { color: var(--color-text-secondary); font-size: 12px; }
.st-tag { background: var(--color-accent); color: #fff; padding: 1px 6px; border-radius: var(--radius-sm); font-size: 11px; font-weight: 600; }
.dr-summary { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/* Category */
.dr-category { margin-bottom: 14px; }
.dr-cat-h { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.dr-cat-dot { width: 8px; height: 8px; border-radius: 50%; }
.dot-red { background: var(--color-up); }
.dot-yellow { background: #eab308; }
.dot-green { background: var(--color-down); }
.dr-cat-name { font-weight: 600; font-size: 13px; }

/* Items */
.dr-items { display: flex; flex-direction: column; gap: 4px; }
.dr-item { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 6px 10px; border-radius: var(--radius-sm); font-size: 12px; background: var(--color-bg-subtle); }
.dr-item-left { display: flex; align-items: center; gap: 6px; min-width: 140px; }
.dr-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.dr-indicator { font-weight: 500; white-space: nowrap; }
.dr-item-right { display: flex; align-items: center; gap: 8px; margin-left: auto; }
.dr-current { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
.dr-threshold { color: var(--color-text-secondary); font-size: 11px; }
.dr-detail { width: 100%; color: var(--color-text-secondary); font-size: 11px; padding-left: 18px; }
.dr-item-danger { border-left: 3px solid var(--color-up); }
.dr-item-warn { border-left: 3px solid #eab308; }
.dr-item-safe { border-left: 3px solid var(--color-down); }
</style>
