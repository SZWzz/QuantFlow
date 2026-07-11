<script setup lang="ts">
import { ref, computed } from 'vue'
import { PanelHeader, PanelTable, EmptyState } from '@/terminal/components/panel'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { logger } from '@/lib/logger'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const { control: addToWfControl } = useAddToWorkflow(props.panelId)

interface StrategyDef {
  id: string
  name: string
  category: string
  description: string
  params: { name: string; type: string; default: string; description: string }[]
}

interface ScanResult {
  symbol: string
  name: string
  score: number
  signal: string
  last_price: number
  change_pct: number
  volume: number
  matched_conditions: string[]
}

const strategies: StrategyDef[] = [
  { id: 'golden_cross', name: '金叉选股', category: '均线系统', description: '短期均线上穿长期均线',
    params: [{ name: 'fast', type: 'int', default: '5', description: '快线' },
             { name: 'slow', type: 'int', default: '20', description: '慢线' }] },
  { id: 'macd_golden', name: 'MACD金叉', category: 'MACD', description: 'MACD快线上穿慢线',
    params: [{ name: 'fast', type: 'int', default: '12', description: '快线' },
             { name: 'slow', type: 'int', default: '26', description: '慢线' },
             { name: 'signal', type: 'int', default: '9', description: '信号线' }] },
  { id: 'volume_break', name: '放量突破', category: '量价', description: '成交量突破N日均量 + 价格上涨',
    params: [{ name: 'n', type: 'int', default: '20', description: '均量周期' },
             { name: 'ratio', type: 'float', default: '1.5', description: '量比阈值' }] },
  { id: 'breakout_high', name: '突破前高', category: '形态', description: '价格突破N日最高价',
    params: [{ name: 'n', type: 'int', default: '60', description: '周期' }] },
  { id: 'oversold_bounce', name: '超跌反弹', category: '超买超卖', description: 'RSI/KDJ低位反弹信号',
    params: [{ name: 'rsi_period', type: 'int', default: '14', description: 'RSI周期' },
             { name: 'rsi_threshold', type: 'int', default: '30', description: 'RSI阈值' }] },
  { id: 'bullish_engulf', name: '看涨吞没', category: 'K线形态', description: '阳线实体吞没前日阴线',
    params: [] },
  { id: 'ma_support', name: '均线支撑', category: '均线系统', description: '回踩均线不破反弹',
    params: [{ name: 'ma', type: 'int', default: '60', description: '均线周期' },
             { name: 'tolerance', type: 'float', default: '0.02', description: '容差' }] },
  { id: 'multi_factor', name: '多因子综合', category: '综合', description: '多因子加权打分排名',
    params: [{ name: 'top_n', type: 'int', default: '20', description: '返回前N' }] },
]

const marketList = ref([
  { code: 'all', name: '全市场' },
  { code: 'sh', name: '上证A股' },
  { code: 'sz', name: '深证A股' },
  { code: 'cyb', name: '创业板' },
  { code: 'kcb', name: '科创板' },
])

const selectedMarket = ref('all')
const selectedStrategy = ref<StrategyDef | null>(null)
const paramValues = ref<Record<string, string>>({})
const scanning = ref(false)
const loadError = ref('')
const results = ref<ScanResult[]>([])
const sortField = ref<'score' | 'change_pct' | 'volume'>('score')
const sortDir = ref<'asc' | 'desc'>('desc')

const sortedResults = computed(() => {
  const list = [...results.value]
  list.sort((a, b) => {
    const va = a[sortField.value]
    const vb = b[sortField.value]
    return (vb - va) * (sortDir.value === 'desc' ? 1 : -1)
  })
  return list
})

const resultSummary = computed(() => {
  const avgScore = results.value.length
    ? results.value.reduce((s: number, r: ScanResult) => s + r.score, 0) / results.value.length
    : 0
  const positive = results.value.filter((r: ScanResult) => r.change_pct > 0).length
  return { total: results.value.length, avgScore, positive }
})

function selectStrategy(strategy: StrategyDef) {
  selectedStrategy.value = strategy
  paramValues.value = {}
  for (const p of strategy.params) {
    paramValues.value[p.name] = p.default
  }
  results.value = []
}

async function startScan() {
  scanning.value = true
  loadError.value = ''
  results.value = []

  try {
    const app = (window as any).go?.main?.App
    if (!app) return
    const result = await app.ScanStocks(selectedStrategy.value?.id || 'momentum')
    if (result?.results) {
      results.value = result.results.map((r: any) => ({
        symbol: r.symbol,
        name: r.name,
        score: r.score || r.sharpe || 0,
        signal: r.signal || 'hold',
        last_price: r.price || r.last_price || 0,
        change_pct: r.change_pct || 0,
        volume: r.volume || 0,
        matched_conditions: r.matched_conditions || [],
      }))
    }
  } catch (e: any) {
    logger.error('[Scanner] scan failed:', e)
    loadError.value = e?.message || String(e)
    results.value = []
  } finally {
    scanning.value = false
  }
}

function toggleSort(field: 'score' | 'change_pct' | 'volume') {
  if (sortField.value === field) {
    sortDir.value = sortDir.value === 'desc' ? 'asc' : 'desc'
  } else {
    sortField.value = field
    sortDir.value = 'desc'
  }
}

function goBack() {
  selectedStrategy.value = null
  results.value = []
}

function formatScore(v: number): string {
  return (v * 100).toFixed(1) + '%'
}

function formatPct(v: number): string {
  return (v > 0 ? '+' : '') + v.toFixed(2) + '%'
}

function formatVol(v: number): string {
  if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'
  if (v >= 1e4) return (v / 1e4).toFixed(1) + '万'
  return v.toFixed(0)
}

const tableColumns = [
  { key: 'symbol', label: '代码', width: 80 },
  { key: 'name', label: '名称', flex: 1 },
  { key: 'score', label: '评分', width: 80, align: 'right' as const },
  { key: 'change_pct', label: '涨跌幅', width: 90, align: 'right' as const, colorize: true },
  { key: 'volume', label: '成交量', width: 100, align: 'right' as const, format: 'volume' as const },
  { key: 'signal', label: '信号', width: 60 },
  { key: 'matched_conditions', label: '匹配条件', flex: 2 },
]

function tableData() {
  return sortedResults.value.map(r => ({
    symbol: r.symbol,
    name: r.name,
    score: formatScore(r.score),
    change_pct: r.change_pct,
    volume: r.volume,
    signal: r.signal === 'buy' ? '买入' : r.signal === 'sell' ? '卖出' : '观望',
    matched_conditions: r.matched_conditions.join(', '),
  }))
}
</script>

<template>
  <div class="scanner-panel">
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <PanelHeader
      :title="selectedStrategy ? selectedStrategy.name : 'Stock Scanner'"
      :subtitle="selectedStrategy ? selectedStrategy.category : '策略选股'"
      :controls="addToWfControl ? [addToWfControl] : []"
    >
      <template #controls>
        <div v-if="selectedStrategy" class="header-controls">
          <select v-model="selectedMarket" class="market-select">
            <option v-for="m in marketList" :key="m.code" :value="m.code">{{ m.name }}</option>
          </select>
          <button @click="startScan" :disabled="scanning" class="scan-btn">
            {{ scanning ? '扫描中...' : '开始扫描' }}
          </button>
          <button @click="goBack" class="back-btn">返回</button>
        </div>
      </template>
    </PanelHeader>

    <!-- Strategy Selection Grid -->
    <div v-if="!selectedStrategy" class="strategy-grid">
      <div class="strategy-list">
        <button
          v-for="s in strategies"
          :key="s.id"
          @click="selectStrategy(s)"
          class="strategy-card"
        >
          <span class="card-category">{{ s.category }}</span>
          <span class="card-name">{{ s.name }}</span>
          <span class="card-desc">{{ s.description }}</span>
        </button>
      </div>
    </div>

    <!-- Strategy Detail & Results -->
    <div v-else class="strategy-detail">
      <div class="detail-header">
        <span class="strat-name">{{ selectedStrategy.name }}</span>
        <span class="strat-category">{{ selectedStrategy.category }}</span>
        <span class="strat-desc">{{ selectedStrategy.description }}</span>
      </div>

      <!-- Params -->
      <div v-if="selectedStrategy.params.length" class="params-bar">
        <div v-for="p in selectedStrategy.params" :key="p.name" class="param-item">
          <label>{{ p.name }}</label>
          <input v-model="paramValues[p.name]" type="text" class="param-input" />
        </div>
      </div>

      <!-- Summary -->
      <div v-if="results.length" class="result-summary">
        <span class="summary-stat">结果: {{ resultSummary.total }} 只</span>
        <span class="summary-stat">均分: {{ formatScore(resultSummary.avgScore) }}</span>
        <span class="summary-stat up">上涨: {{ resultSummary.positive }}</span>
      </div>

      <!-- Results Table -->
      <PanelTable
        v-if="results.length"
        :columns="tableColumns"
        :data="tableData()"
        :loading="scanning"
        striped
      />
      <EmptyState
        v-else-if="!scanning"
        icon="search"
        title="选择参数后点击开始扫描"
      />
      <EmptyState
        v-else
        icon="loader"
        title="正在扫描全市场..."
      />
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; border-radius: var(--radius-sm); background: var(--color-up-soft); color: var(--color-up); font-size: 12px; }
.scanner-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: var(--panel-padding);
  gap: 10px;
  overflow-y: auto;
  background: var(--color-bg-panel);
}
.header-controls { display: flex; gap: 6px; align-items: center; }
.market-select {
  padding: 4px 8px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border);
  color: var(--color-text-primary);
  font-size: 12px;
}
.scan-btn {
  padding: 4px 16px;
  background: var(--color-accent);
  color: var(--color-text-primary);
  border: none;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  border-radius: var(--radius-md);
}
.scan-btn:disabled { opacity: 0.5; cursor: default; }
.back-btn {
  padding: 4px 8px;
  background: var(--color-bg-panel);
  color: var(--color-text-tertiary);
  border: 1px solid var(--color-border);
  cursor: pointer;
  font-size: 12px;
  border-radius: var(--radius-md);
}
.strategy-grid {
  flex: 1;
  overflow-y: auto;
}
.strategy-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
}
.strategy-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  padding: 10px 12px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border);
  cursor: pointer;
  text-align: left;
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}
.strategy-card:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}
.card-category {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
}
.card-name { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.card-desc { font-size: 11px; color: var(--color-text-tertiary); }
.strategy-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}
.detail-header {
  display: flex;
  gap: 10px;
  align-items: baseline;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--color-border);
}
.strat-name { font-size: 15px; font-weight: 700; color: var(--color-text-primary); }
.strat-category { font-size: 11px; color: var(--color-text-tertiary); }
.strat-desc { font-size: 11px; color: var(--color-accent); }
.params-bar {
  display: flex;
  gap: 12px;
  padding: 6px 8px;
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  flex-wrap: wrap;
  border-radius: var(--radius-md);
}
.param-item {
  display: flex;
  gap: 4px;
  align-items: center;
  font-size: 12px;
}
.param-item label {
  font-weight: 600;
  color: var(--color-text-tertiary);
  min-width: 40px;
}
.param-input {
  width: 60px;
  padding: 2px 6px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  font-size: 12px;
  border-radius: var(--radius-sm);
}
.result-summary {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.summary-stat.up { color: var(--color-up); }
</style>
