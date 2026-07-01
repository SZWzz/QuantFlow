<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useDataStore } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'
import { detectMarket } from '@/lib/wails'
import { PanelHeader } from '@/terminal/components/panel'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const { fetchWithCache } = usePanelCache()

const WS_KEY = 'quantflow-watchlist'
const defaultSymbols = ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276']

function loadSymbols(): string[] {
  try {
    const saved = localStorage.getItem(WS_KEY)
    if (saved) { const arr = JSON.parse(saved); if (Array.isArray(arr) && arr.length > 0) return arr }
  } catch(e) { console.error('[Watchlist] localStorage:', e) }
  return [...defaultSymbols]
}

function saveSymbols(syms: string[]) {
  localStorage.setItem(WS_KEY, JSON.stringify(syms))
}

const symbols = ref<string[]>(loadSymbols())
const quotes = ref<Record<string, any>>({})
const loading = ref<Record<string, boolean>>({})

async function refreshQuote(sym: string) {
  loading.value[sym] = true
  try {
    const { data: result } = await fetchWithCache(`quote:${detectMarket(sym)}:${sym}`, () => (window as any).go?.main?.App?.GetQuote(detectMarket(sym), sym), 60 * 1000)
    const snapshot = Array.isArray(result) ? result[0] : result
    quotes.value[sym] = {
      symbol: snapshot?.symbol ?? sym,
      name: snapshot?.name || quotes.value[sym]?.name || sym,
      last: snapshot?.last ?? 0,
      change: snapshot?.change ?? 0,
      changePct: snapshot?.change_pct ?? snapshot?.changePct ?? 0,
    }
  } catch(e) {
    console.error('[Watchlist] fetch:', e)
    if (!quotes.value[sym]) {
      quotes.value[sym] = { symbol: sym, name: sym, last: 0, change: 0, changePct: 0 }
    }
  } finally {
    loading.value[sym] = false
  }
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter(s => s !== sym)
  saveSymbols(symbols.value)
}

function selectSymbol(sym: string) {
  ctx.setGroupSymbol(pg.groupId, sym)
}

async function refreshAll() {
  await Promise.all(symbols.value.map(sym => refreshQuote(sym)))
}

function formatPrice(p: number): string {
  if (typeof p !== 'number' || p === 0) return '--'
  return p.toFixed(2)
}

function formatChange(c: number, pct: number): string {
  if (typeof c !== 'number' || typeof pct !== 'number') return '--'
  if (c === 0 && pct === 0) return '--'
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)} (${sign}${pct.toFixed(2)}%)`
}

async function onWatchlistChanged() {
  symbols.value = loadSymbols()
  await Promise.all(symbols.value.map(sym => refreshQuote(sym)))
}

onMounted(async () => {
  window.addEventListener('watchlist-changed', onWatchlistChanged)
  try {
    const app = (window as any).go?.main?.App
    if (app?.SearchSymbols) {
      await Promise.all(symbols.value.map(async (sym) => {
        const { data: results } = await fetchWithCache(`search:${sym}`, () => app.SearchSymbols(sym, 1), 5 * 60 * 1000)
        if (Array.isArray(results) && results.length > 0 && results[0].name) {
          if (!quotes.value[sym]) {
            quotes.value[sym] = { symbol: sym, name: results[0].name, last: 0, change: 0, changePct: 0 }
          } else if (quotes.value[sym].name === sym) {
            quotes.value[sym].name = results[0].name
          }
        }
      }))
    }
  } catch { /* best-effort */ }
  await Promise.all(symbols.value.map(sym => refreshQuote(sym)))
})

onUnmounted(() => {
  window.removeEventListener('watchlist-changed', onWatchlistChanged)
})
</script>

<template>
  <div class="watchlist-panel">
    <PanelHeader
      :title="$t('watchlist.title')"
      :controls="[
        { icon: 'refresh', label: $t('common.refresh'), action: refreshAll },
      ]"
    />
    <div class="symbol-list">
      <div
        v-for="sym in symbols" :key="sym"
        class="symbol-row"
        :class="{
          up: (quotes[sym]?.change || 0) >= 0,
          down: (quotes[sym]?.change || 0) < 0,
          active: ctx.getGroupSymbol(pg.groupId) === sym,
        }"
        @click="selectSymbol(sym)"
      >
        <div class="symbol-info">
          <span class="symbol-code">{{ sym }}</span>
          <span class="symbol-name">{{ quotes[sym]?.name || sym }}</span>
        </div>
        <div v-if="loading[sym]" class="symbol-price"><span class="no-data">--</span></div>
        <div v-else-if="quotes[sym]" class="symbol-price">
          <span class="last">{{ formatPrice(quotes[sym].last) }}</span>
          <span class="change">{{ formatChange(quotes[sym].change, quotes[sym].changePct) }}</span>
        </div>
        <div v-else class="symbol-price"><span class="no-data">--</span></div>
        <button class="remove-btn" @click.stop="removeSymbol(sym)" :title="$t('common.delete')">✕</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.watchlist-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
}
.symbol-list { flex: 1; overflow-y: auto; }
.symbol-row {
  display: flex; align-items: center; padding: var(--panel-padding-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  cursor: pointer; transition: background var(--transition-fast);
}
.symbol-row:hover { background: var(--color-bg-hover); }
.symbol-row.active {
  background: var(--color-accent-soft);
  border-left: 2px solid var(--color-accent);
  padding-left: 6px;
}
.symbol-row.up .symbol-code { color: var(--color-up); }
.symbol-row.down .symbol-code { color: var(--color-down); }
.symbol-row.up .last { color: var(--color-up); }
.symbol-row.down .last { color: var(--color-down); }
.symbol-info {
  flex: 1; display: flex; flex-direction: column; gap: 1px; min-width: 0;
}
.symbol-code { font-weight: 600; font-size: var(--font-base); color: var(--color-text-primary); }
.symbol-name {
  font-size: 11px; color: var(--color-text-tertiary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.symbol-price { text-align: right; margin-right: 8px; }
.last { font-size: var(--font-base); font-weight: 500; font-variant-numeric: tabular-nums; }
.change { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.symbol-row.up .change { color: var(--color-up); }
.symbol-row.down .change { color: var(--color-down); }
.no-data { font-size: var(--font-sm); color: var(--color-text-tertiary); }
.remove-btn {
  background: none; border: none; color: var(--color-text-tertiary);
  cursor: pointer; font-size: 12px; padding: 2px 4px; opacity: 0;
  transition: opacity var(--transition-fast);
}
.symbol-row:hover .remove-btn { opacity: 0.6; }
.remove-btn:hover { opacity: 1; color: var(--color-down); }
</style>
