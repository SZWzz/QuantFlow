<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { PanelHeader, LoadingState, EmptyState, ErrorState } from '@/terminal/components/panel'
import { logger } from '@/lib/logger'

const { t } = useI18n()
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)

// ── Market selector ──
type Market = 'CN' | 'US'
const market = ref<Market>('CN')

// ══════ CN (A-share) options ══════
const cnLoading = ref(false)
const cnError = ref('')
const cnOptions = ref<any[]>([])

const SOURCE = 'akshare'
const DATA_TYPE = 'options'

async function loadCNData() {
  cnLoading.value = true; cnError.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (app?.FetchData) {
      const { data: result } = await (usePanelCache()).fetchWithCache<any>('options_data', () => app.FetchData(SOURCE, DATA_TYPE, [], '', '', {}), 5 * 60 * 1000)
      if (result?.data) {
        const parsed = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        if (parsed?.success === false) {
          cnError.value = parsed.error || '数据获取失败'
        } else {
          cnOptions.value = Array.isArray(parsed) ? parsed : (parsed?.data || [])
        }
      } else if (result?.error) cnError.value = result.error
    }
  } catch (e: any) { cnError.value = e.message || '加载失败' }
  finally { cnLoading.value = false }
}

function fmt(v: any): string {
  if (v == null || v === '') return '-'
  const n = typeof v === 'string' ? parseFloat(v) : v
  if (isNaN(n)) return String(v)
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  return n.toFixed(n % 1 === 0 ? 0 : 2)
}

// ══════ US options chain ══════
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

const usSymbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || 'AAPL')
const { name: usName } = useStockName(usSymbol)
const usRows = ref<OptionRow[]>([])
const usLoading = ref(false)
const usError = ref<string | null>(null)
const selectedExpiry = ref('')

const { fetchWithCache } = usePanelCache()

const expiries = computed(() => [...new Set(usRows.value.map(r => r.expiry))].sort())

const filtered = computed(() => {
  if (!selectedExpiry.value) return usRows.value
  return usRows.value.filter(r => r.expiry === selectedExpiry.value)
})

const calls = computed(() => filtered.value.filter(r => r.type === 'call').sort((a, b) => a.strike - b.strike))
const puts = computed(() => filtered.value.filter(r => r.type === 'put').sort((a, b) => a.strike - b.strike))

async function fetchUSData() {
  const app = (window as any).go?.main?.App
  if (!app?.GetUSOptionChain) return
  usLoading.value = true
  usError.value = null
  try {
    const { data } = await fetchWithCache<any>('us_options:' + usSymbol.value, () => app.GetUSOptionChain(usSymbol.value))
    usRows.value = (data || []).map((r: any) => ({
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
    usError.value = String(e)
    usRows.value = []
  } finally {
    usLoading.value = false
  }
}

function onUSSymbolClick() {
  ctx.setGroupSymbol(pg.groupId, usSymbol.value)
}

const loading = computed(() => market.value === 'CN' ? cnLoading.value : usLoading.value)

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== usSymbol.value) { usSymbol.value = newSym; fetchUSData() }
})
</script>

<template>
  <div class="options-panel">
    <div class="panel-header">
      <span class="title">期权</span>
      <!-- Market selector -->
      <div class="market-selector">
        <button :class="['market-tab', { active: market === 'CN' }]" @click="market = 'CN'">A股</button>
        <button :class="['market-tab', { active: market === 'US' }]" @click="market = 'US'; if (!usRows.length) fetchUSData()">美股</button>
      </div>
      <!-- US symbol input -->
      <template v-if="market === 'US'">
        <span class="symbol-badge">{{ usSymbol }} {{ usName }}</span>
        <input v-model="usSymbol" class="sym-input" :placeholder="t('common.symbol')" @change="fetchUSData" @click="onUSSymbolClick" />
      </template>
      <button class="btn-sm" @click="market === 'CN' ? loadCNData() : fetchUSData()" :disabled="loading">⟳ 刷新</button>
    </div>

    <!-- ── CN content ── -->
    <template v-if="market === 'CN'">
      <div class="panel-body">
        <div v-if="cnLoading" class="state">加载中...</div>
        <div v-else-if="cnError" class="state error">{{ cnError }}</div>
        <div v-else-if="cnOptions.length === 0" class="state">暂无数据</div>

        <template v-else>
          <div class="table-wrap">
            <table class="opt-table">
              <thead>
                <tr>
                  <th>合约代码</th>
                  <th>合约简称</th>
                  <th>标的</th>
                  <th>类型</th>
                  <th>行权价</th>
                  <th>到期日</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in cnOptions" :key="row['合约编码'] || row['合约交易代码']">
                  <td class="td-code">{{ row['合约交易代码'] || '-' }}</td>
                  <td>{{ row['合约简称'] || '-' }}</td>
                  <td>{{ row['标的券名称及代码'] || '-' }}</td>
                  <td :class="row['类型'] === '认购' ? 'up' : row['类型'] === '认沽' ? 'down' : ''">{{ row['类型'] || '-' }}</td>
                  <td>{{ row['行权价'] != null ? fmt(row['行权价']) : '-' }}</td>
                  <td>{{ row['到期日'] || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>
      </div>
    </template>

    <!-- ── US content ── -->
    <template v-if="market === 'US'">
      <div v-if="expiries.length > 0" class="expiry-bar">
        <label class="expiry-label">{{ t('misc.expiry') }}:</label>
        <select v-model="selectedExpiry" class="expiry-select">
          <option value="">{{ t('common.all') }}</option>
          <option v-for="e in expiries" :key="e" :value="e">{{ e }}</option>
        </select>
      </div>

      <LoadingState v-if="usLoading && usRows.length === 0" type="table" />

      <div v-else-if="usError" class="error-state">
        <span class="error-text">{{ usError }}</span>
        <button class="retry-btn" @click="fetchUSData">{{ t('common.retry') }}</button>
      </div>

      <template v-else-if="calls.length > 0 || puts.length > 0">
        <div class="chain-grid">
          <div class="chain-side">
            <div class="side-header calls-header">CALL</div>
            <div class="table-wrapper">
              <div class="table-header-row">
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
              <div class="table-header-row">
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
    </template>
  </div>
</template>

<style scoped>
.options-panel { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-panel); color: var(--color-text-primary); font-size: var(--font-sm); overflow: hidden; }

.title { font-weight: 600; font-size: var(--font-sm); }
.btn-sm { padding: 2px 8px; font-size: var(--font-xs); border: 1px solid var(--color-border); border-radius: var(--radius-sm); background: transparent; color: var(--color-text-secondary); cursor: pointer; }
.btn-sm:hover { background: var(--color-bg-hover); }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }

/* Market selector */
.market-selector { display: flex; gap: 0; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); overflow: hidden; }
.market-tab { padding: 2px 10px; border: none; background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: var(--font-xs); font-weight: 500; }
.market-tab + .market-tab { border-left: 1px solid var(--color-border-strong); }
.market-tab.active { color: var(--color-accent); background: var(--color-accent-soft); }

/* CN */
.panel-body { flex: 1; overflow: auto; padding: 12px; }
.state { display: flex; align-items: center; justify-content: center; height: 100%; color: var(--color-text-tertiary); font-size: var(--font-sm); }
.state.error { color: var(--color-danger); }
.table-wrap { overflow-x: auto; }
.opt-table { width: 100%; border-collapse: collapse; font-size: var(--font-sm); font-variant-numeric: tabular-nums; }
.opt-table th { text-align: right; padding: 4px 8px; color: var(--color-text-tertiary); font-weight: 500; border-bottom: 1px solid var(--color-border-subtle); white-space: nowrap; }
.opt-table th:first-child { text-align: left; }
.opt-table td { text-align: right; padding: 4px 8px; border-bottom: 1px solid var(--color-border-subtle); }
.opt-table tr:hover td { background: var(--color-bg-hover); }
.td-code { text-align: left !important; color: var(--color-text-secondary); font-family: monospace; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

/* US */
.symbol-badge { font-size: var(--font-xs); padding: 2px 8px; border-radius: var(--radius-sm); background: var(--color-accent-soft); color: var(--color-accent); font-family: monospace; }
.sym-input { padding: 2px 6px; font-size: var(--font-xs); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); width: 70px; }

.expiry-bar { display: flex; align-items: center; gap: 6px; padding: 6px 12px; flex-shrink: 0; border-bottom: 1px solid var(--color-border-subtle); }
.expiry-label { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.expiry-select { padding: 2px 4px; font-size: var(--font-xs); border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); }

.chain-grid { flex: 1; display: grid; grid-template-columns: 1fr 1fr; gap: 8px; overflow: hidden; padding: 0 8px; }
.chain-side { display: flex; flex-direction: column; overflow: hidden; }
.side-header { font-size: var(--font-xs); font-weight: 700; text-transform: uppercase; padding: 4px 0; margin-bottom: 2px; border-bottom: 1px solid var(--color-border-strong); flex-shrink: 0; }
.calls-header { color: var(--color-down); }
.puts-header { color: var(--color-up); }

.table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header-row { display: flex; padding: 3px 0; border-bottom: 1px solid var(--color-border-strong); font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; flex-shrink: 0; }
.table-body { flex: 1; overflow-y: auto; font-size: var(--font-xs); }
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
.error-text { color: var(--color-up); font-size: var(--font-xs); max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.retry-btn { padding: 4px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm); background: var(--color-bg-elevated); color: var(--color-text-primary); cursor: pointer; font-size: var(--font-xs); }
</style>
