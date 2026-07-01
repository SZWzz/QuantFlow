<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { marketChangeColor } from '@/lib/composables/useMarketColors'
import { PanelHeader, PanelTable, EmptyState, LoadingState } from '@/terminal/components/panel'

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

const { fetchWithCache } = usePanelCache()
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
    const { data: result } = await fetchWithCache(`abnormal_stocks:${market.value}`, () => app.GetAbnormalStocks(market.value))
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

const tableColumns = [
  { key: 'symbol', label: '代码', align: 'left' as const, width: 80 },
  { key: 'name', label: '名称', align: 'left' as const, width: 80 },
  { key: 'price', label: '价格', align: 'right' as const, width: 70, format: 'price' as const },
  { key: 'change_pct', label: '涨跌幅', align: 'right' as const, width: 80, colorize: true, formatter: (v: number) => formatPct(v) },
  { key: 'volume', label: '成交量', align: 'right' as const, flex: 1, format: 'volume' as const },
]
</script>

<template>
  <div class="limit-up-down-panel">
    <PanelHeader
      :title="$t('misc.limit_up_down')"
      :tabs="[
        { key: 'SH', label: 'SH' },
        { key: 'SZ', label: 'SZ' },
      ]"
      :active-tab="market"
      :controls="[
        { label: `${$t('misc.limit_up')}: ${limitUpStocks.length}`, action: () => {}, title: '涨停数' },
        { label: `${$t('misc.limit_down')}: ${limitDownStocks.length}`, action: () => {}, title: '跌停数' },
        { label: autoRefresh ? '自动(30s)' : '手动', action: () => autoRefresh = !autoRefresh, title: '切换自动刷新' },
        { icon: 'refresh', action: refresh, loading: loading.valueOf(), title: '刷新' },
      ]"
      @tab-change="switchMarket"
    />

    <div class="filter-bar">
      <button
        v-for="f in [
          { key: 'all', label: $t('common.all') },
          { key: 'limit-up', label: '涨停' },
          { key: 'limit-down', label: '跌停' },
        ]"
        :key="f.key"
        :class="['filter-btn', { active: filter === f.key }]"
        @click="filter = f.key as typeof filter"
      >
        {{ f.label }}
      </button>
    </div>

    <LoadingState
      v-if="loading && stocks.length === 0"
      type="table"
      :rows="6"
      :cols="5"
    />

    <EmptyState
      v-else-if="filteredStocks.length === 0"
      icon="search"
      :title="$t('misc.no_limit_stocks') || '暂无涨跌停股票'"
    />

    <PanelTable
      v-else
      :columns="tableColumns"
      :data="filteredStocks"
      :striped="true"
      :loading="loading"
      @row-click="(row) => onSymbolClick(row.symbol)"
    />
  </div>
</template>

<style scoped>
.limit-up-down-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
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

.limit-up-down-panel :deep(.panel-table-wrapper) {
  flex: 1;
  overflow: hidden;
}

.limit-up-down-panel :deep(.clickable) {
  cursor: pointer;
}

.limit-up-down-panel :deep(.clickable):hover {
  text-decoration: underline;
  color: var(--color-accent);
}
</style>
