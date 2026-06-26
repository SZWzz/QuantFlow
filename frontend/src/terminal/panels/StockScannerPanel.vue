<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

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
const results = ref<ScanResult[]>([])
const sortField = ref<'score' | 'change_pct' | 'volume'>('score')
const sortDir = ref<'asc' | 'desc'>('desc')

const sortedResults = computed(() => {
  const list = [...results.value]
  list.sort((a, b) => {
    const va = a[sortField.value]
    const vb = b[sortField.value]
    if (typeof va === 'string') return va.localeCompare(vb as string)
    return (va as number) - (vb as number)
  })
  if (sortDir.value === 'desc') list.reverse()
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

function startScan() {
  scanning.value = true
  results.value = []

  // Demo results — in production this calls the Go sidecar
  setTimeout(() => {
    results.value = []
    scanning.value = false
  }, 1500)
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
</script>

<template>
  <div class="scanner-panel">
    <div class="panel-header">
      <div class="header-left">
        <h3>Stock Scanner</h3>
        <span class="subtitle">策略选股</span>
      </div>
      <div v-if="selectedStrategy" class="header-controls">
        <select v-model="selectedMarket" class="market-select">
          <option v-for="m in marketList" :key="m.code" :value="m.code">{{ m.name }}</option>
        </select>
        <button @click="startScan" :disabled="scanning" class="scan-btn">
          {{ scanning ? '扫描中...' : '开始扫描' }}
        </button>
        <button @click="goBack" class="back-btn">返回</button>
      </div>
    </div>

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
        <span class="summary-stat positive">上涨: {{ resultSummary.positive }}</span>
      </div>

      <!-- Results Table -->
      <div class="results-table">
        <table v-if="results.length" class="data-table">
          <thead>
            <tr>
              <th>代码</th>
              <th>名称</th>
              <th @click="toggleSort('score')" class="sortable">
                评分 {{ sortField === 'score' ? (sortDir === 'desc' ? '↓' : '↑') : '' }}
              </th>
              <th @click="toggleSort('change_pct')" class="sortable">
                涨跌幅 {{ sortField === 'change_pct' ? (sortDir === 'desc' ? '↓' : '↑') : '' }}
              </th>
              <th @click="toggleSort('volume')" class="sortable">
                成交量 {{ sortField === 'volume' ? (sortDir === 'desc' ? '↓' : '↑') : '' }}
              </th>
              <th>信号</th>
              <th>匹配条件</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in sortedResults" :key="r.symbol">
              <td class="mono">{{ r.symbol }}</td>
              <td>{{ r.name }}</td>
              <td class="mono score">{{ formatScore(r.score) }}</td>
              <td :class="r.change_pct >= 0 ? 'positive' : 'negative'">
                {{ formatPct(r.change_pct) }}
              </td>
              <td class="mono">{{ formatVol(r.volume) }}</td>
              <td>
                <span :class="r.signal === 'buy' ? 'tag-buy' : r.signal === 'sell' ? 'tag-sell' : 'tag-hold'">
                  {{ r.signal === 'buy' ? '买入' : r.signal === 'sell' ? '卖出' : '观望' }}
                </span>
              </td>
              <td class="conditions-cell">
                <span v-for="c in r.matched_conditions" :key="c" class="condition-tag">{{ c }}</span>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else-if="!scanning" class="empty-hint">
          选择参数后点击"开始扫描"
        </p>
        <p v-else class="scanning-hint">正在扫描全市场...</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scanner-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px;
  gap: 10px;
  overflow-y: auto;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.header-left h3 { margin: 0; font-size: 15px; }
.subtitle { font-size: 11px; color: var(--term-fg-dim); margin-left: 6px; }
.header-controls { display: flex; gap: 6px; align-items: center; }
.market-select {
  padding: 4px 8px;
  background: var(--term-bg);
  border: 1px solid var(--term-border);
  color: var(--term-fg);
  font-size: 12px;
}
.scan-btn {
  padding: 4px 16px;
  background: var(--term-accent);
  color: #fff;
  border: none;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
}
.scan-btn:disabled { opacity: 0.5; cursor: default; }
.back-btn {
  padding: 4px 8px;
  background: var(--term-bg);
  color: var(--term-fg-dim);
  border: 1px solid var(--term-border);
  cursor: pointer;
  font-size: 12px;
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
  background: var(--term-bg);
  border: 1px solid var(--term-border);
  cursor: pointer;
  text-align: left;
}
.strategy-card:hover {
  border-color: var(--term-accent);
  background: var(--term-accent-dim);
}
.card-category {
  font-size: 10px;
  color: var(--term-fg-dim);
  text-transform: uppercase;
}
.card-name { font-size: 14px; font-weight: 600; }
.card-desc { font-size: 11px; color: var(--term-fg-dim); }
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
  border-bottom: 1px solid var(--term-border);
}
.strat-name { font-size: 15px; font-weight: 700; }
.strat-category { font-size: 11px; color: var(--term-fg-dim); }
.strat-desc { font-size: 11px; color: var(--term-accent); }
.params-bar {
  display: flex;
  gap: 12px;
  padding: 6px 8px;
  background: var(--term-bg-dim);
  border: 1px solid var(--term-border);
  flex-wrap: wrap;
}
.param-item {
  display: flex;
  gap: 4px;
  align-items: center;
  font-size: 12px;
}
.param-item label {
  font-weight: 600;
  color: var(--term-fg-dim);
  min-width: 40px;
}
.param-input {
  width: 60px;
  padding: 2px 6px;
  border: 1px solid var(--term-border);
  background: var(--term-bg);
  color: var(--term-fg);
  font-size: 12px;
}
.result-summary {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--term-fg-dim);
}
.summary-stat.positive { color: #4ade80; }
.results-table {
  flex: 1;
  overflow: auto;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.data-table th, .data-table td {
  padding: 4px 8px;
  border-bottom: 1px solid var(--term-border);
  text-align: left;
}
.data-table th { color: var(--term-fg-dim); font-weight: 600; }
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--term-accent); }
.mono { font-family: monospace; }
.score { color: var(--term-accent); font-weight: 600; }
.positive { color: #4ade80; }
.negative { color: #f87171; }
.tag-buy { color: #4ade80; font-weight: 600; }
.tag-sell { color: #f87171; font-weight: 600; }
.tag-hold { color: var(--term-fg-dim); }
.conditions-cell {
  display: flex;
  gap: 3px;
  flex-wrap: wrap;
  max-width: 160px;
}
.condition-tag {
  font-size: 10px;
  padding: 1px 4px;
  background: var(--term-bg-dim);
  border: 1px solid var(--term-border);
}
.empty-hint, .scanning-hint {
  color: var(--term-fg-dim);
  font-style: italic;
  text-align: center;
  padding: 30px 0;
}
.scanning-hint { animation: pulse 1.5s infinite; }
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
