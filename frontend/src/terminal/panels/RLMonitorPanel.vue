<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import VChart from 'vue-echarts'
import * as echarts from 'echarts'

defineProps<{ panelId: string; params?: Record<string, any> }>()

// 算法 selector
const algorithm = ref<'ppo' | 'dqn' | 'sac'>('ppo')
const algorithms = ['ppo', 'dqn', 'sac'] as const

// Training state
const isTraining = ref(false)
const episode = ref(0)
const total回合数 = ref(100)
const rewards = ref<number[]>([])
const sharpes = ref<number[]>([])
const currentReward = ref(0)
const currentSharpe = ref(0)

let intervalId: ReturnType<typeof setInterval> | null = null

function startTraining() {
  if (isTraining.value) return
  isTraining.value = true
  episode.value = 0
  rewards.value = []
  sharpes.value = []

  intervalId = setInterval(() => {
    episode.value++
    const r = (Math.random() - 0.48) * 0.02
    const s = ((r / 0.01) * Math.sqrt(252)) * (0.5 + Math.random() * 0.5)
    rewards.value.push(r)
    sharpes.value.push(s)
    currentReward.value = r
    currentSharpe.value = s

    if (episode.value >= total回合数.value) {
      pauseTraining()
    }
  }, 200)
}

function pauseTraining() {
  isTraining.value = false
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

function saveModel() {
  // Placeholder: save trained model
}

onUnmounted(() => {
  if (intervalId) clearInterval(intervalId)
})

const rewardChartOption = computed(() => ({
  backgroundColor: 'transparent',
  title: { text: 'Reward per 回合', textStyle: { color: '#c9d1d9', fontSize: 13 }, left: 'center' },
  grid: { top: 40, right: 20, bottom: 30, left: 50 },
  xAxis: { type: 'value', name: '回合', axisLabel: { color: '#5a6380', fontSize: 10 } },
  yAxis: { type: 'value', name: 'Reward', axisLabel: { color: '#5a6380', fontSize: 10 } },
  series: [{
    type: 'line',
    data: rewards.value.map((r, i) => [i + 1, r]),
    smooth: true,
    lineStyle: { color: '#3fb950', width: 2 },
    areaStyle: {
      color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
        { offset: 0, color: 'rgba(63,185,80,0.3)' },
        { offset: 1, color: 'rgba(63,185,80,0.02)' },
      ]),
    },
    symbol: 'none',
  }],
}))

const sharpeChartOption = computed(() => ({
  backgroundColor: 'transparent',
  title: { text: 'Sharpe Ratio', textStyle: { color: '#c9d1d9', fontSize: 13 }, left: 'center' },
  grid: { top: 40, right: 20, bottom: 30, left: 50 },
  xAxis: { type: 'value', name: '回合', axisLabel: { color: '#5a6380', fontSize: 10 } },
  yAxis: { type: 'value', name: 'Sharpe', axisLabel: { color: '#5a6380', fontSize: 10 } },
  series: [{
    type: 'line',
    data: sharpes.value.map((s, i) => [i + 1, s]),
    smooth: true,
    lineStyle: { color: '#58a6ff', width: 2 },
    symbol: 'none',
  }],
}))

const fmt = (n: number, dec = 2) => n.toFixed(dec)
</script>

<template>
  <div class="rl-monitor-panel">
    <div class="header">
      <h3 class="panel-title">强化学习监控</h3>
      <div class="controls">
        <select v-model="algorithm" class="algo-select" :disabled="isTraining">
          <option v-for="a in algorithms" :key="a" :value="a">{{ a.toUpperCase() }}</option>
        </select>
        <label class="ep-label">
          回合数
          <input v-model.number="total回合数" type="number" min="10" max="5000" step="10" class="ep-input" :disabled="isTraining" />
        </label>
        <button v-if="!isTraining" class="btn btn-start" @click="startTraining">开始</button>
        <button v-else class="btn btn-pause" @click="pauseTraining">暂停</button>
        <button class="btn btn-save" @click="saveModel" :disabled="rewards.length === 0">保存</button>
      </div>
    </div>

    <div class="kpi-row">
      <div class="kpi-card">
        <span class="kpi-label">回合</span>
        <span class="kpi-value">{{ episode }} / {{ total回合数 }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">最新奖励</span>
        <span class="kpi-value" :style="{ color: currentReward >= 0 ? '#3fb950' : '#f85149' }">{{ fmt(currentReward, 6) }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">最新夏普</span>
        <span class="kpi-value" :style="{ color: currentSharpe >= 1 ? '#3fb950' : '#f0883e' }">{{ fmt(currentSharpe, 4) }}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">算法</span>
        <span class="kpi-value mono">{{ algorithm.toUpperCase() }}</span>
      </div>
    </div>

    <div class="charts-grid">
      <div class="chart-box">
        <VChart :option="rewardChartOption" autoresize style="height: 220px" />
      </div>
      <div class="chart-box">
        <VChart :option="sharpeChartOption" autoresize style="height: 220px" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.rl-monitor-panel {
  padding: 12px;
  background: var(--bg);
  height: 100%;
  overflow-y: auto;
  font-variant-numeric: tabular-nums;
}

.header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.panel-title {
  font-size: 12px;
  color: var(--muted);
  text-transform: uppercase;
  margin: 0;
}

.controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.algo-select {
  background: var(--card);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
}

.ep-label {
  font-size: 11px;
  color: var(--muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.ep-input {
  width: 60px;
  background: var(--card);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 3px 6px;
  border-radius: 4px;
  font-size: 11px;
}

.btn {
  padding: 5px 14px;
  border: none;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-start { background: #3fb950; color: #0d1117; }
.btn-pause { background: #f0883e; color: #0d1117; }
.btn-save { background: #58a6ff; color: #0d1117; }

.kpi-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-bottom: 12px;
}

.kpi-card {
  padding: 10px;
  background: var(--card);
  border-radius: 4px;
}

.kpi-label {
  display: block;
  font-size: 10px;
  color: var(--muted);
  text-transform: uppercase;
  margin-bottom: 4px;
}

.kpi-value {
  font-size: 17px;
  font-weight: 700;
  color: var(--text);
}

.kpi-value.mono {
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.chart-box {
  background: var(--card);
  border-radius: 4px;
  padding: 8px;
}
</style>
