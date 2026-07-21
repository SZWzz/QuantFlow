<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { getIcon } from '@/lib/icons'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { PanelHeader, LoadingState, EmptyState } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '000001')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const { control: addToWfControl, addToWorkflow } = useAddToWorkflow(props.panelId)
const loading = ref(false); const error = ref(''); const data = ref<any>(null)

const SOURCE = 'akshare'
const latestDay = computed(() => { if (!Array.isArray(data.value) || data.value.length === 0) return null; return data.value[data.value.length - 1] })
const flowCards = computed(() => { if (!latestDay.value) return []; const d = latestDay.value; return [{ label: '主力净流入', netAmount: d['主力净流入-净额'] ?? 0, netRatio: d['主力净流入-净占比'] ?? 0 }, { label: '超大单净流入', netAmount: d['超大单净流入-净额'] ?? 0, netRatio: d['超大单净流入-净占比'] ?? 0 }, { label: '大单净流入', netAmount: d['大单净流入-净额'] ?? 0, netRatio: d['大单净流入-净占比'] ?? 0 }, { label: '中单净流入', netAmount: d['中单净流入-净额'] ?? 0, netRatio: d['中单净流入-净占比'] ?? 0 }, { label: '小单净流入', netAmount: d['小单净流入-净额'] ?? 0, netRatio: d['小单净流入-净占比'] ?? 0 }] })

function isPositive(v: any): boolean { const n = Number(v); return !isNaN(n) && n >= 0 }
function formatAmount(v: number): string { if (v == null || isNaN(v)) return '--'; const abs = Math.abs(v); return (v < 0 ? '-' : '') + (abs >= 1e8 ? (abs / 1e8).toFixed(2) + '亿' : abs >= 1e4 ? (abs / 1e4).toFixed(1) + '万' : abs.toFixed(0)); }
function formatRatio(v: number): string { if (v == null || isNaN(v)) return '--'; return (v >= 0 ? '+' : '') + (v * 100).toFixed(2) + '%' }

async function loadData() { loading.value = true; error.value = ''; try { const app = (window as any).go?.main?.App; if (!app?.GetFundFlow) return; const { data: result } = await fetchWithCache<any>('fundflow:' + symbol.value, () => app.GetFundFlow(symbol.value)); data.value = result && Array.isArray(result) ? result : Array.from(result || []) } catch (e: any) { error.value = e?.message || String(e) } finally { loading.value = false } }

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId]?.activeSymbol, (newSym) => { if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() } })
onMounted(loadData)
</script>

<template>
  <div class="fundflow-panel">
    <PanelHeader :title="`${symbol} ${name}`" subtitle="资金流向">
      <template #controls>
        <button v-if="addToWfControl" class="btn btn-sm" @click="addToWorkflow()" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
        <span v-if="latestDay" class="latest-date">{{ latestDay['日期'] }}</span>
        <button class="btn btn-sm" @click="loadData">🔄 刷新</button>
      </template>
    </PanelHeader>

    <div class="panel-body">
      <LoadingState v-if="loading" type="card" :rows="4" />
      <EmptyState v-else-if="error" title="加载失败" :description="error" />
      <EmptyState v-else-if="!data || !Array.isArray(data) || data.length === 0" title="选择标的查看数据" />

      <template v-else>
        <div class="summary-row" v-if="latestDay">
          <span class="summary-close">收盘 {{ latestDay['收盘价']?.toFixed(2) }}</span>
          <span :class="['summary-change', isPositive(latestDay['涨跌幅']) ? 'up' : 'down']">{{ isPositive(latestDay['涨跌幅']) ? '+' : '' }}{{ latestDay['涨跌幅']?.toFixed(2) }}%</span>
        </div>

        <div class="card-grid">
          <div v-for="card in flowCards" :key="card.label" class="flow-card">
            <div class="card-label">{{ card.label }}</div>
            <div :class="['card-amount', isPositive(card.netAmount) ? 'up' : 'down']">{{ isPositive(card.netAmount) ? '+' : '' }}{{ formatAmount(card.netAmount) }}</div>
            <div :class="['card-ratio', isPositive(card.netRatio) ? 'up' : 'down']"><span class="arrow">{{ isPositive(card.netRatio) ? '↑' : '↓' }}</span>{{ formatRatio(card.netRatio) }}</div>
          </div>
        </div>

        <div class="history-section">
          <h4 class="section-title">历史明细</h4>
          <div class="table-wrap">
            <table class="flow-table">
              <thead><tr><th>日期</th><th>收盘</th><th>涨跌幅</th><th>主力净流入</th><th>占比</th><th>超大单</th><th>大单</th><th>中单</th><th>小单</th></tr></thead>
              <tbody><tr v-for="row in data" :key="row['日期']"><td class="cell-date">{{ row['日期'] }}</td><td>{{ row['收盘价']?.toFixed(2) }}</td><td :class="isPositive(row['涨跌幅']) ? 'up' : 'down'">{{ isPositive(row['涨跌幅']) ? '+' : '' }}{{ row['涨跌幅']?.toFixed(2) }}%</td><td :class="isPositive(row['主力净流入-净额']) ? 'up' : 'down'">{{ formatAmount(row['主力净流入-净额']) }}</td><td :class="isPositive(row['主力净流入-净占比']) ? 'up' : 'down'">{{ formatRatio(row['主力净流入-净占比']) }}</td><td :class="isPositive(row['超大单净流入-净额']) ? 'up' : 'down'">{{ formatAmount(row['超大单净流入-净额']) }}</td><td :class="isPositive(row['大单净流入-净额']) ? 'up' : 'down'">{{ formatAmount(row['大单净流入-净额']) }}</td><td :class="isPositive(row['中单净流入-净额']) ? 'up' : 'down'">{{ formatAmount(row['中单净流入-净额']) }}</td><td :class="isPositive(row['小单净流入-净额']) ? 'up' : 'down'">{{ formatAmount(row['小单净流入-净额']) }}</td></tr></tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.fundflow-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.latest-date { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.panel-body { flex: 1; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }
.summary-row { display: flex; align-items: center; gap: var(--space-md); padding: var(--space-sm) var(--space-md); margin-bottom: var(--space-md); background: var(--color-bg-subtle); border-radius: var(--radius-md); }
.summary-close { font-size: var(--font-lg); font-weight: 600; }
.summary-change { font-size: var(--font-sm); font-weight: 600; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }
.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: var(--space-sm); margin-bottom: var(--space-lg); }
.flow-card { padding: var(--space-md); border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); text-align: center; }
.card-label { font-size: var(--font-xs); color: var(--color-text-secondary); margin-bottom: var(--space-xs); }
.card-amount { font-size: var(--font-lg); font-weight: 700; font-variant-numeric: tabular-nums; margin-bottom: var(--space-xs); }
.card-ratio { font-size: var(--font-xs); font-weight: 500; font-variant-numeric: tabular-nums; }
.card-ratio .arrow { font-weight: 700; margin-right: var(--space-xs); }
.history-section { margin-top: var(--space-sm); }
.history-section .section-title { display: block; margin-bottom: var(--space-sm); }
.table-wrap { overflow-x: auto; }
.flow-table { width: 100%; border-collapse: collapse; font-size: var(--font-xs); font-variant-numeric: tabular-nums; }
.flow-table th { text-align: right; padding: var(--space-xs) var(--space-sm); color: var(--color-text-tertiary); font-weight: 500; border-bottom: 1px solid var(--color-border-subtle); white-space: nowrap; }
.flow-table th:first-child { text-align: left; }
.flow-table td { text-align: right; padding: var(--space-xs) var(--space-sm); border-bottom: 1px solid var(--color-border-subtle); }
.cell-date { text-align: left !important; color: var(--color-text-secondary); }
.flow-table tr:hover td { background: var(--color-bg-hover); }
</style>
