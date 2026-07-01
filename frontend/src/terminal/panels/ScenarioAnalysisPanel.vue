<script setup lang="ts">
import { ref, computed } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const portfolio = usePortfolioStore()

interface Scenario {
  name: string
  description: string
  shocks: { factor: string; change: number }[]
}

const scenarios: Scenario[] = [
  {
    name: '市场大跌 (-15%)',
    description: '系统性风险，全面下跌',
    shocks: [
      { factor: 'equity', change: -0.15 },
      { factor: 'volatility', change: 0.3 },
    ],
  },
  {
    name: '利率上升 (+100bp)',
    description: '美联储加息，债券下跌，成长股承压',
    shocks: [
      { factor: 'bond', change: -0.03 },
      { factor: 'equity_growth', change: -0.08 },
      { factor: 'equity_value', change: 0.02 },
    ],
  },
  {
    name: '商品暴涨',
    description: '地缘冲突，能源/金属飙升',
    shocks: [
      { factor: 'commodity', change: 0.2 },
      { factor: 'equity_energy', change: 0.1 },
      { factor: 'equity_consumer', change: -0.05 },
    ],
  },
  {
    name: '加密货币崩盘 (-30%)',
    description: '监管/交易所风险，加密资产暴跌',
    shocks: [
      { factor: 'crypto', change: -0.3 },
      { factor: 'equity_tech', change: -0.05 },
    ],
  },
  {
    name: '经济复苏',
    description: '经济超预期，周期股领涨',
    shocks: [
      { factor: 'equity', change: 0.1 },
      { factor: 'bond', change: -0.02 },
      { factor: 'commodity', change: 0.05 },
    ],
  },
]

const selectedScenario = ref(0)
const capital = ref(1000000)
const result = ref<{ before: number; after: number; change: number; changePct: number; details: { label: string; before: number; after: number }[] } | null>(null)
const loading = ref(false)

const scenarioResult = computed(() => {
  if (!result.value) return null
  return result.value
})

function runScenario() {
  loading.value = true
  const scenario = scenarios[selectedScenario.value]
  const totalShock = scenario.shocks.reduce((sum, s) => sum + Math.abs(s.change), 0) / scenario.shocks.length
  const estimatedChange = totalShock * 0.8 // simplified: 80% of avg shock applied to portfolio
  const sign = scenario.shocks.reduce((s, sh) => s + sh.change, 0) >= 0 ? 1 : -1
  const change = capital.value * estimatedChange * sign
  const after = capital.value + change

  result.value = {
    before: capital.value,
    after,
    change,
    changePct: (change / capital.value) * 100,
    details: scenario.shocks.map(s => ({
      label: s.factor,
      before: capital.value,
      after: capital.value * (1 + s.change * 0.5),
    })),
  }
  loading.value = false
}

function formatMoney(v: number): string {
  return '$' + v.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })
}
</script>

<template>
  <div class="scenario-analysis-panel">
    <div class="panel-header">
      <h3>{{ $t('misc.scenario_analysis') }}</h3>
    </div>

    <div class="params-section">
      <div class="param-row">
        <label>{{ $t('misc.capital') }}</label>
        <input v-model.number="capital" type="number" class="capital-input" />
      </div>
      <div class="param-row">
        <label>{{ $t('misc.scenario') }}</label>
        <div class="scenario-tabs">
          <button v-for="(s, idx) in scenarios" :key="s.name"
            :class="['sc-tab', { active: selectedScenario === idx }]"
            @click="selectedScenario = idx">
            {{ s.name }}
          </button>
        </div>
      </div>
      <div class="scenario-desc">{{ scenarios[selectedScenario].description }}</div>
      <button class="run-btn" @click="runScenario" :disabled="loading">{{ $t('misc.run_scenario') }}</button>
    </div>

    <SkeletonPanel v-if="loading" type="card" :rows="2" />

    <div v-else-if="scenarioResult" class="result-section">
      <div class="result-cards">
        <div class="res-card">
          <div class="res-label">{{ $t('misc.before') }}</div>
          <div class="res-value">{{ formatMoney(scenarioResult.before) }}</div>
        </div>
        <div class="res-card">
          <div class="res-label">{{ $t('misc.after') }}</div>
          <div class="res-value" :class="scenarioResult.change >= 0 ? 'up' : 'down'">{{ formatMoney(scenarioResult.after) }}</div>
        </div>
        <div class="res-card">
          <div class="res-label">{{ $t('misc.change') }}</div>
          <div class="res-value" :class="scenarioResult.change >= 0 ? 'up' : 'down'">
            {{ scenarioResult.change >= 0 ? '+' : '' }}{{ formatMoney(scenarioResult.change) }}
          </div>
        </div>
        <div class="res-card">
          <div class="res-label">{{ $t('misc.change_pct') }}</div>
          <div class="res-value" :class="scenarioResult.changePct >= 0 ? 'up' : 'down'">
            {{ scenarioResult.changePct >= 0 ? '+' : '' }}{{ scenarioResult.changePct.toFixed(2) }}%
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">{{ $t('misc.set_params_tip') }}</div>
  </div>
</template>

<style scoped>
.scenario-analysis-panel {
  padding: 12px; height: 100%; display: flex; flex-direction: column;
  color: var(--color-text, var(--color-border)); background: var(--color-bg-panel, var(--color-bg-panel)); overflow: hidden;
}
.panel-header { margin-bottom: 12px; }
.panel-header h3 { margin: 0; font-size: 14px; font-weight: 600; }
.params-section { margin-bottom: 16px; display: flex; flex-direction: column; gap: 10px; }
.param-row { display: flex; align-items: center; gap: 8px; }
.param-row label { font-size: 12px; color: var(--color-text-secondary); min-width: 60px; }
.capital-input { padding: 4px 8px; font-size: 12px; border: 1px solid var(--color-border-strong); border-radius: 4px; background: var(--color-bg-elevated); color: var(--color-text-primary); width: 120px; }
.scenario-tabs { display: flex; flex-wrap: wrap; gap: 4px; }
.sc-tab {
  padding: 2px 8px; border: 1px solid var(--color-border-strong); border-radius: 4px;
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: 10px;
}
.sc-tab.active { color: var(--color-accent); border-color: var(--color-accent); background: rgba(59,130,246,0.1); }
.scenario-desc { font-size: 11px; color: var(--color-text-tertiary); }
.run-btn {
  align-self: flex-start; padding: 6px 20px; border: none; border-radius: 4px;
  background: var(--color-accent); color: var(--color-text-primary); cursor: pointer; font-size: 12px; font-weight: 500;
}
.run-btn:hover { background: var(--color-accent); }
.run-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.empty-state { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-tertiary); font-size: 13px; }

.result-section { flex: 1; }
.result-cards { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }
.res-card { padding: 12px; border: 1px solid var(--color-border-subtle); border-radius: 8px; text-align: center; }
.res-label { font-size: 10px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.res-value { font-size: 16px; font-weight: 700; font-variant-numeric: tabular-nums; }
.up { color: var(--color-up); }
.down { color: var(--color-down); }
</style>
