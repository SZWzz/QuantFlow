<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useMLStore } from '@/stores/ml'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const mlStore = useMLStore()
const ctx = useSymbolContext()
const pg = ctx.getOrCreatePanelGroup(props.panelId)
const selectedModelId = ref('')
const selectedSymbol = ref(props.params?.symbol || ctx.getGroupSymbol(pg.groupId) || '')

const chartData = ref({
  distribution: [] as number[],
  icTimeline: [] as number[],
  scatter: [] as [number, number][],
  quantile: [] as number[],
})

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
    <div class="controls">
      <select v-model="selectedModelId" class="filter-select">
        <option v-for="m in mlStore.readyModels" :key="m.id" :value="m.id">{{ m.name }}</option>
      </select>
      <input v-model="selectedSymbol" placeholder="Symbol (e.g. AAPL)" class="search-input" />
      <button @click="mlStore.fetchPredictions(selectedModelId, selectedSymbol); buildCharts()" class="btn">加载</button>
    </div>
    <div class="charts-grid">
      <div class="chart-box">
        <h4>预测分布</h4>
        <div class="histogram">
          <div v-for="(count, i) in chartData.distribution" :key="i" class="bar" :style="{ height: (count / Math.max(...chartData.distribution, 1) * 100) + '%' }"></div>
        </div>
      </div>
      <div class="chart-box">
        <h4>IC 走势</h4>
        <div class="line-chart-placeholder">IC plot — {{ chartData.icTimeline.length }} points</div>
      </div>
      <div class="chart-box">
        <h4>预测 vs 实际</h4>
        <div class="line-chart-placeholder">Scatter — {{ chartData.scatter.length }} points</div>
      </div>
      <div class="chart-box">
        <h4>分位数收益</h4>
        <div class="line-chart-placeholder">{{ chartData.quantile.join(', ') }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prediction-dashboard { padding: 8px; height: 100%; display: flex; flex-direction: column; }
.controls { display: flex; gap: 8px; margin-bottom: 8px; }
.search-input { padding: 4px 8px; border: 1px solid var(--border-color); border-radius: 4px; }
.filter-select { padding: 4px 8px; }
.btn { padding: 4px 12px; border: 1px solid var(--border-color); border-radius: 4px; cursor: pointer; }
.charts-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; flex: 1; }
.chart-box { border: 1px solid var(--border-color); border-radius: 4px; padding: 8px; }
.chart-box h4 { margin: 0 0 8px 0; font-size: 0.9em; }
.histogram { display: flex; align-items: flex-end; height: 150px; gap: 1px; }
.bar { flex: 1; background: #4a90d9; min-height: 1px; border-radius: 1px 1px 0 0; }
.line-chart-placeholder { height: 150px; display: flex; align-items: center; justify-content: center; color: var(--text-muted); }
</style>
