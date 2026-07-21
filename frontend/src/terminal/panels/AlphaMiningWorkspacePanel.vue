<script setup lang="ts">
import { ref } from 'vue'
import { useMLStore } from '@/stores/ml'
import { PanelHeader, PanelTable, EmptyState } from '@/terminal/components/panel'

const mlStore = useMLStore()
const 已选Factors = ref<string[]>([])
const popSize = ref(200)
const generations = ref(50)
const crossoverRate = ref(0.7)
const mutationRate = ref(0.1)
const topK = ref(20)
const fitnessMetric = ref('ic')
const miningLoading = ref(false)
const miningError = ref('')

const availableFactors = [
  'momentum_1m', 'momentum_3m', 'momentum_6m', 'momentum_12m', 'rsi_alpha',
  'ma_cross', 'macd_divergence', 'trend_strength', 'adx_alpha', 'price_channel',
  'volatility_20d', 'volatility_60d', 'atr_alpha', 'bollinger_position', 'parkinson_vol',
  'volume_ratio', 'volume_trend', 'obv_alpha', 'mfi_alpha', 'vwap_deviation',
  'size_factor', 'sector_neutral_momentum', 'industry_relative', 'turnover_alpha', 'amplitude_alpha',
]

function toggleFactor(name: string) {
  const idx = 已选Factors.value.indexOf(name)
  if (idx >= 0) 已选Factors.value.splice(idx, 1)
  else 已选Factors.value.push(name)
}

async function runMining() {
  miningLoading.value = true; miningError.value = ''
  try {
    await mlStore.runAlphaMining({
      factorNames: 已选Factors.value, factorData: {}, returnsData: {},
      populationSize: popSize.value, generations: generations.value, topK: topK.value,
    })
  } catch (e: any) {
    miningError.value = e?.message || String(e)
  } finally {
    miningLoading.value = false
  }
}

function registerFactor(factor: { formula: string }) {
  window.dispatchEvent(new CustomEvent('quantflow:register-factor', {
    detail: { formula: factor.formula }
  }))
}

const tableColumns = [
  { key: 'formula', label: 'Formula', align: 'left' as const, formatter: (v: string) => v },
  { key: 'ic', label: 'IC', align: 'right' as const, formatter: (v: number) => v?.toFixed(4) ?? '--' },
  { key: 'ir', label: 'IR', align: 'right' as const, formatter: (v: number) => v?.toFixed(4) ?? '--' },
  { key: 'sharpe', label: 'Sharpe', align: 'right' as const, formatter: (v: number) => v?.toFixed(4) ?? '--' },
]
</script>

<template>
  <div class="alpha-mining-panel">
    <PanelHeader
      :title="$t('ml.alpha_mining')"
      :subtitle="已选Factors.length + ' ' + $t('common.selected')"
    />

    <div class="panel-content">
      <div class="factor-pool">
        <h4>基础因子池 ({{ 已选Factors.length }} 已选)</h4>
        <div class="factor-chips">
          <span
            v-for="f in availableFactors"
            :key="f"
            :class="['chip', { active: 已选Factors.includes(f) }]"
            @click="toggleFactor(f)"
          >{{ f }}</span>
        </div>
      </div>

      <div class="gp-config">
        <h4>{{ $t('ml.genetic_config') }}</h4>
        <div class="config-grid">
          <label>{{ $t('ml.population') }}: <input v-model.number="popSize" type="number" min="10" max="1000" /></label>
          <label>{{ $t('ml.generations') }}: <input v-model.number="generations" type="number" min="5" max="200" /></label>
          <label>{{ $t('ml.crossover') }}: <input v-model.number="crossoverRate" type="number" min="0" max="1" step="0.05" /></label>
          <label>{{ $t('ml.mutation') }}: <input v-model.number="mutationRate" type="number" min="0" max="1" step="0.05" /></label>
          <label>{{ $t('ml.top_k') }}: <input v-model.number="topK" type="number" min="1" max="50" /></label>
          <label>适应度:
            <select v-model="fitnessMetric">
              <option value="ic">{{ $t('ml.ic') }}</option><option value="ir">{{ $t('ml.ir') }}</option>
              <option value="sharpe">Sharpe</option><option value="composite">综合</option>
            </select>
          </label>
        </div>
      </div>

      <button
        @click="runMining"
        :disabled="miningLoading || mlStore.miningRunning || 已选Factors.length < 2"
        class="btn btn-primary btn-run"
      >
        {{ miningLoading ? '挖掘中...' : '开始挖掘' }}
      </button>

      <div v-if="miningError" class="panel-error">{{ miningError }}</div>

      <div v-if="mlStore.discoveredFactors.length" class="results">
        <h4>{{ $t('ml.discovered_factors') }}</h4>
        <PanelTable
          :columns="tableColumns"
          :data="mlStore.discoveredFactors"
          :striped="true"
        >
          <template #action="{ row }">
            <button @click="registerFactor(row)" class="btn btn-ghost btn-sm">
              {{ $t('ml.register') }}
            </button>
          </template>
        </PanelTable>
      </div>

      <EmptyState
        v-else-if="!mlStore.miningRunning"
        icon="search"
        :title="$t('ml.no_discovered') || '暂无挖掘结果'"
        :description="$t('ml.select_factors_hint') || '选择至少2个因子并点击开始挖掘'"
      />
    </div>
  </div>
</template>

<style scoped>
.alpha-mining-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--panel-padding);
}

.factor-pool { margin-bottom: var(--space-lg); }
.factor-pool h4 { font-size: var(--font-sm); color: var(--color-text-secondary); margin: 0 0 var(--space-sm) 0; }

.factor-chips { display: flex; flex-wrap: wrap; gap: var(--space-xs); }

.chip {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
  background: var(--color-bg-subtle);
  transition: all var(--transition-fast);
}

.chip:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
}

.chip.active {
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-color: var(--color-accent);
}

.gp-config { margin-bottom: var(--space-lg); }
.gp-config h4 { font-size: var(--font-sm); color: var(--color-text-secondary); margin: 0 0 var(--space-sm) 0; }

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
}

.config-grid label {
  display: flex;
  flex-direction: column;
  font-size: var(--font-sm);
  gap: var(--space-xs);
  color: var(--color-text-secondary);
}

.config-grid input,
.config-grid select {
  padding: var(--space-sm);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-input);
  color: var(--color-text-primary);
  font-size: var(--font-sm);
  font-family: inherit;
}

.config-grid input:focus,
.config-grid select:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.btn-run {
  margin-bottom: var(--space-lg);
  padding: var(--space-sm) var(--space-xl);
  font-size: var(--font-base);
}

.results { flex: 1; }
.results h4 { font-size: var(--font-sm); color: var(--color-text-secondary); margin: 0 0 var(--space-sm) 0; }

.btn-sm {
  padding: 2px 8px;
  font-size: var(--font-xs);
}

.formula :deep(.td) {
  font-family: 'JetBrains Mono', monospace;
  font-size: var(--font-xs);
}
</style>
