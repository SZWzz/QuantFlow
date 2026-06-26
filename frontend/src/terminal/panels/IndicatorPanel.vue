<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

interface IndicatorDef {
  id: string
  name: string
  category: string
  outputs: string[]
  params: { name: string; type: string; default: string; description: string }[]
}

interface ResultRow {
  date: string
  [key: string]: any
}

const indicators: IndicatorDef[] = [
  { id: 'kdj', name: 'KDJ', category: '超买超卖', outputs: ['K', 'D', 'J'],
    params: [{ name: 'n', type: 'int', default: '9', description: '周期' },
             { name: 'm1', type: 'int', default: '3', description: 'K平滑' },
             { name: 'm2', type: 'int', default: '3', description: 'D平滑' }] },
  { id: 'dmi', name: 'DMI', category: '趋势', outputs: ['PDI', 'MDI', 'ADX', 'ADXR'],
    params: [{ name: 'n', type: 'int', default: '14', description: '周期' },
             { name: 'm', type: 'int', default: '6', description: 'ADXR平滑' }] },
  { id: 'atr', name: 'ATR', category: '波动', outputs: ['ATR'],
    params: [{ name: 'n', type: 'int', default: '14', description: '周期' }] },
  { id: 'wr', name: 'WR', category: '超买超卖', outputs: ['WR1', 'WR2'],
    params: [{ name: 'n1', type: 'int', default: '10', description: '周期1' },
             { name: 'n2', type: 'int', default: '6', description: '周期2' }] },
  { id: 'cci', name: 'CCI', category: '超买超卖', outputs: ['CCI'],
    params: [{ name: 'n', type: 'int', default: '14', description: '周期' }] },
  { id: 'obv', name: 'OBV', category: '量价', outputs: ['OBV'], params: [] },
  { id: 'mfi', name: 'MFI', category: '量价', outputs: ['MFI'],
    params: [{ name: 'n', type: 'int', default: '14', description: '周期' }] },
  { id: 'sar', name: 'SAR', category: '趋势', outputs: ['SAR'], params: [] },
  { id: 'vwap', name: 'VWAP', category: '量价', outputs: ['VWAP'], params: [] },
  { id: 'aroon', name: 'AROON', category: '趋势', outputs: ['AROON_UP', 'AROON_DOWN'],
    params: [{ name: 'n', type: 'int', default: '14', description: '周期' }] },
  { id: 'brar', name: 'BRAR', category: '能量', outputs: ['BR', 'AR'],
    params: [{ name: 'n', type: 'int', default: '26', description: '周期' }] },
  { id: 'mass', name: 'MASS', category: '趋势', outputs: ['MASS'],
    params: [{ name: 'n', type: 'int', default: '25', description: '周期' },
             { name: 'm', type: 'int', default: '9', description: '平滑' }] },
  { id: 'psy', name: 'PSY', category: '心理', outputs: ['PSY'],
    params: [{ name: 'n', type: 'int', default: '12', description: '周期' }] },
  { id: 'roc', name: 'ROC', category: '动量', outputs: ['ROC'],
    params: [{ name: 'n', type: 'int', default: '12', description: '周期' }] },
  { id: 'bias', name: 'BIAS', category: '趋势', outputs: ['BIAS6', 'BIAS12', 'BIAS24'], params: [] },
  { id: 'bbi', name: 'BBI', category: '趋势', outputs: ['BBI'], params: [] },
  { id: 'asi', name: 'ASI', category: '能量', outputs: ['ASI'], params: [] },
  { id: 'zhuoyao', name: 'ZHUOYAO', category: '量价', outputs: ['ZHUOYAO_20', 'ZHUOYAO_60', 'ZHUOYAO_120'], params: [] },
]

const categories = computed(() => {
  const cats = new Map<string, IndicatorDef[]>()
  for (const ind of indicators) {
    const list = cats.get(ind.category) || []
    list.push(ind)
    cats.set(ind.category, list)
  }
  return Array.from(cats.entries())
})

const selectedCategory = ref<string | null>(null)
const selectedIndicator = ref<IndicatorDef | null>(null)
const symbol = ref('')
const loading = ref(false)
const results = ref<ResultRow[]>([])
const paramValues = ref<Record<string, string>>({})

const filteredIndicators = computed(() => {
  if (!selectedCategory.value) return indicators
  return indicators.filter((i: IndicatorDef) => i.category === selectedCategory.value)
})

function selectIndicator(ind: IndicatorDef) {
  selectedIndicator.value = ind
  paramValues.value = {}
  for (const p of ind.params) {
    paramValues.value[p.name] = p.default
  }
  results.value = []
}

function runComputation() {
  loading.value = true
  // In production, this calls the Go sidecar via the indicator workflow nodes
  setTimeout(() => {
    results.value = []
    loading.value = false
  }, 800)
}

function goBack() {
  selectedIndicator.value = null
  results.value = []
}
</script>

<template>
  <div class="indicator-panel">
    <div class="panel-header">
      <div class="header-left">
        <h3>Indicator Panel</h3>
        <span class="subtitle">技术指标</span>
      </div>
      <div v-if="selectedIndicator" class="symbol-input">
        <input
          v-model="symbol"
          type="text"
          placeholder="股票代码"
          @keyup.enter="runComputation"
          class="symbol-field"
        />
        <button @click="runComputation" :disabled="loading || !symbol" class="query-btn">
          {{ loading ? '计算中...' : '计算' }}
        </button>
        <button @click="goBack" class="back-btn">返回</button>
      </div>
    </div>

    <!-- Category / Indicator Grid -->
    <div v-if="!selectedIndicator" class="indicator-grid">
      <div v-for="[cat, items] in categories" :key="cat" class="category-group">
        <h4 class="cat-title">{{ cat }}</h4>
        <div class="indicator-list">
          <button
            v-for="ind in items"
            :key="ind.id"
            @click="selectIndicator(ind)"
            class="indicator-chip"
            :title="ind.outputs.join(', ')"
          >
            <span class="chip-name">{{ ind.name }}</span>
            <span class="chip-outputs">→ {{ ind.outputs.join(', ') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Indicator Detail -->
    <div v-else class="indicator-detail">
      <div class="detail-header">
        <span class="ind-name">{{ selectedIndicator.name }}</span>
        <span class="ind-category">{{ selectedIndicator.category }}</span>
        <span class="ind-outputs">输出: {{ selectedIndicator.outputs.join(', ') }}</span>
      </div>

      <!-- Params editor -->
      <div v-if="selectedIndicator.params.length" class="params-section">
        <div v-for="p in selectedIndicator.params" :key="p.name" class="param-row">
          <label :title="p.description">{{ p.name }}</label>
          <input
            v-model="paramValues[p.name]"
            type="text"
            class="param-field"
          />
          <span class="param-desc">{{ p.description }}</span>
        </div>
      </div>

      <!-- Results Table -->
      <div v-if="results.length" class="results-table">
        <table class="data-table">
          <thead>
            <tr>
              <th>日期</th>
              <th v-for="o in selectedIndicator.outputs" :key="o">{{ o }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(r, idx) in results" :key="idx">
              <td>{{ r.date }}</td>
              <td v-for="o in selectedIndicator.outputs" :key="o" class="mono">
                {{ r[o.toLowerCase()]?.toFixed?.(4) ?? r[o.toLowerCase()] ?? '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else-if="!loading" class="empty-hint">
        输入股票代码并点击"计算"查看结果
      </p>
    </div>
  </div>
</template>

<style scoped>
.indicator-panel {
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
.header-left h3 {
  margin: 0;
  font-size: 15px;
}
.subtitle {
  font-size: 11px;
  color: var(--term-fg-dim);
  margin-left: 6px;
}
.symbol-input {
  display: flex;
  gap: 4px;
}
.symbol-field {
  width: 120px;
  padding: 4px 8px;
  border: 1px solid var(--term-border);
  background: var(--term-bg);
  color: var(--term-fg);
  font-size: 13px;
}
.query-btn {
  padding: 4px 12px;
  background: var(--term-accent);
  color: #fff;
  border: none;
  cursor: pointer;
  font-size: 13px;
}
.query-btn:disabled { opacity: 0.5; cursor: default; }
.back-btn {
  padding: 4px 8px;
  background: var(--term-bg);
  color: var(--term-fg-dim);
  border: 1px solid var(--term-border);
  cursor: pointer;
  font-size: 12px;
}
.indicator-grid {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.category-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cat-title {
  margin: 0;
  font-size: 12px;
  color: var(--term-fg-dim);
  text-transform: uppercase;
  padding-bottom: 2px;
  border-bottom: 1px solid var(--term-border);
}
.indicator-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.indicator-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: 6px 10px;
  background: var(--term-bg);
  border: 1px solid var(--term-border);
  cursor: pointer;
  font-size: 12px;
  text-align: left;
  min-width: 100px;
}
.indicator-chip:hover {
  border-color: var(--term-accent);
  background: var(--term-accent-dim);
}
.chip-name {
  font-weight: 600;
  color: var(--term-fg);
}
.chip-outputs {
  font-size: 10px;
  color: var(--term-fg-dim);
  margin-top: 2px;
}
.indicator-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail-header {
  display: flex;
  gap: 12px;
  align-items: baseline;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--term-border);
}
.ind-name { font-size: 16px; font-weight: 700; }
.ind-category { font-size: 11px; color: var(--term-fg-dim); }
.ind-outputs { font-size: 11px; color: var(--term-accent); }
.params-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  background: var(--term-bg-dim);
  border: 1px solid var(--term-border);
}
.param-row {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 12px;
}
.param-row label {
  width: 50px;
  font-weight: 600;
  color: var(--term-fg);
}
.param-field {
  width: 80px;
  padding: 3px 6px;
  border: 1px solid var(--term-border);
  background: var(--term-bg);
  color: var(--term-fg);
  font-size: 12px;
}
.param-desc {
  font-size: 11px;
  color: var(--term-fg-dim);
}
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
  text-align: right;
}
.data-table th:first-child, .data-table td:first-child { text-align: left; }
.data-table th {
  color: var(--term-fg-dim);
  font-weight: 600;
  text-align: right;
}
.mono { font-family: monospace; }
.empty-hint {
  color: var(--term-fg-dim);
  font-style: italic;
  text-align: center;
  padding: 30px 0;
}
</style>
