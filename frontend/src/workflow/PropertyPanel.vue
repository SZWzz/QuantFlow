<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWorkflowStore } from '@/stores/workflow'
import { useTerminalStore } from '@/stores/terminal'
import { paramLabel } from '@/workflow/nodeLabels'

const { t } = useI18n()
const workflow = useWorkflowStore()
const terminal = useTerminalStore()

const credentialNames = ref<string[]>([])
onMounted(async () => {
  try {
    const app = (window as any).go?.main?.App
    if (app?.ListCredentialNames) credentialNames.value = await app.ListCredentialNames()
  } catch { /* ignore */ }
})

function isCredentialParam(key: string): boolean {
  const lower = key.toLowerCase()
  return lower.includes('api_key') || lower.includes('secret') || lower.includes('token') || lower === 'credential'
}

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

const visibleParams = computed(() => {
  if (!selectedNode.value?.data.params) return {}
  const p: Record<string, any> = {}
  for (const [k, v] of Object.entries(selectedNode.value.data.params)) {
    if (!k.startsWith('_')) p[k] = v
  }
  return p
})

const errorStrategy = computed(() => selectedNode.value?.data.params?._onError || 'stop')
const retryCount = computed(() => selectedNode.value?.data.params?._retryCount || 3)

function setErrorStrategy(s: string) {
  if (!selectedNode.value) return
  if (!selectedNode.value.data.params) selectedNode.value.data.params = {}
  if (s === 'stop') {
    delete selectedNode.value.data.params._onError
    delete selectedNode.value.data.params._retryCount
  } else {
    selectedNode.value.data.params._onError = s
    if (s === 'retry' && !selectedNode.value.data.params._retryCount) {
      selectedNode.value.data.params._retryCount = 3
    }
  }
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
  terminal.openPanel(panelId, { symbol: node.data.params?.symbol || '600519' })
}
</script>

<template>
  <div class="property-panel">
    <div class="panel-header">
      <h3>{{ t('workflow.properties') }}</h3>
    </div>

    <div v-if="selectedNode" class="panel-content">
      <div class="prop-section">
        <div class="prop-row">
          <span class="prop-label">{{ t('workflow.id') }}</span>
          <span class="prop-value mono">{{ selectedNode.id }}</span>
        </div>
        <div class="prop-row">
          <span class="prop-label">{{ t('workflow.type') }}</span>
          <span class="prop-value type-badge">{{ selectedNode.data.nodeType }}</span>
        </div>
        <button class="pin-btn" @click="pinToTerminal">{{ t('workflow.pin_terminal') }}</button>
      </div>

      <!-- Error handling strategy -->
      <div class="prop-section">
        <h4 class="section-title">{{ t('workflow.error_handling') }}</h4>
        <div class="error-strategy-row">
          <select class="strategy-select" :value="errorStrategy" @change="setErrorStrategy(($event.target as HTMLSelectElement).value)">
            <option value="stop">⏹ {{ t('workflow.error_stop') }}</option>
            <option value="skip">⏭ {{ t('workflow.error_skip') }}</option>
            <option value="retry">🔄 {{ t('workflow.error_retry') }}</option>
          </select>
          <div v-if="errorStrategy === 'retry'" class="retry-config">
            <label class="retry-label">{{ t('workflow.retry_count') }}</label>
            <input
              class="retry-input"
              type="number"
              min="1" max="10"
              :value="retryCount"
              @input="updateParam('_retryCount', ($event.target as HTMLInputElement).value)"
            />
          </div>
        </div>
      </div>

      <div v-if="selectedNode.data.inputs" class="prop-section">
        <h4 class="section-title">{{ t('workflow.input_ports') }}</h4>
        <div v-for="port in selectedNode.data.inputs" :key="port" class="port-item">
          <span class="port-dir">◀</span> {{ port }}
        </div>
      </div>

      <div v-if="selectedNode.data.outputs" class="prop-section">
        <h4 class="section-title">{{ t('workflow.output_ports') }}</h4>
        <div v-for="port in selectedNode.data.outputs" :key="port" class="port-item">
          <span class="port-dir">▶</span> {{ port }}
        </div>
      </div>

      <div v-if="Object.keys(visibleParams).length" class="prop-section">
        <h4 class="section-title">{{ t('workflow.parameters') }}</h4>
        <div v-for="(val, key) in visibleParams" :key="key" class="param-row">
          <label class="param-label">{{ paramLabel(key, t) }}</label>
          <input
            class="param-input"
            :value="formatValue(val)"
            @input="updateParam(key, ($event.target as HTMLInputElement).value)"
            placeholder="—"
          />
        </div>
      </div>

      <div v-if="selectedNode.data.status && selectedNode.data.status !== 'idle'" class="prop-section">
        <h4 class="section-title">{{ t('workflow.status') }}</h4>
        <div class="status-badge" :class="selectedNode.data.status">
          {{ selectedNode.data.status }}
        </div>
        <div v-if="selectedNode.data.error" class="error-text">{{ selectedNode.data.error }}</div>
      </div>
    </div>

    <div v-else class="no-selection">
      <p>{{ t('workflow.select_node') }}</p>
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
.type-badge { padding: 1px 6px; background: var(--color-bg-subtle); border: 1px solid var(--color-border); border-radius: var(--radius-sm); }

.section-title {
  font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase;
  letter-spacing: 0.5px; margin-bottom: 6px; padding-bottom: 3px;
  border-bottom: 1px solid var(--color-border);
}

.param-row { display: flex; flex-direction: column; gap: 3px; margin-bottom: 6px; }
.param-label { font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.5px; }
.param-input {
  padding: 4px 8px; background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: 12px; outline: none;
}
.param-input:focus { border-color: #58a6ff; }

.port-item { font-size: 11px; color: var(--color-text-secondary); padding: 2px 0; }
.port-dir { color: var(--color-text-tertiary); }

.status-badge { display: inline-block; padding: 2px 8px; border-radius: var(--radius-sm); font-size: 11px; font-weight: 600; }
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
  color: #3fb950; border-radius: var(--radius-sm); font-size: 11px; cursor: pointer;
}
.pin-btn:hover { background: rgba(63,185,80,0.2); }

.error-strategy-row { display: flex; flex-direction: column; gap: 6px; }
.strategy-select {
  width: 100%; padding: 5px 8px; background: var(--color-bg-input);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: 12px; outline: none; cursor: pointer;
}
.strategy-select:focus { border-color: #58a6ff; }
.retry-config { display: flex; align-items: center; gap: 8px; }
.retry-label { font-size: 10px; color: var(--color-text-tertiary); }
.retry-input {
  width: 60px; padding: 4px 8px; background: var(--color-bg-input);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: 12px; outline: none;
}
.retry-input:focus { border-color: #58a6ff; }
.cred-select { width: 100%; padding: 4px 8px; background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: 12px; outline: none; cursor: pointer; }
.cred-select:focus { border-color: #58a6ff; }
</style>
