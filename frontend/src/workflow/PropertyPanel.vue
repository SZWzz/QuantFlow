<script setup lang="ts">
import { computed } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'
import { useTerminalStore } from '@/stores/terminal'

const workflow = useWorkflowStore()
const terminal = useTerminalStore()

const selectedNode = computed(() => {
  if (!workflow.selectedNodeId) return null
  return workflow.nodes.find((n) => n.id === workflow.selectedNodeId) || null
})

function updateParam(key: string, value: string) {
  if (!selectedNode.value) return
  const node = workflow.nodes.find((n) => n.id === workflow.selectedNodeId)
  if (node) {
    if (!node.data.params) node.data.params = {}
    // Preserve numeric type: if the old value was a number, coerce back.
    const old = node.data.params[key]
    if (typeof old === 'number') {
      const parsed = Number(value)
      node.data.params[key] = isNaN(parsed) ? value : parsed
    } else if (typeof old === 'boolean') {
      node.data.params[key] = value === 'true'
    } else {
      node.data.params[key] = value
    }
  }
}

function formatValue(value: unknown): string {
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function formatDuration(n: number | undefined): string {
  if (n === undefined) return ''
  return (n / 1000).toFixed(2) + 'µs'
}

function pinToTerminal() {
  if (!selectedNode.value) return
  const node = selectedNode.value
  const panelMap: Record<string, string> = {
    data_loader: 'candlestick',
    sma: 'candlestick',
    cross_signal: 'order-entry',
    log_output: 'system-monitor',
    loop: 'watchlist',
  }
  const panelId = panelMap[node.data.nodeType] || 'system-monitor'
  terminal.openPanel(panelId, { symbol: 'AAPL' })
}
</script>

<template>
  <div class="property-panel">
    <div class="panel-header">
      <h3>Properties</h3>
    </div>

    <div v-if="selectedNode" class="panel-content">
      <div class="prop-section">
        <div class="prop-row">
          <span class="prop-label">ID</span>
          <span class="prop-value mono">{{ selectedNode.id }}</span>
        </div>
        <div class="prop-row">
          <span class="prop-label">Type</span>
          <span class="prop-value type-badge">{{ selectedNode.data.nodeType }}</span>
        </div>
        <button class="pin-btn" @click="pinToTerminal">📌 Pin to Terminal</button>
      </div>

      <div v-if="selectedNode.data.params && Object.keys(selectedNode.data.params).length > 0" class="prop-section">
        <h4 class="section-title">Parameters</h4>
        <div v-for="(value, key) in selectedNode.data.params" :key="key" class="param-row">
          <label class="param-label">{{ key }}</label>
          <input
            class="param-input"
            :value="value"
            @input="updateParam(key, ($event.target as HTMLInputElement).value)"
            type="text"
          />
        </div>
      </div>

      <div v-if="selectedNode.data.inputs" class="prop-section">
        <h4 class="section-title">Input Ports</h4>
        <div v-for="port in selectedNode.data.inputs" :key="port" class="port-item">
          <span class="port-dir">◀</span> {{ port }}
        </div>
      </div>

      <div v-if="selectedNode.data.outputs" class="prop-section">
        <h4 class="section-title">Output Ports</h4>
        <div v-for="port in selectedNode.data.outputs" :key="port" class="port-item">
          <span class="port-dir">▶</span> {{ port }}
        </div>
      </div>

      <div v-if="selectedNode.data.status && selectedNode.data.status !== 'idle'" class="prop-section">
        <h4 class="section-title">Status</h4>
        <div class="status-badge" :class="selectedNode.data.status">
          {{ selectedNode.data.status }}
        </div>
        <div v-if="selectedNode.data.error" class="error-text">{{ selectedNode.data.error }}</div>
      </div>
    </div>

    <div v-else class="no-selection">
      <p>Select a node to view its properties</p>
    </div>
  </div>
</template>

<style scoped>
.property-panel {
  width: 240px;
  background: var(--color-bg-input);
  border-left: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  padding: 10px;
  border-bottom: 1px solid var(--color-border);
}

.panel-header h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-text-tertiary);
  letter-spacing: 0.5px;
}

.panel-content { padding: 8px; overflow-y: auto; flex: 1; }

.prop-section { margin-bottom: 12px; }

.prop-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 4px 0; font-size: 12px;
}
.prop-label { color: var(--color-text-tertiary); }
.prop-value { color: var(--color-text-primary); }
.prop-value.mono { font-family: monospace; font-size: 11px; }
.type-badge { padding: 1px 6px; background: var(--color-bg-subtle); border: 1px solid var(--color-border); border-radius: 3px; }

.section-title {
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase;
  letter-spacing: 0.5px; margin-bottom: 6px; padding-bottom: 3px;
  border-bottom: 1px solid var(--color-border);
}

.param-row { display: flex; flex-direction: column; gap: 3px; margin-bottom: 6px; }
.param-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; }
.param-input {
  padding: 4px 8px; background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-primary); font-size: 12px; outline: none;
}
.param-input:focus { border-color: #58a6ff; }

.port-item { font-size: 11px; color: var(--color-text-secondary); padding: 2px 0; }
.port-dir { color: var(--color-text-tertiary); }

.status-badge { display: inline-block; padding: 2px 8px; border-radius: 3px; font-size: 11px; font-weight: 600; }
.status-badge.running { background: rgba(240,136,62,0.15); color: #f0883e; }
.status-badge.success { background: rgba(63,185,80,0.15); color: #3fb950; }
.status-badge.failed { background: rgba(248,81,73,0.15); color: #f85149; }

.error-text { font-size: 10px; color: #f85149; margin-top: 4px; word-break: break-all; }

.no-selection {
  flex: 1; display: flex; align-items: center; justify-content: center;
  padding: 20px; text-align: center;
}
.no-selection p { font-size: 12px; color: #3a4a6c; }

.pin-btn {
  width: 100%; padding: 4px 8px; margin-top: 6px;
  background: rgba(63,185,80,0.1); border: 1px solid rgba(63,185,80,0.3);
  color: #3fb950; border-radius: 4px; font-size: 11px; cursor: pointer;
}
.pin-btn:hover { background: rgba(63,185,80,0.2); }
</style>
