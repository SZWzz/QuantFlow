<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { useDataStore } from '@/stores/data'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { PanelHeader } from '@/terminal/components/panel'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWebSocket } from '@/lib/composables/useWebSocket'
import { useI18n } from 'vue-i18n'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
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
const CONFIG_KEY = 'quantflow-watchlist-config'
const defaultSymbols = ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276']

interface QuoteRow {
  symbol: string; name: string; last: number; open: number; high: number; low: number
  change: number; changePct: number; volume: number; turnover: number
  turnoverRate: number; volumeRatio: number; amplitude: number; prevLast?: number
}

interface ColumnDef {
  key: string; label: string; visible: boolean; sortable: boolean; format?: 'price' | 'percent' | 'volume' | 'number'
}

interface SortState { key: string; dir: 'asc' | 'desc' | null }

const defaultColumns: ColumnDef[] = [
  { key: 'symbol', label: 'common.symbol', visible: true, sortable: false },
  { key: 'name', label: 'common.name', visible: true, sortable: false },
  { key: 'last', label: 'common.price', visible: true, sortable: true, format: 'price' },
  { key: 'changePct', label: 'quote.change_pct', visible: true, sortable: true, format: 'percent' },
  { key: 'change', label: 'quote.change', visible: false, sortable: true, format: 'price' },
  { key: 'speed', label: 'watchlist.speed', visible: true, sortable: true, format: 'percent' },
  { key: 'volumeRatio', label: 'kline.volume_ratio', visible: true, sortable: true, format: 'number' },
  { key: 'turnoverRate', label: 'kline.turnover', visible: false, sortable: true, format: 'percent' },
  { key: 'amplitude', label: 'kline.amplitude', visible: false, sortable: true, format: 'percent' },
  { key: 'volume', label: 'common.volume', visible: false, sortable: true, format: 'volume' },
  { key: 'turnover', label: 'quote.turnover', visible: false, sortable: true, format: 'volume' },
  { key: 'high', label: 'quote.high', visible: false, sortable: true, format: 'price' },
  { key: 'low', label: 'quote.low', visible: false, sortable: true, format: 'price' },
]

function loadColumns(): ColumnDef[] {
  try {
    const saved = localStorage.getItem(CONFIG_KEY)
    if (saved) {
      const parsed = JSON.parse(saved)
      if (Array.isArray(parsed)) return parsed
    }
  } catch {}
  return defaultColumns.map(c => ({ ...c }))
}

function saveColumns(cols: ColumnDef[]) {
  localStorage.setItem(CONFIG_KEY, JSON.stringify(cols))
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
const loading = ref<Record<string, boolean>>({})
const initialLoadDone = ref(false)
const sort = ref<SortState>({ key: '', dir: null })
const columns = ref<ColumnDef[]>(loadColumns())
const showColumnSettings = ref(false)
const expandedGroups = reactive<Record<string, boolean>>({ CN: true, HK: true, US: true, CRYPTO: true })
const pollingActive = ref(true)

// Context menu
const ctxMenu = ref<{ x: number; y: number; symbol: string } | null>(null)

// Drag state
const dragSymbol = ref<string | null>(null)

const visibleColumns = computed(() => columns.value.filter(c => c.visible))

const gridTemplateCols = computed(() => {
  const sizes: Record<string, string> = {
    symbol: '70px', name: '1fr', last: '72px', changePct: '76px',
    change: '68px', speed: '64px', volumeRatio: '56px', turnoverRate: '64px',
    amplitude: '64px', volume: '72px', turnover: '80px', high: '68px', low: '68px',
  }
  const cols = visibleColumns.value.map(c => sizes[c.key] || '60px')
  cols.push('28px')
  return cols.join(' ')
})

function getValue(sym: string, key: string): number {
  const q = quotes[sym]
  if (!q) return 0
  switch (key) {
    case 'last': return q.last
    case 'change': return q.change
    case 'changePct': return q.changePct
    case 'speed': return q.prevLast != null && q.prevLast > 0 ? (q.last - q.prevLast) / q.prevLast * 100 : 0
    case 'volumeRatio': return q.volumeRatio
    case 'turnoverRate': return q.turnoverRate
    case 'amplitude': return q.amplitude
    case 'volume': return q.volume
    case 'turnover': return q.turnover
    case 'high': return q.high
    case 'low': return q.low
    default: return 0
  }
}

const displaySymbols = computed(() => {
  const s = sort.value
  if (!s.dir || !s.key) return symbols.value
  return [...symbols.value].sort((a, b) => {
    const va = getValue(a, s.key); const vb = getValue(b, s.key)
    if (va === vb) return 0
    return s.dir === 'asc' ? (va - vb) : (vb - va)
  })
})

const groups = computed(() => {
  const map: Record<string, string[]> = { CN: [], HK: [], US: [], CRYPTO: [] }
  for (const sym of displaySymbols.value) {
    const m = detectMarket(sym)
    if (map[m]) map[m].push(sym); else map.CN.push(sym)
  }
  return map
})

function toggleSort(key: string) {
  if (sort.value.key === key) {
    if (sort.value.dir === 'asc') sort.value.dir = 'desc'
    else if (sort.value.dir === 'desc') sort.value = { key: '', dir: null }
    else sort.value.dir = 'asc'
  } else {
    sort.value = { key, dir: 'asc' }
  }
}

function toggleGroup(market: string) {
  expandedGroups[market] = !expandedGroups[market]
}

function formatCell(sym: string, col: ColumnDef): string {
  const q = quotes[sym]
  if (!q) return '--'
  const v = getValue(sym, col.key)
  if (v == null || (v === 0 && col.key !== 'volumeRatio' && col.key !== 'last')) return '--'
  switch (col.format) {
    case 'price': return v.toFixed(2)
    case 'percent': return (v >= 0 ? '+' : '') + v.toFixed(2) + '%'
    case 'volume': {
      if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
      if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
      return v.toFixed(0)
    }
    default: return String(v)
  }
}

function rowClasses(sym: string) {
  const q = quotes[sym]
  const chg = q?.change ?? 0
  const active = ctx.getGroupSymbol(pg.groupId) === sym
  return { up: chg >= 0, down: chg < 0, active }
}

function cellColor(sym: string, key: string): string {
  if (key === 'change' || key === 'changePct' || key === 'speed') {
    const v = getValue(sym, key)
    if (v > 0) return 'var(--color-up)'
    if (v < 0) return 'var(--color-down)'
  }
  return 'var(--color-text-primary)'
}

async function refreshQuote(sym: string) {
  loading.value[sym] = true
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
  delete loading.value[sym]
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
  delete loading.value[sym]
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter(s => s !== sym)
  saveSymbols(symbols.value)
  window.dispatchEvent(new CustomEvent('watchlist-changed'))
}

function selectSymbol(sym: string) {
  ctx.setGroupSymbol(pg.groupId, sym)
}

async function refreshAll() {
  await Promise.all(symbols.value.map(sym => refreshQuote(sym)))
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

// Drag & drop
function onDragStart(e: DragEvent, sym: string) {
  dragSymbol.value = sym
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}
function onDragOver(e: DragEvent, _sym: string) {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
}
function onDrop(e: DragEvent, targetSym: string) {
  if (!dragSymbol.value || dragSymbol.value === targetSym) return
  const list = [...symbols.value]
  const fromIdx = list.indexOf(dragSymbol.value)
  const toIdx = list.indexOf(targetSym)
  if (fromIdx < 0 || toIdx < 0) return
  list.splice(fromIdx, 1)
  list.splice(toIdx, 0, dragSymbol.value)
  symbols.value = list
  saveSymbols(list)
  dragSymbol.value = null
}

function onVisibility() {
  pollingActive.value = !document.hidden
  if (!pollingActive.value) return
  refreshAll()
}

// Column settings
function toggleColumn(key: string) {
  const col = columns.value.find(c => c.key === key)
  if (col) { col.visible = !col.visible; saveColumns(columns.value) }
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
  ws.disconnect()
})
</script>

<template>
  <div class="watchlist-panel" data-testid="watchlist-panel">
    <PanelHeader
      :title="$t('watchlist.title')"
      :controls="controls"
    />

    <!-- Empty state -->
    <div v-if="symbols.length === 0" class="empty-state" data-testid="watchlist-empty">
      <div class="empty-icon">📋</div>
      <div class="empty-text">{{ $t('watchlist.empty') }}</div>
    </div>

    <!-- Table -->
    <div v-else class="symbol-table">
      <!-- Header row -->
      <div class="table-header" :style="{ gridTemplateColumns: gridTemplateCols }">
        <div v-for="col in visibleColumns" :key="col.key"
          class="th" :class="{ sortable: col.sortable, asc: sort.key === col.key && sort.dir === 'asc', desc: sort.key === col.key && sort.dir === 'desc' }"
          :style="{ cursor: col.sortable ? 'pointer' : 'default' }"
          @click="col.sortable && toggleSort(col.key)">
          {{ $t(col.label) }}
          <span v-if="sort.key === col.key" class="sort-arrow">{{ sort.dir === 'asc' ? '↑' : '↓' }}</span>
        </div>
        <div class="th-actions">
          <button class="col-settings-btn" @click.stop="showColumnSettings = !showColumnSettings" :title="$t('watchlist.column_settings')">⚙</button>
          <div v-if="showColumnSettings" class="col-settings-popover" @click.stop>
            <div class="popover-title">{{ $t('watchlist.column_settings') }}</div>
            <label v-for="col in columns" :key="col.key" class="col-toggle">
              <input type="checkbox" :checked="col.visible" @change="toggleColumn(col.key)" />
              <span>{{ $t(col.label) }}</span>
            </label>
          </div>
        </div>
      </div>

      <!-- Group accordion -->
      <template v-for="(syms, mkt) in groups" :key="mkt">
        <div v-if="syms.length" class="group-header" @click="toggleGroup(mkt)">
          <span class="group-arrow">{{ expandedGroups[mkt] ? '▼' : '▶' }}</span>
          <span class="group-label">{{ $t('watchlist.group_' + mkt.toLowerCase()) }}</span>
          <span class="group-count">{{ syms.length }}</span>
        </div>

        <template v-if="expandedGroups[mkt]">
          <!-- Loading skeleton -->
          <div v-if="!initialLoadDone" v-for="i in Math.min(syms.length, 3)" :key="'skel-' + mkt + '-' + i" class="table-row skeleton" :style="{ gridTemplateColumns: gridTemplateCols }">
            <div v-for="col in visibleColumns" :key="col.key" class="td"><div class="skeleton-bar"></div></div>
            <div class="td-actions"></div>
          </div>

          <!-- Data rows -->
          <div v-for="sym in syms" v-else :key="sym"
            class="table-row" :class="rowClasses(sym)" data-testid="watchlist-row"
            :style="{ gridTemplateColumns: gridTemplateCols }"
            @click="selectSymbol(sym)"
            @contextmenu.prevent="openContextMenu($event, sym)"
            draggable="true"
            @dragstart="onDragStart($event, sym)"
            @dragover="onDragOver($event, sym)"
            @drop="onDrop($event, sym)">
            <template v-for="col in visibleColumns" :key="col.key">
              <div v-if="col.key === 'symbol'" class="td code" :style="{ color: cellColor(sym, 'changePct') }">{{ sym }}</div>
              <div v-else-if="col.key === 'name'" class="td name"><span class="name-text">{{ quotes[sym]?.name || sym }}</span></div>
              <div v-else class="td" :style="{ color: cellColor(sym, col.key) }">
                {{ loading[sym] ? '--' : formatCell(sym, col) }}
              </div>
            </template>
            <div class="td-actions">
              <button class="remove-btn" @click.stop="removeSymbol(sym)" :title="$t('common.delete')">✕</button>
            </div>
          </div>
        </template>
      </template>
    </div>

    <!-- Polling indicator -->
    <div v-if="!pollingActive && symbols.length" class="polling-badge">{{ $t('watchlist.polling_paused') }}</div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="ctxMenu" class="context-menu" :style="{ left: ctxMenu.x + 'px', top: ctxMenu.y + 'px' }">
        <div class="menu-item" @click="contextOpenKline">{{ $t('watchlist.context_open_kline') }}</div>
        <div class="menu-sep"></div>
        <div class="menu-item" @click="contextOpenAnalysis('dupont-analysis')">杜邦分析</div>
        <div class="menu-item" @click="contextOpenAnalysis('shareholder-analysis')">股东分析</div>
        <div class="menu-item" @click="contextOpenAnalysis('event-study')">事件分析</div>
        <div class="menu-sep"></div>
        <div class="menu-item" @click="contextCopyCode">{{ $t('watchlist.context_copy') }}</div>
        <div class="menu-sep"></div>
        <div class="menu-item danger" @click="contextDelete">{{ $t('watchlist.context_delete') }}</div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.watchlist-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--color-bg-panel); position: relative;
}

.empty-icon { font-size: 32px; opacity: 0.4; }
.empty-text { font-size: var(--font-sm); text-align: center; padding: 0 24px; line-height: 1.5; }

.symbol-table { flex: 1; overflow-y: auto; font-size: var(--font-xs); }

/* Header */
.table-header {
  display: grid; gap: 0; position: sticky; top: 0; z-index: 2;
  background: var(--color-bg-elevated); border-bottom: 1px solid var(--color-border-strong);
  padding: 0 8px; font-size: 10px; color: var(--color-text-tertiary);
  grid-template-columns: repeat(auto-fit, minmax(0, 1fr));
}
.th { padding: 6px 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; user-select: none; }
.th.sortable:hover { color: var(--color-text-primary); }
.sort-arrow { margin-left: 2px; font-size: 9px; }
.th-actions { display: flex; align-items: center; justify-content: flex-end; padding: 2px; position: relative; }
.col-settings-btn { background: none; border: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 13px; padding: 2px 4px; border-radius: var(--radius-sm); }
.col-settings-btn:hover { color: var(--color-text-primary); background: var(--color-bg-hover); }

.col-settings-popover {
  position: absolute; top: 100%; right: 0; z-index: 100;
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md); padding: 8px; min-width: 140px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}
.popover-title { font-size: 11px; font-weight: 600; margin-bottom: 6px; color: var(--color-text-primary); }
.col-toggle { display: flex; align-items: center; gap: 6px; padding: 3px 0; font-size: 11px; cursor: pointer; color: var(--color-text-secondary); }
.col-toggle:hover { color: var(--color-text-primary); }
.col-toggle input { margin: 0; }

/* Group header */
.group-header {
  display: flex; align-items: center; gap: 6px;
  padding: 4px 10px; cursor: pointer; user-select: none;
  background: var(--color-bg-elevated); border-bottom: 1px solid var(--color-border-subtle);
  font-size: 11px; color: var(--color-text-secondary); position: sticky; top: 28px; z-index: 1;
}
.group-header:hover { color: var(--color-text-primary); }
.group-arrow { font-size: 8px; width: 12px; }
.group-count { margin-left: auto; font-size: 10px; color: var(--color-text-tertiary); }

/* Row */
.table-row {
  display: grid; gap: 0; align-items: center;
  padding: 0 8px; cursor: pointer; transition: background var(--transition-fast);
  border-bottom: 1px solid var(--color-border-subtle);
  min-height: 32px;
}
.table-row:hover { background: var(--color-bg-hover); }
.table-row.active {
  background: var(--color-accent-soft);
  border-left: 2px solid var(--color-accent);
  padding-left: 6px;
}
.table-row.dragging { opacity: 0.4; }
.td { padding: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-variant-numeric: tabular-nums; }
.td.code { font-weight: 600; font-size: var(--font-xs); }
.td.name { overflow: hidden; }
.name-text { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; display: block; color: var(--color-text-tertiary); font-size: 10px; }
.td-actions { display: flex; align-items: center; justify-content: flex-end; }
.remove-btn {
  background: none; border: none; color: var(--color-text-tertiary);
  cursor: pointer; font-size: 10px; padding: 2px 4px; opacity: 0;
  transition: opacity var(--transition-fast);
}
.table-row:hover .remove-btn { opacity: 0.5; }
.remove-btn:hover { opacity: 1 !important; color: var(--color-down); }

/* Skeleton */
.skeleton-bar {
  height: 10px; background: var(--color-border-strong); border-radius: 4px;
  animation: shimmer 1.2s ease-in-out infinite;
}
@keyframes shimmer { 0% { opacity: 0.3; } 50% { opacity: 0.7; } 100% { opacity: 0.3; } }

/* Polling badge */
.polling-badge {
  position: absolute; bottom: 8px; left: 50%; transform: translateX(-50%);
  font-size: 9px; color: var(--color-text-tertiary); background: var(--color-bg-elevated);
  padding: 2px 8px; border-radius: 8px; border: 1px solid var(--color-border-subtle);
}

/* Context menu */
.context-menu {
  position: fixed; z-index: var(--z-tooltip);
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md); padding: 4px 0;
  min-width: 120px; box-shadow: var(--shadow-md);
}
.menu-item { padding: 6px 14px; font-size: 12px; cursor: pointer; color: var(--color-text-primary); transition: background 0.1s; }
.menu-item:hover { background: var(--color-accent); color: #fff; }
.menu-item.danger:hover { background: var(--color-down); }
.menu-sep { height: 1px; margin: 4px 8px; background: var(--color-border-subtle); }
</style>
