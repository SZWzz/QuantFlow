<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, LoadingState, ErrorState, EmptyState } from '@/terminal/components/panel'
import ErrorBoundary from '@/terminal/components/ErrorBoundary.vue'

const { t } = useI18n()
const app = useWailsApp()

type Market = 'CN' | 'HK'
interface IPOItem {
  code: string; name: string; issue_price: number; pe: number;
  subscription_date: string; listing_date: string; lottery_rate: number; status: string;
}
type CNTabKey = 'today_apply' | 'upcoming' | 'recent'
interface HKIPOItem {
  股票代码: string; 名称: string; 招股价: number; 发行价: number; 入场费: number;
  每手股数: number; 招股日期: string; 上市日期: string; 认购倍数: number; 一手中签率: number;
}
type HKTabKey = 'subscribing' | 'upcoming_listing' | 'recent_perf'
function toDateStr(d: string): string { if (!d) return '--'; const parts = d.split(/[-T ]/); return parts.length >= 3 ? parts[0] + '-' + parts[1] + '-' + parts[2] : d }
function formatLotteryRate(r: number): string { if (r == null) return '--'; return (r * 100).toFixed(4) + '%' }
function hkToDateStr(d: string): string { if (!d) return '--'; const m = d.match(/(\d{4})-(\d{2})-(\d{2})/); return m ? m[2] + '-' + m[3] : d }
function formatHKPrice(v: number): string { if (v == null) return '--'; return v.toFixed(2) + ' HKD' }
function formatHKMoney(v: number): string { if (v == null) return '--'; return v.toFixed(0) + ' HKD' }
function formatMultiple(v: number): string { if (v == null) return '--'; return v.toFixed(1) + 'x' }
function formatHKLotteryRate(v: number): string { if (v == null) return '--'; return (v * 100).toFixed(1) + '%' }

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const market = ref<Market>('CN')
const activeTab = ref<CNTabKey>('today_apply')
const hkActiveTab = ref<HKTabKey>('subscribing')
const { fetchWithCache } = usePanelCache()
const cnLoading = ref(false); const cnError = ref('')
const allData = ref<IPOItem[]>([]); const cnFiltered = computed(() => allData.value.filter(r => r.status === activeTab.value))
const subscriptionData = ref<HKIPOItem[]>([]); const listingData = ref<HKIPOItem[]>([]); const recentPerfData = ref<HKIPOItem[]>([])
const hkLoading = ref(false); const hkError = ref('')
const hkHasData = computed(() => subscriptionData.value.length + listingData.value.length + recentPerfData.value.length > 0)
const hkFiltered = computed(() => hkActiveTab.value === 'subscribing' ? subscriptionData.value : hkActiveTab.value === 'upcoming_listing' ? listingData.value : recentPerfData.value)

const loading = computed(() => market.value === 'CN' ? cnLoading.value : hkLoading.value)
function doFetch() { if (market.value === 'CN') fetchCNData(); else fetchHKData() }

async function fetchCNData() { cnLoading.value = true; cnError.value = ''; try { const key = 'ipo_cn_data'; const { data } = await fetchWithCache<IPOItem[]>(key, () => app!.GetIPOData('CN') as Promise<IPOItem[]>, 1800000); allData.value = data || [] } catch (e: any) { cnError.value = e?.message || String(e); allData.value = [] } finally { cnLoading.value = false } }
async function fetchHKData() { hkLoading.value = true; hkError.value = ''; try { if (app?.GetHKIPOCalendar) { const { data } = await fetchWithCache<any>('ipo_hk_data', () => app.GetHKIPOCalendar(new Date().getFullYear()), 1800000); subscriptionData.value = data?.subscription || []; listingData.value = data?.listing || []; recentPerfData.value = data?.recent || [] } } catch (e: any) { hkError.value = e?.message || String(e) } finally { hkLoading.value = false } }
function switchTab(key: string) { activeTab.value = key as CNTabKey }
function switchHKTab(key: string) { hkActiveTab.value = key as HKTabKey }
function onSymbolClick(code: string) { ctx.setGroupSymbol(ctx.getOrCreatePanelGroup(props.panelId).groupId, code) }

onMounted(fetchCNData)
</script>

<template>
  <ErrorBoundary :panel-id="panelId">
    <div class="ipo-calendar-panel">
      <PanelHeader title="IPO日历">
        <template #controls>
          <div class="market-selector">
            <button :class="['market-tab', { active: market === 'CN' }]" @click="market = 'CN'">A股</button>
            <button :class="['market-tab', { active: market === 'HK' }]" @click="market = 'HK'; if (!subscriptionData.length && !listingData.length) fetchHKData()">港股</button>
          </div>
          <button class="btn btn-sm" @click="doFetch" :disabled="loading">⟳</button>
        </template>
      </PanelHeader>

      <div class="sub-tabs">
        <template v-if="market === 'CN'">
          <button :class="['btn btn-sm', { 'btn-primary': activeTab === 'today_apply' }]" @click="switchTab('today_apply')">{{ $t('panels.today_apply') }}</button>
          <button :class="['btn btn-sm', { 'btn-primary': activeTab === 'upcoming' }]" @click="switchTab('upcoming')">{{ $t('panels.upcoming') }}</button>
          <button :class="['btn btn-sm', { 'btn-primary': activeTab === 'recent' }]" @click="switchTab('recent')">{{ $t('panels.recent') }}</button>
        </template>
        <template v-if="market === 'HK'">
          <button :class="['btn btn-sm', { 'btn-primary': hkActiveTab === 'subscribing' }]" @click="switchHKTab('subscribing')">{{ t('misc.subscribing') }}</button>
          <button :class="['btn btn-sm', { 'btn-primary': hkActiveTab === 'upcoming_listing' }]" @click="switchHKTab('upcoming_listing')">{{ t('misc.upcoming_listing') }}</button>
          <button :class="['btn btn-sm', { 'btn-primary': hkActiveTab === 'recent_perf' }]" @click="switchHKTab('recent_perf')">{{ t('misc.recent_perf') }}</button>
        </template>
      </div>

      <!-- CN content -->
      <template v-if="market === 'CN'">
        <ErrorState v-if="cnError" :description="cnError" @retry="fetchCNData" />
        <LoadingState v-else-if="cnLoading && !allData.length" type="table" :rows="8" />
        <EmptyState v-else-if="cnFiltered.length === 0" title="暂无数据" />
        <div v-else class="table-wrapper">
          <div class="table-header"><span class="col-code">{{ $t('common.symbol') }}</span><span class="col-name">{{ $t('common.name') }}</span><span class="col-price">{{ $t('common.price') }}</span><span class="col-pe">PE</span><span class="col-sub-date">{{ $t('common.subscription_date') }}</span><span class="col-list-date">{{ $t('common.listing_date') }}</span><span class="col-lottery">{{ $t('panels.lottery_rate') }}</span><span class="col-status">{{ $t('common.status') }}</span></div>
          <div class="table-body">
            <div v-for="row in cnFiltered" :key="row.code + row.subscription_date" class="table-row">
              <span class="col-code clickable" @click="onSymbolClick(row.code)">{{ row.code }}</span><span class="col-name">{{ row.name }}</span><span class="col-price">{{ row.issue_price ? row.issue_price.toFixed(2) : '--' }}</span><span class="col-pe">{{ row.pe ? row.pe.toFixed(2) : '--' }}</span><span class="col-sub-date">{{ toDateStr(row.subscription_date) }}</span><span class="col-list-date">{{ toDateStr(row.listing_date) }}</span><span class="col-lottery">{{ formatLotteryRate(row.lottery_rate) }}</span><span class="col-status">{{ row.status || '--' }}</span>
            </div>
          </div>
        </div>
      </template>

      <!-- HK content -->
      <template v-if="market === 'HK'">
        <ErrorState v-if="hkError" :description="hkError" @retry="fetchHKData" />
        <LoadingState v-else-if="hkLoading && !hkHasData" type="table" :rows="8" />
        <EmptyState v-else-if="hkFiltered.length === 0" title="暂无IPO数据" />
        <div v-else class="table-wrapper">
          <div class="table-header"><span class="col-code">代码</span><span class="col-name">名称</span><span class="col-offer-price">招股价/发行价</span><span class="col-entry-fee">入场费</span><span class="col-lot-size">每手股数</span><span class="col-sub-date">招股日期</span><span class="col-list-date">上市日期</span><span class="col-multiple">认购倍数</span><span class="col-lottery">一手中签率</span></div>
          <div class="table-body">
            <div v-for="(row, idx) in hkFiltered" :key="row.股票代码 + (row.上市日期 || idx)" class="table-row">
              <span class="col-code clickable" @click="onSymbolClick(row.股票代码)">{{ row.股票代码 }}</span><span class="col-name">{{ row.名称 }}</span><span class="col-offer-price">{{ formatHKPrice(row.招股价 || row.发行价) }}</span><span class="col-entry-fee">{{ formatHKMoney(row.入场费) }}</span><span class="col-lot-size">{{ row.每手股数 || '--' }}</span><span class="col-sub-date">{{ hkToDateStr(row.招股日期) }}</span><span class="col-list-date">{{ hkToDateStr(row.上市日期) }}</span><span class="col-multiple">{{ formatMultiple(row.认购倍数) }}</span><span class="col-lottery">{{ formatHKLotteryRate(row.一手中签率) }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </ErrorBoundary>
</template>

<style scoped>
.ipo-calendar-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.market-selector { display: flex; gap: 0; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); overflow: hidden; }
.market-tab { padding: var(--space-xs) var(--space-md); border: none; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: var(--font-xs); font-weight: 500; }
.market-tab + .market-tab { border-left: 1px solid var(--color-border-strong); }
.market-tab.active { color: var(--color-accent); background: var(--color-accent-soft); }
.sub-tabs { display: flex; gap: var(--space-xs); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); }
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: var(--space-xs) 0; border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: var(--font-xs); }
.table-row { display: flex; padding: var(--space-xs) 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-hover); }
.col-code { width: 72px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 64px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-price { width: 56px; text-align: right; }
.col-pe { width: 52px; text-align: right; }
.col-sub-date { width: 80px; text-align: center; }
.col-list-date { width: 80px; text-align: center; }
.col-lottery { width: 64px; text-align: right; }
.col-status { flex: 1; min-width: 0; text-align: center; color: var(--color-text-secondary); }
.col-offer-price { width: 64px; text-align: right; }
.col-entry-fee { width: 72px; text-align: right; }
.col-lot-size { width: 52px; text-align: right; }
.col-multiple { width: 60px; text-align: right; }
</style>
