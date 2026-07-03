<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Handle, Position } from '@vue-flow/core'
import { nodeLabel, paramLabel } from '@/workflow/nodeLabels'

const { t } = useI18n()

const props = defineProps<{
  id: string
  data: {
    nodeType: string
    label: string
    params: Record<string, any>
    inputs?: string[]
    outputs?: string[]
    status?: 'idle' | 'running' | 'success' | 'failed'
    error?: string
  }
  selected?: boolean
}>()

const emit = defineEmits<{
  (e: 'updateParams', params: Record<string, any>): void
}>()

function portLabel(name: string): string { return t(`workflow.port_${name}`) !== `workflow.port_${name}` ? t(`workflow.port_${name}`) : name }

const CAT_ICONS: Record<string, string> = {
  data: '📊', indicator: '📈', signal: '⚡', trading: '💰', risk: '🛡️',
  portfolio: '📦', strategy: '🧠', ml: '🤖', ai: '🤖', output: '📝', control: '🔀',
  utility: '🔧', research: '🔍', alternative_data: '🛰️', notify: '🔔', schedule: '⏰',
  backtest: '📉', alpha: '⭐',
}
const CAT_COLORS: Record<string, string> = {
  data: '#58a6ff', indicator: '#3fb950', signal: '#f0883e', trading: '#e94560',
  risk: '#ef4444', portfolio: '#14b8a6', strategy: '#06b6d4', ml: '#a855f7',
  ai: '#ec4899', output: '#a371f7', control: '#6366f1', utility: '#64748b',
  research: '#0ea5e9', alternative_data: '#84cc16', notify: '#f97316', schedule: '#6366f1',
  backtest: '#8b5cf6', alpha: '#f59e0b',
}

const category = computed(() => (props.data as any).category || 'utility')
const catIcon = computed(() => CAT_ICONS[category.value] || '🔹')
const categoryColor = computed(() => CAT_COLORS[category.value] || CAT_COLORS.utility)
const statusClass = computed(() => `status-${props.data.status || 'idle'}`)

// ── Inline param editing ──
const editingParam = ref<string | null>(null)
const editValue = ref('')

function startEdit(key: string) {
  editingParam.value = key
  const v = (props.data.params || {})[key]
  editValue.value = v !== undefined && v !== null ? String(v) : ''
}

function commitEdit() {
  if (!editingParam.value) return
  const key = editingParam.value
  const raw = editValue.value.trim()
  let val: any = raw
  if (raw === 'true') val = true
  else if (raw === 'false') val = false
  else if (raw !== '' && !isNaN(Number(raw))) val = Number(raw)
  else if (raw === '') val = null
  const newParams = { ...props.data.params, [key]: val }
  if (val === null) delete newParams[key]
  props.data.params = newParams
  emit('updateParams', newParams)
  editingParam.value = null
}
function cancelEdit() { editingParam.value = null }

const paramKeys = computed(() => Object.keys(props.data.params || {}))
const inputPorts = computed(() => (props.data.inputs && props.data.inputs.length) ? props.data.inputs : ['input'])
const outputPorts = computed(() => (props.data.outputs && props.data.outputs.length) ? props.data.outputs : ['output'])
const maxPorts = computed(() => Math.max(inputPorts.value.length, outputPorts.value.length, 1))

const HEADER_H = 30
const PORT_ROW_H = 26
const paramsH = computed(() => paramKeys.value.length > 0 ? paramKeys.value.length * 20 + 6 : 0)

// Position each port dot at the correct vertical offset
function portTop(idx: number): string {
  return (HEADER_H + paramsH.value + idx * PORT_ROW_H + PORT_ROW_H / 2) + 'px'
}
</script>

<template>
  <div class="custom-node" :class="[statusClass, { selected }]">
    <!-- Header -->
    <div class="node-header" :style="{ background: categoryColor }">
      <span class="cat-icon">{{ catIcon }}</span>
      <span class="node-type">{{ nodeLabel(data.nodeType) }}</span>
    </div>

    <!-- Inline editable params — click any param to edit directly on the card -->
    <div v-if="paramKeys.length > 0" class="node-params">
      <div class="params-hint">{{ t('workflow.click_to_edit') }}</div>
      <div v-for="key in paramKeys" :key="key" class="param-row" @click.stop="startEdit(key)">
        <template v-if="editingParam === key">
          <input class="param-input" v-model="editValue" @keyup.enter="commitEdit" @keyup.escape="cancelEdit" @blur="commitEdit" autofocus @click.stop />
        </template>
        <template v-else>
          <span class="param-key">{{ paramLabel(key, t) }}</span>
          <span class="param-val">{{ (data.params || {})[key] }}</span>
          <span class="edit-hint">✎</span>
        </template>
      </div>
    </div>

    <!-- Input handles on left edge -->
    <Handle
      v-for="(port, idx) in inputPorts"
      :key="'in-' + port"
      :type="'target'"
      :position="Position.Left"
      :id="port"
      :style="{ top: portTop(idx) }"
      class="port-dot port-dot-left"
    />
    <!-- Output handles on right edge -->
    <Handle
      v-for="(port, idx) in outputPorts"
      :key="'out-' + port"
      :type="'source'"
      :position="Position.Right"
      :id="port"
      :style="{ top: portTop(idx) }"
      class="port-dot port-dot-right"
    />
    <!-- Port labels -->
    <div class="node-ports">
      <div
        v-for="i in maxPorts"
        :key="'pr-' + i"
        class="port-row"
      >
        <span class="port-label left-label">
          {{ inputPorts[i - 1] ? portLabel(inputPorts[i - 1]) : '' }}
        </span>
        <span class="port-label right-label">
          {{ outputPorts[i - 1] ? portLabel(outputPorts[i - 1]) : '' }}
        </span>
      </div>
    </div>

    <!-- Badges -->
    <div class="node-badges">
      <span v-if="(data as any).badges?.pin" class="badge pin-badge" :title="t('workflow.pinned_output')">📌</span>
      <span v-if="(data as any).mode === 2" class="badge disabled-badge" :title="t('workflow.disabled')">⏸</span>
    </div>

    <!-- Status -->
    <div v-if="data.status === 'running'" class="running-indicator" />
    <div v-if="data.status === 'success'" class="success-check">✓</div>
    <div v-if="data.status === 'failed'" class="failed-mark">✗ {{ data.error }}</div>
  </div>
</template>

<style scoped>
.custom-node {
  background: #1c2333;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  min-width: 170px;
  max-width: 240px;
  font-size: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: border-color 0.15s, box-shadow 0.15s;
  position: relative;
  overflow: visible;
}

.custom-node.selected {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.3);
}

.custom-node.status-running { border-color: #f0883e; animation: pulse 1.5s ease-in-out infinite; }
.custom-node.status-success { border-color: #3fb950; }
.custom-node.status-failed { border-color: #f85149; }

@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }

.node-header {
  padding: 6px 12px; border-radius: 7px 7px 0 0;
  color: #fff; font-weight: 600; font-size: 12px;
  display: flex; align-items: center; gap: 6px;
}

/* ── Inline params ── */
.node-params {
  padding: 4px 12px; border-bottom: 1px solid rgba(255,255,255,.06);
  background: rgba(0,0,0,.15);
}

/* ── Badges ── */
.node-badges { position: absolute; top: -8px; right: -8px; display: flex; gap: 2px; z-index: 30; }
.badge { font-size: 12px; line-height: 1; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5)); }
.param-row { display: flex; align-items: center; gap: 6px; padding: 1px 0; cursor: pointer; }
.param-row { position: relative; }
.param-row:hover { background: rgba(88,166,255,.12); border-radius: 3px; }
.param-row:hover .edit-hint { opacity: 1; }
.edit-hint {
  position: absolute; right: 4px; font-size: 9px; color: var(--color-accent);
  opacity: 0; transition: opacity .15s;
}
.params-hint {
  font-size: 9px; color: #6b7a8f;
  padding: 0 0 3px; font-style: italic;
}
.param-key { font-size: 11px; color: #8b949e; flex-shrink: 0; }
.param-key::after { content: ':'; margin-right: 1px; }
.param-val { font-size: 12px; color: #e6edf3; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.param-input { width: 100%; padding: 2px 6px; border: 1px solid var(--color-accent); border-radius: 3px; background: #1a2a3a; color: #e6edf3; font-size: 11px; font-family: monospace; outline: none; }

/* ── Port rows ── */
.node-ports { padding: 4px 0; }
.port-row {
  display: flex; justify-content: space-between; align-items: center;
  height: 26px; padding: 0 12px;
}
.port-label { font-size: 10px; color: var(--color-text-tertiary); user-select: none; white-space: nowrap; }
.right-label { text-align: right; }

/* ── Port dots on card edges ── */
.port-dot {
  width: 10px !important; height: 10px !important;
  background: #58a6ff !important;
  border: 2px solid #171b26 !important;
  border-radius: 50% !important;
  position: absolute !important;
  z-index: 20;
}
.port-dot:hover { transform: scale(1.6); background: #79c0ff !important; }
.port-dot-left { left: -6px; }
.port-dot-right { right: -6px; }

/* ── Status ── */
.running-indicator { position: absolute; bottom: 4px; right: 4px; width: 8px; height: 8px; background: #f0883e; border-radius: 50%; }
.success-check { position: absolute; bottom: 2px; right: 6px; color: #3fb950; font-size: 14px; font-weight: bold; }
.failed-mark { position: absolute; bottom: 2px; left: 6px; color: #f85149; font-size: 10px; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
