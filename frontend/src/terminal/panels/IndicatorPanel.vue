<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useStockName } from '@/lib/composables/useStockName'
import { PanelHeader, PanelTable, EmptyState, type Column } from '@/terminal/components/panel'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const props = defineProps<{
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
const { name } = useStockName(symbol)
const loading = ref(false)
const results = ref<ResultRow[]>([])
const paramValues = ref<Record<string, string>>({})
const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loaded')

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

async function runComputation() {
  if (!symbol.value || !selectedIndicator.value) return
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app) return
    const result = await app.ComputeIndicator(symbol.value, selectedIndicator.value.id, paramValues.value)
    if (result?.data) {
      results.value = convertToRows(result.data)
    }
  } catch (e) {
    console.error('[Indicator] compute failed:', e)
  } finally {
    loading.value = false
  }
}

function convertToRows(data: any[]): ResultRow[] {
  if (!Array.isArray(data)) return []
  return data.map((item: any) => {
    const row: ResultRow = { date: item.date || item.Date || '' }
    for (const key of Object.keys(item)) {
      if (key !== 'date' && key !== 'Date') {
        row[key.toLowerCase()] = item[key]
      }
    }
    return row
  })
}

function goBack() {
  selectedIndicator.value = null
  results.value = []
}

const tableColumns = computed(() => {
  if (!selectedIndicator.value) return []
  const cols: Column[] = [
    { key: 'date', label: '日期', align: 'left', width: 100 },
  ]
  for (const o of selectedIndicator.value.outputs) {
    cols.push({
      key: o.toLowerCase(),
      label: o,
      align: 'left' as const,
      formatter: (v: number) => v?.toFixed?.(4) ?? v ?? '-',
    })
  }
  return cols
})

const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const { t } = useI18n()
const headerControls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  if (selectedIndicator.value) {
    list.push({ icon: 'chevron-left', label: t('common.back'), action: goBack })
  }
  return list
})
</script>

<template>
  <PanelShell :state="state">
    <template #loaded>
      <div class="indicator-panel">
        <PanelHeader
          :title="selectedIndicator ? selectedIndicator.name : 'Indicator Panel'"
          :subtitle="selectedIndicator ? selectedIndicator.category : '技术指标'"
          :controls="headerControls"
        />

        <div class="panel-body">
          <!-- Symbol input (when indicator selected) -->
          <div v-if="selectedIndicator" class="symbol-bar">
            <span v-if="symbol" class="symbol-tag">{{ symbol }} {{ name }}</span>
            <div class="symbol-input">
              <input
                v-model="symbol"
                type="text"
                placeholder="股票代码"
                @keyup.enter="runComputation"
                class="symbol-field"
              />
              <button
                @click="runComputation"
                :disabled="loading || !symbol"
                class="btn btn-primary"
              >
                {{ loading ? '计算中...' : '计算' }}
              </button>
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
              <PanelTable
                :columns="tableColumns"
                :data="results"
                :striped="true"
                :loading="loading"
              />
            </div>
            <EmptyState
              v-else-if="!loading"
              icon="chart"
              title="暂无计算结果"
              description="输入股票代码并点击计算查看结果"
            />
          </div>
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.indicator-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--panel-padding);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.symbol-bar {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  flex-wrap: wrap;
}

.symbol-tag {
  font-size: var(--font-sm);
  color: var(--color-accent);
  font-weight: 600;
}

.symbol-input {
  display: flex;
  gap: var(--space-xs);
  margin-left: auto;
}

.symbol-field {
  width: 120px;
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-input);
  color: var(--color-text-primary);
  font-size: var(--font-sm);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.symbol-field:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.indicator-grid {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.category-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.cat-title {
  margin: 0;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border);
  font-weight: 600;
}

.indicator-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-xs);
}

.indicator-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: var(--space-sm) var(--space-md);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-sm);
  text-align: left;
  min-width: 100px;
  transition: all var(--transition-fast);
}

.indicator-chip:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.chip-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.chip-outputs {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.indicator-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  overflow: hidden;
}

.detail-header {
  display: flex;
  gap: var(--space-md);
  align-items: baseline;
  padding-bottom: var(--space-sm);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
}

.ind-name {
  font-size: var(--font-lg);
  font-weight: 700;
  color: var(--color-text-primary);
}

.ind-category {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

.ind-outputs {
  font-size: var(--font-xs);
  color: var(--color-accent);
  font-weight: 600;
}

.params-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  padding: var(--space-md);
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.param-row {
  display: flex;
  gap: var(--space-sm);
  align-items: center;
  font-size: var(--font-sm);
}

.param-row label {
  width: 50px;
  font-weight: 600;
  color: var(--color-text-primary);
  flex-shrink: 0;
}

.param-field {
  width: 80px;
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-input);
  color: var(--color-text-primary);
  font-size: var(--font-sm);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.param-field:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.param-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

.results-table {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.results-table :deep(.td) {
  font-family: 'JetBrains Mono', monospace;
  font-size: var(--font-xs);
}
</style>
