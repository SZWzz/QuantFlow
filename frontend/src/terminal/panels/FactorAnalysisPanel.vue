<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { PanelHeader } from '@/terminal/components/panel'
import { useChartTheme } from '@/lib/composables/useChartTheme'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { useWailsApp } from '@/lib/composables/useWailsApp'

const props = defineProps<{
  panelId: string
  params?: Record<string, any>
}>()
const { control: addToWfControl } = useAddToWorkflow(props.panelId)
const chartTheme = useChartTheme()

const searchQuery = ref('')
const selectedCategory = ref<string | null>(null)
const factorsLoading = ref(false)

// Factor catalog — mirrors Python sidecar's registered factors
const DEFAULT_FACTORS: any[] = [
  { name: 'momentum_20d', category: 'momentum', description: '20-day price momentum', params: { period: '20' } },
  { name: 'momentum_60d', category: 'momentum', description: '60-day price momentum', params: { period: '60' } },
  { name: 'momentum_120d', category: 'momentum', description: '120-day price momentum', params: { period: '120' } },
  { name: 'momentum_5d_minus_20d', category: 'momentum', description: 'Short minus medium momentum', params: {} },
  { name: 'rsi_14', category: 'momentum', description: '14-day Relative Strength Index', params: { period: '14' } },
  { name: 'sma_5', category: 'trend', description: '5-day Simple Moving Average', params: { period: '5' } },
  { name: 'sma_20', category: 'trend', description: '20-day Simple Moving Average', params: { period: '20' } },
  { name: 'sma_60', category: 'trend', description: '60-day Simple Moving Average', params: { period: '60' } },
  { name: 'sma_5_minus_sma_20', category: 'trend', description: 'Golden/death cross signal', params: {} },
  { name: 'macd_12_26_9', category: 'trend', description: 'MACD histogram (12/26/9)', params: {} },
  { name: 'atr_14', category: 'volatility', description: '14-day Average True Range', params: { period: '14' } },
  { name: 'volatility_20d', category: 'volatility', description: '20-day annualized volatility', params: { period: '20' } },
  { name: 'volatility_60d', category: 'volatility', description: '60-day annualized volatility', params: { period: '60' } },
  { name: 'bollinger_width_20', category: 'volatility', description: 'Bollinger Band width', params: { period: '20' } },
  { name: 'max_drawdown_60d', category: 'volatility', description: '60-day max drawdown from peak', params: { period: '60' } },
  { name: 'volume_ratio_5d', category: 'volume', description: '5-day volume ratio', params: { period: '5' } },
  { name: 'volume_ratio_20d', category: 'volume', description: '20-day volume ratio', params: { period: '20' } },
  { name: 'obv', category: 'volume', description: 'On-Balance Volume', params: {} },
  { name: 'vwap_deviation', category: 'volume', description: 'Deviation from VWAP', params: {} },
  { name: 'turnover_20d', category: 'volume', description: '20-day turnover proxy', params: { period: '20' } },
  { name: 'zscore_momentum_20d', category: 'cross_sectional', description: 'Z-score of momentum (cross-section)', params: {} },
  { name: 'rank_momentum_20d', category: 'cross_sectional', description: 'Percentile rank of momentum', params: {} },
  { name: 'zscore_volatility_20d', category: 'cross_sectional', description: 'Z-score of volatility (cross-section)', params: {} },
  { name: 'zscore_volume_ratio_5d', category: 'cross_sectional', description: 'Z-score of volume ratio', params: {} },
  { name: 'size_factor', category: 'cross_sectional', description: 'Log turnover size proxy', params: {} },
]

const factors = ref<any[]>([...DEFAULT_FACTORS])

async function loadFactors() {
  factorsLoading.value = true
  try {
    const app = useWailsApp()
    if (app?.ListFactors) {
      const result = await app.ListFactors()
      if (Array.isArray(result) && result.length > 0) {
        factors.value = result
        return
      }
    }
  } catch (e) { console.warn('Factor list fetch failed:', e) }
  finally { factorsLoading.value = false }
}

onMounted(loadFactors)

const categories = computed(() => {
  const cats = new Map<string, number>()
  for (const f of factors.value) {
    cats.set(f.category, (cats.get(f.category) || 0) + 1)
  }
  return Array.from(cats.entries()).map(([name, count]) => ({ name, count }))
})

const filteredFactors = computed(() => {
  let list = factors.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter((f) =>
      f.name.toLowerCase().includes(q) || f.description.toLowerCase().includes(q) || f.category.toLowerCase().includes(q)
    )
  }
  if (selectedCategory.value) {
    list = list.filter((f) => f.category === selectedCategory.value)
  }
  return list
})

function selectCategory(cat: string | null) {
  selectedCategory.value = selectedCategory.value === cat ? null : cat
}

function categoryColor(cat: string): string {
  const colors: Record<string, string> = {
    momentum: chartTheme.palette[0],
    trend: chartTheme.palette[1],
    volatility: chartTheme.palette[2],
    volume: chartTheme.palette[4],
    cross_sectional: chartTheme.palette[3],
  }
  return colors[cat] || 'var(--color-text-tertiary)'
}
</script>

<template>
  <div class="factor-panel">
    <PanelHeader
      :title="$t('ml.factor_analysis')"
      :controls="addToWfControl ? [addToWfControl] : []"
    />
    <!-- Search -->
    <div class="search-bar">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="Search factors..."
        class="search-input"
      />
    </div>

    <!-- Category Filter -->
    <div class="category-filters">
      <button
        v-for="cat in categories"
        :key="cat.name"
        :class="['cat-chip', { active: selectedCategory === cat.name }]"
        :style="{ borderColor: categoryColor(cat.name) }"
        @click="selectCategory(cat.name)"
      >
        <span class="cat-dot" :style="{ background: categoryColor(cat.name) }"></span>
        {{ cat.name }}
        <span class="cat-count">{{ cat.count }}</span>
      </button>
    </div>

    <!-- Factor List -->
    <div class="factor-list">
      <div
        v-for="factor in filteredFactors"
        :key="factor.name"
        class="factor-item"
      >
        <div class="factor-header">
          <span
            class="factor-category-dot"
            :style="{ background: categoryColor(factor.category) }"
          ></span>
          <span class="factor-name">{{ factor.name }}</span>
          <span class="factor-category">{{ factor.category }}</span>
        </div>
        <div class="factor-desc">{{ factor.description }}</div>
        <div v-if="Object.keys(factor.params).length > 0" class="factor-params">
          <span v-for="(v, k) in factor.params" :key="k" class="param-tag">
            {{ k }}={{ v }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="filteredFactors.length === 0" class="empty-state">
      No factors match "{{ searchQuery }}"
    </div>

    <!-- Summary -->
    <div class="summary-bar">
      {{ factors.length }} factors &middot; {{ categories.length }} categories
    </div>
  </div>
</template>

<style scoped>
.factor-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.search-bar { margin-bottom: var(--space-sm); padding: 0 var(--panel-padding); }
.search-input { width: 100%; padding: var(--space-sm) var(--space-md); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-md); color: var(--color-text-primary); font-size: var(--font-xs); outline: none; }
.search-input:focus { border-color: var(--color-accent); }
.search-input::placeholder { color: var(--color-text-tertiary); }
.category-filters { display: flex; flex-wrap: wrap; gap: var(--space-xs); margin-bottom: var(--space-sm); padding: 0 var(--panel-padding); }
.cat-chip { display: flex; align-items: center; gap: var(--space-xs); padding: var(--space-xs) var(--space-sm); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-lg); font-size: var(--font-xs); color: var(--color-text-tertiary); cursor: pointer; transition: all 0.15s; }
.cat-chip:hover { border-color: var(--color-accent); color: var(--color-text-primary); }
.cat-chip.active { background: var(--color-bg-subtle); color: var(--color-text-primary); }
.cat-dot { width: 6px; height: 6px; border-radius: 50%; }
.cat-count { color: var(--color-text-tertiary); font-size: var(--font-xs); }
.factor-list { flex: 1; overflow-y: auto; }
.factor-item { padding: var(--space-sm) var(--space-md); border-bottom: 1px solid var(--color-bg-subtle); transition: background 0.15s; }
.factor-item:hover { background: var(--color-bg-subtle); }
.factor-header { display: flex; align-items: center; gap: var(--space-sm); margin-bottom: var(--space-xs); }
.factor-category-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.factor-name { font-weight: 600; font-family: var(--font-mono); font-size: var(--font-xs); }
.factor-category { margin-left: auto; font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; }
.factor-desc { color: var(--color-text-tertiary); font-size: var(--font-xs); margin-left: var(--space-lg); margin-bottom: var(--space-xs); }
.factor-params { margin-left: var(--space-lg); display: flex; gap: var(--space-xs); flex-wrap: wrap; }
.param-tag { padding: 0 var(--space-xs); background: var(--color-bg-subtle); border-radius: var(--radius-sm); font-size: var(--font-xs); color: var(--color-accent); font-family: var(--font-mono); }
.summary-bar { padding: var(--space-sm) 0; border-top: 1px solid var(--color-border); font-size: var(--font-xs); color: var(--color-text-tertiary); text-align: center; }
</style>
