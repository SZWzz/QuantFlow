<script setup lang="ts">
import { ref } from 'vue'
import { useMLStore } from '@/stores/ml'

const mlStore = useMLStore()
const selectedFactors = ref<string[]>([])
const popSize = ref(200)
const generations = ref(50)
const crossoverRate = ref(0.7)
const mutationRate = ref(0.1)
const topK = ref(20)
const fitnessMetric = ref('ic')

const availableFactors = [
  'momentum_1m', 'momentum_3m', 'momentum_6m', 'momentum_12m', 'rsi_alpha',
  'ma_cross', 'macd_divergence', 'trend_strength', 'adx_alpha', 'price_channel',
  'volatility_20d', 'volatility_60d', 'atr_alpha', 'bollinger_position', 'parkinson_vol',
  'volume_ratio', 'volume_trend', 'obv_alpha', 'mfi_alpha', 'vwap_deviation',
  'size_factor', 'sector_neutral_momentum', 'industry_relative', 'turnover_alpha', 'amplitude_alpha',
]

function toggleFactor(name: string) {
  const idx = selectedFactors.value.indexOf(name)
  if (idx >= 0) selectedFactors.value.splice(idx, 1)
  else selectedFactors.value.push(name)
}

async function runMining() {
  await mlStore.runAlphaMining({
    factorNames: selectedFactors.value, factorData: {}, returnsData: {},
    populationSize: popSize.value, generations: generations.value, topK: topK.value,
  })
}

function registerFactor(factor: { formula: string }) {
  window.dispatchEvent(new CustomEvent('quantflow:register-factor', {
    detail: { formula: factor.formula }
  }))
}
</script>

<template>
  <div class="alpha-mining-panel">
    <h3>Alpha Mining Workspace</h3>
    <div class="factor-pool">
      <h4>Base Factor Pool ({{ selectedFactors.length }} selected)</h4>
      <div class="factor-chips">
        <span v-for="f in availableFactors" :key="f"
              :class="['chip', { active: selectedFactors.includes(f) }]"
              @click="toggleFactor(f)">{{ f }}</span>
      </div>
    </div>
    <div class="gp-config">
      <h4>Genetic Programming Config</h4>
      <div class="config-grid">
        <label>Population: <input v-model.number="popSize" type="number" min="10" max="1000" /></label>
        <label>Generations: <input v-model.number="generations" type="number" min="5" max="200" /></label>
        <label>Crossover: <input v-model.number="crossoverRate" type="number" min="0" max="1" step="0.05" /></label>
        <label>Mutation: <input v-model.number="mutationRate" type="number" min="0" max="1" step="0.05" /></label>
        <label>Top K: <input v-model.number="topK" type="number" min="1" max="50" /></label>
        <label>Fitness:
          <select v-model="fitnessMetric">
            <option value="ic">IC</option><option value="ir">IR</option>
            <option value="sharpe">Sharpe</option><option value="composite">Composite</option>
          </select>
        </label>
      </div>
    </div>
    <button @click="runMining" :disabled="mlStore.miningRunning || selectedFactors.length < 2" class="btn-run">
      {{ mlStore.miningRunning ? 'Mining...' : 'Start Mining' }}
    </button>
    <div v-if="mlStore.discoveredFactors.length" class="results">
      <h4>Discovered Factors</h4>
      <table><thead><tr><th>Formula</th><th>IC</th><th>IR</th><th>Sharpe</th><th>Action</th></tr></thead>
        <tbody><tr v-for="(f, i) in mlStore.discoveredFactors" :key="i">
          <td class="formula">{{ f.formula }}</td><td>{{ f.ic?.toFixed(4) }}</td>
          <td>{{ f.ir?.toFixed(4) }}</td><td>{{ f.sharpe?.toFixed(4) }}</td>
          <td><button @click="registerFactor(f)" class="btn btn-sm">Register</button></td>
        </tr></tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.alpha-mining-panel { padding: 12px; height: 100%; overflow-y: auto; }
.factor-chips { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 12px; }
.chip { padding: 2px 8px; border: 1px solid var(--border-color); border-radius: 12px; cursor: pointer; font-size: 0.85em; }
.chip.active { background: #4a90d9; color: white; border-color: #4a90d9; }
.config-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 12px; }
.config-grid label { display: flex; flex-direction: column; font-size: 0.9em; gap: 2px; }
.config-grid input, .config-grid select { padding: 4px; border: 1px solid var(--border-color); border-radius: 4px; }
.btn-run { padding: 8px 24px; background: #4a90d9; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 1em; }
.btn-run:disabled { opacity: 0.5; cursor: not-allowed; }
.results table { width: 100%; border-collapse: collapse; margin-top: 8px; }
.results th, .results td { padding: 6px 8px; text-align: left; border-bottom: 1px solid var(--border-color); font-size: 0.9em; }
.formula { font-family: monospace; font-size: 0.85em; max-width: 300px; overflow-x: auto; }
.btn-sm { padding: 2px 8px; font-size: 0.85em; cursor: pointer; }
</style>
