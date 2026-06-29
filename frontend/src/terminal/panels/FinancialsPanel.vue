<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { name } = useStockName(symbol)
const loading = ref(false)
const error = ref('')
const result = ref<any>(null)

const periods = computed(() => result.value?.periods || [])
const anomalyFlags = computed(() => result.value?.anomaly_flags || [])
const scoreBreakdown = computed(() => result.value?.score_breakdown || [])
const latestPeriod = computed(() => periods.value[0]?.period || '')
const groupedAnomalies = computed(() => {
  const groups: Record<string, { type: string; level: string; count: number; detail: string; latest: boolean }> = {}
  for (const a of anomalyFlags.value) {
    const key = a.type
    if (!groups[key]) { groups[key] = { type: a.type, level: a.level, count: 0, detail: a.detail, latest: false } }
    groups[key].count++
    if (a.level === 'high') groups[key].level = 'high'
    if (a.period === latestPeriod.value) { groups[key].latest = true; groups[key].detail = a.detail }
  }
  return Object.values(groups).sort((a, b) => (a.latest === b.latest ? 0 : a.latest ? -1 : 1))
})
const healthScore = computed(() => result.value?.health_score ?? 0)
const healthGrade = computed(() => result.value?.health_grade ?? '--')
const metrics = computed(() => result.value?.metrics || {})

const scoreColor = computed(() => {
  const s = healthScore.value
  if (s >= 80) return '#22c55e'; if (s >= 60) return '#3b82f6'
  if (s >= 40) return '#f59e0b'; return '#ef4444'
})

const columns = ['period', 'revenue', 'net_profit', 'roe', 'debt_ratio', 'profit_margin']
const colMeta: Record<string, { label: string; fmt: (v: any) => string }> = {
  period: { label: '报告期', fmt: (v: string) => (v || '').slice(0, 7) },
  revenue: { label: '营收(亿)', fmt: (v: number) => v ? (v / 1e8).toFixed(2) : '--' },
  net_profit: { label: '净利润(亿)', fmt: (v: number) => v ? (v / 1e8).toFixed(2) : '--' },
  roe: { label: 'ROE%', fmt: (v: number) => v != null ? v.toFixed(1) : '--' },
  debt_ratio: { label: '负债率%', fmt: (v: number) => v != null ? v.toFixed(1) : '--' },
  profit_margin: { label: '净利率%', fmt: (v: number) => v != null ? v.toFixed(1) : '--' },
}

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetFinancialAnalysis) return
    const resp = await app.GetFinancialAnalysis(symbol.value)
    result.value = resp?.data ? JSON.parse(resp.data) : resp
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
  <div class="fin-panel">
    <div class="panel-header">
      <h3>财务报表</h3>
      <div class="header-right">
        <span class="symbol-badge">{{ symbol }} {{ name }}</span>
        <button class="refresh-btn" @click="loadData" :disabled="loading">⟳</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading && periods.length === 0" type="table" :rows="5" />
    <div v-else-if="error" class="status error">{{ error }}</div>
    <div v-else-if="!loading && periods.length === 0" class="status">{{ result?.error || '暂无财务数据 — 输入 A 股代码查看' }}</div>

    <div v-else class="content-area">
      <div class="score-card">
        <div class="score-ring" :style="{ color: scoreColor }">
          <span class="score-num">{{ healthScore }}</span>
          <span class="score-label">健康评分</span>
        </div>
        <div class="score-grade" :style="{ color: scoreColor }">{{ healthGrade }}</div>
        <div class="score-metrics">
          <div v-for="(b, i) in scoreBreakdown" :key="i" class="metric-item">
            <span class="m-label">{{ b.item }}</span>
            <span class="m-value" :style="{ color: b.effect >= 0 ? '#4ade80' : '#f87171' }">{{ b.effect > 0 ? '+' : '' }}{{ b.effect }}</span>
          </div>
        </div>
      </div>
      <div v-if="groupedAnomalies.length" class="anomaly-section">
        <div class="section-title">⚠️ 异常标记</div>
        <div v-for="(a, i) in groupedAnomalies" :key="i" class="anomaly-row" :class="a.level">
          <span class="a-type">{{ a.type }}</span>
          <span class="a-detail">{{ a.detail }}</span>
          <span v-if="!a.latest" class="a-count">近 {{ a.count }} 期中 {{ a.count }} 期</span>
          <span v-else-if="a.count > 1" class="a-count">本期 · 持续 {{ a.count }} 期</span>
          <span v-else class="a-count">本期</span>
        </div>
      </div>
      <div class="table-wrapper">
        <div class="table-header"><span v-for="col in columns" :key="col" class="th-cell">{{ colMeta[col].label }}</span></div>
        <div class="table-body"><div v-for="p in periods" :key="p.period" class="table-row"><span v-for="col in columns" :key="col" class="td-cell">{{ colMeta[col].fmt(p[col]) }}</span></div></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fin-panel { padding: 12px; height: 100%; display: flex; flex-direction: column; color: var(--color-text,#e5e7eb); background: var(--color-bg-panel,#1a1a2e); overflow: hidden; }
.panel-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-right { display: flex; align-items: center; gap: 8px; }
.symbol-badge { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: rgba(59,130,246,0.15); color: #60a5fa; font-family: monospace; }
.refresh-btn { padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.status { display: flex; align-items: center; justify-content: center; flex: 1; color: var(--color-text-tertiary); font-size: 13px; }
.status.error { color: var(--color-error); }
.content-area { flex: 1; overflow-y: auto; }
.score-card { display: flex; align-items: center; gap: 16px; padding: 12px 16px; background: var(--color-bg-elevated); border-radius: 8px; margin-bottom: 10px; }
.score-ring { display: flex; flex-direction: column; align-items: center; justify-content: center; width: 64px; height: 64px; border-radius: 50%; border: 3px solid currentColor; }
.score-num { font-size: 20px; font-weight: 700; }
.score-label { font-size: 9px; color: var(--color-text-tertiary); }
.score-grade { font-size: 16px; font-weight: 600; }
.score-metrics { display: flex; gap: 16px; margin-left: auto; }
.metric-item { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.m-label { font-size: 9px; color: var(--color-text-tertiary); text-transform: uppercase; }
.m-value { font-size: 14px; font-weight: 600; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.anomaly-section { margin-bottom: 10px; }
.section-title { font-size: 10px; font-weight: 600; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 4px; }
.anomaly-row { display: flex; gap: 8px; padding: 4px 8px; border-radius: 4px; font-size: 11px; margin-bottom: 2px; }
.anomaly-row.high { background: rgba(239,68,68,0.1); color: #fca5a5; }
.anomaly-row.medium { background: rgba(245,158,11,0.1); color: #fcd34d; }
.a-type { font-weight: 600; flex-shrink: 0; }
.a-detail { color: var(--color-text-secondary); }
.a-count { margin-left: auto; font-size: 10px; color: var(--color-text-tertiary); background: rgba(255,255,255,0.06); padding: 0 6px; border-radius: 3px; }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: 4px 0; border-bottom: 2px solid var(--color-border-strong); font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row { display: flex; padding: 3px 0; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-elevated); }
.th-cell, .td-cell { flex: 1; padding: 0 4px; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-variant-numeric: tabular-nums; }
.th-cell:first-child, .td-cell:first-child { text-align: left; }
</style>
