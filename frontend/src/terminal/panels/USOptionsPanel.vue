<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { logger } from '@/lib/logger'

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

interface OptionRow {
  expiry: string
  strike: number
  type: 'call' | 'put'
  bid: number
  ask: number
  last: number
  volume: number
  open_interest: number
  implied_vol: number
  delta: number
  gamma: number
  theta: number
  vega: number
}

const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name } = useStockName(symbol)
const rows = ref<OptionRow[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const selectedExpiry = ref('')

const { fetchWithCache } = usePanelCache()

const expiries = computed(() => [...new Set(rows.value.map(r => r.expiry))].sort())

const filtered = computed(() => {
  if (!selectedExpiry.value) return rows.value
  return rows.value.filter(r => r.expiry === selectedExpiry.value)
})

const calls = computed(() => filtered.value.filter(r => r.type === 'call').sort((a, b) => a.strike - b.strike))
const puts = computed(() => filtered.value.filter(r => r.type === 'put').sort((a, b) => a.strike - b.strike))

async function fetchData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetUSOptionChain) return
  loading.value = true
  error.value = null
  try {
    const { data } = await fetchWithCache<any>('us_options:' + symbol.value, () => app.GetUSOptionChain(symbol.value))
    rows.value = (data || []).map((r: any) => ({
      expiry: r.expiry || '',
      strike: r.strike || 0,
      type: r.type === 'put' ? 'put' : 'call',
      bid: r.bid || 0,
      ask: r.ask || 0,
      last: r.last || 0,
      volume: r.volume || 0,
      open_interest: r.open_interest || 0,
      implied_vol: r.implied_vol || 0,
      delta: r.delta || 0,
      gamma: r.gamma || 0,
      theta: r.theta || 0,
      vega: r.vega || 0,
    }))
    if (expiries.value.length > 0) {
      selectedExpiry.value = expiries.value[0]
    }
  } catch (e) {
    logger.error('[USOptions]', e)
    error.value = String(e)
    rows.value = []
  } finally {
    loading.value = false
  }
}

function onSymbolClick() {
  ctx.setGroupSymbol(pg.groupId, symbol.value)
}

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; fetchData() }
})
onMounted(fetchData)
</script>

<template>
  <div class="us-options-panel">
    <div class="panel-header">
      <h3>{{ t('misc.us_option_chain') }}</h3>
      <span class="symbol-badge">{{ symbol }} {{ name }}</span>
      <input v-model="symbol" class="sym-input" :placeholder="t('common.symbol')" @change="fetchData" @click="onSymbolClick" />
      <button class="refresh-btn" @click="fetchData" :disabled="loading">⟳</button>
    </div>

    <div v-if="expiries.length > 0" class="expiry-bar">
      <label class="expiry-label">{{ t('misc.expiry') }}:</label>
      <select v-model="selectedExpiry" class="expiry-select">
        <option value="">{{ t('common.all') }}</option>
        <option v-for="e in expiries" :key="e" :value="e">{{ e }}</option>
      </select>
    </div>

    <SkeletonPanel v-if="loading && rows.length === 0" type="table" />

    <div v-else-if="error" class="error-state">
      <span class="error-text">{{ error }}</span>
      <button class="retry-btn" @click="fetchData">{{ t('common.retry') }}</button>
    </div>

    <template v-else-if="calls.length > 0 || puts.length > 0">
      <div class="chain-grid">
        <div class="chain-side">
          <div class="side-header calls-header">CALL</div>
          <div class="table-wrapper">
            <div class="table-header">
              <span class="col-strike">{{ t('misc.strike') }}</span>
              <span class="col-last">{{ t('quote.last') }}</span>
              <span class="col-bidask">{{ t('quote.bid') }}/{{ t('quote.ask') }}</span>
              <span class="col-vol">{{ t('common.volume') }}</span>
              <span class="col-oi">{{ t('misc.open_interest') }}</span>
              <span class="col-iv">{{ t('misc.implied_vol') }}</span>
              <span class="col-grk">Δ/Γ</span>
            </div>
            <div class="table-body">
              <div v-for="r in calls" :key="r.strike + r.expiry" class="table-row">
                <span class="col-strike">{{ r.strike.toFixed(2) }}</span>
                <span class="col-last">{{ r.last ? r.last.toFixed(2) : '--' }}</span>
                <span class="col-bidask">{{ r.bid.toFixed(2) }}/{{ r.ask.toFixed(2) }}</span>
                <span class="col-vol">{{ r.volume || '--' }}</span>
                <span class="col-oi">{{ r.open_interest || '--' }}</span>
                <span class="col-iv">{{ (r.implied_vol * 100).toFixed(1) }}%</span>
                <span class="col-grk">{{ r.delta.toFixed(2) }}/{{ r.gamma.toFixed(3) }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="chain-side">
          <div class="side-header puts-header">PUT</div>
          <div class="table-wrapper">
            <div class="table-header">
              <span class="col-strike">{{ t('misc.strike') }}</span>
              <span class="col-last">{{ t('quote.last') }}</span>
              <span class="col-bidask">{{ t('quote.bid') }}/{{ t('quote.ask') }}</span>
              <span class="col-vol">{{ t('common.volume') }}</span>
              <span class="col-oi">{{ t('misc.open_interest') }}</span>
              <span class="col-iv">{{ t('misc.implied_vol') }}</span>
              <span class="col-grk">Δ/Γ</span>
            </div>
            <div class="table-body">
              <div v-for="r in puts" :key="r.strike + r.expiry" class="table-row">
                <span class="col-strike">{{ r.strike.toFixed(2) }}</span>
                <span class="col-last">{{ r.last ? r.last.toFixed(2) : '--' }}</span>
                <span class="col-bidask">{{ r.bid.toFixed(2) }}/{{ r.ask.toFixed(2) }}</span>
                <span class="col-vol">{{ r.volume || '--' }}</span>
                <span class="col-oi">{{ r.open_interest || '--' }}</span>
                <span class="col-iv">{{ (r.implied_vol * 100).toFixed(1) }}%</span>
                <span class="col-grk">{{ r.delta.toFixed(2) }}/{{ r.gamma.toFixed(3) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="empty-state">{{ t('common.no_data') }}</div>
  </div>
</template>

<style scoped>
.us-options-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-shrink: 0; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.sym-input { padding: 2px 6px; font-size: 11px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); width: 70px; }
.refresh-btn { margin-left: auto; padding: 4px 10px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 13px; }
.refresh-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }

.expiry-bar { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; flex-shrink: 0; }
.expiry-label { font-size: 11px; color: var(--color-text-tertiary); }
.expiry-select { padding: 2px 4px; font-size: 11px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); }

.chain-grid { flex: 1; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; overflow: hidden; }
.chain-side { display: flex; flex-direction: column; overflow: hidden; }
.side-header { font-size: 10px; font-weight: 700; text-transform: uppercase; padding: 4px 0; margin-bottom: 2px; border-bottom: 1px solid var(--color-border-strong); flex-shrink: 0; }
.calls-header { color: var(--color-down); }
.puts-header { color: var(--color-up); }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header { display: flex; padding: 3px 0; border-bottom: 1px solid var(--color-border-strong); font-size: 9px; color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: 11px; }
.table-row { display: flex; padding: 2px 0; align-items: center; border-bottom: 1px solid var(--color-border-subtle); }
.table-row:hover { background: var(--color-bg-elevated); }

.col-strike { width: 55px; text-align: right; font-variant-numeric: tabular-nums; padding-right: 4px; }
.col-last { width: 50px; text-align: right; font-variant-numeric: tabular-nums; }
.col-bidask { width: 72px; text-align: right; font-variant-numeric: tabular-nums; }
.col-vol { width: 48px; text-align: right; font-variant-numeric: tabular-nums; }
.col-oi { width: 48px; text-align: right; font-variant-numeric: tabular-nums; }
.col-iv { width: 48px; text-align: right; font-variant-numeric: tabular-nums; }
.col-grk { width: 58px; text-align: right; font-variant-numeric: tabular-nums; }

.error-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; }
.error-text { color: var(--color-up); font-size: 11px; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.retry-btn { padding: 4px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: 11px; }
</style>
