<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import ErrorBoundary from '@/terminal/components/ErrorBoundary.vue'

const { t } = useI18n()

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

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

type TabKey = 'subscribing' | 'upcoming_listing' | 'recent_perf'

const activeTab = ref<TabKey>('subscribing')
const subscriptionData = ref<HKIPOItem[]>([])
const listingData = ref<HKIPOItem[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const year = ref(new Date().getFullYear())

const hasData = computed(() => subscriptionData.value.length > 0 || listingData.value.length > 0)

const filteredData = computed(() => {
  const now = new Date()
  switch (activeTab.value) {
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

function toDateStr(d: string): string {
  if (!d) return '--'
  return d.slice(0, 10)
}

function normalizeItem(r: any): HKIPOItem {
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

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetHKIPOCalendar) return
  loading.value = true
  error.value = null
  try {
    const result = await app.GetHKIPOCalendar(year.value)
    const subRaw = result?.subscription?.data || []
    const listRaw = result?.listing?.data || []
    subscriptionData.value = subRaw.map((r: any) => normalizeItem(r))
    listingData.value = listRaw.map((r: any) => normalizeItem(r))
  } catch (e: any) {
    console.error('[HKIPOPanel]', e)
    error.value = e.message || String(e)
    subscriptionData.value = []
    listingData.value = []
  } finally {
    loading.value = false
  }
}

function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function switchTab(tab: TabKey) {
  activeTab.value = tab
}

function formatPrice(v: number): string {
  if (!v) return '--'
  return v.toFixed(2)
}

function formatMoney(v: number): string {
  if (!v) return '--'
  return 'HK$' + v.toLocaleString('zh-HK')
}

function formatMultiple(v: number): string {
  if (!v) return '--'
  return v.toFixed(2) + 'x'
}

function formatLotteryRate(v: number): string {
  if (!v) return '--'
  return v.toFixed(2) + '%'
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <ErrorBoundary :panel-id="panelId">
    <div class="hk-ipo-panel">
      <div class="panel-header">
        <h3>{{ t('misc.hk_ipo') }}</h3>
        <div class="header-tabs">
          <button :class="['tab', { active: activeTab === 'subscribing' }]" @click="switchTab('subscribing')">{{ t('misc.subscribing') }}</button>
          <button :class="['tab', { active: activeTab === 'upcoming_listing' }]" @click="switchTab('upcoming_listing')">{{ t('misc.upcoming_listing') }}</button>
          <button :class="['tab', { active: activeTab === 'recent_perf' }]" @click="switchTab('recent_perf')">{{ t('misc.recent_perf') }}</button>
        </div>
        <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
      </div>

      <div v-if="error" class="error-state">
        <span class="error-icon">⚠</span>
        <span class="error-text">{{ error }}</span>
        <button class="retry-btn" @click="fetchData">{{ t('common.retry') }}</button>
      </div>

      <SkeletonPanel v-else-if="loading && !hasData" type="table" :rows="8" />

      <div v-else-if="filteredData.length === 0" class="empty-state">
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
          <div v-for="(row, idx) in filteredData" :key="row.股票代码 + (row.上市日期 || idx)" class="table-row">
            <span class="col-code clickable" @click="onSymbolClick(row.股票代码)">{{ row.股票代码 }}</span>
            <span class="col-name">{{ row.名称 }}</span>
            <span class="col-offer-price">{{ formatPrice(row.招股价 || row.发行价) }}</span>
            <span class="col-entry-fee">{{ formatMoney(row.入场费) }}</span>
            <span class="col-lot-size">{{ row.每手股数 || '--' }}</span>
            <span class="col-sub-date">{{ toDateStr(row.招股日期) }}</span>
            <span class="col-list-date">{{ toDateStr(row.上市日期) }}</span>
            <span class="col-multiple">{{ formatMultiple(row.认购倍数) }}</span>
            <span class="col-lottery">{{ formatLotteryRate(row.一手中签率) }}</span>
          </div>
        </div>
      </div>
    </div>
  </ErrorBoundary>
</template>

<style scoped>
.hk-ipo-panel {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg-panel, #1a1a2e);
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
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.header-tabs .tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.refresh-btn {
  margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.error-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
}
.error-icon { font-size: 24px; }
.error-text { max-width: 300px; text-align: center; word-break: break-all; color: var(--color-warning, #f59e0b); }
.retry-btn {
  padding: 6px 16px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 12px;
}
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  color: var(--color-text-tertiary); font-size: 13px;
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
.col-code.clickable { cursor: pointer; color: #60a5fa; }
.col-name { width: 56px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-offer-price { width: 64px; text-align: right; }
.col-entry-fee { width: 72px; text-align: right; }
.col-lot-size { width: 52px; text-align: right; }
.col-sub-date { width: 80px; text-align: center; }
.col-list-date { width: 80px; text-align: center; }
.col-multiple { width: 60px; text-align: right; }
.col-lottery { flex: 1; min-width: 0; text-align: right; }
</style>
