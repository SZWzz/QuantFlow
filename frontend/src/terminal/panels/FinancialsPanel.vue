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

const scoreColor = computed(() => {
  const s = healthScore.value
  if (s >= 80) return '#22c55e'; if (s >= 60) return '#3b82f6'
  if (s >= 40) return '#f59e0b'; return '#ef4444'
})

// ── Table columns definition ───────────────────────────────

interface ColDef {
  key: string
  label: string
  fmt: (v: any) => string
  div?: number // divisor for display (e.g. 1e8 for 亿)
  suffix?: string
}

const mainCols: ColDef[] = [
  { key: 'period', label: '报告期', fmt: (v: string) => (v || '').slice(0, 7) },
  { key: 'revenue', label: '营收', fmt: (v: number) => v ? (v / 1e8).toFixed(2) : '--', suffix: '亿' },
  { key: 'gross_margin', label: '毛利率', fmt: (v: number) => v != null ? v.toFixed(1) : '--', suffix: '%' },
  { key: 'net_profit', label: '净利润', fmt: (v: number) => v ? (v / 1e8).toFixed(2) : '--', suffix: '亿' },
  { key: 'parent_profit', label: '归母净利', fmt: (v: number) => v ? (v / 1e8).toFixed(2) : '--', suffix: '亿' },
  { key: 'total_assets', label: '总资产', fmt: (v: number) => v ? (v / 1e8).toFixed(1) : '--', suffix: '亿' },
  { key: 'roe', label: 'ROE', fmt: (v: number) => v != null ? v.toFixed(1) : '--', suffix: '%' },
  { key: 'profit_margin', label: '净利率', fmt: (v: number) => v != null ? v.toFixed(1) : '--', suffix: '%' },
  { key: 'debt_ratio', label: '负债率', fmt: (v: number) => v != null ? v.toFixed(1) : '--', suffix: '%' },
]

// Columns that display YoY change inline under the value
const yoyCols = new Set(['revenue', 'gross_margin', 'net_profit', 'parent_profit'])

// ── YoY comparison ─────────────────────────────────────────

function yoy(current: number, prev: number): number | null {
  if (!prev || !current) return null
  return ((current - prev) / Math.abs(prev)) * 100
}

function periodKey(period: string): string {
  // Extract month-day signature: "2024-06-30" → "06-30", "2024Q2" → "Q2"
  const m = period.match(/(\d{2})-(\d{2})$/)
  if (m) return m[1] + '-' + m[2]
  const q = period.match(/[Qq][1-4]$/)
  if (q) return q[0].toUpperCase()
  return period
}

function yoyChange(p: any, key: string): number | null {
  const pk = periodKey(p.period)
  // Skip period-key extraction failures (e.g. annual-only data)
  if (pk === p.period) return null
  const cur = p[key]
  if (cur == null) return null
  // Find same period last year by matching month-day signature
  for (const prev of periods.value) {
    if (prev === p) continue
    if (periodKey(prev.period) === pk) {
      const prv = prev[key]
      if (prv == null || !prv) return null
      if (key === 'roe' || key === 'profit_margin' || key === 'debt_ratio' || key === 'gross_margin') {
        return +(cur - prv).toFixed(1)
      }
      return yoy(cur, prv)
    }
  }
  return null
}

function yoySuffix(key: string): string {
  return (key === 'roe' || key === 'profit_margin' || key === 'debt_ratio' || key === 'gross_margin') ? 'pp' : '%'
}

function yoyColor(v: number | null): string {
  if (v == null) return ''
  return v >= 0 ? '#4ade80' : '#f87171'
}

function yoySign(v: number | null): string {
  if (v == null || v <= 0) return ''
  return '+'
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
      <!-- Left: Table -->
      <div class="table-col">
        <div class="table-wrapper">
          <div class="table-header">
            <span v-for="col in mainCols" :key="col.key" class="th-cell">
              {{ col.label }}
              <span v-if="col.suffix" class="th-unit">({{ col.suffix }})</span>
            </span>
          </div>
          <div class="table-body">
            <div v-for="p in periods" :key="p.period" class="table-row">
              <span class="td-cell td-period">{{ (p.period || '').slice(0, 7) }}</span>
              <template v-for="col in mainCols.slice(1)" :key="col.key">
                <span class="td-cell">
                  <span class="td-val">{{ col.fmt(p[col.key]) }}</span>
                  <span v-if="yoyCols.has(col.key) && yoyChange(p, col.key) != null"
                        class="td-yoy"
                        :style="{ color: yoyColor(yoyChange(p, col.key)) }">
                    {{ yoySign(yoyChange(p, col.key)) }}{{ yoyChange(p, col.key)?.toFixed(1) }}{{ yoySuffix(col.key) }}
                  </span>
                </span>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: Scoring -->
      <div class="score-col">
        <div class="score-card">
          <div class="score-ring-wrap">
            <div class="score-ring" :style="{ borderColor: scoreColor }">
              <span class="score-num" :style="{ color: scoreColor }">{{ healthScore }}</span>
            </div>
            <div class="score-grade" :style="{ color: scoreColor }">{{ healthGrade }}</div>
          </div>
          <div class="score-breakdown">
            <div class="sb-title">评分细项</div>
            <div v-for="(b, i) in scoreBreakdown" :key="i" class="sb-row">
              <span class="sb-label">{{ b.item }}</span>
              <span class="sb-val" :style="{ color: b.effect >= 0 ? '#4ade80' : '#f87171' }">
                {{ b.effect > 0 ? '+' : '' }}{{ b.effect }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="groupedAnomalies.length" class="anomaly-section">
          <div class="section-title">⚠️ 异常标记</div>
          <div v-for="(a, i) in groupedAnomalies" :key="i" class="anomaly-row" :class="a.level">
            <div class="a-top">
              <span class="a-type">{{ a.type }}</span>
              <span v-if="!a.latest" class="a-count">近 {{ a.count }} 期</span>
              <span v-else-if="a.count > 1" class="a-count">本期 · 持续 {{ a.count }} 期</span>
              <span v-else class="a-count">本期</span>
            </div>
            <div class="a-detail">{{ a.detail }}</div>
          </div>
        </div>
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

.content-area { flex: 1; overflow: hidden; display: flex; gap: 12px; min-height: 0; }

/* ── Left: Table column ── */
.table-col { flex: 2; display: flex; flex-direction: column; overflow: hidden; min-width: 0; }
.table-wrapper { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 2px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row { display: flex; padding: 3px 0; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-elevated); }
.th-cell { flex: 1; padding: 0 4px; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-variant-numeric: tabular-nums; }
.th-cell:first-child { text-align: left; flex: 0.7; }
.td-period { text-align: left; flex: 0.7; padding: 0 4px; font-variant-numeric: tabular-nums; font-size: 12px; display: flex; align-items: center; }
.th-unit { font-weight: 400; color: var(--color-text-tertiary); }
.td-cell { flex: 1; display: flex; flex-direction: column; align-items: flex-end; justify-content: center; gap: 1px; padding: 1px 4px; }
.td-val { font-size: 12px; }
.td-yoy { font-size: 10px; font-weight: 600; line-height: 1; }

/* ── Right: Score column ── */
.score-col { flex: 0 0 180px; display: flex; flex-direction: column; gap: 10px; overflow-y: auto; }
.score-card {
  display: flex; flex-direction: column; align-items: center; gap: 12px;
  padding: 14px; background: var(--color-bg-elevated); border-radius: 8px;
}
.score-ring-wrap { display: flex; flex-direction: column; align-items: center; gap: 6px; }
.score-ring {
  display: flex; align-items: center; justify-content: center;
  width: 72px; height: 72px; border-radius: 50%; border: 3px solid;
}
.score-num { font-size: 22px; font-weight: 700; }
.score-grade { font-size: 18px; font-weight: 600; letter-spacing: 2px; }
.score-breakdown { width: 100%; }
.sb-title { font-size: 10px; font-weight: 600; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 6px; text-align: center; }
.sb-row { display: flex; justify-content: space-between; padding: 3px 0; font-size: 11px; }
.sb-label { color: var(--color-text-secondary); }
.sb-val { font-weight: 600; font-variant-numeric: tabular-nums; }

.anomaly-section { }
.section-title { font-size: 10px; font-weight: 600; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 4px; }
.anomaly-row { padding: 5px 8px; border-radius: 4px; font-size: 11px; margin-bottom: 3px; }
.anomaly-row.high { background: rgba(239,68,68,0.1); }
.anomaly-row.medium { background: rgba(245,158,11,0.1); }
.a-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; }
.a-type { font-weight: 600; color: #fca5a5; }
.anomaly-row.medium .a-type { color: #fcd34d; }
.a-detail { font-size: 10px; color: var(--color-text-secondary); }
.a-count { font-size: 10px; color: var(--color-text-tertiary); background: rgba(255,255,255,0.06); padding: 0 6px; border-radius: 3px; }
</style>
