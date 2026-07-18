<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, watch } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { PanelHeader, PanelTable, EmptyState, type Column } from '@/terminal/components/panel'
import type { FlashClass } from '@/lib/composables/useFlashOnUpdate'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWebSocket } from '@/lib/composables/useWebSocket'
import { useI18n } from 'vue-i18n'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()
const ws = useWebSocket()
const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/market`
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const { t } = useI18n()

const controls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', label: t('common.refresh'), action: refreshAll })
  return list
})

const WS_KEY = 'quantflow-watchlist'
const defaultSymbols = ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276']

interface QuoteRow {
  symbol: string; name: string; last: number; open: number; high: number; low: number
  change: number; changePct: number; volume: number; turnover: number
  turnoverRate: number; volumeRatio: number; amplitude: number; prevLast?: number
}

/** PanelTable 行；0 值视为无数据（显示 '--'），与迁移前 formatCell 的零值处理一致 */
interface WatchRow {
  symbol: string
  name: string
  last?: number
  changePct?: number
  turnover?: number
}

function loadSymbols(): string[] {
  try {
    const saved = localStorage.getItem(WS_KEY)
    if (saved) { const arr = JSON.parse(saved); if (Array.isArray(arr) && arr.length > 0) return arr }
  } catch {}
  return [...defaultSymbols]
}

function saveSymbols(syms: string[]) {
  localStorage.setItem(WS_KEY, JSON.stringify(syms))
}

const symbols = ref<string[]>(loadSymbols())
const quotes = reactive<Record<string, QuoteRow>>({})
const initialLoadDone = ref(false)
const pollingActive = ref(true)

// Context menu
const ctxMenu = ref<{ x: number; y: number; symbol: string } | null>(null)

const tableColumns = computed<Column[]>(() => [
  { key: 'symbol', label: t('common.symbol'), width: 70 },
  { key: 'name', label: t('common.name'), flex: 1 },
  { key: 'last', label: t('common.price'), align: 'right', format: 'price', width: 72 },
  { key: 'changePct', label: t('quote.change_pct'), align: 'right', format: 'percent', colorize: true, width: 76 },
  { key: 'turnover', label: t('quote.turnover'), align: 'right', format: 'volume', width: 80 },
])

const tableRows = computed<WatchRow[]>(() =>
  symbols.value.map(sym => {
    const q = quotes[sym]
    return {
      symbol: sym,
      name: q?.name || sym,
      last: q?.last,
      changePct: q?.changePct || undefined,
      turnover: q?.turnover || undefined,
    }
  }),
)

// 涨跌闪烁：watch 每个 symbol 的现价，变化时在该行根元素加 .flash-up/.flash-down，600ms 后移除。
// 语义与 useFlashOnUpdate 一致：前后值均为有限数值才闪烁；窗口内同向变化只重排清除
// 计时器（class 字符串不变，CSS 动画不重启，避免高频行情频闪）。
// prev === 0 视为无基线跳过——名称解析阶段会以 last: 0 占位，避免初始加载整表闪烁。
const flashMap = reactive<Record<string, FlashClass>>({})
const flashTimers: Record<string, ReturnType<typeof setTimeout>> = {}

watch(
  () => Object.fromEntries(symbols.value.map(sym => [sym, quotes[sym]?.last] as const)),
  (next, prev) => {
    for (const sym of Object.keys(next)) {
      const n = next[sym]
      const p = prev[sym]
      if (typeof n !== 'number' || typeof p !== 'number' || p === 0) continue
      if (!Number.isFinite(n) || !Number.isFinite(p) || n === p) continue
      flashMap[sym] = n > p ? 'flash-up' : 'flash-down'
      clearTimeout(flashTimers[sym])
      flashTimers[sym] = setTimeout(() => { flashMap[sym] = '' }, 600)
    }
  },
)

function rowClass(row: WatchRow): string {
  const cls: string[] = []
  if (ctx.getGroupSymbol(pg.groupId) === row.symbol) cls.push('active')
  const f = flashMap[row.symbol]
  if (f) cls.push(f)
  return cls.join(' ')
}

async function refreshQuote(sym: string) {
  try {
    const { data: result } = await fetchWithCache(`quote:${detectMarket(sym)}:${sym}`, () => (window as any).go?.main?.App?.GetQuote(detectMarket(sym), sym), 10 * 1000)
    const snap = Array.isArray(result) ? result[0] : result
    if (!snap) return
    const prev = quotes[sym]
    quotes[sym] = {
      symbol: snap.symbol ?? sym,
      name: snap.name || prev?.name || sym,
      last: snap.last ?? 0,
      open: snap.open ?? 0,
      high: snap.high ?? 0,
      low: snap.low ?? 0,
      change: snap.change ?? 0,
      changePct: snap.change_pct ?? snap.changePct ?? 0,
      volume: snap.volume ?? 0,
      turnover: snap.turnover ?? 0,
      turnoverRate: snap.turnover_rate ?? 0,
      volumeRatio: snap.volume_ratio ?? 0,
      amplitude: snap.amplitude ?? 0,
      prevLast: prev?.last,
    }
  } catch { /* best-effort */ }
}

function handleWSQuote(topic: string, data: any) {
  const parts = topic.split(':')
  const sym = parts[parts.length - 1]
  if (!sym || !symbols.value.includes(sym)) return
  const snap = Array.isArray(data) ? data[0] : data
  if (!snap) return
  const prev = quotes[sym]
  quotes[sym] = {
    symbol: snap.symbol ?? sym,
    name: snap.name || prev?.name || sym,
    last: snap.last ?? 0,
    open: snap.open ?? 0,
    high: snap.high ?? 0,
    low: snap.low ?? 0,
    change: snap.change ?? 0,
    changePct: snap.change_pct ?? snap.changePct ?? 0,
    volume: snap.volume ?? 0,
    turnover: snap.turnover ?? 0,
    turnoverRate: snap.turnover_rate ?? 0,
    volumeRatio: snap.volume_ratio ?? 0,
    amplitude: snap.amplitude ?? 0,
    prevLast: prev?.last,
  }
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter(s => s !== sym)
  saveSymbols(symbols.value)
  clearTimeout(flashTimers[sym])
  delete flashTimers[sym]
  delete flashMap[sym]
  window.dispatchEvent(new CustomEvent('watchlist-changed'))
}

function selectSymbol(sym: string) {
  ctx.setGroupSymbol(pg.groupId, sym)
}

async function refreshAll() {
  await Promise.all(symbols.value.map(sym => refreshQuote(sym)))
}

function onRowClick(row: WatchRow) {
  selectSymbol(row.symbol)
}

function onRowContextMenu(row: WatchRow, e: MouseEvent) {
  e.preventDefault()
  openContextMenu(e, row.symbol)
}

// Context menu
function openContextMenu(e: MouseEvent, sym: string) {
  ctxMenu.value = { x: e.clientX, y: e.clientY, symbol: sym }
}
function closeContextMenu() { ctxMenu.value = null }

function contextOpenKline() {
  if (!ctxMenu.value) return
  selectSymbol(ctxMenu.value.symbol)
  closeContextMenu()
}
function contextCopyCode() {
  if (!ctxMenu.value) return
  navigator.clipboard.writeText(ctxMenu.value.symbol).catch(() => {})
  closeContextMenu()
}
function contextDelete() {
  if (!ctxMenu.value) return
  removeSymbol(ctxMenu.value.symbol)
  closeContextMenu()
}
function contextOpenAnalysis(panelId: string) {
  if (!ctxMenu.value) return
  selectSymbol(ctxMenu.value.symbol)
  terminal.openPanel(panelId, { symbol: ctxMenu.value.symbol })
  closeContextMenu()
}

function onVisibility() {
  pollingActive.value = !document.hidden
  if (!pollingActive.value) return
  refreshAll()
}

async function onWatchlistChanged() {
  symbols.value = loadSymbols()
  await refreshAll()
}

onMounted(async () => {
  window.addEventListener('watchlist-changed', onWatchlistChanged)
  document.addEventListener('click', closeContextMenu)

  try {
    const app = (window as any).go?.main?.App
    if (app?.SearchSymbols) {
      await Promise.all(symbols.value.map(async (sym) => {
        const { data: results } = await fetchWithCache(`search:${sym}`, () => app.SearchSymbols(sym, 1), 5 * 60 * 1000)
        if (Array.isArray(results) && results.length > 0 && results[0].name) {
          if (!quotes[sym]) {
            quotes[sym] = { symbol: sym, name: results[0].name, last: 0, open: 0, high: 0, low: 0, change: 0, changePct: 0, volume: 0, turnover: 0, turnoverRate: 0, volumeRatio: 0, amplitude: 0 }
          } else if (quotes[sym].name === sym) {
            quotes[sym].name = results[0].name
          }
        }
      }))
    }
  } catch { /* best-effort */ }

  // Initial data fetch via Wails IPC (instant, not waiting for WS)
  await refreshAll()
  initialLoadDone.value = true

  // Subscribe to real-time updates via WebSocket
  ws.connect(wsUrl, symbols.value.map(sym => `market:quote:${detectMarket(sym)}:${sym}`))
  ws.onMessage('*', (msg: any) => {
    if (msg.topic?.startsWith('market:quote:')) {
      handleWSQuote(msg.topic, msg.data)
    }
  })
  pollingActive.value = true
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  window.removeEventListener('watchlist-changed', onWatchlistChanged)
  document.removeEventListener('click', closeContextMenu)
  document.removeEventListener('visibilitychange', onVisibility)
  for (const k of Object.keys(flashTimers)) clearTimeout(flashTimers[k])
  ws.disconnect()
})
</script>

<template>
  <div class="watchlist-panel" data-testid="watchlist-panel">
    <PanelHeader
      :title="t('watchlist.title')"
      :controls="controls"
    />

    <EmptyState
      v-if="symbols.length === 0"
      :title="t('watchlist.empty')"
      data-testid="watchlist-empty"
    />

    <PanelTable
      v-else
      :columns="tableColumns"
      :data="initialLoadDone ? tableRows : []"
      :loading="!initialLoadDone"
      :striped="false"
      clickable
      row-test-id="watchlist-row"
      :row-class="rowClass"
      @row-click="onRowClick"
      @row-contextmenu="onRowContextMenu"
    >
      <template #action="{ row }">
        <button class="remove-btn" :title="t('common.delete')" @click.stop="removeSymbol(row.symbol)">✕</button>
      </template>
    </PanelTable>

    <!-- Polling indicator -->
    <div v-if="!pollingActive && symbols.length" class="polling-badge">{{ t('watchlist.polling_paused') }}</div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="ctxMenu" class="context-menu" :style="{ left: ctxMenu.x + 'px', top: ctxMenu.y + 'px' }">
        <div class="menu-item" @click="contextOpenKline">{{ t('watchlist.context_open_kline') }}</div>
        <div class="menu-sep"></div>
        <div class="menu-item" @click="contextOpenAnalysis('dupont-analysis')">杜邦分析</div>
        <div class="menu-item" @click="contextOpenAnalysis('shareholder-analysis')">股东分析</div>
        <div class="menu-item" @click="contextOpenAnalysis('event-study')">事件分析</div>
        <div class="menu-sep"></div>
        <div class="menu-item" @click="contextCopyCode">{{ t('watchlist.context_copy') }}</div>
        <div class="menu-sep"></div>
        <div class="menu-item danger" @click="contextDelete">{{ t('watchlist.context_delete') }}</div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.watchlist-panel {
  height: 100%; display: flex; flex-direction: column; overflow: hidden;
  position: relative;
}

/* Selected row (rows live inside PanelTable, hence :deep) */
.watchlist-panel :deep(.table-row.active) {
  background: var(--color-accent-soft);
  border-left: 2px solid var(--color-accent);
}

/* Remove button (#action slot content renders in this scope) */
.remove-btn {
  background: none; border: none; color: var(--color-text-tertiary);
  cursor: pointer; font-size: var(--font-xs); padding: var(--space-xs);
  opacity: 0; transition: opacity var(--transition-fast);
}
.table-row:hover .remove-btn { opacity: 0.5; }
.remove-btn:hover { opacity: 1 !important; color: var(--color-down); }

/* Polling badge */
.polling-badge {
  position: absolute; bottom: var(--space-sm); left: 50%; transform: translateX(-50%);
  font-size: var(--font-xs); color: var(--color-text-tertiary); background: var(--color-bg-elevated);
  padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-md); border: 1px solid var(--color-border-subtle);
}

/* Context menu */
.context-menu {
  position: fixed; z-index: var(--z-tooltip);
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md); padding: var(--space-xs) 0;
  min-width: 120px;
}
.menu-item { padding: var(--space-xs) var(--space-md); font-size: var(--font-xs); cursor: pointer; color: var(--color-text-primary); transition: background var(--transition-fast); }
.menu-item:hover { background: var(--color-accent); color: var(--color-text-inverse); }
.menu-item.danger:hover { background: var(--color-down); }
.menu-sep { height: 1px; margin: var(--space-xs) var(--space-sm); background: var(--color-border-subtle); }
</style>
