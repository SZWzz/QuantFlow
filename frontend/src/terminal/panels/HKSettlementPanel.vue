<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, LoadingState, EmptyState, ErrorState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const { fetchWithCache } = usePanelCache()

interface HKSettlementInfo {
  settlement_days: number; stamp_duty: number; exchange_fee: number; sfc_levy: number
  trading_fee: number; frc_levy: number; has_price_limits: boolean; lot_size_min: number
  currency: string; description: string
}
interface CalendarEntry { trade_date: string; remarks: string; [key: string]: any }
interface CalendarResult { holidays: CalendarEntry[]; trading_days: CalendarEntry[] }

const activeTab = ref<'settlement_rules' | 'fee_calculator' | 'trade_calendar'>('settlement_rules')
const settlementInfo = ref<HKSettlementInfo | null>(null)
const loading = ref(false)
const tradeAmount = ref(100000)
const calendarYear = ref(new Date().getFullYear())
const calendarMonth = ref(0)
const calendarEntries = ref<CalendarEntry[]>([])
const calendarLoading = ref(false)
const filteredCalendar = computed(() => calendarMonth.value === 0 ? calendarEntries.value : calendarEntries.value.filter((e: CalendarEntry) => { const m = parseInt((e.trade_date || '').split('-')[1]); return m === calendarMonth.value }))

const fees = computed(() => {
  const si = settlementInfo.value
  if (!si) return { items: [] as { label: string; rate: number; amount: number }[], total: 0 }
  const items = [
    { label: t('common.stamp_duty'), rate: si.stamp_duty, amount: tradeAmount.value * si.stamp_duty },
    { label: t('common.exchange_fee'), rate: si.exchange_fee, amount: tradeAmount.value * si.exchange_fee },
    { label: t('common.sfc_levy'), rate: si.sfc_levy, amount: tradeAmount.value * si.sfc_levy },
    { label: t('common.trading_fee'), rate: si.trading_fee, amount: tradeAmount.value * si.trading_fee },
    { label: t('common.frc_levy'), rate: si.frc_levy, amount: tradeAmount.value * si.frc_levy },
  ]
  return { items, total: items.reduce((s, i) => s + i.amount, 0) }
})

function formatRate(r: number): string { if (r == null) return '--'; return (r * 100).toFixed(4) + '%' }
function formatDate(d: string): string { if (!d) return '--'; const parts = d.split(/[-T ]/); return parts.length >= 3 ? parts[0] + '-' + parts[1] + '-' + parts[2] : d }

async function fetchSettlementInfo() {
  loading.value = true
  try {
    const { data } = await fetchWithCache<HKSettlementInfo>('hk_settlement_info', () => (window as any).go.main.App.GetHKSettlementInfo(), 3600000)
    settlementInfo.value = data || null
  } catch { settlementInfo.value = null }
  finally { loading.value = false }
}

async function fetchCalendar() {
  calendarLoading.value = true
  try {
    const { data } = await fetchWithCache<CalendarResult>('hk_calendar:' + calendarYear.value, () => (window as any).go.main.App.GetHKTradeCalendar(calendarYear.value), 3600000)
    calendarEntries.value = [...(data?.trading_days || []), ...(data?.holidays || [])]
  } catch { calendarEntries.value = [] }
  finally { calendarLoading.value = false }
}

function switchTab(key: string) { activeTab.value = key as typeof activeTab.value }

onMounted(() => { fetchSettlementInfo() })

const tabs = [
  { key: 'settlement_rules', label: t('panels.settlement_rules') },
  { key: 'fee_calculator', label: t('panels.fee_calculator') },
  { key: 'trade_calendar', label: t('panels.trade_calendar') },
]
</script>

<template>
  <div class="hk-settlement-panel">
    <PanelHeader :tabs="tabs" :active-tab="activeTab" @tab-change="switchTab" />

    <LoadingState v-if="loading" type="card" :rows="3" />
    <ErrorState v-else-if="!loading && !settlementInfo && activeTab !== 'trade_calendar'" description="结算信息加载失败" @retry="fetchSettlementInfo" />

    <template v-else-if="activeTab === 'settlement_rules' && settlementInfo">
      <div class="settlement-timeline">
        <div class="timeline-item"><span class="timeline-label">T</span><span class="timeline-desc">{{ t('common.trade_date') }}</span></div>
        <div class="timeline-arrow">→</div>
        <div class="timeline-item"><span class="timeline-label">T+{{ settlementInfo.settlement_days }}</span><span class="timeline-desc">{{ t('common.settlement_date') }}</span></div>
      </div>
      <div class="fee-table">
        <div class="fee-table-header"><span class="fee-col-name">{{ t('common.fee_type') }}</span><span class="fee-col-rate">{{ t('common.rate') }}</span></div>
        <div class="fee-table-body">
          <div class="fee-row"><span class="fee-col-name">{{ t('common.stamp_duty') }}</span><span class="fee-col-rate">{{ formatRate(settlementInfo.stamp_duty) }}</span></div>
          <div class="fee-row"><span class="fee-col-name">{{ t('common.exchange_fee') }}</span><span class="fee-col-rate">{{ formatRate(settlementInfo.exchange_fee) }}</span></div>
          <div class="fee-row"><span class="fee-col-name">{{ t('common.sfc_levy') }}</span><span class="fee-col-rate">{{ formatRate(settlementInfo.sfc_levy) }}</span></div>
          <div class="fee-row"><span class="fee-col-name">{{ t('common.trading_fee') }}</span><span class="fee-col-rate">{{ formatRate(settlementInfo.trading_fee) }}</span></div>
          <div class="fee-row"><span class="fee-col-name">{{ t('common.frc_levy') }}</span><span class="fee-col-rate">{{ formatRate(settlementInfo.frc_levy) }}</span></div>
        </div>
      </div>
      <div class="rule-cards">
        <div class="rule-card"><span class="rule-label">{{ t('common.settlement_mode') }}</span><span class="rule-value">T+{{ settlementInfo.settlement_days }}</span></div>
        <div class="rule-card"><span class="rule-label">{{ t('common.lot_size') }}</span><span class="rule-value">{{ settlementInfo.lot_size_min }}</span></div>
        <div class="rule-card"><span class="rule-label">{{ t('common.currency') }}</span><span class="rule-value">{{ settlementInfo.currency }}</span></div>
      </div>
      <div v-if="settlementInfo.description" class="description">{{ settlementInfo.description }}</div>
    </template>

    <template v-else-if="activeTab === 'fee_calculator' && settlementInfo">
      <div class="calculator-section">
        <div class="input-row"><label class="input-label">{{ t('panels.trade_amount') }} (HKD)</label><input v-model.number="tradeAmount" type="number" class="amount-input" min="0" /></div>
        <div class="fee-table">
          <div class="fee-table-header"><span class="fee-col-name">{{ t('common.fee_type') }}</span><span class="fee-col-rate">{{ t('common.rate') }}</span><span class="fee-col-amount">{{ t('common.amount') }}</span></div>
          <div class="fee-table-body">
            <div v-for="fee in fees.items" :key="fee.label" class="fee-row"><span class="fee-col-name">{{ fee.label }}</span><span class="fee-col-rate">{{ formatRate(fee.rate) }}</span><span class="fee-col-amount">{{ fee.amount.toFixed(2) }}</span></div>
          </div>
        </div>
        <div class="total-row"><span class="total-label">{{ t('panels.total_fees') }}</span><span class="total-amount">{{ fees.total.toFixed(2) }} HKD</span></div>
      </div>
    </template>

    <template v-else-if="activeTab === 'trade_calendar'">
      <div class="calendar-controls">
        <input v-model.number="calendarYear" type="number" class="year-input" min="2000" max="2100" @change="fetchCalendar" />
        <select v-model.number="calendarMonth" class="month-select"><option :value="0">{{ t('common.all') }}</option><option v-for="m in 12" :key="m" :value="m">{{ m }}</option></select>
        <button class="btn btn-sm" @click="fetchCalendar" :disabled="calendarLoading">⟳</button>
      </div>
      <LoadingState v-if="calendarLoading" type="table" :rows="8" />
      <EmptyState v-else-if="filteredCalendar.length === 0" title="暂无数据" />
      <div v-else class="table-wrapper">
        <div class="table-header"><span class="col-date">{{ t('common.date') }}</span><span class="col-info">{{ t('common.remarks') }}</span></div>
        <div class="table-body">
          <div v-for="(entry, idx) in filteredCalendar" :key="idx" class="table-row"><span class="col-date">{{ formatDate(entry.trade_date || entry['日期']) }}</span><span class="col-info">{{ entry.remarks || entry['备注'] || entry.holiday || entry['节日'] || '--' }}</span></div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.hk-settlement-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.settlement-timeline { display: flex; align-items: center; gap: var(--space-md); padding: var(--space-lg); margin-bottom: var(--space-md); background: var(--color-bg-elevated); border-radius: var(--radius-md); flex-shrink: 0; }
.timeline-item { display: flex; flex-direction: column; align-items: center; gap: var(--space-xs); }
.timeline-label { font-size: var(--font-xl); font-weight: 700; color: var(--color-accent); }
.timeline-desc { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.timeline-arrow { font-size: var(--font-xl); color: var(--color-text-tertiary); }
.fee-table { margin-bottom: var(--space-md); flex-shrink: 0; }
.fee-table-header { display: flex; padding: var(--space-xs) 0; border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.fee-table-body { font-size: var(--font-xs); }
.fee-row { display: flex; padding: var(--space-xs) 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.fee-row:hover { background: var(--color-bg-hover); }
.fee-col-name { flex: 1; min-width: 0; }
.fee-col-rate { width: 100px; text-align: right; font-variant-numeric: tabular-nums; }
.fee-col-amount { width: 100px; text-align: right; font-variant-numeric: tabular-nums; }
.rule-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: var(--space-sm); margin-bottom: var(--space-md); flex-shrink: 0; }
.rule-card { display: flex; flex-direction: column; gap: var(--space-xs); padding: var(--space-md); background: var(--color-bg-elevated); border-radius: var(--radius-md); }
.rule-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.rule-value { font-size: var(--font-lg); font-weight: 600; color: var(--color-accent); }
.description { font-size: var(--font-xs); color: var(--color-text-secondary); line-height: 1.5; }
.calculator-section { display: flex; flex-direction: column; gap: var(--space-md); padding: var(--space-sm) var(--panel-padding); }
.input-row { display: flex; align-items: center; gap: var(--space-sm); flex-shrink: 0; }
.input-label { font-size: var(--font-xs); color: var(--color-text-secondary); white-space: nowrap; }
.amount-input { padding: var(--space-xs) var(--space-sm); font-size: var(--font-sm); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); width: 160px; font-variant-numeric: tabular-nums; }
.total-row { display: flex; justify-content: flex-end; align-items: center; gap: var(--space-md); padding: var(--space-sm) 0; border-top: 1px solid var(--color-border-strong); }
.total-label { font-size: var(--font-xs); font-weight: 600; color: var(--color-text-primary); }
.total-amount { font-size: var(--font-lg); font-weight: 700; color: var(--color-accent); }
.calendar-controls { display: flex; gap: var(--space-sm); align-items: center; margin-bottom: var(--space-sm); flex-shrink: 0; padding: 0 var(--panel-padding); }
.year-input { padding: var(--space-xs) var(--space-sm); font-size: var(--font-xs); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); width: 70px; }
.month-select { padding: var(--space-xs) var(--space-sm); font-size: var(--font-xs); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; margin: 0 var(--panel-padding); }
.table-header { display: flex; padding: var(--space-xs) 0; border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: var(--font-xs); }
.table-row { display: flex; padding: var(--space-xs) 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-hover); }
.col-date { width: 100px; }
.col-info { flex: 1; min-width: 0; color: var(--color-text-secondary); }
</style>
