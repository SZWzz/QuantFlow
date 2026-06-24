<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDataStore } from '@/stores/data'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const dataStore = useDataStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

const symbols = ref<string[]>(['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276'])
const newSymbol = ref('')
const quotes = ref<Record<string, any>>({})
const loading = ref<Record<string, boolean>>({})

async function refreshQuote(sym: string) {
  loading.value[sym] = true
  try {
    const [snapshot, _source] = await (window as any).go.main.App.GetQuote({}, 'CN', sym)
    quotes.value[sym] = {
      symbol: snapshot.symbol ?? sym,
      last: snapshot.last ?? 0,
      change: snapshot.change ?? 0,
      changePct: snapshot.change_pct ?? snapshot.changePct ?? 0,
    }
  } catch {
    quotes.value[sym] = null as any
  } finally {
    loading.value[sym] = false
  }
}

function addSymbol() {
  const sym = newSymbol.value.trim().toUpperCase()
  if (sym && !symbols.value.includes(sym)) {
    symbols.value.push(sym)
    refreshQuote(sym)
  }
  newSymbol.value = ''
}

function removeSymbol(sym: string) {
  symbols.value = symbols.value.filter(s => s !== sym)
}

function selectSymbol(sym: string) {
  ctx.setGroupSymbol(pg.groupId, sym)
}

function formatPrice(p: number): string { return p.toFixed(2) }

function formatChange(c: number, pct: number): string {
  const sign = c >= 0 ? '+' : ''
  return `${sign}${c.toFixed(2)} (${sign}${pct.toFixed(2)}%)`
}

onMounted(() => {
  symbols.value.forEach(sym => refreshQuote(sym))
})
</script>

<template>
  <div class="watchlist-panel">
    <div class="panel-toolbar">
      <input v-model="newSymbol" type="text" placeholder="Add symbol..." class="symbol-input" @keyup.enter="addSymbol" />
      <button class="add-btn" @click="addSymbol">+</button>
    </div>
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
          <span class="symbol-name">{{ sym }}</span>
        </div>
        <div v-if="loading[sym]" class="symbol-price"><span class="no-data">--</span></div>
        <div v-else-if="quotes[sym]" class="symbol-price">
          <span class="last">{{ formatPrice(quotes[sym].last) }}</span>
          <span class="change">{{ formatChange(quotes[sym].change, quotes[sym].changePct) }}</span>
        </div>
        <div v-else class="symbol-price"><span class="no-data">--</span></div>
        <button class="remove-btn" @click.stop="removeSymbol(sym)" title="Remove">✕</button>
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
.panel-toolbar {
  display: flex; gap: 4px; padding: 8px;
  border-bottom: 1px solid var(--color-border);
}
.symbol-input {
  flex: 1; padding: 4px 8px;
  background: var(--color-bg-input); border: 1px solid var(--color-border);
  border-radius: var(--radius-sm); color: var(--color-text-primary);
  font-size: var(--font-xs); outline: none;
}
.symbol-input:focus { border-color: var(--color-accent); }
.add-btn {
  padding: 4px 10px; background: var(--color-bg-subtle);
  border: 1px solid var(--color-border); color: var(--color-accent);
  border-radius: var(--radius-sm); cursor: pointer; font-size: 14px; font-weight: bold;
}
.symbol-list { flex: 1; overflow-y: auto; }
.symbol-row {
  display: flex; align-items: center; padding: 6px 8px;
  border-bottom: 1px solid var(--color-border-subtle);
  cursor: pointer; transition: background var(--transition-fast);
}
.symbol-row:hover { background: var(--color-bg-hover); }
.symbol-row.active {
  background: var(--color-accent-soft);
  border-left: 2px solid var(--color-accent);
  padding-left: 6px;
}
.symbol-row.up .symbol-name { color: var(--color-up); }
.symbol-row.down .symbol-name { color: var(--color-down); }
.symbol-row.up .last { color: var(--color-up); }
.symbol-row.down .last { color: var(--color-down); }
.symbol-info { flex: 1; }
.symbol-name { font-weight: 600; font-size: var(--font-base); color: var(--color-text-primary); }
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
