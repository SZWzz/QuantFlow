<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { usePortfolioStore, type DailyReport } from '@/stores/portfolio'
import { GetDailyReport, ListDailyReports, GenerateDailyReport, ExportReportCSV } from '@/lib/wails'
import { getIcon } from '@/lib/icons'

const portfolio = usePortfolioStore()

const selectedDate = ref(new Date().toISOString().slice(0, 10))
const report = ref<DailyReport | null>(null)
const history = ref<DailyReport[]>([])
const loading = ref(false)
const notes = ref('')
const showHistory = ref(false)

async function loadReport(date: string) {
  loading.value = true
  try {
    const r = await GetDailyReport(date)
    report.value = r || null
    notes.value = r?.notes || ''
  } catch {
    try {
      const r = await GenerateDailyReport(date)
      report.value = r || null
      notes.value = ''
    } catch {
      report.value = null
    }
  } finally {
    loading.value = false
  }
}

async function loadHistory() {
  try {
    history.value = await ListDailyReports(30)
  } catch {
    history.value = []
  }
}

async function handleGenerate() {
  loading.value = true
  try {
    const r = await GenerateDailyReport(selectedDate.value)
    report.value = r || null
    notes.value = ''
    loadHistory()
  } catch {
    report.value = null
  } finally {
    loading.value = false
  }
}

async function handleExport() {
  if (!report.value) return
  try {
    await ExportReportCSV(report.value.date)
  } catch { /* ignore */ }
}

const pnlClass = computed(() => {
  if (!report.value) return ''
  return report.value.day_pnl >= 0 ? 'pnl-positive' : 'pnl-negative'
})

onMounted(() => {
  loadReport(selectedDate.value)
  loadHistory()
})
</script>

<template>
  <div class="daily-report-panel">
    <div class="panel-header">
      <h3>📋 日结报告</h3>
      <div class="header-actions">
        <input v-model="selectedDate" type="date" class="date-input" />
        <button class="btn" @click="handleGenerate" :disabled="loading">
          {{ loading ? '生成中...' : '生成报告' }}
        </button>
        <button class="btn btn-secondary" @click="showHistory = !showHistory">
          历史报告
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="report" class="report-content">
      <div class="report-summary">
        <div class="summary-card" :class="pnlClass">
          <span class="summary-label">今日盈亏</span>
          <span class="summary-value">
            {{ report.day_pnl >= 0 ? '+' : '' }}{{ report.day_pnl?.toFixed(2) }}
            <span class="summary-pct">({{ report.day_pnl_percent?.toFixed(2) }}%)</span>
          </span>
        </div>
        <div class="summary-card">
          <span class="summary-label">累计盈亏</span>
          <span class="summary-value">
            {{ report.total_pnl >= 0 ? '+' : '' }}{{ report.total_pnl?.toFixed(2) }}
            <span class="summary-pct">({{ report.total_pnl_percent?.toFixed(2) }}%)</span>
          </span>
        </div>
        <div class="summary-card">
          <span class="summary-label">持仓市值</span>
          <span class="summary-value">¥{{ report.market_value?.toFixed(2) }}</span>
        </div>
        <div class="summary-card">
          <span class="summary-label">交易</span>
          <span class="summary-value">{{ report.trades }} 笔 | 佣金 ¥{{ report.commission?.toFixed(2) }}</span>
        </div>
      </div>

      <div v-if="report.best_trade || report.worst_trade" class="trade-highlights">
        <div v-if="report.best_trade" class="highlight best">
          🏆 最佳: {{ report.best_trade.symbol }}
          +{{ report.best_trade.pnl?.toFixed(2) }}
        </div>
        <div v-if="report.worst_trade" class="highlight worst">
          😞 最差: {{ report.worst_trade.symbol }}
          {{ report.worst_trade.pnl?.toFixed(2) }}
        </div>
      </div>

      <div v-if="report.positions?.length" class="positions-section">
        <h4>持仓 ({{ report.positions.length }})</h4>
        <div v-for="pos in report.positions" :key="pos.symbol" class="position-row">
          <span class="pos-symbol">{{ pos.symbol }}</span>
          <span class="pos-qty">{{ pos.quantity }}</span>
          <span class="pos-val">¥{{ pos.market_val?.toFixed(2) }}</span>
          <span class="pos-pnl" :class="pos.pnl >= 0 ? 'pnl-positive' : 'pnl-negative'">
            {{ pos.pnl >= 0 ? '+' : '' }}{{ pos.pnl_pct?.toFixed(2) }}%
          </span>
        </div>
      </div>

      <div class="report-footer">
        <button class="btn" @click="handleExport">导出 CSV</button>
        <span class="report-date">{{ report.date }}</span>
      </div>
    </div>

    <div v-else class="empty-state">
      选择日期并点击"生成报告"以创建日结报告
    </div>

    <!-- History panel -->
    <div v-if="showHistory" class="history-panel">
      <h4>历史报告</h4>
      <div v-if="history.length === 0" class="empty-state">暂无历史报告</div>
      <div
        v-for="h in history"
        :key="h.date"
        class="history-item"
        @click="selectedDate = h.date; showHistory = false; loadReport(h.date)"
      >
        <span class="history-date">{{ h.date }}</span>
        <span class="history-pnl" :class="h.day_pnl >= 0 ? 'pnl-positive' : 'pnl-negative'">
          {{ h.day_pnl >= 0 ? '+' : '' }}{{ h.day_pnl?.toFixed(2) }}
        </span>
        <span class="history-trades">{{ h.trades }} 笔</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.daily-report-panel {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header-actions { display: flex; gap: 8px; align-items: center; }
.date-input {
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 12px;
}
.btn {
  padding: 4px 12px;
  border: 1px solid var(--color-accent);
  background: var(--color-accent);
  color: #fff;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
}
.btn-secondary { background: transparent; color: var(--color-accent); }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.loading { text-align: center; padding: 32px; color: var(--color-text-tertiary); font-size: 13px; }
.report-content { display: flex; flex-direction: column; gap: 16px; }

.report-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}
.summary-card {
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-subtle);
}
.summary-label { display: block; font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.summary-value { font-size: 16px; font-weight: 700; font-family: 'JetBrains Mono', monospace; }
.summary-pct { font-size: 11px; font-weight: 400; opacity: 0.7; }
.pnl-positive { color: var(--color-success); }
.pnl-negative { color: var(--color-danger); }

.trade-highlights { display: flex; gap: 8px; }
.highlight { flex: 1; padding: 8px 12px; border-radius: var(--radius-sm); font-size: 12px; font-weight: 600; }
.highlight.best { background: var(--color-success-soft); border: 1px solid var(--color-success); color: var(--color-success); }
.highlight.worst { background: var(--color-danger-soft); border: 1px solid var(--color-danger); color: var(--color-danger); }

.positions-section h4 { font-size: 13px; margin-bottom: 8px; }
.position-row {
  display: flex; gap: 12px; align-items: center;
  padding: 6px 0; border-bottom: 1px solid var(--color-border);
  font-size: 12px;
}
.pos-symbol { font-weight: 600; width: 60px; }
.pos-qty { width: 50px; text-align: right; color: var(--color-text-secondary); }
.pos-val { width: 80px; text-align: right; font-family: 'JetBrains Mono', monospace; }
.pos-pnl { width: 60px; text-align: right; font-weight: 600; }

.report-footer { display: flex; justify-content: space-between; align-items: center; }
.report-date { font-size: 11px; color: var(--color-text-tertiary); }

.history-panel { border-top: 1px solid var(--color-border); padding-top: 12px; }
.history-panel h4 { font-size: 13px; margin-bottom: 8px; }
.history-item {
  display: flex; gap: 16px; align-items: center;
  padding: 6px 8px; cursor: pointer; border-radius: var(--radius-sm);
  font-size: 12px;
}
.history-item:hover { background: var(--color-bg-hover); }
.history-date { font-weight: 600; width: 80px; }
.history-pnl { width: 100px; font-family: 'JetBrains Mono', monospace; font-weight: 600; }
.history-trades { color: var(--color-text-secondary); }
</style>
