<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'
import { useStockName } from '@/lib/composables/useStockName'
import { usePanelCache } from '@/lib/composables/usePanelCache'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '000001')
const { name } = useStockName(symbol)
const { fetchWithCache } = usePanelCache()
const loading = ref(false)
const error = ref('')
const data = ref<any>(null)

const SOURCE = 'akshare'
const DATA_TYPE = 'fundflow'

interface FlowItem {
  label: string
  netAmount: number
  netRatio: number
}

const latestDay = computed(() => {
  if (!data.value || !Array.isArray(data.value) || data.value.length === 0) return null
  return data.value[0]
})

const flowCards = computed<FlowItem[]>(() => {
  const day = latestDay.value
  if (!day) return []
  return [
    { label: '主力净流入', netAmount: day['主力净流入-净额'] ?? 0, netRatio: day['主力净流入-净占比'] ?? 0 },
    { label: '超大单净流入', netAmount: day['超大单净流入-净额'] ?? 0, netRatio: day['超大单净流入-净占比'] ?? 0 },
    { label: '大单净流入', netAmount: day['大单净流入-净额'] ?? 0, netRatio: day['大单净流入-净占比'] ?? 0 },
    { label: '中单净流入', netAmount: day['中单净流入-净额'] ?? 0, netRatio: day['中单净流入-净占比'] ?? 0 },
    { label: '小单净流入', netAmount: day['小单净流入-净额'] ?? 0, netRatio: day['小单净流入-净占比'] ?? 0 },
  ]
})

async function loadData() {
  loading.value = true; error.value = ''
  try {
    const w = (window as any)
    if (w?.go?.main?.App?.FetchData) {
      const { data: result } = await fetchWithCache('fundflow:' + symbol.value, async () => {
        return await w.go.main.App.FetchData(SOURCE, DATA_TYPE, [symbol.value], '', '', {})
      })
      if (result?.data) {
        const parsed = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        if (parsed?.success === false) {
          error.value = parsed.error || '数据获取失败'
        } else {
          const raw = Array.isArray(parsed) ? parsed : (parsed?.data || parsed?.data?.records || [])
          // Sort descending (newest first) for latestDay and table display
          data.value = [...raw].sort((a: any, b: any) => {
            if (a['日期'] < b['日期']) return 1
            if (a['日期'] > b['日期']) return -1
            return 0
          })
        }
      }
      else if (result?.error) error.value = result.error
    }
  } catch (e: any) { error.value = e.message || '加载失败' }
  finally { loading.value = false }
}

function formatAmount(v: number): string {
  if (Math.abs(v) >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (Math.abs(v) >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

function formatRatio(v: number): string {
  return (v >= 0 ? '+' : '') + v.toFixed(2) + '%'
}

function isPositive(v: number): boolean {
  return v > 0
}

watch(symbol, loadData)
watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (newSym && newSym !== symbol.value) { symbol.value = newSym; loadData() }
})
onMounted(loadData)
</script>

<template>
  <div class="fundflow-panel">
    <div class="panel-header">
      <div class="header-left">
        <span class="symbol">{{ symbol }}</span>
        <span class="stock-name">{{ name }}</span>
        <span class="badge">资金流向</span>
      </div>
      <div class="header-right">
        <span v-if="latestDay" class="latest-date">{{ latestDay['日期'] }}</span>
        <button class="btn-sm" @click="loadData">🔄 刷新</button>
      </div>
    </div>

    <div class="panel-body">
      <div v-if="loading" class="empty-state">加载中...</div>
      <div v-else-if="error" class="empty-state error">{{ error }}</div>
      <div v-else-if="!data || !Array.isArray(data) || data.length === 0" class="empty-state">选择标的查看数据</div>

      <template v-else>
        <!-- Summary row: latest close & change -->
        <div class="summary-row" v-if="latestDay">
          <span class="summary-close">收盘 {{ latestDay['收盘价']?.toFixed(2) }}</span>
          <span :class="['summary-change', isPositive(latestDay['涨跌幅']) ? 'up' : 'down']">
            {{ isPositive(latestDay['涨跌幅']) ? '+' : '' }}{{ latestDay['涨跌幅']?.toFixed(2) }}%
          </span>
        </div>

        <!-- Flow cards grid -->
        <div class="card-grid">
          <div v-for="card in flowCards" :key="card.label" class="flow-card">
            <div class="card-label">{{ card.label }}</div>
            <div :class="['card-amount', isPositive(card.netAmount) ? 'up' : 'down']">
              {{ isPositive(card.netAmount) ? '+' : '' }}{{ formatAmount(card.netAmount) }}
            </div>
            <div :class="['card-ratio', isPositive(card.netRatio) ? 'up' : 'down']">
              <span class="arrow">{{ isPositive(card.netRatio) ? '↑' : '↓' }}</span>
              {{ formatRatio(card.netRatio) }}
            </div>
          </div>
        </div>

        <!-- Daily history table -->
        <div class="history-section">
          <div class="section-title">历史明细</div>
          <div class="table-wrap">
            <table class="flow-table">
              <thead>
                <tr>
                  <th>日期</th>
                  <th>收盘</th>
                  <th>涨跌幅</th>
                  <th>主力净流入</th>
                  <th>占比</th>
                  <th>超大单</th>
                  <th>大单</th>
                  <th>中单</th>
                  <th>小单</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in data" :key="row['日期']">
                  <td class="cell-date">{{ row['日期'] }}</td>
                  <td>{{ row['收盘价']?.toFixed(2) }}</td>
                  <td :class="isPositive(row['涨跌幅']) ? 'up' : 'down'">
                    {{ isPositive(row['涨跌幅']) ? '+' : '' }}{{ row['涨跌幅']?.toFixed(2) }}%
                  </td>
                  <td :class="isPositive(row['主力净流入-净额']) ? 'up' : 'down'">
                    {{ formatAmount(row['主力净流入-净额']) }}
                  </td>
                  <td :class="isPositive(row['主力净流入-净占比']) ? 'up' : 'down'">
                    {{ formatRatio(row['主力净流入-净占比']) }}
                  </td>
                  <td :class="isPositive(row['超大单净流入-净额']) ? 'up' : 'down'">
                    {{ formatAmount(row['超大单净流入-净额']) }}
                  </td>
                  <td :class="isPositive(row['大单净流入-净额']) ? 'up' : 'down'">
                    {{ formatAmount(row['大单净流入-净额']) }}
                  </td>
                  <td :class="isPositive(row['中单净流入-净额']) ? 'up' : 'down'">
                    {{ formatAmount(row['中单净流入-净额']) }}
                  </td>
                  <td :class="isPositive(row['小单净流入-净额']) ? 'up' : 'down'">
                    {{ formatAmount(row['小单净流入-净额']) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.fundflow-panel {
  display: flex; flex-direction: column; height: 100%;
  background: var(--color-bg-panel); color: var(--color-text-primary);
  font-size: 13px;
}
.panel-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 12px; border-bottom: 1px solid var(--color-border);
}
.header-left { display: flex; align-items: center; gap: 8px; }
.header-right { display: flex; align-items: center; gap: 8px; }
.symbol { font-weight: 600; font-size: 14px; }
.stock-name { color: var(--color-text-secondary); font-size: 12px; }
.badge {
  font-size: 10px; background: var(--color-accent); color: var(--color-text-primary);
  padding: 2px 8px; border-radius: 10px;
}
.latest-date { font-size: 11px; color: var(--color-text-tertiary); }
.btn-sm {
  padding: 2px 8px; font-size: 11px;
  border: 1px solid var(--color-border); border-radius: 4px;
  background: transparent; color: var(--color-text-secondary); cursor: pointer;
}
.btn-sm:hover { background: var(--color-bg-hover); }

.panel-body { flex: 1; overflow-y: auto; padding: 12px; }
.empty-state {
  display: flex; align-items: center; justify-content: center;
  height: 100%; color: var(--color-text-tertiary); font-size: 13px;
}
.empty-state.error { color: var(--color-error); }

.summary-row {
  display: flex; align-items: center; gap: 12px;
  padding: 8px 12px; margin-bottom: 12px;
  background: var(--color-bg-subtle); border-radius: 6px;
}
.summary-close { font-size: 15px; font-weight: 600; }
.summary-change { font-size: 14px; font-weight: 600; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }

.card-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px; margin-bottom: 16px;
}
.flow-card {
  padding: 12px; border: 1px solid var(--color-border-subtle);
  border-radius: 8px; text-align: center;
}
.card-label { font-size: 11px; color: var(--color-text-secondary); margin-bottom: 4px; }
.card-amount { font-size: 16px; font-weight: 700; font-variant-numeric: tabular-nums; margin-bottom: 2px; }
.card-ratio { font-size: 12px; font-weight: 500; font-variant-numeric: tabular-nums; }
.card-ratio .arrow { font-weight: 700; margin-right: 2px; }

.history-section { margin-top: 8px; }
.section-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.table-wrap { overflow-x: auto; }
.flow-table {
  width: 100%; border-collapse: collapse; font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.flow-table th {
  text-align: right; padding: 4px 8px;
  color: var(--color-text-tertiary); font-weight: 500;
  border-bottom: 1px solid var(--color-border-subtle);
  white-space: nowrap;
}
.flow-table th:first-child { text-align: left; }
.flow-table td {
  text-align: right; padding: 4px 8px;
  border-bottom: 1px solid var(--color-border-subtle);
}
.cell-date { text-align: left !important; color: var(--color-text-secondary); }
.flow-table tr:hover td { background: var(--color-bg-hover); }
</style>
