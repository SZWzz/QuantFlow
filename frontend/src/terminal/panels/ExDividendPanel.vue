<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, EmptyState, ErrorState, LoadingState } from '@/terminal/components/panel'
import { useWailsApp } from '@/lib/composables/useWailsApp'

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
const app = useWailsApp()
const activeTab = ref<Tab>('today')
const data = ref<ExDividendStock[]>([])
const loading = ref(false)
const loadError = ref('')

const tabs = computed(() => [
  { key: 'today', label: t('panels.today_ex') },
  { key: 'week', label: t('panels.this_week_ex') },
  { key: 'month', label: t('panels.this_month_ex') },
])

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

function onTabChange(key: string) {
  activeTab.value = key as Tab
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
    <PanelHeader
      :title="t('panels.ex_dividend')"
      :tabs="tabs"
      :active-tab="activeTab"
      @tab-change="onTabChange"
    />

    <ErrorState v-if="loadError" :description="loadError" @retry="fetchData" />
    <LoadingState v-else-if="loading && data.length === 0" type="table" :rows="6" :cols="8" />
    <EmptyState v-else-if="data.length === 0" :title="t('panels.no_data')" />

    <!-- 保留自绘表格：代码列为单元格级符号联动点击、股息率 >3% 阈值高亮，PanelTable 均无法表达 -->
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
.ex-dividend-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex;
  padding: var(--space-xs) var(--panel-padding);
  border-bottom: 1px solid var(--color-border);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: var(--font-xs); padding: 0 var(--panel-padding); }
.table-row {
  display: flex;
  padding: var(--space-xs) 0;
  align-items: center;
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
