<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { logger } from '@/lib/logger'
import { PanelHeader, PanelTable, LoadingState, EmptyState, ErrorState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// ── Top-level tab ──
const activeTab = ref<'limit' | 'abnormal' | 'dragon'>(
  (props.params?.tab as any) || 'limit'
)
const topTabs = [
  { id: 'limit', key: 'misc.limit_up_down' },
  { id: 'abnormal', key: 'misc.abnormal_stocks' },
  { id: 'dragon', key: 'misc.dragon_tiger' },
]

// ── Shared helpers ──
const { fetchWithCache } = usePanelCache()

function formatPct(pct: number): string {
  return (pct >= 0 ? '+' : '') + pct.toFixed(2) + '%'
}

function formatVolume(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return String(v)
}

function formatTurnover(t: number): string {
  if (typeof t !== 'number') return '--'
  return t.toFixed(2) + '%'
}

function formatAmount(v: number): string {
  const abs = Math.abs(v)
  if (abs >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

function isTradingHours(): boolean {
  const now = new Date()
  const day = now.getDay()
  if (day === 0 || day === 6) return false
  const h = now.getHours()
  const m = now.getMinutes()
  const t = h * 60 + m
  return (t >= 9 * 60 + 30 && t <= 11 * 60 + 30) || (t >= 13 * 60 && t <= 15 * 60)
}

// ═══════════════════════════════════════════════════════════
// Tab 1: 涨跌停 (LimitUpDown)
// ═══════════════════════════════════════════════════════════

interface LimitStock {
  symbol: string
  name: string
  price: number
  change_pct: number
  volume: number
  turnover: number
}

const limitMarket = ref<'SH' | 'SZ'>(props.params?.market || 'SH')
const limitFilter = ref<'all' | 'limit-up' | 'limit-down'>('all')
const limitStocks = ref<LimitStock[]>([])
const limitLoading = ref(false)
const limitError = ref('')
const limitAutoRefresh = ref(true)
let limitTimer: ReturnType<typeof setInterval> | null = null

function detectLimit(pct: number): 'up' | 'down' | null {
  if (pct >= 9.8) return 'up'
  if (pct <= -9.8) return 'down'
  return null
}

const limitUpStocks = computed(() => limitStocks.value.filter(s => detectLimit(s.change_pct) === 'up'))
const limitDownStocks = computed(() => limitStocks.value.filter(s => detectLimit(s.change_pct) === 'down'))
const limitFilteredStocks = computed(() => {
  if (limitFilter.value === 'limit-up') return limitUpStocks.value
  if (limitFilter.value === 'limit-down') return limitDownStocks.value
  return limitStocks.value
})

async function refreshLimit() {
  const app = (window as any).go?.main?.App
  if (!app) return
  limitLoading.value = true
  limitError.value = ''
  try {
    const { data: result } = await fetchWithCache(`abnormal_stocks:${limitMarket.value}`, () => app.GetAbnormalStocks(limitMarket.value))
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    limitStocks.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      change_pct: i.change_pct || i.changePct || 0,
      volume: i.volume || 0,
      turnover: i.turnover || 0,
    }))
  } catch (e: any) {
    console.error('[MarketScanner:Limit]', e)
    limitError.value = e?.message || String(e)
    limitStocks.value = []
  } finally {
    limitLoading.value = false
  }
}

function switchLimitMarket(mkt: string) {
  if (mkt !== 'SH' && mkt !== 'SZ') return
  limitMarket.value = mkt
  refreshLimit()
}

const limitTableColumns = [
  { key: 'symbol', label: '代码', align: 'left' as const, width: 80 },
  { key: 'name', label: '名称', align: 'left' as const, width: 80 },
  { key: 'price', label: '价格', align: 'right' as const, width: 70, format: 'price' as const },
  { key: 'change_pct', label: '涨跌幅', align: 'right' as const, width: 80, colorize: true, formatter: (v: number) => formatPct(v) },
  { key: 'volume', label: '成交量', align: 'right' as const, flex: 1, format: 'volume' as const },
]

// ═══════════════════════════════════════════════════════════
// Tab 2: 异动监控 (AbnormalStocks)
// ═══════════════════════════════════════════════════════════

interface AbnormalStock {
  symbol: string
  name: string
  price: number
  change_pct: number
  reason: string
  volume: number
  turnover: number
}

const abMarket = ref<'SH' | 'SZ'>(props.params?.market || 'SH')
const abStocks = ref<AbnormalStock[]>([])
const abLoading = ref(false)
const abAutoRefresh = ref(true)
let abTimer: ReturnType<typeof setInterval> | null = null

async function refreshAbnormal() {
  const app = (window as any).go?.main?.App
  if (!app) return
  abLoading.value = true
  try {
    const { data: result } = await fetchWithCache(`abnormal_stocks:${abMarket.value}`, () => app.GetAbnormalStocks(abMarket.value))
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    abStocks.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      change_pct: i.change_pct || i.changePct || 0,
      reason: i.reason || '',
      volume: i.volume || 0,
      turnover: i.turnover || 0,
    }))
  } catch (e) {
    logger.error('[MarketScanner:Abnormal]', e)
    abStocks.value = []
  } finally {
    abLoading.value = false
  }
}

function switchAbMarket(mkt: 'SH' | 'SZ') {
  abMarket.value = mkt
  refreshAbnormal()
}

function toggleAbAutoRefresh() {
  abAutoRefresh.value = !abAutoRefresh.value
}

// ═══════════════════════════════════════════════════════════
// Tab 3: 龙虎榜 (DragonTiger)
// ═══════════════════════════════════════════════════════════

interface DeptDetail {
  name: string
  net_amount: number
}

interface DragonTigerStock {
  code: string
  name: string
  close: number
  change_pct: number
  net_buy: number
  reason: string
  turnover: number
  dept_buy_top5: DeptDetail[]
  dept_sell_top5: DeptDetail[]
  dept_total_top5: DeptDetail[]
}

const dtActiveTab = ref<'daily' | 'history'>('daily')
const dtDate = ref(new Date().toISOString().slice(0, 10))
const dtMinNetBuy = ref(0)
const dtStocks = ref<DragonTigerStock[]>([])
const dtHistoryData = ref<DragonTigerStock[]>([])
const dtLoading = ref(false)
const dtError = ref('')
const dtHistoryLoading = ref(false)
const dtExpandedRow = ref<string | null>(null)
const dtHistorySymbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')

watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (sym) => {
  if (sym && dtActiveTab.value === 'history') {
    dtHistorySymbol.value = sym
    fetchDtHistory()
  }
})

async function fetchDtDaily() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDailyDragonTiger) return
  dtError.value = ''
  dtLoading.value = true
  try {
    const { data: result } = await fetchWithCache<any>(`dragon_tiger:${dtDate.value}:${dtMinNetBuy.value}`, () => app.GetDailyDragonTiger(dtDate.value, dtMinNetBuy.value), 5 * 60 * 1000)
    const raw = Array.isArray(result) ? result : (result?.stocks || [])
    dtStocks.value = raw.map((s: any) => ({
      code: s.code || '',
      name: s.name || '',
      close: s.close || 0,
      change_pct: s.change_pct || 0,
      net_buy: s.net_buy || 0,
      reason: s.reason || '',
      turnover: s.turnover || 0,
      dept_buy_top5: (s.dept_buy_top5 || []).map((d: any) => ({ name: d.name, net_amount: d.net_amount })),
      dept_sell_top5: (s.dept_sell_top5 || []).map((d: any) => ({ name: d.name, net_amount: d.net_amount })),
      dept_total_top5: (s.dept_total_top5 || []).map((d: any) => ({ name: d.name, net_amount: d.net_amount })),
    }))
  } catch (e: any) {
    dtError.value = e?.message || String(e)
    dtStocks.value = []
  } finally {
    dtLoading.value = false
  }
}

async function fetchDtHistory() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDragonTiger || !dtHistorySymbol.value) return
  dtError.value = ''
  dtHistoryLoading.value = true
  try {
    const { data: result } = await fetchWithCache<any>(`dragon_tiger_history:${dtHistorySymbol.value}:${dtDate.value}`, () => app.GetDragonTiger(dtHistorySymbol.value, dtDate.value, 20), 5 * 60 * 1000)
    const raw = Array.isArray(result) ? result : (result?.records || [])
    dtHistoryData.value = raw.map((s: any) => ({
      code: s.code || dtHistorySymbol.value,
      name: s.name || '',
      close: s.close || 0,
      change_pct: s.change_pct || 0,
      net_buy: s.net_buy || 0,
      reason: s.reason || '',
      turnover: s.turnover || 0,
      dept_buy_top5: [],
      dept_sell_top5: [],
      dept_total_top5: [],
    }))
  } catch (e: any) {
    dtError.value = e?.message || String(e)
    dtHistoryData.value = []
  } finally {
    dtHistoryLoading.value = false
  }
}

function toggleDtRow(code: string) {
  dtExpandedRow.value = dtExpandedRow.value === code ? null : code
}

function onDtSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function switchDtSubTab(tab: 'daily' | 'history') {
  dtActiveTab.value = tab
  if (tab === 'daily') fetchDtDaily()
  else fetchDtHistory()
}

// ═══════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════

onMounted(() => {
  // Initialize the active tab's data
  if (activeTab.value === 'limit') refreshLimit()
  else if (activeTab.value === 'abnormal') refreshAbnormal()
  else fetchDtDaily()

  // Auto-refresh timers for limit and abnormal tabs
  limitTimer = setInterval(() => {
    if (limitAutoRefresh.value && isTradingHours()) refreshLimit()
  }, 30000)
  abTimer = setInterval(() => {
    if (abAutoRefresh.value && isTradingHours()) refreshAbnormal()
  }, 30000)
})

onUnmounted(() => {
  if (limitTimer) clearInterval(limitTimer)
  if (abTimer) clearInterval(abTimer)
})

// Switch tabs triggers data fetch
function switchTopTab(tab: 'limit' | 'abnormal' | 'dragon') {
  activeTab.value = tab
  if (tab === 'limit' && limitStocks.value.length === 0) refreshLimit()
  else if (tab === 'abnormal' && abStocks.value.length === 0) refreshAbnormal()
  else if (tab === 'dragon' && dtStocks.value.length === 0) fetchDtDaily()
}
</script>

<template>
  <div class="market-scanner-panel">
    <!-- Top-level tab bar -->
    <div class="top-tab-bar">
      <button
        v-for="tab in topTabs"
        :key="tab.id"
        :class="['top-tab-btn', { active: activeTab === tab.id }]"
        @click="switchTopTab(tab.id as 'limit' | 'abnormal' | 'dragon')"
      >{{ $t(tab.key) }}</button>
    </div>

    <div class="tab-content">
      <!-- ═══════════════ Tab 1: 涨跌停 ═══════════════ -->
      <div v-if="activeTab === 'limit'" class="tab-pane limit-pane">
        <PanelHeader
          :title="$t('misc.limit_up_down')"
          :tabs="[
            { key: 'SH', label: 'SH' },
            { key: 'SZ', label: 'SZ' },
          ]"
          :active-tab="limitMarket"
          :controls="[
            { label: `${$t('misc.limit_up')}: ${limitUpStocks.length}`, action: () => {}, title: '涨停数' },
            { label: `${$t('misc.limit_down')}: ${limitDownStocks.length}`, action: () => {}, title: '跌停数' },
            { label: limitAutoRefresh ? '自动(30s)' : '手动', action: () => limitAutoRefresh = !limitAutoRefresh, title: '切换自动刷新' },
            { icon: 'refresh', action: refreshLimit, loading: limitLoading.valueOf(), title: '刷新' },
          ]"
          @tab-change="switchLimitMarket"
        />

        <div v-if="limitError" class="panel-error">{{ limitError }}</div>
        <div class="filter-bar">
          <button
            v-for="f in [
              { key: 'all', label: $t('common.all') },
              { key: 'limit-up', label: '涨停' },
              { key: 'limit-down', label: '跌停' },
            ]"
            :key="f.key"
            :class="['filter-btn', { active: limitFilter === f.key }]"
            @click="limitFilter = f.key as typeof limitFilter"
          >
            {{ f.label }}
          </button>
        </div>

        <LoadingState
          v-if="limitLoading && limitStocks.length === 0"
          type="table"
          :rows="6"
          :cols="5"
        />

        <EmptyState
          v-else-if="limitFilteredStocks.length === 0"
          icon="search"
          :title="$t('misc.no_limit_stocks') || '暂无涨跌停股票'"
        />

        <PanelTable
          v-else
          :columns="limitTableColumns"
          :data="limitFilteredStocks"
          :striped="true"
          :loading="limitLoading"
        />
      </div>

      <!-- ═══════════════ Tab 2: 异动监控 ═══════════════ -->
      <div v-if="activeTab === 'abnormal'" class="tab-pane abnormal-pane">
        <div class="pane-header">
          <h3>{{ $t('misc.abnormal_stocks') }}</h3>
          <div class="market-tabs">
            <button :class="['mkt-tab', { active: abMarket === 'SH' }]" @click="switchAbMarket('SH')">SH</button>
            <button :class="['mkt-tab', { active: abMarket === 'SZ' }]" @click="switchAbMarket('SZ')">SZ</button>
          </div>
          <div class="header-controls">
            <button class="auto-btn" :class="{ active: abAutoRefresh }" @click="toggleAbAutoRefresh">
              自动 {{ abAutoRefresh ? '(30s)' : '' }}
            </button>
            <button class="refresh-btn" @click="refreshAbnormal" :disabled="abLoading">
              {{ abLoading ? '...' : '⟳' }}
            </button>
          </div>
        </div>

        <div v-if="abLoading && abStocks.length === 0" class="status-msg">{{ $t('common.loading') }}</div>
        <div v-else-if="abStocks.length === 0" class="status-msg">{{ $t('misc.no_abnormal_stocks') }}</div>
        <div v-else class="stocks-table-wrapper">
          <div class="table-header-row">
            <span class="col symbol-col">{{ $t('common.symbol') }}</span>
            <span class="col name-col">{{ $t('common.name') }}</span>
            <span class="col price-col">{{ $t('common.price') }}</span>
            <span class="col change-col">{{ $t('quote.change_pct') }}</span>
            <span class="col reason-col">{{ $t('misc.abnormal_reason') }}</span>
            <span class="col vol-col">{{ $t('common.volume') }}</span>
            <span class="col turnover-col">{{ $t('misc.turnover') }}</span>
          </div>
          <div class="table-body">
            <div v-for="(s, idx) in abStocks" :key="s.symbol + idx" class="table-row">
              <span class="col symbol-col">{{ s.symbol }}</span>
              <span class="col name-col">{{ s.name }}</span>
              <span class="col price-col">{{ s.price.toFixed(2) }}</span>
              <span class="col change-col" :style="{ color: marketChangeColor('600519', s.change_pct) }">{{ formatPct(s.change_pct) }}</span>
              <span class="col reason-col" :title="s.reason">{{ s.reason }}</span>
              <span class="col vol-col">{{ formatVolume(s.volume) }}</span>
              <span class="col turnover-col">{{ formatTurnover(s.turnover) }}</span>
            </div>
          </div>
        </div>

        <div v-if="abLoading && abStocks.length > 0" class="refreshing-indicator">{{ $t('common.loading') }}</div>
      </div>

      <!-- ═══════════════ Tab 3: 龙虎榜 ═══════════════ -->
      <div v-if="activeTab === 'dragon'" class="tab-pane dragon-pane">
        <div class="pane-header">
          <h3>{{ $t('misc.dragon_tiger') }}</h3>
          <div class="header-tabs">
            <button :class="['sub-tab', { active: dtActiveTab === 'daily' }]" @click="switchDtSubTab('daily')">{{ $t('misc.daily_board') }}</button>
            <button :class="['sub-tab', { active: dtActiveTab === 'history' }]" @click="switchDtSubTab('history')">{{ $t('misc.stock_history') }}</button>
          </div>
          <div class="header-controls">
            <template v-if="dtActiveTab === 'daily'">
              <input v-model="dtDate" type="date" class="date-input" @change="fetchDtDaily" />
              <input v-model.number="dtMinNetBuy" type="number" class="min-input" placeholder="min(亿)" @change="fetchDtDaily" />
            </template>
            <template v-else>
              <input v-model="dtHistorySymbol" class="symbol-input" placeholder="代码" @change="fetchDtHistory" />
            </template>
            <button class="refresh-btn" @click="dtActiveTab === 'daily' ? fetchDtDaily() : fetchDtHistory()" :disabled="dtLoading || dtHistoryLoading">⟳</button>
          </div>
        </div>

        <div v-if="dtError" class="error-state" @click="dtActiveTab === 'daily' ? fetchDtDaily() : fetchDtHistory()">{{ dtError }} ⟳</div>

        <LoadingState v-else-if="dtLoading && dtActiveTab === 'daily'" type="table" :rows="8" />
        <LoadingState v-else-if="dtHistoryLoading && dtActiveTab === 'history'" type="table" :rows="8" />

        <template v-else-if="dtActiveTab === 'daily'">
          <div v-if="dtStocks.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
          <div v-else class="table-wrapper">
            <div class="table-header">
              <span class="col-code">{{ $t('common.symbol') }}</span>
              <span class="col-name">{{ $t('common.name') }}</span>
              <span class="col-price">{{ $t('common.price') }}</span>
              <span class="col-pct">{{ $t('quote.change_pct') }}</span>
              <span class="col-netbuy">{{ $t('misc.net_buy') }}</span>
              <span class="col-reason">{{ $t('misc.reason') }}</span>
            </div>
            <div class="table-body">
              <template v-for="s in dtStocks" :key="s.code">
                <div class="table-row" :class="{ expanded: dtExpandedRow === s.code }" @click="toggleDtRow(s.code)">
                  <span class="col-code clickable" @click.stop="onDtSymbolClick(s.code)">{{ s.code }}</span>
                  <span class="col-name">{{ s.name }}</span>
                  <span class="col-price">{{ s.close.toFixed(2) }}</span>
                  <span class="col-pct" :style="{ color: marketChangeColor(s.code, s.change_pct) }">{{ formatPct(s.change_pct) }}</span>
                  <span class="col-netbuy" :class="s.net_buy >= 0 ? 'up' : 'down'">{{ formatAmount(s.net_buy) }}</span>
                  <span class="col-reason" :title="s.reason">{{ s.reason }}</span>
                </div>
                <div v-if="dtExpandedRow === s.code" class="expand-detail">
                  <div class="detail-section">
                    <div class="detail-title">{{ $t('misc.buy_top5') }}</div>
                    <div class="detail-list">
                      <div v-for="d in s.dept_buy_top5" :key="d.name" class="detail-item">
                        <span class="dept-name">{{ d.name }}</span>
                        <span class="dept-amount up">{{ formatAmount(d.net_amount) }}</span>
                      </div>
                      <div v-if="s.dept_buy_top5.length === 0" class="detail-empty">--</div>
                    </div>
                  </div>
                  <div class="detail-section">
                    <div class="detail-title">{{ $t('misc.sell_top5') }}</div>
                    <div class="detail-list">
                      <div v-for="d in s.dept_sell_top5" :key="d.name" class="detail-item">
                        <span class="dept-name">{{ d.name }}</span>
                        <span class="dept-amount down">{{ formatAmount(d.net_amount) }}</span>
                      </div>
                      <div v-if="s.dept_sell_top5.length === 0" class="detail-empty">--</div>
                    </div>
                  </div>
                  <div class="detail-section">
                    <div class="detail-title">{{ $t('misc.dept_total_top5') }}</div>
                    <div class="detail-list">
                      <div v-for="d in s.dept_total_top5" :key="d.name" class="detail-item">
                        <span class="dept-name">{{ d.name }}</span>
                        <span class="dept-amount">{{ formatAmount(d.net_amount) }}</span>
                      </div>
                      <div v-if="s.dept_total_top5.length === 0" class="detail-empty">--</div>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </template>

        <template v-else>
          <div v-if="dtHistoryData.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
          <div v-else class="table-wrapper">
            <div class="table-header">
              <span class="col-date">{{ $t('common.date') }}</span>
              <span class="col-price">{{ $t('common.price') }}</span>
              <span class="col-pct">{{ $t('quote.change_pct') }}</span>
              <span class="col-netbuy">{{ $t('misc.net_buy') }}</span>
              <span class="col-reason">{{ $t('misc.reason') }}</span>
            </div>
            <div class="table-body">
              <div v-for="s in dtHistoryData" :key="s.code + s.close" class="table-row">
                <span class="col-date">{{ s.reason?.slice(0, 10) || '--' }}</span>
                <span class="col-price">{{ s.close.toFixed(2) }}</span>
                <span class="col-pct" :style="{ color: marketChangeColor(s.code, s.change_pct) }">{{ formatPct(s.change_pct) }}</span>
                <span class="col-netbuy" :class="s.net_buy >= 0 ? 'up' : 'down'">{{ formatAmount(s.net_buy) }}</span>
                <span class="col-reason">{{ s.reason }}</span>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.market-scanner-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  color: var(--color-text-primary);
  background: var(--color-bg-panel);
}

/* ── Top-level tab bar (underline, same style as StockResearchPanel) ── */
.top-tab-bar {
  display: flex;
  gap: 2px;
  padding: 0 var(--panel-padding);
  border-bottom: 1px solid var(--color-border-strong);
  flex-shrink: 0;
  overflow-x: auto;
}

.top-tab-btn {
  padding: 6px 14px;
  border: none;
  background: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: var(--font-sm);
  border-bottom: 2px solid transparent;
  white-space: nowrap;
}

.top-tab-btn:hover {
  color: var(--color-text-primary);
}

.top-tab-btn.active {
  color: var(--color-text-primary);
  border-bottom-color: var(--color-accent);
}

/* ── Tab content ── */
.tab-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tab-pane {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ═══════════════ Tab 1: Limit Up/Down ═══════════════ */
.limit-pane {
  color: var(--color-text-primary);
  background: var(--color-bg-panel);
}

.filter-bar {
  display: flex;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.filter-btn {
  padding: 2px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.filter-btn:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.filter-btn.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.limit-pane :deep(.panel-table-wrapper) {
  flex: 1;
  overflow: hidden;
}

.limit-pane :deep(.clickable) {
  cursor: pointer;
}

.limit-pane :deep(.clickable):hover {
  text-decoration: underline;
  color: var(--color-accent);
}

/* ═══════════════ Tab 2: Abnormal Stocks ═══════════════ */
.abnormal-pane {
  padding: 16px;
  background: var(--color-bg, var(--color-bg-panel));
}

.pane-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-shrink: 0;
}

.pane-header h3 {
  margin: 0;
  font-size: var(--font-sm);
  font-weight: 600;
}

.market-tabs {
  display: flex;
  gap: 4px;
}

.mkt-tab {
  padding: 2px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
}

.mkt-tab.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: rgba(59, 130, 246, 0.1);
}

.header-controls {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-left: auto;
}

.auto-btn {
  padding: 2px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
}

.auto-btn.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
}

.refresh-btn {
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: var(--font-sm);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.status-msg {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: var(--color-text-tertiary);
  font-size: var(--font-sm);
}

.stocks-table-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.table-header-row {
  display: flex;
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border-strong);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  flex-shrink: 0;
}

.table-body {
  flex: 1;
  overflow-y: auto;
  font-size: var(--font-sm);
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-strong) transparent;
}

.table-row {
  display: flex;
  padding: 3px 0;
  align-items: center;
  border-bottom: 1px solid var(--color-border-subtle);
}

.table-row:hover {
  background: var(--color-bg-elevated);
}

.col {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.symbol-col { width: 72px; }
.name-col { width: 72px; }
.price-col { width: 64px; text-align: right; }
.change-col { width: 64px; text-align: right; font-weight: 500; }
.reason-col { flex: 1; min-width: 0; color: var(--color-text-secondary); }
.vol-col { width: 70px; text-align: right; color: var(--color-text-secondary); }
.turnover-col { width: 60px; text-align: right; color: var(--color-text-tertiary); }

.refreshing-indicator {
  padding: 4px 0;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-align: center;
  flex-shrink: 0;
}

/* ═══════════════ Tab 3: Dragon Tiger ═══════════════ */
.dragon-pane {
  padding: 12px;
  background: var(--color-bg-panel, var(--color-bg-panel));
}

.header-tabs {
  display: flex;
  gap: 4px;
}

.header-tabs .sub-tab {
  padding: 2px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
}

.header-tabs .sub-tab.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: rgba(59, 130, 246, 0.1);
}

.date-input,
.min-input,
.symbol-input {
  padding: 2px 6px;
  font-size: var(--font-xs);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  width: 100px;
}

.min-input { width: 70px; }
.symbol-input { width: 70px; }

.error-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  color: var(--color-danger);
  font-size: var(--font-sm);
  cursor: pointer;
}

.dragon-pane .table-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dragon-pane .table-header {
  display: flex;
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border-strong);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  flex-shrink: 0;
}

.dragon-pane .table-body {
  flex: 1;
  overflow-y: auto;
  font-size: var(--font-sm);
}

.dragon-pane .table-row {
  cursor: pointer;
}

.dragon-pane .table-row.expanded {
  background: var(--color-bg-elevated);
}

.col-code { width: 64px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-price { width: 60px; text-align: right; }
.col-pct { width: 60px; text-align: right; font-weight: 500; }
.col-netbuy { width: 70px; text-align: right; font-weight: 500; }
.col-date { width: 80px; }

.up { color: var(--color-up); }
.down { color: var(--color-down); }

.expand-detail {
  display: flex;
  gap: 12px;
  padding: 8px 12px;
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border-subtle);
}

.detail-section { flex: 1; min-width: 0; }
.detail-title {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin-bottom: 4px;
  text-transform: uppercase;
}

.detail-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-xs);
  padding: 2px 0;
}

.dept-name {
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dept-amount {
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.detail-empty {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

/* ── Deep scoped width fix for PanelHeader internal column classes ── */
/* Ensure the limit pane's PanelTable inherits proper sizing */
.limit-pane :deep(.panel-table-wrapper) :deep(.col-code) { width: 64px; }
.limit-pane :deep(.panel-table-wrapper) :deep(.col-name) { width: 64px; }
</style>
