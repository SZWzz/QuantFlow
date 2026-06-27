<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import type { PositionDetail } from '@/stores/portfolio'

import VChart from 'vue-echarts'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

// --- Store ---
const store = usePortfolioStore()

// --- State ---
const positions = ref<PositionDetail[]>([])
const currentAllocation = ref<{ by_market: Record<string, number>; by_sector: Record<string, number> }>({
  by_market: {},
  by_sector: {},
})
const targetAllocation = ref<Array<{ id: number; name: string; targetWeight: number }>>([])
let nextId = 1

// --- Computed ---
const totalPortfolioValue = computed(() =>
  positions.value.reduce((s, p) => s + p.market_price * p.quantity, 0),
)

const totalTargetWeight = computed(() =>
  targetAllocation.value.reduce((s, t) => s + t.targetWeight, 0),
)

const marketColors: Record<string, string> = {
  CN: '#ef4444',
  US: '#3b82f6',
  HK: '#22c55e',
  CRYPTO: '#f59e0b',
}

const marketOrder = ['CN', 'US', 'HK', 'CRYPTO']

const sortedMarketAllocation = computed(() => {
  const byMarket = currentAllocation.value.by_market
  return marketOrder
    .filter((m) => byMarket[m] !== undefined && byMarket[m] > 0)
    .map((m) => ({ name: m, pct: byMarket[m], color: marketColors[m] || 'var(--color-text-tertiary)' }))
})

const sortedSectorAllocation = computed(() => {
  const bySector = currentAllocation.value.by_sector
  return Object.entries(bySector)
    .filter(([, v]) => v > 0)
    .sort(([, a], [, b]) => b - a)
    .map(([name, pct], i) => ({
      name,
      pct,
      color: ['#ef4444', '#3b82f6', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899', '#06b6d4'][i % 7],
    }))
})

const tradeList = computed(() => {
  const total = totalPortfolioValue.value
  if (!total || targetAllocation.value.length === 0) return []

  // Build current weight map by market
  const currentByMarket: Record<string, number> = {}
  for (const pos of positions.value) {
    const mkt = pos.market
    const mv = pos.market_price * pos.quantity
    currentByMarket[mkt] = (currentByMarket[mkt] || 0) + mv / total * 100
  }

  // Build position lookup by market (use largest position as representative)
  const repByMarket: Record<string, PositionDetail> = {}
  for (const pos of positions.value) {
    const existing = repByMarket[pos.market]
    if (!existing || pos.market_price * pos.quantity > existing.market_price * existing.quantity) {
      repByMarket[pos.market] = pos
    }
  }

  const trades: Array<{
    symbol: string
    name: string
    market: string
    currentWeight: number
    targetWeight: number
    deltaValue: number
    action: 'Buy' | 'Sell'
    shares: number
  }> = []

  for (const t of targetAllocation.value) {
    const currentWeight = currentByMarket[t.name] || 0
    const deltaPct = t.targetWeight - currentWeight
    const deltaValue = (deltaPct / 100) * total
    const action = deltaValue >= 0 ? 'Buy' : 'Sell'
    const abs删除ta = Math.abs(deltaValue)

    const rep = repByMarket[t.name]
    const price = rep ? rep.market_price : 100
    const rawShares = price > 0 ? abs删除ta / price : 0
    // CN stocks trade in lots of 100
    const lotSize = t.name === 'CN' ? 100 : 1
    const shares = Math.floor(rawShares / lotSize) * lotSize

    trades.push({
      symbol: rep ? rep.symbol : t.name,
      name: rep ? (rep.name || '') : '',
      market: t.name,
      currentWeight: Math.round(currentWeight * 10) / 10,
      targetWeight: Math.round(t.targetWeight * 10) / 10,
      deltaValue: Math.round(deltaValue * 100) / 100,
      action,
      shares: shares > 0 ? shares : 0,
    })
  }

  return trades.filter((t) => Math.abs(t.deltaValue) > 0.01)
})

// ECharts donut option (rendered only if VChart is available)
const donutOption = computed(() => {
  const data = sortedMarketAllocation.value.map((a) => ({
    name: a.name,
    value: a.pct,
    itemStyle: { color: a.color },
  }))
  return {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item' as const,
      backgroundColor: 'var(--color-bg-elevated)',
      borderColor: 'var(--color-border-strong)',
      textStyle: { color: '#e5e7eb', fontSize: 11 },
      formatter: '{b}: {c}%',
    },
    series: [{
      type: 'pie',
      radius: ['45%', '72%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 2, borderColor: 'var(--color-bg-panel)', borderWidth: 2 },
      label: { show: true, position: 'outside' as const, color: 'var(--color-text-secondary)', fontSize: 9, formatter: '{b}\n{d}%' },
      labelLine: { lineStyle: { color: 'var(--color-border-strong)' } },
      data,
    }],
  }
})

// --- Methods ---
function addTargetRow() {
  targetAllocation.value.push({ id: nextId++, name: '', targetWeight: 0 })
}

function removeTargetRow(id: number) {
  targetAllocation.value = targetAllocation.value.filter((t) => t.id !== id)
}

function generateOrders() {
  const buys = tradeList.value.filter((t) => t.action === 'Buy')
  const sells = tradeList.value.filter((t) => t.action === 'Sell')
  window.alert(`Generated ${buys.length} buy orders, ${sells.length} sell orders`)
}

function fmtMoney(n: number): string {
  const abs = Math.abs(n)
  const sign = n < 0 ? '-' : ''
  if (abs >= 1e6) return sign + '$' + (abs / 1e6).toFixed(2) + 'M'
  if (abs >= 1e3) return sign + '$' + (abs / 1e3).toFixed(1) + 'K'
  return sign + '$' + abs.toFixed(2)
}

// --- Init ---
onMounted(async () => {
  await store.fetchAllocation()
  await store.fetchPositions()
  positions.value = (store.positions as any).value || store.positions

  const alloc = (store.allocation as any)?.value || store.allocation
  if (alloc) {
    currentAllocation.value = {
      by_market: alloc.by_market || {},
      by_sector: alloc.by_sector || {},
    }
  }

  // Init target from current
  const markets = currentAllocation.value.by_market
  targetAllocation.value = Object.entries(markets)
    .filter(([, v]) => v > 0)
    .map(([name, weight]) => ({
      id: nextId++,
      name,
      targetWeight: Math.round(weight * 10) / 10,
    }))
})
</script>

<template>
  <div class="rebalance-panel">
    <h2 class="panel-title">组合再平衡</h2>

    <!-- Section A: 当前配置 -->
    <div class="section">
      <div class="section-header">
        <span class="section-label">当前配置</span>
      </div>

      <div class="alloc-grid">
        <div class="alloc-bars">
          <div class="sub-label">按市场</div>
          <div v-for="item in sortedMarketAllocation" :key="item.name" class="alloc-row">
            <span class="alloc-name" :style="{ color: item.color }">{{ item.name }}</span>
            <div class="alloc-bar-track">
              <div
                class="alloc-bar-fill"
                :style="{ width: item.pct + '%', background: item.color }"
              />
            </div>
            <span class="alloc-pct">{{ item.pct.toFixed(1) }}%</span>
          </div>

          <div v-if="sortedSectorAllocation.length" class="sub-label" style="margin-top: 12px;">按行业</div>
          <div v-for="item in sortedSectorAllocation" :key="item.name" class="alloc-row">
            <span class="alloc-name" :style="{ color: item.color }">{{ item.name }}</span>
            <div class="alloc-bar-track">
              <div
                class="alloc-bar-fill"
                :style="{ width: item.pct + '%', background: item.color }"
              />
            </div>
            <span class="alloc-pct">{{ item.pct.toFixed(1) }}%</span>
          </div>
        </div>

        <!-- Donut chart -->
        <div class="alloc-donut">
          <VChart :option="donutOption" autoresize style="height: 180px;" />
        </div>
      </div>
    </div>

    <!-- Section B: 目标配置 -->
    <div class="section">
      <div class="section-header">
        <span class="section-label">目标配置</span>
        <span class="weight-summary" :class="{ 'weight-warn': Math.abs(totalTargetWeight - 100) > 0.5 }">
          {{ $t('common.total') }}: {{ totalTargetWeight.toFixed(1) }}%
          <span v-if="Math.abs(totalTargetWeight - 100) > 0.5" class="warn-text"> (应为 100%)</span>
        </span>
      </div>

      <table class="target-table">
        <thead>
          <tr>
            <th>{{ $t('misc.asset_market') }}</th>
            <th class="num">Target Weight (%)</th>
            <th class="action-col">{{ $t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in targetAllocation" :key="row.id">
            <td>
              <input
                v-model="row.name"
                type="text"
                class="target-input text-input"
                placeholder="e.g. CN"
              />
            </td>
            <td class="num">
              <input
                v-model.number="row.targetWeight"
                type="number"
                min="0"
                max="100"
                step="0.1"
                class="target-input num-input"
              />
            </td>
            <td class="action-col">
              <button class="btn-delete" @click="removeTargetRow(row.id)">{{ $t('common.delete') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
      <button class="btn-add" @click="addTargetRow">{{ $t('trade.add_row') }}</button>
    </div>

    <!-- Section C: 交易清单 -->
    <div class="section">
      <div class="section-header">
        <span class="section-label">交易清单</span>
        <span class="trade-summary">
          {{ $t('portfolio.total_value') }}: {{ fmtMoney(totalPortfolioValue) }}
        </span>
      </div>

      <div class="table-wrap">
        <table class="trade-table">
          <thead>
            <tr>
              <th>{{ $t('portfolio.symbol') }}</th>
              <th>{{ $t('portfolio.market') }}</th>
              <th class="num">Curr. Wt</th>
              <th class="num">Targ. Wt</th>
              <th class="num">{{ $t('common.change') }}</th>
              <th class="action-col">{{ $t('common.actions') }}</th>
              <th class="num">{{ $t('portfolio.quantity') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(trade, i) in tradeList" :key="i">
              <td class="trade-symbol">{{ trade.symbol }} - {{ trade.name }}</td>
              <td>
                <span class="market-badge" :style="{ color: marketColors[trade.market] }">
                  {{ trade.market }}
                </span>
              </td>
              <td class="num">{{ trade.currentWeight }}%</td>
              <td class="num">{{ trade.targetWeight }}%</td>
              <td class="num" :class="trade.action === 'Buy' ? 'up' : 'down'">
                {{ trade.deltaValue >= 0 ? '+' : '' }}{{ fmtMoney(trade.deltaValue) }}
              </td>
              <td class="action-col">
                <span :class="['action-tag', trade.action === 'Buy' ? 'buy-tag' : 'sell-tag']">
                  {{ trade.action }}
                </span>
              </td>
              <td class="num">{{ trade.shares }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="tradeList.length === 0" class="empty-state">
        无需调整，当前配置已匹配目标
      </div>

      <button class="btn-generate" @click="generateOrders" :disabled="tradeList.length === 0">
        {{ $t('trade.place_order') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.rebalance-panel {
  height: 100%;
  overflow-y: auto;
  padding: 12px;
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.panel-title {
  font-size: 15px;
  font-weight: 700;
  color: #f9fafb;
  margin: 0 0 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--color-border-strong);
}

.section {
  background: #1a1f2e;
  border: 1px solid var(--color-border-strong);
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 10px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.section-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-secondary);
  font-weight: 600;
}

/* --- 当前配置 Bars --- */
.alloc-grid {
  display: flex;
  gap: 12px;
}

.alloc-bars {
  flex: 1;
  min-width: 0;
}

.alloc-donut {
  flex: 0 0 200px;
}

.sub-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  margin-bottom: 6px;
}

.alloc-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 5px;
}

.alloc-name {
  width: 55px;
  font-size: 11px;
  font-weight: 600;
  text-align: right;
  flex-shrink: 0;
}

.alloc-bar-track {
  flex: 1;
  height: 12px;
  background: var(--color-bg-elevated);
  border-radius: 3px;
  overflow: hidden;
  border: 1px solid var(--color-border-strong);
}

.alloc-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
  min-width: 2px;
}

.alloc-pct {
  width: 44px;
  font-size: 11px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
  text-align: left;
}

/* --- Section B: Target Table --- */
.weight-summary {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.weight-summary.weight-warn {
  color: #f59e0b;
}

.warn-text {
  color: #ef4444;
}

.target-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
  margin-bottom: 6px;
}

.target-table th {
  text-align: left;
  padding: 5px 6px;
  font-size: 10px;
  color: var(--color-text-tertiary);
  font-weight: 500;
  text-transform: uppercase;
  border-bottom: 1px solid var(--color-border-strong);
}

.target-table th.num,
.target-table th.action-col {
  text-align: center;
}

.target-table td {
  padding: 4px 6px;
  border-bottom: 1px solid var(--color-bg-elevated);
}

.target-table td.num {
  text-align: center;
}

.target-table td.action-col {
  text-align: center;
}

.target-input {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-primary);
  border-radius: 4px;
  padding: 4px 8px;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  width: 100%;
  outline: none;
  transition: border-color 0.15s;
}

.target-input:focus {
  border-color: #3b82f6;
}

.text-input {
  min-width: 80px;
}

.num-input {
  width: 80px;
  text-align: center;
}

.btn-delete {
  background: transparent;
  border: 1px solid var(--color-border-strong);
  color: #ef4444;
  padding: 3px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 10px;
  transition: background 0.15s;
}

.btn-delete:hover {
  background: rgba(239, 68, 68, 0.15);
}

.btn-add {
  background: transparent;
  border: 1px dashed var(--color-border-strong);
  color: var(--color-text-secondary);
  padding: 5px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  width: 100%;
  transition: border-color 0.15s, color 0.15s;
}

.btn-add:hover {
  border-color: #3b82f6;
  color: #3b82f6;
}

/* --- Section C: 交易清单 --- */
.trade-summary {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.table-wrap {
  overflow-x: auto;
  margin-bottom: 8px;
}

.trade-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}

.trade-table th {
  text-align: left;
  padding: 5px 8px;
  font-size: 10px;
  color: var(--color-text-tertiary);
  font-weight: 500;
  text-transform: uppercase;
  border-bottom: 1px solid var(--color-border-strong);
  white-space: nowrap;
}

.trade-table th.num {
  text-align: right;
}

.trade-table th.action-col {
  text-align: center;
}

.trade-table td {
  padding: 5px 8px;
  border-bottom: 1px solid var(--color-bg-elevated);
  white-space: nowrap;
}

.trade-table td.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.trade-table td.action-col {
  text-align: center;
}

.trade-symbol {
  font-weight: 600;
  color: #f9fafb;
}

.market-badge {
  font-size: 10px;
  font-weight: 600;
}

.action-tag {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
}

.buy-tag {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.12);
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.sell-tag {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.up { color: #22c55e; }
.down { color: #ef4444; }

.empty-state {
  text-align: center;
  padding: 16px;
  color: var(--color-text-tertiary);
  font-size: 12px;
  font-style: italic;
}

.btn-generate {
  background: #3b82f6;
  border: none;
  color: #fff;
  padding: 8px 20px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  width: 100%;
  transition: background 0.15s;
}

.btn-generate:hover:not(:disabled) {
  background: #2563eb;
}

.btn-generate:disabled {
  background: var(--color-border-strong);
  color: var(--color-text-tertiary);
  cursor: not-allowed;
}
</style>
