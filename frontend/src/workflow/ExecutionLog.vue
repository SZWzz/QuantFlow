<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'

const workflow = useWorkflowStore()
const logContainer = ref<HTMLElement | null>(null)

const entries = computed(() => {
  const result: { id: string; type: string; status: string; duration?: number; error?: string; output?: any }[] = []
  for (const [nodeId, status] of workflow.nodeStatuses) {
    result.push({
      id: nodeId,
      type: 'node',
      status: status.status,
      duration: status.duration,
      error: status.error,
      output: workflow.nodeOutputs.get(nodeId),
    })
  }
  return result
})

function outputNodeType(id: string): string {
  const node = (workflow.nodes as any[]).find((n: any) => n.id === id)
  return node?.data?.nodeType || ''
}

function getBacktestEquityCurve(): number[] {
  for (const entry of entries.value) {
    if (outputNodeType(entry.id) === 'backtest' && entry.output?.equity_curve) {
      return entry.output.equity_curve
    }
  }
  return []
}

function getChartDataEquity(): number[] {
  for (const entry of entries.value) {
    if (outputNodeType(entry.id) === 'chart_data' && entry.output?.chart_json) {
      try {
        const opt = JSON.parse(typeof entry.output.chart_json === 'string' ? entry.output.chart_json : JSON.stringify(entry.output.chart_json))
        if (opt.series?.[0]?.data) return opt.series[0].data
      } catch { /* fall through */ }
    }
  }
  return []
}

const equityData = computed(() => {
  const data = getBacktestEquityCurve()
  if (data.length > 0) return data
  return getChartDataEquity()
})

function fmtPrice(v: number): string {
  if (v >= 1e6) return (v / 1e4).toFixed(0) + '万'
  if (v >= 1e4) return (v / 1e4).toFixed(2) + '万'
  return v.toFixed(2)
}

const chartInfo = computed(() => {
  if (equityData.value.length < 2) return null
  const data = equityData.value
  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const w = 400, h = 180
  const padL = 52, padR = 8, padT = 8, padB = 16
  const pw = w - padL - padR
  const ph = h - padT - padB

  const points = data.map((v, i) => {
    const x = padL + (i / (data.length - 1)) * pw
    const y = padT + ph - ((v - min) / range) * ph
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')

  const nLabels = 5
  const labels: { y: number; label: string }[] = []
  for (let i = 0; i < nLabels; i++) {
    const pct = i / (nLabels - 1)
    const val = min + (max - min) * (1 - pct)
    const y = padT + ph * pct
    labels.push({ y, label: fmtPrice(val) })
  }
  return { polyline: points, labels, w, h }
})

function clearLog() {
  workflow.resetExecution()
}

watch(
  () => entries.value.length,
  () => {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight
      }
    })
  }
)

function statusIcon(s: string): string {
  switch (s) {
    case 'success': return '✓'
    case 'failed': return '✗'
    case 'running': return '⟳'
    case 'skipped': return '−'
  }
  return '•'
}

function statusColor(s: string): string {
  switch (s) {
    case 'success': return 'var(--wf-success)'
    case 'failed': return 'var(--wf-danger)'
    case 'running': return 'var(--wf-warn)'
    case 'skipped': return 'var(--wf-skipped)'
  }
  return 'var(--wf-skipped)'
}

function formatOutput(key: any, val: any): string {
  if (typeof val === 'number') return val.toFixed(6)
  if (typeof val === 'string') return val.length > 300 ? val.slice(0, 300) + '…' : val
  if (Array.isArray(val)) return `[${val.length} items]`
  if (typeof val === 'object' && val !== null) return JSON.stringify(val).slice(0, 200)
  return String(val)
}
</script>

<template>
  <div class="execution-log">
    <div class="log-header">
      <span class="log-title">Execution Log</span>
      <div class="log-actions">
        <span v-if="workflow.executionStatus === 'running'" class="running-badge">Running...</span>
        <span v-else-if="workflow.executionStatus === 'completed'" class="done-badge">Done</span>
        <span v-else-if="workflow.executionStatus === 'failed'" class="fail-badge">Failed</span>
        <button class="clear-btn" @click="clearLog">Clear</button>
      </div>
    </div>

    <div ref="logContainer" class="log-content">
      <div v-if="entries.length === 0" class="empty-log">
        Run a workflow to see execution output
      </div>

      <!-- inline equity curve chart with axes -->
      <div v-if="chartInfo" class="chart-output">
        <div class="chart-label">📈 净值曲线 ({{ equityData.length }} bars)</div>
        <svg :viewBox="`0 0 ${chartInfo.w} ${chartInfo.h}`" class="equity-svg" preserveAspectRatio="xMidYMid meet">
          <g v-for="(l, i) in chartInfo.labels" :key="'g'+i">
            <line x1="52" :x2="chartInfo.w-8" :y1="l.y" :y2="l.y" style="stroke: var(--wf-chart-grid)" stroke-width="0.5" stroke-dasharray="3,3" />
            <text x="50" :y="l.y+3" text-anchor="end" style="fill: var(--wf-chart-text); font-size: var(--font-xs)">{{ l.label }}</text>
          </g>
          <polyline :points="chartInfo.polyline" fill="none" style="stroke: var(--wf-success)" stroke-width="1.5" />
        </svg>
      </div>

      <div v-for="entry in entries" :key="entry.id" class="log-entry">
        <span class="entry-icon" :style="{ color: statusColor(entry.status) }">
          {{ statusIcon(entry.status) }}
        </span>
        <span class="entry-node">{{ entry.id }}</span>
        <span class="entry-status" :style="{ color: statusColor(entry.status) }">{{ entry.status }}</span>
        <span v-if="entry.duration" class="entry-time">{{ (entry.duration / 1000).toFixed(2) }}µs</span>
        <span v-if="entry.error" class="entry-error">{{ entry.error }}</span>
      </div>

      <!-- log_output: structured display -->
      <div v-for="entry in entries.filter(e => e.output && outputNodeType(e.id) === 'log_output')" :key="'log-'+entry.id" class="chart-output">
        <div class="chart-label">📋 {{ entry.id }}</div>
        <div class="chart-inline">
          <div v-for="(val, key) in entry.output" :key="key" class="output-row">
            <span class="output-key">{{ key }}:</span>
            <span class="output-val">{{ formatOutput(key, val) }}</span>
          </div>
        </div>
      </div>

      <!-- other node outputs -->
      <div v-for="entry in entries.filter(e => e.output && outputNodeType(e.id) !== 'chart_data' && outputNodeType(e.id) !== 'log_output')" :key="'out-'+entry.id" class="chart-output">
        <div class="chart-label">📦 {{ entry.id }} output</div>
        <div class="chart-inline">
          <div v-for="(val, key) in entry.output" :key="key" class="output-row">
            <span class="output-key">{{ key }}:</span>
            <span class="output-val">{{ formatOutput(key, val) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.execution-log {
  background: var(--color-bg-input);
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  max-height: 500px;
  flex-shrink: 0;
}

.log-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; background: var(--color-bg-input); border-bottom: 1px solid var(--color-border);
}

.log-title { font-size: var(--font-xs); font-weight: 600; color: var(--color-text-primary); text-transform: uppercase; letter-spacing: 0.5px; }
.log-actions { display: flex; align-items: center; gap: 8px; }

.running-badge { font-size: var(--font-xs); color: var(--wf-warn); animation: pulse 1s ease-in-out infinite; }
.done-badge { font-size: var(--font-xs); color: var(--wf-success); }
.fail-badge { font-size: var(--font-xs); color: var(--wf-danger); }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.clear-btn {
  padding: 2px 8px; background: transparent; border: 1px solid var(--color-border);
  color: var(--color-text-tertiary); border-radius: var(--radius-sm); font-size: var(--font-xs); cursor: pointer;
}
.clear-btn:hover { color: var(--color-text-primary); border-color: var(--color-border-strong); }

.log-content { flex: 1; overflow-y: auto; padding: 6px; font-family: monospace; }

.empty-log { padding: 16px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; font-family: system-ui; }

.log-entry { display: flex; align-items: center; gap: 6px; padding: 2px 4px; font-size: var(--font-xs); }
.entry-icon { font-size: var(--font-xs); width: 12px; text-align: center; }
.entry-node { color: var(--color-text-primary); min-width: 80px; }
.entry-status { font-weight: 500; }
.entry-time { color: var(--color-text-tertiary); margin-left: auto; }
.entry-error { color: var(--wf-danger); font-size: var(--font-xs); }

.chart-output { padding: 4px 8px; margin: 2px 0; background: var(--color-bg-panel); border-radius: var(--radius-sm); }
.chart-label { font-size: var(--font-xs); font-weight: 600; color: var(--color-text-secondary); margin-bottom: 2px; }
.chart-inline { display: flex; flex-direction: column; gap: 1px; }
.output-row { font-size: var(--font-xs); color: var(--color-text-primary); }
.output-key { color: var(--color-text-tertiary); margin-right: 4px; }
.output-val { word-break: break-all; }

.equity-svg {
  width: 100%;
  height: auto;
  max-height: 160px;
}
</style>
