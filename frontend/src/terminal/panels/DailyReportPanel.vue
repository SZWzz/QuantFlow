<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePortfolioStore, type DailyReport } from '@/stores/portfolio'
import { GetDailyReport, ListDailyReports, GenerateDailyReport, ExportReportCSV } from '@/lib/wails'
import { getIcon } from '@/lib/icons'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

const portfolio = usePortfolioStore()
const selectedDate = ref(new Date().toISOString().slice(0, 10))
const report = ref<DailyReport | null>(null); const history = ref<DailyReport[]>([])
const loading = ref(false); const notes = ref(''); const showHistory = ref(false)

async function loadReport(date: string) { loading.value = true; try { const r = await GetDailyReport(date); report.value = r || null; notes.value = r?.notes || '' } catch { try { const r = await GenerateDailyReport(date); report.value = r || null; notes.value = '' } catch { report.value = null } } finally { loading.value = false } }
async function loadHistory() { try { history.value = await ListDailyReports(30) } catch { history.value = [] } }
async function handleGenerate() { loading.value = true; try { const r = await GenerateDailyReport(selectedDate.value); report.value = r || null; notes.value = ''; loadHistory() } catch { report.value = null } finally { loading.value = false } }
async function handleExport() { if (!report.value) return; try { await ExportReportCSV(report.value.date) } catch { /* ignore */ } }

const pnlClass = computed(() => { if (!report.value) return ''; return report.value.day_pnl >= 0 ? 'pnl-positive' : 'pnl-negative' })

onMounted(() => { loadReport(selectedDate.value); loadHistory() })
</script>

<template>
  <div class="daily-report-panel">
    <PanelHeader title="日结报告">
      <template #controls>
        <input v-model="selectedDate" type="date" class="date-input" />
        <button class="btn btn-primary" @click="handleGenerate" :disabled="loading">{{ loading ? '生成中...' : '生成报告' }}</button>
        <button class="btn btn-sm" @click="showHistory = !showHistory">历史报告</button>
      </template>
    </PanelHeader>

    <LoadingState v-if="loading" type="card" :rows="3" />

    <div v-else-if="report" class="report-content">
      <div class="report-summary">
        <div class="summary-card" :class="pnlClass"><span class="summary-label">今日盈亏</span><span class="summary-value">{{ report.day_pnl >= 0 ? '+' : '' }}{{ report.day_pnl?.toFixed(2) }}<span class="summary-pct">({{ report.day_pnl_percent?.toFixed(2) }}%)</span></span></div>
        <div class="summary-card"><span class="summary-label">累计盈亏</span><span class="summary-value">{{ report.total_pnl >= 0 ? '+' : '' }}{{ report.total_pnl?.toFixed(2) }}<span class="summary-pct">({{ report.total_pnl_percent?.toFixed(2) }}%)</span></span></div>
        <div class="summary-card"><span class="summary-label">持仓市值</span><span class="summary-value">¥{{ report.market_value?.toFixed(2) }}</span></div>
        <div class="summary-card"><span class="summary-label">交易</span><span class="summary-value">{{ report.trades }} 笔 | 佣金 ¥{{ report.commission?.toFixed(2) }}</span></div>
      </div>

      <div v-if="report.best_trade || report.worst_trade" class="trade-highlights">
        <div v-if="report.best_trade" class="highlight best">🏆 最佳: {{ report.best_trade.symbol }} +{{ report.best_trade.pnl?.toFixed(2) }}</div>
        <div v-if="report.worst_trade" class="highlight worst">😞 最差: {{ report.worst_trade.symbol }} {{ report.worst_trade.pnl?.toFixed(2) }}</div>
      </div>

      <div v-if="report.positions?.length" class="positions-section">
        <h4>持仓 ({{ report.positions.length }})</h4>
        <div v-for="pos in report.positions" :key="pos.symbol" class="position-row"><span class="pos-symbol">{{ pos.symbol }}</span><span class="pos-qty">{{ pos.quantity }}</span><span class="pos-val">¥{{ pos.market_val?.toFixed(2) }}</span><span class="pos-pnl" :class="pos.pnl >= 0 ? 'pnl-positive' : 'pnl-negative'">{{ pos.pnl >= 0 ? '+' : '' }}{{ pos.pnl_pct?.toFixed(2) }}%</span></div>
      </div>

      <div class="report-footer"><button class="btn btn-primary" @click="handleExport">导出 CSV</button><span class="report-date">{{ report.date }}</span></div>
    </div>

    <EmptyState v-else title="选择日期并生成报告" description="选择日期并点击「生成报告」以创建日结报告" />

    <div v-if="showHistory" class="history-panel">
      <h4>历史报告</h4>
      <EmptyState v-if="history.length === 0" title="暂无历史报告" />
      <div v-for="h in history" :key="h.date" class="history-item" @click="selectedDate = h.date; showHistory = false; loadReport(h.date)">
        <span class="history-date">{{ h.date }}</span><span class="history-pnl" :class="h.day_pnl >= 0 ? 'pnl-positive' : 'pnl-negative'">{{ h.day_pnl >= 0 ? '+' : '' }}{{ h.day_pnl?.toFixed(2) }}</span><span class="history-trades">{{ h.trades }} 笔</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.daily-report-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.header-actions { display: flex; gap: var(--space-sm); align-items: center; }
.date-input { padding: var(--space-xs) var(--space-sm); border: 1px solid var(--color-border); border-radius: var(--radius-sm); background: var(--color-bg-panel); color: var(--color-text-primary); font-size: var(--font-xs); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.report-content { display: flex; flex-direction: column; gap: var(--space-lg); flex: 1; overflow-y: auto; padding: var(--space-sm) var(--panel-padding); }
.report-summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: var(--space-sm); }
.summary-card { padding: var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-bg-subtle); }
.summary-label { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); margin-bottom: var(--space-xs); }
.summary-value { font-size: var(--font-lg); font-weight: 700; font-family: var(--font-mono); }
.summary-pct { font-size: var(--font-xs); font-weight: 400; opacity: 0.7; }
.pnl-positive { color: var(--color-down); }
.pnl-negative { color: var(--color-up); }
.trade-highlights { display: flex; gap: var(--space-sm); }
.highlight { flex: 1; padding: var(--space-sm) var(--space-md); border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 600; }
.highlight.best { background: var(--color-down-soft); border: 1px solid var(--color-down); color: var(--color-down); }
.highlight.worst { background: var(--color-up-soft); border: 1px solid var(--color-up); color: var(--color-up); }
.positions-section h4 { font-size: var(--font-sm); margin-bottom: var(--space-sm); }
.position-row { display: flex; gap: var(--space-md); align-items: center; padding: var(--space-sm) 0; border-bottom: 1px solid var(--color-border); font-size: var(--font-xs); }
.pos-symbol { font-weight: 600; width: 60px; }
.pos-qty { width: 50px; text-align: right; color: var(--color-text-secondary); }
.pos-val { width: 80px; text-align: right; font-family: var(--font-mono); }
.pos-pnl { width: 60px; text-align: right; font-weight: 600; }
.report-footer { display: flex; justify-content: space-between; align-items: center; padding: var(--space-sm) var(--panel-padding); border-top: 1px solid var(--color-border-subtle); }
.report-date { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.history-panel { border-top: 1px solid var(--color-border); padding: var(--space-md) var(--panel-padding); }
.history-panel h4 { font-size: var(--font-sm); margin-bottom: var(--space-sm); }
.history-item { display: flex; gap: var(--space-lg); align-items: center; padding: var(--space-sm); cursor: pointer; border-radius: var(--radius-sm); font-size: var(--font-xs); }
.history-item:hover { background: var(--color-bg-hover); }
.history-date { font-weight: 600; width: 80px; }
.history-pnl { width: 100px; font-family: var(--font-mono); font-weight: 600; }
.history-trades { color: var(--color-text-secondary); }
</style>
