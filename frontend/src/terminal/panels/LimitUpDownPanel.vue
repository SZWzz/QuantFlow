<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface LimitStock {
  symbol: string
  name: string
  price: number
  change_pct: number
  volume: number
  turnover: number
}

const market = ref<'SH' | 'SZ'>(props.params?.market || 'SH')
const filter = ref<'all' | 'limit-up' | 'limit-down'>('all')
const stocks = ref<LimitStock[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
let timer: ReturnType<typeof setInterval> | null = null

function isTradingHours(): boolean {
  const now = new Date()
  const day = now.getDay()
  if (day === 0 || day === 6) return false
  const h = now.getHours()
  const m = now.getMinutes()
  const t = h * 60 + m
  return (t >= 9 * 60 + 30 && t <= 11 * 60 + 30) || (t >= 13 * 60 && t <= 15 * 60)
}

function detectLimit(pct: number): 'up' | 'down' | null {
  if (pct >= 9.8) return 'up'
  if (pct <= -9.8) return 'down'
  return null
}

const limitUpStocks = computed(() => stocks.value.filter(s => detectLimit(s.change_pct) === 'up'))
const limitDownStocks = computed(() => stocks.value.filter(s => detectLimit(s.change_pct) === 'down'))
const filteredStocks = computed(() => {
  if (filter.value === 'limit-up') return limitUpStocks.value
  if (filter.value === 'limit-down') return limitDownStocks.value
  return stocks.value
})

async function refresh() {
  const app = (window as any).go?.main?.App
  if (!app) return
  loading.value = true
  try {
    const result = await app.GetAbnormalStocks(market.value)
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    stocks.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      change_pct: i.change_pct || i.changePct || 0,
      volume: i.volume || 0,
      turnover: i.turnover || 0,
    }))
  } catch (e) {
    console.error('[LimitUpDown]', e)
    stocks.value = []
  } finally {
    loading.value = false
  }
}

function onSymbolClick(code: string) {
  ctx.setGroupSymbol(pg.groupId, code)
}

function switchMarket(mkt: 'SH' | 'SZ') {
  market.value = mkt
  refresh()
}

function formatPct(pct: number): string {
  return (pct >= 0 ? '+' : '') + pct.toFixed(2) + '%'
}

function formatVolume(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return String(v)
}

onMounted(() => {
  refresh()
  timer = setInterval(() => {
    if (autoRefresh.value && isTradingHours()) refresh()
  }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="limit-up-down-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.limit_up_down') }}</h3>
      <div class="market-tabs">
        <button :class="['mkt-tab', { active: market === 'SH' }]" @click="switchMarket('SH')">SH</button>
        <button :class="['mkt-tab', { active: market === 'SZ' }]" @click="switchMarket('SZ')">SZ</button>
      </div>
      <div class="filter-tabs">
        <button :class="['f-tab', { active: filter === 'all' }]" @click="filter = 'all'">{{ $t('common.all') }}</button>
        <button :class="['f-tab', { active: filter === 'limit-up' }]" @click="filter = 'limit-up'">涨停</button>
        <button :class="['f-tab', { active: filter === 'limit-down' }]" @click="filter = 'limit-down'">跌停</button>
      </div>
      <div class="header-controls">
        <span class="stat-badge up">{{ $t('misc.limit_up') }}: {{ limitUpStocks.length }}</span>
        <span class="stat-badge down">{{ $t('misc.limit_down') }}: {{ limitDownStocks.length }}</span>
        <button class="auto-btn" :class="{ active: autoRefresh }" @click="autoRefresh = !autoRefresh">
          {{ autoRefresh ? '自动(30s)' : '手动' }}
        </button>
        <button class="refresh-btn" @click="refresh" :disabled="loading">⟳</button>
      </div>
    </div>

    <SkeletonPanel v-if="loading && stocks.length === 0" type="table" :rows="6" />

    <div v-else-if="filteredStocks.length === 0" class="empty-state">{{ $t('misc.no_limit_stocks') }}</div>

    <div v-else class="table-wrapper">
      <div class="table-header">
        <span class="col-code">{{ $t('common.symbol') }}</span>
        <span class="col-name">{{ $t('common.name') }}</span>
        <span class="col-price">{{ $t('common.price') }}</span>
        <span class="col-pct">{{ $t('quote.change_pct') }}</span>
        <span class="col-vol">{{ $t('common.volume') }}</span>
      </div>
      <div class="table-body">
        <div v-for="s in filteredStocks" :key="s.symbol" class="table-row">
          <span class="col-code clickable" @click="onSymbolClick(s.symbol)">{{ s.symbol }}</span>
          <span class="col-name">{{ s.name }}</span>
          <span class="col-price">{{ s.price.toFixed(2) }}</span>
          <span class="col-pct" :class="detectLimit(s.change_pct) === 'up' ? 'up' : 'down'" :style="{ color: marketChangeColor(s.symbol, s.change_pct) }">
            {{ formatPct(s.change_pct) }}
          </span>
          <span class="col-vol">{{ formatVolume(s.volume) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.limit-up-down-panel {
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
  flex-wrap: wrap;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.market-tabs, .filter-tabs { display: flex; gap: 4px; }
.mkt-tab, .f-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.mkt-tab.active, .f-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.header-controls { display: flex; gap: 6px; align-items: center; margin-left: auto; }
.stat-badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 10px;
}
.stat-badge.up { color: #dc2626; background: rgba(220,38,38,0.1); }
.stat-badge.down { color: #16a34a; background: rgba(22,163,74,0.1); }
.auto-btn {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.auto-btn.active { color: #60a5fa; border-color: #3b82f6; }
.refresh-btn {
  padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px;
}
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state {
  flex: 1; display: flex; align-items: center; justify-content: center;
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
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col-code { width: 64px; }
.col-code.clickable { cursor: pointer; color: #60a5fa; }
.col-name { width: 60px; }
.col-price { width: 60px; text-align: right; }
.col-pct { width: 64px; text-align: right; font-weight: 500; }
.col-vol { flex: 1; min-width: 0; text-align: right; color: var(--color-text-secondary); }
.up { color: #dc2626; }
.down { color: #16a34a; }
</style>
