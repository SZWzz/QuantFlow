<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const { fetchWithCache } = usePanelCache()
const app = (window as any).go?.main?.App

interface HKSettlementInfo {
  market: string
  settlement_days: number
  stamp_duty: number
  exchange_fee: number
  sfc_levy: number
  trading_fee: number
  frc_levy: number
  has_price_limits: boolean
  lot_size_min: number
  currency: string
  description: string
}

interface CalendarEntry {
  trade_date: string
  [key: string]: any
}

interface CalendarResult {
  data: CalendarEntry[]
}

const activeTab = ref<'settlement_rules' | 'fee_calculator' | 'trade_calendar'>('settlement_rules')
const loading = ref(false)
const settlementInfo = ref<HKSettlementInfo | null>(null)
const calendarData = ref<CalendarEntry[]>([])
const calendarLoading = ref(false)

const tradeAmount = ref<number>(100000)

const currentYear = new Date().getFullYear()
const calendarYear = ref(currentYear)
const calendarMonth = ref<number>(0)

async function fetchSettlementInfo() {
  if (!app?.GetHKSettlementInfo) return
  loading.value = true
  try {
    const { data: result } = await fetchWithCache('hk_settlement_info', () => app.GetHKSettlementInfo(), 30 * 60 * 1000)
    settlementInfo.value = result as HKSettlementInfo
  } catch (e) {
    console.error('[HKSettlement]', e)
    settlementInfo.value = null
  } finally {
    loading.value = false
  }
}

async function fetchCalendar() {
  if (!app?.GetHKTradingCalendar) return
  calendarLoading.value = true
  try {
    const { data: result } = await fetchWithCache(`hk_trade_calendar:${calendarYear.value}`, () => app.GetHKTradingCalendar(calendarYear.value), 30 * 60 * 1000)
    const raw = result as CalendarResult
    calendarData.value = Array.isArray(raw.data) ? raw.data : []
  } catch (e) {
    console.error('[HKSettlement calendar]', e)
    calendarData.value = []
  } finally {
    calendarLoading.value = false
  }
}

const fees = computed(() => {
  const amt = tradeAmount.value || 0
  const info = settlementInfo.value
  if (!info) return { items: [], total: 0 }
  const items = [
    { label: t('common.stamp_duty'), rate: info.stamp_duty, amount: amt * info.stamp_duty },
    { label: t('common.exchange_fee'), rate: info.exchange_fee, amount: amt * info.exchange_fee },
    { label: t('common.sfc_levy'), rate: info.sfc_levy, amount: amt * info.sfc_levy },
    { label: t('common.trading_fee'), rate: info.trading_fee, amount: amt * info.trading_fee },
    { label: t('common.frc_levy'), rate: info.frc_levy, amount: amt * info.frc_levy },
  ]
  const total = items.reduce((s, i) => s + i.amount, 0)
  return { items, total }
})

const filteredCalendar = computed(() => {
  if (!calendarMonth.value) return calendarData.value
  return calendarData.value.filter((entry) => {
    const dateStr = entry.trade_date || entry['日期'] || ''
    const m = dateStr.slice(5, 7)
    return parseInt(m, 10) === calendarMonth.value
  })
})

function formatDate(d: string): string {
  return d ? d.slice(0, 10) : '--'
}

function formatRate(rate: number): string {
  return (rate * 100).toFixed(4) + '%'
}

function switchTab(tab: 'settlement_rules' | 'fee_calculator' | 'trade_calendar') {
  activeTab.value = tab
  if (tab === 'trade_calendar' && calendarData.value.length === 0) {
    fetchCalendar()
  }
}

onMounted(() => {
  fetchSettlementInfo()
})
</script>

<template>
  <div class="hk-settlement-panel">
    <div class="panel-header">
      <h3>{{ t('panels.hk_settlement') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'settlement_rules' }]" @click="switchTab('settlement_rules')">{{ t('panels.settlement_rules') }}</button>
        <button :class="['tab', { active: activeTab === 'fee_calculator' }]" @click="switchTab('fee_calculator')">{{ t('panels.fee_calculator') }}</button>
        <button :class="['tab', { active: activeTab === 'trade_calendar' }]" @click="switchTab('trade_calendar')">{{ t('panels.trade_calendar') }}</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading" type="card" />

    <template v-else-if="activeTab === 'settlement_rules' && settlementInfo">
      <div class="settlement-timeline">
        <div class="timeline-item">
          <span class="timeline-label">T</span>
          <span class="timeline-desc">{{ t('common.trade_date') }}</span>
        </div>
        <div class="timeline-arrow">→</div>
        <div class="timeline-item">
          <span class="timeline-label">T+{{ settlementInfo.settlement_days }}</span>
          <span class="timeline-desc">{{ t('common.settlement_date') }}</span>
        </div>
      </div>

      <div class="fee-table">
        <div class="fee-table-header">
          <span class="fee-col-name">{{ t('common.fee_type') }}</span>
          <span class="fee-col-rate">{{ t('common.rate') }}</span>
        </div>
        <div class="fee-table-body">
          <div class="fee-row">
            <span class="fee-col-name">{{ t('common.stamp_duty') }}</span>
            <span class="fee-col-rate">{{ formatRate(settlementInfo.stamp_duty) }}</span>
          </div>
          <div class="fee-row">
            <span class="fee-col-name">{{ t('common.exchange_fee') }}</span>
            <span class="fee-col-rate">{{ formatRate(settlementInfo.exchange_fee) }}</span>
          </div>
          <div class="fee-row">
            <span class="fee-col-name">{{ t('common.sfc_levy') }}</span>
            <span class="fee-col-rate">{{ formatRate(settlementInfo.sfc_levy) }}</span>
          </div>
          <div class="fee-row">
            <span class="fee-col-name">{{ t('common.trading_fee') }}</span>
            <span class="fee-col-rate">{{ formatRate(settlementInfo.trading_fee) }}</span>
          </div>
          <div class="fee-row">
            <span class="fee-col-name">{{ t('common.frc_levy') }}</span>
            <span class="fee-col-rate">{{ formatRate(settlementInfo.frc_levy) }}</span>
          </div>
        </div>
      </div>

      <div class="rule-cards">
        <div class="rule-card">
          <span class="rule-label">{{ t('common.settlement_mode') }}</span>
          <span class="rule-value">T+{{ settlementInfo.settlement_days }}</span>
        </div>
        <div class="rule-card">
          <span class="rule-label">{{ t('common.price_limits') }}</span>
          <span class="rule-value">{{ settlementInfo.has_price_limits ? t('common.yes') : t('common.no') }}</span>
        </div>
        <div class="rule-card">
          <span class="rule-label">{{ t('common.lot_size') }}</span>
          <span class="rule-value">{{ settlementInfo.lot_size_min }}</span>
        </div>
        <div class="rule-card">
          <span class="rule-label">{{ t('common.currency') }}</span>
          <span class="rule-value">{{ settlementInfo.currency }}</span>
        </div>
      </div>

      <div v-if="settlementInfo.description" class="description">{{ settlementInfo.description }}</div>
    </template>

    <template v-else-if="activeTab === 'fee_calculator' && settlementInfo">
      <div class="calculator-section">
        <div class="input-row">
          <label class="input-label">{{ t('panels.trade_amount') }} (HKD)</label>
          <input v-model.number="tradeAmount" type="number" class="amount-input" min="0" />
        </div>

        <div class="fee-table">
          <div class="fee-table-header">
            <span class="fee-col-name">{{ t('common.fee_type') }}</span>
            <span class="fee-col-rate">{{ t('common.rate') }}</span>
            <span class="fee-col-amount">{{ t('common.amount') }}</span>
          </div>
          <div class="fee-table-body">
            <div v-for="fee in fees.items" :key="fee.label" class="fee-row">
              <span class="fee-col-name">{{ fee.label }}</span>
              <span class="fee-col-rate">{{ formatRate(fee.rate) }}</span>
              <span class="fee-col-amount">{{ fee.amount.toFixed(2) }}</span>
            </div>
          </div>
        </div>

        <div class="total-row">
          <span class="total-label">{{ t('panels.total_fees') }}</span>
          <span class="total-amount">{{ fees.total.toFixed(2) }} HKD</span>
        </div>
      </div>
    </template>

    <template v-else-if="activeTab === 'trade_calendar'">
      <div class="calendar-controls">
        <input v-model.number="calendarYear" type="number" class="year-input" min="2000" max="2100" @change="fetchCalendar" />
        <select v-model.number="calendarMonth" class="month-select">
          <option :value="0">{{ t('common.all') }}</option>
          <option v-for="m in 12" :key="m" :value="m">{{ m }}</option>
        </select>
        <button class="refresh-btn" @click="fetchCalendar" :disabled="calendarLoading">⟳</button>
      </div>

      <SkeletonPanel v-if="calendarLoading" type="table" :rows="8" />

      <div v-else-if="filteredCalendar.length === 0" class="empty-state">{{ t('common.no_data') }}</div>

      <div v-else class="table-wrapper">
        <div class="table-header">
          <span class="col-date">{{ t('common.date') }}</span>
          <span class="col-info">{{ t('common.remarks') }}</span>
        </div>
        <div class="table-body">
          <div v-for="(entry, idx) in filteredCalendar" :key="idx" class="table-row">
            <span class="col-date">{{ formatDate(entry.trade_date || entry['日期']) }}</span>
            <span class="col-info">{{ entry.remarks || entry['备注'] || entry.holiday || entry['节日'] || '--' }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.hk-settlement-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}

.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }

.settlement-timeline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  margin-bottom: 12px;
  background: var(--color-bg-elevated);
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
.timeline-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
.timeline-label {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-accent);
}
.timeline-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
}
.timeline-arrow {
  font-size: 20px;
  color: var(--color-text-tertiary);
}

.fee-table {
  margin-bottom: 12px;
  flex-shrink: 0;
}
.fee-table-header {
  display: flex;
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.fee-table-body { font-size: 12px; }
.fee-row {
  display: flex;
  padding: 3px 0;
  align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.fee-row:hover { background: var(--color-bg-elevated); }
.fee-col-name { flex: 1; min-width: 0; }
.fee-col-rate { width: 100px; text-align: right; font-variant-numeric: tabular-nums; }
.fee-col-amount { width: 100px; text-align: right; font-variant-numeric: tabular-nums; }

.rule-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-bottom: 12px;
  flex-shrink: 0;
}
.rule-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  background: var(--color-bg-elevated);
  border-radius: var(--radius-md);
}
.rule-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.rule-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-accent);
}

.description {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.calculator-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.input-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.input-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  white-space: nowrap;
}
.amount-input {
  padding: 4px 8px;
  font-size: 14px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  width: 160px;
  font-variant-numeric: tabular-nums;
}
.total-row {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-top: 1px solid var(--color-border-strong);
}
.total-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text);
}
.total-amount {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-accent);
}

.calendar-controls {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.year-input {
  padding: 2px 6px;
  font-size: 11px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  width: 70px;
}
.month-select {
  padding: 2px 6px;
  font-size: 11px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
}
.refresh-btn {
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex;
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex;
  padding: 3px 0;
  align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-date { width: 100px; }
.col-info { flex: 1; min-width: 0; color: var(--color-text-secondary); }
</style>
