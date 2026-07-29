<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { PanelHeader, LoadingState, ErrorState, EmptyState } from '@/terminal/components/panel'

const { fetchWithCache } = usePanelCache()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol ?? ctx.getGroupSymbol(pg.groupId) ?? '000001')
const { name } = useStockName(symbol)
const loading = ref(false); const error = ref('')
const filings = ref<any[]>([])
const selectedFormType = ref('All')
const formTypes = ['All', '4', '13F-HR', '13D', '13G']
const filteredFilings = computed(() => selectedFormType.value === 'All' ? filings.value : filings.value.filter(f => f.form === selectedFormType.value))

function formBadgeClass(form: string): string { if (!form) return 'badge-gray'; if (form === '4') return 'badge-blue'; if (form === '13F-HR') return 'badge-green'; return 'badge-yellow' }
function openUrl(url: string) { if (url) window.open(url, '_blank') }
function setFormType(ft: string) { selectedFormType.value = ft }

async function loadFilings() { loading.value = true; error.value = ''; try { const app = useWailsApp(); if (!app?.GetSECFilings) { error.value = 'SEC 数据源未连接'; return }; const { data } = await fetchWithCache<any>(`sec:${symbol.value}`, () => app.GetSECFilings(symbol.value), 300000); filings.value = data && Array.isArray(data) ? data : Array.from(data || []) } catch (e: any) { console.error('[DarkPool]', e); error.value = e?.message || String(e) } finally { loading.value = false } }

function handleSymbolSubmit(e: Event) { const target = e.target as HTMLInputElement; if (target?.value) { symbol.value = target.value.trim(); loadFilings() } }
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => { if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadFilings() } })
onMounted(loadFilings)
</script>

<template>
  <div class="darkpool-panel">
    <PanelHeader title="机构交易" :subtitle="`${symbol} ${name}`">
      <template #controls>
        <input class="symbol-input" :value="symbol" placeholder="AAPL" @keyup.enter="handleSymbolSubmit" />
        <button class="btn btn-sm" @click="loadFilings" :disabled="loading">{{ loading ? '...' : '⟳' }}</button>
      </template>
    </PanelHeader>

    <div class="panel-subtitle">SEC 文件中的机构/内部人交易活动</div>

    <div class="filter-bar">
      <div class="filter-buttons"><button v-for="ft in formTypes" :key="ft" :class="['btn btn-sm', { 'btn-primary': selectedFormType === ft }]" @click="setFormType(ft)">{{ ft === 'All' ? $t('common.all') : ft }}</button></div>
    </div>

    <LoadingState v-if="loading" type="table" :rows="5" :cols="4" />
    <ErrorState v-else-if="error" :description="error" @retry="loadFilings" />
    <EmptyState v-else-if="filteredFilings.length === 0" title="暂无机构交易数据" />

    <div v-else class="table-wrapper">
      <table class="filings-table">
        <thead><tr><th>{{ $t('common.date') }}</th><th>{{ $t('sec_form') }}</th><th>{{ $t('filer') }}</th><th>描述</th></tr></thead>
        <tbody><tr v-for="(f, i) in filteredFilings" :key="f.url || i"><td class="date-cell">{{ f.date }}</td><td><span :class="['form-badge', formBadgeClass(f.form)]" @click="openUrl(f.url)">{{ f.form }}</span></td><td class="filer-cell">{{ f.filer }}</td><td class="desc-cell">{{ f.description }}</td></tr></tbody>
      </table>
    </div>

    <div class="panel-footer">{{ $t('data_from_finnhub') }}</div>
  </div>
</template>

<style scoped>
.darkpool-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.symbol-input { width: 100px; padding: var(--space-xs) var(--space-sm); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); font-size: var(--font-sm); }
.panel-subtitle { font-size: var(--font-xs); color: var(--color-text-tertiary); padding: 0 var(--panel-padding); margin-bottom: var(--space-sm); flex-shrink: 0; }
.filter-bar { padding: 0 var(--panel-padding); margin-bottom: var(--space-sm); flex-shrink: 0; }
.filter-buttons { display: flex; gap: var(--space-xs); }
.table-wrapper { flex: 1; overflow: auto; margin: 0 var(--panel-padding); }
.filings-table { width: 100%; border-collapse: collapse; font-size: var(--font-xs); }
.filings-table th { text-align: left; padding: var(--space-sm); color: var(--color-text-secondary); border-bottom: 1px solid var(--color-border-strong); font-weight: 500; white-space: nowrap; }
.filings-table td { padding: var(--space-sm); border-bottom: 1px solid var(--color-border-subtle); color: var(--color-text-primary); }
.date-cell { white-space: nowrap; color: var(--color-text-secondary); }
.form-badge { display: inline-block; padding: var(--space-xs) var(--space-md); border-radius: var(--radius-lg); font-size: var(--font-xs); font-weight: 600; cursor: pointer; }
.badge-blue { background: var(--color-accent-soft); color: var(--color-accent); }
.badge-green { background: var(--color-down-soft); color: var(--color-down); }
.badge-yellow { background: var(--color-accent-soft); color: var(--color-accent); }
.badge-orange { background: var(--color-accent-soft); color: var(--color-accent); }
.badge-gray { background: var(--color-bg-elevated); color: var(--color-text-secondary); }
.filer-cell { white-space: nowrap; font-weight: 500; }
.desc-cell { max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-text-secondary); }
.panel-footer { flex-shrink: 0; padding: var(--space-sm) var(--panel-padding); font-size: var(--font-xs); color: var(--color-text-tertiary); border-top: 1px solid var(--color-border-subtle); margin-top: auto; }
</style>
