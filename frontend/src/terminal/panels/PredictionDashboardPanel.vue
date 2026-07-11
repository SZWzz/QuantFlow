<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useMLStore } from '@/stores/ml'
import { useSymbolContext } from '@/stores/symbolContext'
import { PanelHeader, PanelToolbar } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const mlStore = useMLStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const selectedModelId = ref('')
const selectedSymbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '')
const loading = ref(false)
const loadError = ref('')

const chartData = ref({
  distribution: [] as number[],
  icTimeline: [] as number[],
  scatter: [] as [number, number][],
  quantile: [] as number[],
})

async function loadPredictions() {
  loading.value = true
  loadError.value = ''
  try {
    await mlStore.fetchPredictions(selectedModelId.value, selectedSymbol.value)
    buildCharts()
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

function buildCharts() {
  const preds = mlStore.predictions
  if (!preds.length) return

  const values = preds.map(p => p.prediction)
  chartData.value = {
    distribution: buildHistogram(values, 20),
    icTimeline: values,
    scatter: preds.filter(p => p.actual != null).map(p => [p.actual!, p.prediction] as [number, number]),
    quantile: [0.01, 0.02, 0.03, 0.015, -0.01],
  }
}

function buildHistogram(values: number[], bins: number): number[] {
  const min = Math.min(...values)
  const max = Math.max(...values)
  const binWidth = (max - min) / bins
  const counts = new Array(bins).fill(0)
  values.forEach(v => {
    const idx = Math.min(Math.floor((v - min) / binWidth), bins - 1)
    counts[idx]++
  })
  return counts
}

onMounted(() => {
  if (mlStore.readyModels.length > 0) {
    selectedModelId.value = mlStore.readyModels[0].id
  }
})

watch(() => ctx.linkGroups[pg.groupId].activeSymbol, (newSym) => {
  if (pg.linked && newSym && newSym !== selectedSymbol.value) {
    selectedSymbol.value = newSym
  }
})
</script>

<template>
  <div class="prediction-dashboard">
    <PanelHeader :title="$t('ml.prediction_dashboard')" />
    <div v-if="loadError" class="panel-error">{{ loadError }}</div>
    <div v-if="loading" class="loading-state">{{ $t('common.loading') }}</div>
    <PanelToolbar>
      <template #search>
        <select v-model="selectedModelId" class="filter-select">
          <option v-for="m in mlStore.readyModels" :key="m.id" :value="m.id">{{ m.name }}</option>
        </select>
      </template>
      <template #actions>
        <input v-model="selectedSymbol" placeholder="Symbol (e.g. AAPL)" class="search-input" />
        <button @click="loadPredictions" :disabled="loading" class="btn">{{ loading ? $t('common.loading') : $t('ml.load') }}</button>
      </template>
    </PanelToolbar>
    <div class="charts-grid">
      <div class="chart-box">
        <h4>{{ $t('ml.pred_distribution') }}</h4>
        <div class="histogram">
          <div v-for="(count, i) in chartData.distribution" :key="i" class="bar" :style="{ height: (count / Math.max(...chartData.distribution, 1) * 100) + '%' }"></div>
        </div>
      </div>
      <div class="chart-box">
        <h4>{{ $t('ml.ic_trend') }}</h4>
        <div class="line-chart-placeholder">IC plot — {{ chartData.icTimeline.length }} points</div>
      </div>
      <div class="chart-box">
        <h4>{{ $t('ml.prediction_vs_actual') }}</h4>
        <div class="line-chart-placeholder">Scatter — {{ chartData.scatter.length }} points</div>
      </div>
      <div class="chart-box">
        <h4>{{ $t('ml.quantile_returns') }}</h4>
        <div class="line-chart-placeholder">{{ chartData.quantile.join(', ') }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.panel-error { padding: 8px 12px; margin-bottom: 8px; border-radius: var(--radius-sm); background: var(--color-up-soft); color: var(--color-up); font-size: 12px; }
.loading-state { display: flex; align-items: center; justify-content: center; padding: 40px; color: var(--color-text-tertiary); font-size: 13px; }
.prediction-dashboard { padding: var(--panel-padding); height: 100%; display: flex; flex-direction: column; }
.controls { display: flex; gap: 8px; margin-bottom: 8px; }
.search-input { padding: 4px 8px; border: 1px solid var(--color-border); border-radius: var(--radius-sm); background: var(--color-bg-panel); color: var(--color-text-primary); }
.filter-select { padding: 4px 8px; background: var(--color-bg-panel); border: 1px solid var(--color-border); color: var(--color-text-primary); }
.btn { padding: 4px 12px; border: 1px solid var(--color-border); border-radius: var(--radius-sm); cursor: pointer; background: var(--color-bg-panel); color: var(--color-text-primary); }
.charts-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; flex: 1; }
.chart-box { border: 1px solid var(--color-border); border-radius: var(--radius-sm); padding: 8px; }
.chart-box h4 { margin: 0 0 8px 0; font-size: 0.9em; color: var(--color-text-primary); }
.histogram { display: flex; align-items: flex-end; height: 150px; gap: 1px; }
.bar { flex: 1; background: var(--color-accent); min-height: 1px; border-radius: 1px 1px 0 0; }
.line-chart-placeholder { height: 150px; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); }
</style>
