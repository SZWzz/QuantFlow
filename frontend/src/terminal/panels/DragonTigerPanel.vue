<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

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

const activeTab = ref<'daily' | 'history'>('daily')
const date = ref(new Date().toISOString().slice(0, 10))
const minNetBuy = ref(0)
const stocks = ref<DragonTigerStock[]>([])
const historyData = ref<DragonTigerStock[]>([])
const loading = ref(false)
const loadError = ref('')
const historyLoading = ref(false)
const expandedRow = ref<string | null>(null)
const historySymbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '600519')
const { fetchWithCache } = usePanelCache()

watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (sym) => {
  if (sym && activeTab.value === 'history') {
    historySymbol.value = sym
    fetchHistory()
  }
})

async function fetchDaily() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDailyDragonTiger) return
  loadError.value = ''
  loading.value = true
  try {
    const { data: result } = await fetchWithCache<any>(`dragon_tiger:${date.value}:${minNetBuy.value}`, () => app.GetDailyDragonTiger(date.value, minNetBuy.value), 5 * 60 * 1000)
    const raw = Array.isArray(result) ? result : (result?.stocks || [])
    stocks.value = raw.map((s: any) => ({
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
    loadError.value = e?.message || String(e)
    stocks.value = []
  } finally {
    loading.value = false
  }
}

async function fetchHistory() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDragonTiger || !historySymbol.value) return
  loadError.value = ''
  historyLoading.value = true
  try {
    const { data: result } = await fetchWithCache<any>(`dragon_tiger_history:${historySymbol.value}:${date.value}`, () => app.GetDragonTiger(historySymbol.value, date.value, 20), 5 * 60 * 1000)
    const raw = Array.isArray(result) ? result : (result?.records || [])
    historyData.value = raw.map((s: any) => ({
      code: s.code || historySymbol.value,
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
    loadError.value = e?.message || String(e)
    historyData.value = []
  } finally {
    historyLoading.value = false
  }
}

function toggleRow(code: string) {
  expandedRow.value = expandedRow.value === code ? null : code
}

function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function formatAmount(v: number): string {
  const abs = Math.abs(v)
  if (abs >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (abs >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

function formatPct(pct: number): string {
  return (pct >= 0 ? '+' : '') + pct.toFixed(2) + '%'
}

function switchTab(tab: 'daily' | 'history') {
  activeTab.value = tab
  if (tab === 'daily') fetchDaily()
  else fetchHistory()
}

onMounted(() => fetchDaily())
</script>

<template>
  <div class="dragon-tiger-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.dragon_tiger') }}</h3>
      <div class="header-tabs">
        <button :class="['tab', { active: activeTab === 'daily' }]" @click="switchTab('daily')">{{ $t('misc.daily_board') }}</button>
        <button :class="['tab', { active: activeTab === 'history' }]" @click="switchTab('history')">{{ $t('misc.stock_history') }}</button>
      </div>
      <div class="header-controls">
        <template v-if="activeTab === 'daily'">
          <input v-model="date" type="date" class="date-input" @change="fetchDaily" />
          <input v-model.number="minNetBuy" type="number" class="min-input" placeholder="min(亿)" @change="fetchDaily" />
        </template>
        <template v-else>
          <input v-model="historySymbol" class="symbol-input" placeholder="代码" @change="fetchHistory" />
        </template>
        <button class="refresh-btn" @click="activeTab === 'daily' ? fetchDaily() : fetchHistory()" :disabled="loading || historyLoading">⟳</button>
      </div>
    </div>

    <div v-if="loadError" class="error-state" @click="activeTab === 'daily' ? fetchDaily() : fetchHistory()">{{ loadError }} ⟳</div>

    <SkeletonPanel v-else-if="loading && activeTab === 'daily'" type="table" :rows="8" />
    <SkeletonPanel v-else-if="historyLoading && activeTab === 'history'" type="table" :rows="8" />

    <template v-else-if="activeTab === 'daily'">
      <div v-if="stocks.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
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
          <template v-for="s in stocks" :key="s.code">
            <div class="table-row" :class="{ expanded: expandedRow === s.code }" @click="toggleRow(s.code)">
              <span class="col-code clickable" @click.stop="onSymbolClick(s.code)">{{ s.code }}</span>
              <span class="col-name">{{ s.name }}</span>
              <span class="col-price">{{ s.close.toFixed(2) }}</span>
              <span class="col-pct" :style="{ color: marketChangeColor(s.code, s.change_pct) }">{{ formatPct(s.change_pct) }}</span>
              <span class="col-netbuy" :class="s.net_buy >= 0 ? 'up' : 'down'">{{ formatAmount(s.net_buy) }}</span>
              <span class="col-reason" :title="s.reason">{{ s.reason }}</span>
            </div>
            <div v-if="expandedRow === s.code" class="expand-detail">
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
      <div v-if="historyData.length === 0" class="empty-state">{{ $t('common.no_data') }}</div>
      <div v-else class="table-wrapper">
        <div class="table-header">
          <span class="col-date">{{ $t('common.date') }}</span>
          <span class="col-price">{{ $t('common.price') }}</span>
          <span class="col-pct">{{ $t('quote.change_pct') }}</span>
          <span class="col-netbuy">{{ $t('misc.net_buy') }}</span>
          <span class="col-reason">{{ $t('misc.reason') }}</span>
        </div>
        <div class="table-body">
          <div v-for="s in historyData" :key="s.code + s.close" class="table-row">
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
</template>

<style scoped>
.dragon-tiger-panel {
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
.header-controls { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.date-input, .min-input, .symbol-input {
  padding: 2px 6px; font-size: 11px; border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); width: 100px;
}
.min-input { width: 70px; }
.symbol-input { width: 70px; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 13px;
}
.error-state {
  display: flex; align-items: center; justify-content: center; padding: 12px;
  color: var(--color-error); font-size: 13px; cursor: pointer;
}
.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0;
}
.table-body { flex: 1; overflow-y: auto; font-size: 12px; }
.table-row {
  display: flex; padding: 3px 0; align-items: center; cursor: pointer;
  border-bottom: 1px solid var(--color-border-subtle);
}
.table-row:hover, .table-row.expanded { background: var(--color-bg-elevated); }
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-code { width: 64px; }
.col-code.clickable { cursor: pointer; color: var(--color-accent); }
.col-name { width: 64px; }
.col-price { width: 60px; text-align: right; }
.col-pct { width: 60px; text-align: right; font-weight: 500; }
.col-netbuy { width: 70px; text-align: right; font-weight: 500; }
.col-reason { flex: 1; min-width: 0; color: var(--color-text-secondary); }
.col-date { width: 80px; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }
.expand-detail {
  display: flex; gap: 12px; padding: 8px 12px;
  background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border-subtle);
}
.detail-section { flex: 1; min-width: 0; }
.detail-title { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; text-transform: uppercase; }
.detail-list { display: flex; flex-direction: column; gap: 2px; }
.detail-item { display: flex; justify-content: space-between; font-size: 11px; padding: 2px 0; }
.dept-name { color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dept-amount { font-weight: 500; font-variant-numeric: tabular-nums; }
.detail-empty { font-size: 11px; color: var(--color-text-tertiary); }
</style>
