<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { marketChangeColor } from '@/lib/composables/useMarketColors'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface AbnormalStock {
  symbol: string
  name: string
  price: number
  change_pct: number
  reason: string
  volume: number
  turnover: number
}

const { fetchWithCache } = usePanelCache()
const market = ref<'SH' | 'SZ'>(props.params?.market || 'SH')
const stocks = ref<AbnormalStock[]>([])
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

async function refresh() {
  const app = (window as any).go?.main?.App
  if (!app) return
  loading.value = true
  try {
    const { data: result } = await fetchWithCache(`abnormal_stocks:${market.value}`, () => app.GetAbnormalStocks(market.value))
    const items: any[] = Array.isArray(result) ? result : (result ? [result] : [])
    stocks.value = items.map((i: any) => ({
      symbol: i.symbol || '',
      name: i.name || '',
      price: i.price || 0,
      change_pct: i.change_pct || i.changePct || 0,
      reason: i.reason || '',
      volume: i.volume || 0,
      turnover: i.turnover || 0,
    }))
  } catch(e) {
    console.error('[AbnormalStocks]', e)
    stocks.value = []
  } finally {
    loading.value = false
  }
}

function switchMarket(mkt: 'SH' | 'SZ') {
  market.value = mkt
  refresh()
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
}

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

onMounted(() => {
  refresh()
  timer = setInterval(() => {
    if (autoRefresh.value && isTradingHours()) {
      refresh()
    }
  }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="abnormal-stocks-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.abnormal_stocks') }}</h3>
      <div class="market-tabs">
        <button :class="['mkt-tab', { active: market === 'SH' }]" @click="switchMarket('SH')">SH</button>
        <button :class="['mkt-tab', { active: market === 'SZ' }]" @click="switchMarket('SZ')">SZ</button>
      </div>
      <div class="header-controls">
        <button class="auto-btn" :class="{ active: autoRefresh }" @click="toggleAutoRefresh">
          自动 {{ autoRefresh ? '(30s)' : '' }}
        </button>
        <button class="refresh-btn" @click="refresh" :disabled="loading">
          {{ loading ? '...' : '⟳' }}
        </button>
      </div>
    </div>

    <div v-if="loading && stocks.length === 0" class="status-msg">{{ $t('common.loading') }}</div>
    <div v-else-if="stocks.length === 0" class="status-msg">{{ $t('misc.no_abnormal_stocks') }}</div>
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
        <div v-for="(s, idx) in stocks" :key="s.symbol + idx" class="table-row">
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

    <div v-if="loading && stocks.length > 0" class="refreshing-indicator">{{ $t('common.loading') }}</div>
  </div>
</template>

<style scoped>
.abnormal-stocks-panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
  overflow: hidden;
}
.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.market-tabs { display: flex; gap: 4px; }
.mkt-tab {
  padding: 2px 10px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 11px;
}
.mkt-tab.active { color: #60a5fa; border-color: #3b82f6; background: rgba(59,130,246,0.1); }
.header-controls { display: flex; gap: 6px; align-items: center; margin-left: auto; }
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

.status-msg {
  display: flex; align-items: center; justify-content: center;
  flex: 1; color: var(--color-text-tertiary); font-size: 13px;
}

.stocks-table-wrapper {
  flex: 1; overflow: hidden; display: flex; flex-direction: column;
}
.table-header-row {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase;
  flex-shrink: 0;
}
.table-body {
  flex: 1; overflow-y: auto; font-size: 12px;
  scrollbar-width: thin; scrollbar-color: var(--color-border-strong) transparent;
}
.table-row {
  display: flex; padding: 3px 0; align-items: center;
}
.table-row:hover { background: var(--color-bg-elevated); }
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.symbol-col { width: 72px; }
.name-col { width: 72px; }
.price-col { width: 64px; text-align: right; }
.change-col { width: 64px; text-align: right; font-weight: 500; }
.reason-col { flex: 1; min-width: 0; color: var(--color-text-secondary); }
.vol-col { width: 70px; text-align: right; color: var(--color-text-secondary); }
.turnover-col { width: 60px; text-align: right; color: var(--color-text-tertiary); }

.refreshing-indicator {
  padding: 4px 0; font-size: 10px; color: var(--color-text-tertiary); text-align: center;
}
</style>
