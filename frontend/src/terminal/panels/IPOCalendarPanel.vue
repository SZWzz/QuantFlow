<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import ErrorBoundary from '@/terminal/components/ErrorBoundary.vue'

const { t } = useI18n()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// ── Market selector ──
type Market = 'CN' | 'HK'
const market = ref<Market>('CN')

// ── CN (A-share) IPO ──
interface IPOItem {
  code: string
  name: string
  issue_price: number
  pe: number
  subscription_date: string
  listing_date: string
  lottery_rate: number
  issue_volume: number
  status: string
}

type CNTabKey = 'today_apply' | 'upcoming' | 'recent'
const activeTab = ref<CNTabKey>('today_apply')
const allData = ref<IPOItem[]>([])
const cnLoading = ref(false)
const cnError = ref<string | null>(null)
let cnTimer: ReturnType<typeof setInterval> | null = null

// ── HK IPO ──
interface HKIPOItem {
  股票代码: string
  名称: string
  招股价: number
  发行价: number
  入场费: number
  每手股数: number
  招股日期: string
  上市日期: string
  认购倍数: number
  一手中签率: number
}

type HKTabKey = 'subscribing' | 'upcoming_listing' | 'recent_perf'
const hkActiveTab = ref<HKTabKey>('subscribing')
const subscriptionData = ref<HKIPOItem[]>([])
const listingData = ref<HKIPOItem[]>([])
const hkLoading = ref(false)
const hkError = ref<string | null>(null)
const hkYear = ref(new Date().getFullYear())

const { fetchWithCache } = usePanelCache()

// ── CN helpers ──
function toDateStr(d: string): string {
  return d ? new Date(d).toLocaleDateString('zh-CN') : '--'
}

function toLocalDate(d: string): Date {
  const parts = d.split('-').map(Number)
  return new Date(parts[0], parts[1] - 1, parts[2])
}

function todayStr(): string {
  const d = new Date()
  return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
}

function inDays(d: Date, n: number): Date {
  return new Date(d.getTime() + n * 86400000)
}

const cnFiltered = computed(() => {
  const today = new Date()
  const todayLocal = todayStr()
  switch (activeTab.value) {
    case 'today_apply':
      return allData.value.filter(item => item.subscription_date === todayLocal)
    case 'upcoming': {
      const limit = inDays(today, 14)
      return allData.value.filter(item => {
        if (!item.listing_date) return false
        const ld = toLocalDate(item.listing_date)
        return ld >= today && ld <= limit
      })
    }
    case 'recent': {
      const past = inDays(today, -7)
      return allData.value.filter(item => {
        if (!item.listing_date) return false
        const ld = toLocalDate(item.listing_date)
        return ld >= past && ld <= today
      })
    }
    default:
      return []
  }
})

async function fetchCNData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetIPOCalendar) return
  cnLoading.value = true
  cnError.value = null
  try {
    const start = inDays(new Date(), -30).toISOString().slice(0, 10)
    const end = inDays(new Date(), 30).toISOString().slice(0, 10)
    const { data: result } = await fetchWithCache<any>(`ipo_calendar:${start}:${end}`, () => app.GetIPOCalendar(start, end))
    allData.value = (result || []).map((r: any) => ({
      code: r.code || '',
      name: r.name || '',
      issue_price: r.issue_price || 0,
      pe: r.pe || 0,
      subscription_date: r.subscription_date || '',
      listing_date: r.listing_date || '',
      lottery_rate: r.lottery_rate || 0,
      issue_volume: r.issue_volume || 0,
      status: r.status || '',
    }))
  } catch (e: any) {
    console.error('[IPOCalendar]', e)
    cnError.value = e.message || String(e)
    allData.value = []
  } finally {
    cnLoading.value = false
  }
}

// ── HK helpers ──
function hkToDateStr(d: string): string {
  if (!d) return '--'
  return d.slice(0, 10)
}

function normalizeHKItem(r: any): HKIPOItem {
  return {
    股票代码: r.股票代码 || r.code || '',
    名称: r.名称 || r.name || '',
    招股价: r.招股价 || r.offer_price || 0,
    发行价: r.发行价 || r.issue_price || 0,
    入场费: r.入场费 || r.entry_fee || 0,
    每手股数: r.每手股数 || r.lot_size || 0,
    招股日期: r.招股日期 || r.subscription_date || '',
    上市日期: r.上市日期 || r.listing_date || '',
    认购倍数: r.认购倍数 || r.subscription_multiple || 0,
    一手中签率: r.一手中签率 || r.lottery_rate || 0,
  }
}

const hkHasData = computed(() => subscriptionData.value.length > 0 || listingData.value.length > 0)

const hkFiltered = computed(() => {
  const now = new Date()
  switch (hkActiveTab.value) {
    case 'subscribing':
      return subscriptionData.value.filter(item => {
        if (!item.招股日期) return false
        const end = new Date(item.招股日期)
        return end >= now
      })
    case 'upcoming_listing':
      return listingData.value.filter(item => {
        if (!item.上市日期) return false
        const ld = new Date(item.上市日期)
        return ld >= now
      })
    case 'recent_perf': {
      const past = new Date()
      past.setDate(past.getDate() - 30)
      return listingData.value.filter(item => {
        if (!item.上市日期) return false
        const ld = new Date(item.上市日期)
        return ld >= past && ld <= now
      })
    }
    default:
      return []
  }
})

async function fetchHKData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetHKIPOCalendar) return
  hkLoading.value = true
  hkError.value = null
  try {
    const { data: result } = await fetchWithCache<any>(`hk_ipo:${hkYear.value}`, () => app.GetHKIPOCalendar(hkYear.value), 15 * 60 * 1000)
    const subRaw = result?.subscription?.data || []
    const listRaw = result?.listing?.data || []
    subscriptionData.value = subRaw.map((r: any) => normalizeHKItem(r))
    listingData.value = listRaw.map((r: any) => normalizeHKItem(r))
  } catch (e: any) {
    console.error('[HKIPOPanel]', e)
    hkError.value = e.message || String(e)
    subscriptionData.value = []
    listingData.value = []
  } finally {
    hkLoading.value = false
  }
}

// ── Shared ──
function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function switchTab(tab: CNTabKey) {
  activeTab.value = tab
}

function switchHKTab(tab: HKTabKey) {
  hkActiveTab.value = tab
}

function formatLotteryRate(rate: number): string {
  if (!rate) return '--'
  return (rate * 100).toFixed(3) + '%'
}

function formatVolume(v: number): string {
  if (!v) return '--'
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return String(v)
}

function formatHKPrice(v: number): string {
  if (!v) return '--'
  return v.toFixed(2)
}

function formatHKMoney(v: number): string {
  if (!v) return '--'
  return 'HK$' + v.toLocaleString('zh-HK')
}

function formatMultiple(v: number): string {
  if (!v) return '--'
  return v.toFixed(2) + 'x'
}

function formatHKLotteryRate(v: number): string {
  if (!v) return '--'
  return v.toFixed(2) + '%'
}

const loading = computed(() => market.value === 'CN' ? cnLoading.value : hkLoading.value)

onMounted(() => {
  fetchCNData()
  cnTimer = setInterval(fetchCNData, 60000)
})

onUnmounted(() => {
  if (cnTimer) clearInterval(cnTimer)
})
</script>

<template>
  <ErrorBoundary :panel-id="panelId">
    <div class="ipo-calendar-panel">
      <div class="panel-header">
        <h3>{{ $t('panels.ipo_calendar') }}</h3>
        <!-- Market selector -->
        <div class="market-selector">
          <button :class="['market-tab', { active: market === 'CN' }]" @click="market = 'CN'">A股</button>
          <button :class="['market-tab', { active: market === 'HK' }]" @click="market = 'HK'; if (!subscriptionData.length && !listingData.length) fetchHKData()">港股</button>
        </div>
        <!-- CN tabs -->
        <div v-if="market === 'CN'" class="header-tabs">
          <button :class="['tab', { active: activeTab === 'today_apply' }]" @click="switchTab('today_apply')">{{ $t('panels.today_apply') }}</button>
          <button :class="['tab', { active: activeTab === 'upcoming' }]" @click="switchTab('upcoming')">{{ $t('panels.upcoming') }}</button>
          <button :class="['tab', { active: activeTab === 'recent' }]" @click="switchTab('recent')">{{ $t('panels.recent') }}</button>
        </div>
        <!-- HK tabs -->
        <div v-if="market === 'HK'" class="header-tabs">
          <button :class="['tab', { active: hkActiveTab === 'subscribing' }]" @click="switchHKTab('subscribing')">{{ t('misc.subscribing') }}</button>
          <button :class="['tab', { active: hkActiveTab === 'upcoming_listing' }]" @click="switchHKTab('upcoming_listing')">{{ t('misc.upcoming_listing') }}</button>
          <button :class="['tab', { active: hkActiveTab === 'recent_perf' }]" @click="switchHKTab('recent_perf')">{{ t('misc.recent_perf') }}</button>
        </div>
        <button class="refresh-btn" @click="market === 'CN' ? fetchCNData() : fetchHKData()" :disabled="loading">⟳</button>
      </div>

      <!-- ── CN content ── -->
      <template v-if="market === 'CN'">
        <div v-if="cnError" class="error-state">
          <span class="error-icon">⚠</span>
          <span class="error-text">{{ cnError }}</span>
          <button class="retry-btn" @click="fetchCNData">{{ $t('common.retry') }}</button>
        </div>

        <SkeletonPanel v-else-if="cnLoading && !allData.length" type="table" :rows="8" />

        <div v-else-if="cnFiltered.length === 0" class="empty-state">
          <span class="empty-icon">📋</span>
          <span>{{ $t('panels.no_data') }}</span>
        </div>

        <div v-else class="table-wrapper">
          <div class="table-header">
            <span class="col-code">{{ $t('common.symbol') }}</span>
            <span class="col-name">{{ $t('common.name') }}</span>
            <span class="col-price">{{ $t('common.price') }}</span>
            <span class="col-pe">PE</span>
            <span class="col-sub-date">{{ $t('common.subscription_date') }}</span>
            <span class="col-list-date">{{ $t('common.listing_date') }}</span>
            <span class="col-lottery">{{ $t('panels.lottery_rate') }}</span>
            <span class="col-status">{{ $t('common.status') }}</span>
          </div>
          <div class="table-body">
            <div v-for="row in cnFiltered" :key="row.code + row.subscription_date" class="table-row">
              <span class="col-code clickable" @click="onSymbolClick(row.code)">{{ row.code }}</span>
              <span class="col-name">{{ row.name }}</span>
              <span class="col-price">{{ row.issue_price ? row.issue_price.toFixed(2) : '--' }}</span>
              <span class="col-pe">{{ row.pe ? row.pe.toFixed(2) : '--' }}</span>
              <span class="col-sub-date">{{ toDateStr(row.subscription_date) }}</span>
              <span class="col-list-date">{{ toDateStr(row.listing_date) }}</span>
              <span class="col-lottery">{{ formatLotteryRate(row.lottery_rate) }}</span>
              <span class="col-status">{{ row.status || '--' }}</span>
            </div>
          </div>
        </div>
      </template>

      <!-- ── HK content ── -->
      <template v-if="market === 'HK'">
        <div v-if="hkError" class="error-state">
          <span class="error-icon">⚠</span>
          <span class="error-text">{{ hkError }}</span>
          <button class="retry-btn" @click="fetchHKData">{{ t('common.retry') }}</button>
        </div>

        <SkeletonPanel v-else-if="hkLoading && !hkHasData" type="table" :rows="8" />

        <div v-else-if="hkFiltered.length === 0" class="empty-state">
          <span>{{ t('misc.no_ipo_data') }}</span>
        </div>

        <div v-else class="table-wrapper">
          <div class="table-header">
            <span class="col-code">代码</span>
            <span class="col-name">名称</span>
            <span class="col-offer-price">招股价/发行价</span>
            <span class="col-entry-fee">入场费</span>
            <span class="col-lot-size">每手股数</span>
            <span class="col-sub-date">招股日期</span>
            <span class="col-list-date">上市日期</span>
            <span class="col-multiple">认购倍数</span>
            <span class="col-lottery">一手中签率</span>
          </div>
          <div class="table-body">
            <div v-for="(row, idx) in hkFiltered" :key="row.股票代码 + (row.上市日期 || idx)" class="table-row">
              <span class="col-code clickable" @click="onSymbolClick(row.股票代码)">{{ row.股票代码 }}</span>
              <span class="col-name">{{ row.名称 }}</span>
              <span class="col-offer-price">{{ formatHKPrice(row.招股价 || row.发行价) }}</span>
              <span class="col-entry-fee">{{ formatHKMoney(row.入场费) }}</span>
              <span class="col-lot-size">{{ row.每手股数 || '--' }}</span>
              <span class="col-sub-date">{{ hkToDateStr(row.招股日期) }}</span>
              <span class="col-list-date">{{ hkToDateStr(row.上市日期) }}</span>
              <span class="col-multiple">{{ formatMultiple(row.认购倍数) }}</span>
              <span class="col-lottery">{{ formatHKLotteryRate(row.一手中签率) }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </ErrorBoundary>
</template>

<style scoped>
.ipo-calendar-panel {
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
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; white-space: nowrap; }

/* Market selector */
.market-selector {
  display: flex;
  gap: 0;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.market-tab {
  padding: 2px 10px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 11px;
  font-weight: 500;
}
.market-tab + .market-tab { border-left: 1px solid var(--color-border-strong); }
.market-tab.active { color: var(--color-accent); background: rgba(59,130,246,0.1); }

.header-tabs { display: flex; gap: 4px; }
.header-tabs .tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.refresh-btn {
  margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
}
.error-icon { font-size: 24px; }
.error-text { max-width: 300px; text-align: center; word-break: break-all; color: var(--color-warning, var(--color-accent)); }
.retry-btn {
  padding: 6px 16px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 12px;
}
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
}
.empty-icon { font-size: 24px; }
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
.col-code { width: 72px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 64px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-price { width: 56px; text-align: right; }
.col-pe { width: 52px; text-align: right; }
.col-sub-date { width: 80px; text-align: center; }
.col-list-date { width: 80px; text-align: center; }
.col-lottery { width: 64px; text-align: right; }
.col-status { flex: 1; min-width: 0; text-align: center; color: var(--color-text-secondary); }

/* HK columns */
.col-offer-price { width: 64px; text-align: right; }
.col-entry-fee { width: 72px; text-align: right; }
.col-lot-size { width: 52px; text-align: right; }
.col-multiple { width: 60px; text-align: right; }
</style>
