<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useResearchStore } from '@/stores/research'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loaded')

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const store = useResearchStore()
const loadError = ref('')
const partyFilter = ref('All')
const chamberFilter = ref('All')
const parties = ['All', 'Democrat', 'Republican', 'Independent']
const chambers = ['All', 'House', 'Senate']

const filteredTrades = computed(() => { const trades = store.congressTrades ?? []; return trades.filter((t: any) => { if (partyFilter.value !== 'All' && t.party !== partyFilter.value) return false; if (chamberFilter.value !== 'All' && t.chamber !== chamberFilter.value) return false; return true }) })
const totalBuyVolume = computed(() => { const isBuy = (t: any) => (t.type ?? '').toLowerCase() === 'buy'; const buys = filteredTrades.value.filter(isBuy); const sells = filteredTrades.value.filter((t: any) => (t.type ?? '').toLowerCase() === 'sell'); return `${buys.length} Buys / ${sells.length} Sells` })

onMounted(async () => { loadError.value = ''; try { await store.fetchCongressTrades() } catch (e: any) { loadError.value = e?.message || String(e) } })
function setPartyFilter(p: string) { partyFilter.value = p }
function setChamberFilter(c: string) { chamberFilter.value = c }
async function refresh() { loadError.value = ''; try { await store.fetchCongressTrades() } catch (e: any) { loadError.value = e?.message || String(e) } }
function amountColor(amount: string): string { if (!amount) return 'var(--color-text-tertiary)'; const top = ['$1M-$5M', '$5M+', '$500K-$1M']; return top.some(t => amount.includes(t)) ? 'var(--chart-3)' : 'var(--color-text-primary)' }
</script>

<template>
  <PanelShell :state="state">
    <template #loaded>
      <div class="congress-panel">
        <PanelHeader title="国会交易" :controls="[{ icon: 'refresh', title: '刷新', action: refresh, loading: store.loading }]" />

        <div class="filter-bar">
          <div class="filter-group"><span class="filter-label">{{ $t('research.party') }}</span>
            <div class="filter-buttons"><button v-for="p in parties" :key="p" :class="['btn btn-sm', { 'btn-primary': partyFilter === p }]" @click="setPartyFilter(p)">{{ p === 'All' ? $t('common.all') : p }}</button></div>
          </div>
          <div class="filter-group"><span class="filter-label">{{ $t('research.chamber') }}</span>
            <div class="filter-buttons"><button v-for="c in chambers" :key="c" :class="['btn btn-sm', { 'btn-primary': chamberFilter === c }]" @click="setChamberFilter(c)">{{ c === 'All' ? $t('common.all') : c }}</button></div>
          </div>
        </div>

        <div class="summary-bar" v-if="store.congressTrades"><span>Showing {{ filteredTrades.length }} / {{ store.congressTrades.length }} trades</span><span>{{ totalBuyVolume }}</span></div>

        <div v-if="loadError" class="panel-error">{{ loadError }}</div>
        <LoadingState v-if="store.loading" type="table" :rows="8" :cols="7" />
        <EmptyState v-else-if="!store.congressTrades" title="暂无数据" />
        <div v-else class="panel-content">
          <table v-if="filteredTrades.length > 0" class="congress-table">
            <thead><tr><th>{{ $t('common.name') }}</th><th>{{ $t('research.chamber') }}</th><th>{{ $t('research.party') }}</th><th>{{ $t('quote.symbol') }}</th><th>{{ $t('common.type') }}</th><th>{{ $t('common.amount') }}</th><th>{{ $t('common.date') }}</th></tr></thead>
            <tbody><tr v-for="(t, i) in filteredTrades" :key="t.name + t.symbol + t.date + i"><td class="name-cell">{{ t.name }}</td><td>{{ t.chamber }}</td><td><span :class="['party-badge', (t.party ?? '').toLowerCase()]">{{ t.party }}</span></td><td class="symbol-cell">{{ t.symbol }}</td><td><span :class="['type-badge', (t.type ?? '').toLowerCase()]">{{ t.type }}</span></td><td class="amount-cell" :style="{ color: amountColor(t.amount) }">{{ t.amount }}</td><td class="date-cell">{{ t.date }}</td></tr></tbody>
          </table>
          <EmptyState v-else title="无匹配交易" />
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.congress-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.filter-bar { display: flex; gap: var(--space-lg); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); flex-wrap: wrap; }
.filter-group { display: flex; align-items: center; gap: var(--space-sm); }
.filter-label { font-size: var(--font-xs); color: var(--color-text-primary); font-weight: 500; text-transform: uppercase; }
.filter-buttons { display: flex; gap: var(--space-xs); }
.summary-bar { display: flex; justify-content: space-between; padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle); font-size: var(--font-xs); color: var(--color-text-secondary); }
.panel-error { padding: var(--space-sm) var(--panel-padding); color: var(--color-danger); font-size: var(--font-xs); }
.panel-content { flex: 1; overflow-y: auto; }
.congress-table { width: 100%; border-collapse: collapse; font-size: var(--font-xs); }
.congress-table th { text-align: left; padding: var(--space-sm); color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.congress-table td { padding: var(--space-sm); border-bottom: 1px solid var(--color-border-subtle); color: var(--color-text-primary); }
.name-cell { font-weight: 500; }
.symbol-cell { font-weight: 600; color: var(--color-accent); }
.party-badge { padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-sm); font-size: var(--font-xs); font-weight: 600; }
.party-badge.democrat { background: var(--color-accent-soft); color: var(--color-accent); }
.party-badge.republican { background: var(--color-up-soft); color: var(--color-up); }
.party-badge.independent { background: var(--color-border-subtle); color: var(--color-text-secondary); }
.type-badge { padding: var(--space-xs) var(--space-md); border-radius: var(--radius-lg); font-size: var(--font-xs); font-weight: 600; }
.type-badge.buy { background: var(--color-down-soft); color: var(--color-down); }
.type-badge.sell { background: var(--color-up-soft); color: var(--color-up); }
.amount-cell { font-variant-numeric: tabular-nums; font-weight: 500; }
.date-cell { color: var(--color-text-primary); }
</style>
