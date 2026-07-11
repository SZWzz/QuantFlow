<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface ExDividendStock {
  code: string
  name: string
  ex_date: string
  bonus_rmb: number
  transfer_ratio: number
  bonus_ratio: number
  plan: string
  dividend_yield: number
  close_price: number
}

type Tab = 'today' | 'week' | 'month'

const { fetchWithCache } = usePanelCache()
const app = (window as any).go?.main?.App
const activeTab = ref<Tab>('today')
const data = ref<ExDividendStock[]>([])
const loading = ref(false)
const loadError = ref('')

function pad(n: number): string {
  return n.toString().padStart(2, '0')
}

function getDateRange(tab: Tab): [string, string] {
  const now = new Date()
  const y = now.getFullYear()
  const m = now.getMonth()
  const d = now.getDate()
  switch (tab) {
    case 'today':
      return [`${y}-${pad(m + 1)}-${pad(d)}`, `${y}-${pad(m + 1)}-${pad(d)}`]
    case 'week': {
      const day = now.getDay()
      const diff = day === 0 ? -6 : 1 - day
      const mon = new Date(y, m, d + diff)
      const sun = new Date(y, m, d + diff + 6)
      return [
        `${mon.getFullYear()}-${pad(mon.getMonth() + 1)}-${pad(mon.getDate())}`,
        `${sun.getFullYear()}-${pad(sun.getMonth() + 1)}-${pad(sun.getDate())}`,
      ]
    }
    case 'month': {
      const last = new Date(y, m + 1, 0)
      return [
        `${y}-${pad(m + 1)}-01`,
        `${y}-${pad(m + 1)}-${pad(last.getDate())}`,
      ]
    }
  }
}

async function fetchData() {
  if (!app?.GetExDividendCalendar) return
  loading.value = true
  loadError.value = ''
  try {
    const [start, end] = getDateRange(activeTab.value)
    const { data: result } = await fetchWithCache(`ex_dividend:${start}:${end}`, () => app.GetExDividendCalendar(start, end))
    const raw = Array.isArray(result) ? result : (result ? [result] : [])
    data.value = raw.map((s: any) => ({
      code: s.code || '',
      name: s.name || '',
      ex_date: s.ex_date || '',
      bonus_rmb: s.bonus_rmb || 0,
      transfer_ratio: s.transfer_ratio || 0,
      bonus_ratio: s.bonus_ratio || 0,
      plan: s.plan || '',
      dividend_yield: s.dividend_yield || 0,
      close_price: s.close_price || 0,
    }))
  } catch (e: any) {
    console.error('[ExDividend]', e)
    loadError.value = e?.message || String(e)
    data.value = []
  } finally {
    loading.value = false
  }
}

function switchTab(tab: Tab) {
  activeTab.value = tab
  fetchData()
}

function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function formatBonus(v: number): string {
  return '¥' + v.toFixed(2)
}

function formatYield(v: number): string {
  return v.toFixed(2) + '%'
}

onMounted(fetchData)
</script>

<template>
  <div class="ex-dividend-panel">
    <div class="panel-header">
      <h3>{{ t('panels.ex_dividend') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'today' }]" @click="switchTab('today')">{{ t('panels.today_ex') }}</button>
        <button :class="['tab', { active: activeTab === 'week' }]" @click="switchTab('week')">{{ t('panels.this_week_ex') }}</button>
        <button :class="['tab', { active: activeTab === 'month' }]" @click="switchTab('month')">{{ t('panels.this_month_ex') }}</button>
      </div>
    </div>

    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <SkeletonPanel v-if="loading && data.length === 0" type="table" :rows="6" />

    <div v-else-if="data.length === 0" class="empty-state">
      <span>{{ t('panels.no_data') }}</span>
    </div>

    <div v-else class="table-wrapper">
      <div class="table-header">
        <span class="col-code">{{ t('common.symbol') }}</span>
        <span class="col-name">{{ t('common.name') }}</span>
        <span class="col-date">{{ t('common.date') }}</span>
        <span class="col-bonus">每股派息</span>
        <span class="col-transfer">转增</span>
        <span class="col-bonus-ratio">送股</span>
        <span class="col-yield">{{ t('panels.dividend_yield_col') }}</span>
        <span class="col-plan">进度</span>
      </div>
      <div class="table-body">
        <div v-for="s in data" :key="s.code" class="table-row">
          <span class="col-code clickable" @click="onSymbolClick(s.code)">{{ s.code }}</span>
          <span class="col-name" :title="s.name">{{ s.name }}</span>
          <span class="col-date">{{ s.ex_date }}</span>
          <span class="col-bonus">{{ formatBonus(s.bonus_rmb) }}</span>
          <span class="col-transfer">{{ s.transfer_ratio }}</span>
          <span class="col-bonus-ratio">{{ s.bonus_ratio }}</span>
          <span class="col-yield" :class="{ highlight: s.dividend_yield > 3 }">{{ formatYield(s.dividend_yield) }}</span>
          <span class="col-plan" :title="s.plan">{{ s.plan }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: var(--color-up-soft); color: var(--color-up); font-size: 12px; }
.ex-dividend-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, var(--color-border));
  background: var(--color-bg-panel, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-shrink: 0;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px; gap: 6px;
}
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover { background: var(--color-bg-elevated); }
.col-code { width: 64px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 60px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-date { width: 80px; }
.col-bonus { width: 60px; text-align: right; }
.col-transfer { width: 50px; text-align: right; }
.col-bonus-ratio { width: 50px; text-align: right; }
.col-yield { width: 60px; text-align: right; font-weight: 500; }
.col-yield.highlight { color: var(--color-down); }
.col-plan { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-text-secondary); }
</style>
