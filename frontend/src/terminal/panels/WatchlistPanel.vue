<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, watch, nextTick } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { PanelHeader, PanelTable, EmptyState, LoadingState, type Column } from '@/terminal/components/panel'
import type { FlashClass } from '@/lib/composables/useFlashOnUpdate'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWebSocket } from '@/lib/composables/useWebSocket'
import { useI18n } from 'vue-i18n'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()
const ws = useWebSocket()
const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/market`
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const { t } = useI18n()

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')
const loadError = ref('')

const controls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', label: t('common.refresh'), action: refreshAll })
  // 右键菜单操作的键盘可达入口：对当前选中 symbol 打开同一操作菜单
  list.push({ icon: 'dots', title: t('common.actions'), action: openSymbolActions })
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

interface SortState { key: string; dir: 'asc' | 'desc' | null }

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
const sort = ref<SortState>({ key: '', dir: null })
const expandedGroups = reactive<Record<string, boolean>>({ CN: true, HK: true, US: true, CRYPTO: true })
const pollingActive = ref(true)

// Context menu
const ctxMenu = ref<{ x: number; y: number; symbol: string } | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const rootEl = ref<HTMLElement | null>(null)
/** 打开菜单的触发元素（行或头部按钮），关闭时归还焦点 */
let menuTrigger: HTMLElement | null = null

const tableColumns = computed<Column[]>(() => [
  { key: 'symbol', label: t('common.symbol'), width: 70 },
  { key: 'name', label: t('common.name'), flex: 1 },
  { key: 'last', label: t('common.price'), align: 'right', format: 'price', width: 72, sortable: true },
  { key: 'changePct', label: t('quote.change_pct'), align: 'right', format: 'percent', colorize: true, width: 76, sortable: true },
  { key: 'turnover', label: t('quote.turnover'), align: 'right', format: 'volume', width: 80, sortable: true },
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

// 排序语义与旧版 toggleSort 一致：新列 asc → 同列 desc → 同列清除；仅数值列可排，缺失值按 0
const sortedRows = computed<WatchRow[]>(() => {
  const s = sort.value
  if (!s.dir || !s.key) return tableRows.value
  return [...tableRows.value].sort((a, b) => {
    const va = (a[s.key as keyof WatchRow] as number) ?? 0
    const vb = (b[s.key as keyof WatchRow] as number) ?? 0
    if (va === vb) return 0
    return s.dir === 'asc' ? (va - vb) : (vb - va)
  })
})

function onSortChange(key: string, dir: 'asc' | 'desc' | null) {
  sort.value = dir ? { key, dir } : { key: '', dir: null }
}

const groups = computed(() => {
  const map: Record<string, WatchRow[]> = { CN: [], HK: [], US: [], CRYPTO: [] }
  for (const row of sortedRows.value) {
    const m = detectMarket(row.symbol)
    if (map[m]) map[m].push(row); else map.CN.push(row)
  }
  return map
})

// 仅非空分组，保持 CN/HK/US/CRYPTO 固定顺序；首个展开的分组显示表头
const groupList = computed(() =>
  (['CN', 'HK', 'US', 'CRYPTO'] as const)
    .map(mkt => ({ mkt, rows: groups.value[mkt] }))
    .filter(g => g.rows.length > 0),
)

// 表头跟随第一个展开的分组，避免折叠首组后排序 UI 不可达
const firstExpandedIndex = computed(() => groupList.value.findIndex(g => expandedGroups[g.mkt]))

function toggleGroup(market: string) {
  expandedGroups[market] = !expandedGroups[market]
}

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
  // 键盘场景：删除前定位焦点锚点行（右键菜单触发行或当前焦点行），删除后焦点落到补位的行
  const rowsBefore = Array.from(rootEl.value?.querySelectorAll('[data-testid="watchlist-row"]') ?? []) as HTMLElement[]
  const anchor = (menuTrigger?.matches('[data-testid="watchlist-row"]') ? menuTrigger : document.activeElement) as HTMLElement | null
  const focusIdx = rowsBefore.findIndex(el => el === anchor || el.contains(anchor))
  symbols.value = symbols.value.filter(s => s !== sym)
  saveSymbols(symbols.value)
  clearTimeout(flashTimers[sym])
  delete flashTimers[sym]
  delete flashMap[sym]
  window.dispatchEvent(new CustomEvent('watchlist-changed'))
  if (focusIdx >= 0) {
    nextTick(() => {
      const rowsAfter = Array.from(rootEl.value?.querySelectorAll('[data-testid="watchlist-row"]') ?? []) as HTMLElement[]
      if (rowsAfter.length > 0) rowsAfter[Math.min(focusIdx, rowsAfter.length - 1)].focus()
    })
  }
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

// Context menu：打开后聚焦首个菜单项，关闭时焦点归还触发元素
function focusFirstMenuItem() {
  nextTick(() => menuRef.value?.querySelector<HTMLElement>('[role="menuitem"]')?.focus())
}

function openContextMenu(e: MouseEvent, sym: string) {
  menuTrigger = e.currentTarget instanceof HTMLElement ? e.currentTarget : null
  ctxMenu.value = { x: e.clientX, y: e.clientY, symbol: sym }
  focusFirstMenuItem()
}

/** PanelHeader controls 的键盘可达入口：对当前选中 symbol（缺省取首只）打开同一菜单 */
function openSymbolActions(e: MouseEvent) {
  e.stopPropagation() // 阻止 document click 立即关闭刚打开的菜单
  const sym = ctx.getGroupSymbol(pg.groupId) || symbols.value[0]
  if (!sym) return
  const btn = e.currentTarget instanceof HTMLElement ? e.currentTarget : null
  const rect = btn?.getBoundingClientRect()
  menuTrigger = btn
  ctxMenu.value = { x: rect?.left ?? 0, y: (rect?.bottom ?? 0) + 4, symbol: sym }
  focusFirstMenuItem()
}

function closeContextMenu() {
  if (!ctxMenu.value) return
  const focusInside = menuRef.value?.contains(document.activeElement) ?? false
  ctxMenu.value = null
  if (focusInside && menuTrigger?.isConnected) menuTrigger.focus()
  menuTrigger = null
}

/** 菜单内键盘导航：↑↓ 循环、Home/End、Esc 关闭归还焦点、Tab 关闭 */
function onMenuKeydown(e: KeyboardEvent) {
  const items = Array.from(menuRef.value?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])
  if (e.key === 'Escape') {
    e.preventDefault()
    closeContextMenu()
    return
  }
  if (e.key === 'Tab') {
    closeContextMenu()
    return
  }
  if (items.length === 0) return
  const idx = items.indexOf(document.activeElement as HTMLElement)
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      items[(idx + 1) % items.length].focus()
      break
    case 'ArrowUp':
      e.preventDefault()
      items[(idx - 1 + items.length) % items.length].focus()
      break
    case 'Home':
      e.preventDefault()
      items[0].focus()
      break
    case 'End':
      e.preventDefault()
      items[items.length - 1].focus()
      break
  }
}

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
  // 一次性清理迁移前列设置遗留的 localStorage key（列设置已随 5 列固定规格下线）
  localStorage.removeItem('quantflow-watchlist-config')

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
  try {
    await refreshAll()
    state.value = symbols.value.length > 0 ? 'loaded' : 'empty'
  } catch (e: any) {
    loadError.value = e?.message || String(e)
    state.value = 'error'
  }
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
  <PanelShell :state="state" :error="loadError" data-testid="watchlist-panel" @retry="refreshAll">
    <template #empty>
      <div data-testid="watchlist-empty">{{ t('watchlist.empty') }}</div>
    </template>
    <template #loaded>
      <div class="watchlist-panel" ref="rootEl">
        <PanelHeader
          :title="t('watchlist.title')"
          :controls="controls"
        />

        <div class="watchlist-groups">
          <template v-for="(g, gi) in groupList" :key="g.mkt">
            <button
              type="button"
              class="group-header"
              :aria-expanded="expandedGroups[g.mkt]"
              @click="toggleGroup(g.mkt)"
            >
              <span class="group-arrow" aria-hidden="true">{{ expandedGroups[g.mkt] ? '▼' : '▶' }}</span>
              <span class="group-label">{{ t('watchlist.group_' + g.mkt.toLowerCase()) }}</span>
              <span class="group-count">{{ g.rows.length }}</span>
            </button>
            <PanelTable
              v-if="expandedGroups[g.mkt]"
              :columns="tableColumns"
              :data="g.rows"
              :striped="false"
              clickable
              row-test-id="watchlist-row"
              :row-class="rowClass"
              :sort-key="sort.key"
              :sort-dir="sort.dir"
              :hide-header="gi !== firstExpandedIndex"
              @row-click="onRowClick"
              @row-contextmenu="onRowContextMenu"
              @sort-change="onSortChange"
            >
              <template #action="{ row }">
                <button class="remove-btn" :title="t('common.delete')" @click.stop="removeSymbol(row.symbol)">✕</button>
              </template>
            </PanelTable>
          </template>
        </div>

        <!-- Polling indicator -->
        <div v-if="!pollingActive && symbols.length" class="polling-badge">{{ t('watchlist.polling_paused') }}</div>

        <!-- Context menu -->
        <Teleport to="body">
          <div
            v-if="ctxMenu"
            ref="menuRef"
            class="context-menu"
            role="menu"
            :aria-label="ctxMenu.symbol"
            :style="{ left: ctxMenu.x + 'px', top: ctxMenu.y + 'px' }"
            @keydown="onMenuKeydown"
          >
            <button type="button" class="menu-item" role="menuitem" @click="contextOpenKline">{{ t('watchlist.context_open_kline') }}</button>
            <div class="menu-sep" role="separator"></div>
            <button type="button" class="menu-item" role="menuitem" @click="contextOpenAnalysis('dupont-analysis')">杜邦分析</button>
            <button type="button" class="menu-item" role="menuitem" @click="contextOpenAnalysis('shareholder-analysis')">股东分析</button>
            <button type="button" class="menu-item" role="menuitem" @click="contextOpenAnalysis('event-study')">事件分析</button>
            <div class="menu-sep" role="separator"></div>
            <button type="button" class="menu-item" role="menuitem" @click="contextCopyCode">{{ t('watchlist.context_copy') }}</button>
            <div class="menu-sep" role="separator"></div>
            <button type="button" class="menu-item danger" role="menuitem" @click="contextDelete">{{ t('watchlist.context_delete') }}</button>
          </div>
        </Teleport>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.watchlist-panel {
  height: 100%; display: flex; flex-direction: column; overflow: hidden;
  position: relative;
}

/* Selected row (rows live inside PanelTable, hence :deep)：规范 §3.3 选中行只用 --color-bg-selected，无装饰色条 */
.watchlist-panel :deep(.table-row.active) {
  background: var(--color-bg-selected);
}

/* Group accordion：单一滚动容器，分组内 PanelTable 按内容自然撑高 */
.watchlist-groups {
  flex: 1;
  overflow-y: auto;
}

.watchlist-groups :deep(.panel-table-wrapper) {
  flex: none;
  overflow: visible;
}

.watchlist-groups :deep(.table-body) {
  flex: none;
  overflow: visible;
}

.group-header {
  display: flex; align-items: center; gap: var(--space-xs);
  width: 100%; padding: var(--space-xs) var(--space-sm); cursor: pointer; user-select: none;
  background: var(--color-bg-elevated); border: 0; border-bottom: 1px solid var(--color-border-subtle);
  font-family: inherit; font-size: var(--font-xs); text-align: left; color: var(--color-text-secondary);
  position: sticky; top: 0; z-index: 1;
}
.group-header:hover { color: var(--color-text-primary); }
.group-arrow { font-size: var(--font-xs); width: var(--space-md); }
.group-count { margin-left: auto; font-size: var(--font-xs); color: var(--color-text-tertiary); }

/* Remove button (#action slot content renders in this scope) */
.remove-btn {
  background: none; border: 0; color: var(--color-text-tertiary);
  cursor: pointer; font-size: var(--font-xs); padding: var(--space-xs);
  opacity: 0; transition: opacity var(--transition-fast);
}
.table-row:hover .remove-btn { opacity: 0.5; }
/* 键盘焦点进入行/按钮时同样需要可见，否则键盘用户无法发现删除入口 */
.table-row:focus-within .remove-btn { opacity: 0.5; }
.remove-btn:hover { opacity: 1 !important; color: var(--color-down); }
.remove-btn:focus-visible { opacity: 1 !important; }

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
  box-shadow: var(--shadow-md);
}
.menu-item {
  display: block; width: 100%;
  padding: var(--space-xs) var(--space-md);
  border: 0; background: none;
  font-family: inherit; font-size: var(--font-xs); text-align: left;
  cursor: pointer; color: var(--color-text-primary); transition: background var(--transition-fast);
}
/* 键盘焦点与悬停同视觉：焦点移动即当前项 */
.menu-item:hover, .menu-item:focus-visible { background: var(--color-accent); color: var(--color-text-inverse); outline: none; }
.menu-item.danger:hover, .menu-item.danger:focus-visible { background: var(--color-down); }
.menu-sep { height: 1px; margin: var(--space-xs) var(--space-sm); background: var(--color-border-subtle); }
</style>
