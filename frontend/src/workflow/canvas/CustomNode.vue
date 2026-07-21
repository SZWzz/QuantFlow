<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Handle, Position } from '@vue-flow/core'
import { nodeLabel, paramLabel } from '@/workflow/nodeLabels'
import { cssVar } from '@/lib/cssVar'

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
// 分类色 → themes.css 现有 --cat-* token（与 NodePalette 同一映射；fallback 为亮色值）
const CAT_COLORS: Record<string, string> = {
  data: cssVar('--cat-market', '#2563eb'),
  indicator: cssVar('--cat-chart', '#0891b2'),
  signal: cssVar('--cat-quant', '#d97706'),
  trading: cssVar('--cat-trading', '#059669'),
  risk: cssVar('--color-danger', '#c62828'),
  portfolio: cssVar('--cat-hk', '#0d9488'),
  strategy: cssVar('--cat-chart', '#0891b2'),
  ml: cssVar('--cat-research', '#7c3aed'),
  ai: cssVar('--cat-altdata', '#db2777'),
  output: cssVar('--cat-system', '#475569'),
  control: cssVar('--cat-altdata', '#db2777'),
  utility: cssVar('--cat-system', '#475569'),
  research: cssVar('--cat-market', '#2563eb'),
  alternative_data: cssVar('--cat-altdata', '#db2777'),
  notify: cssVar('--cat-crypto', '#d97706'),
  schedule: cssVar('--cat-portfolio', '#4f46e5'),
  backtest: cssVar('--cat-research', '#7c3aed'),
  alpha: cssVar('--cat-quant', '#d97706'),
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

// Error handling strategy (stored in _ prefixed params)
const errorStrategy = computed(() => props.data.params?._onError || 'stop')
const retryCount = computed(() => props.data.params?._retryCount ?? 3)

function setErrorStrategy(s: string) {
  if (!props.data.params) props.data.params = {}
  if (s === 'stop') {
    delete props.data.params._onError
    delete props.data.params._retryCount
  } else {
    props.data.params._onError = s
    if (s === 'retry' && !props.data.params._retryCount) {
      props.data.params._retryCount = 3
    }
  }
}
function setRetryCount(n: number) {
  if (!props.data.params) props.data.params = {}
  props.data.params._retryCount = n
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
          <input class="param-input" v-model="editValue" @keyup.enter="commitEdit" @keyup.escape="cancelEdit" @blur="commitEdit" autofocus @mousedown.stop />
        </template>
        <template v-else>
          <span class="param-key">{{ paramLabel(key, t) }}</span>
          <span class="param-val">{{ (data.params || {})[key] }}</span>
          <span class="edit-hint">✎</span>
        </template>
      </div>
    </div>

    <!-- Port rows with handles aligned to labels -->
    <div class="node-ports">
      <div
        v-for="i in maxPorts"
        :key="'pr-' + i"
        class="port-row"
      >
        <Handle
          v-if="inputPorts[i - 1]"
          type="target"
          :position="Position.Left"
          :id="inputPorts[i - 1]"
          class="port-dot port-dot-left"
        />
        <span class="port-label left-label">
          {{ inputPorts[i - 1] ? portLabel(inputPorts[i - 1]) : '' }}
        </span>
        <span class="port-label right-label">
          {{ outputPorts[i - 1] ? portLabel(outputPorts[i - 1]) : '' }}
        </span>
        <Handle
          v-if="outputPorts[i - 1]"
          type="source"
          :position="Position.Right"
          :id="outputPorts[i - 1]"
          class="port-dot port-dot-right"
        />
      </div>
    </div>

    <!-- Error handling (only when selected) -->
    <div v-if="selected" class="node-error-section" @mousedown.stop @click.stop>
      <div class="error-row">
        <select class="error-select" :value="errorStrategy" @change="setErrorStrategy(($event.target as HTMLSelectElement).value)" @mousedown.stop>
          <option value="stop">⏹ {{ t('workflow.error_stop') }}</option>
          <option value="skip">⏭ {{ t('workflow.error_skip') }}</option>
          <option value="retry">🔄 {{ t('workflow.error_retry') }}</option>
        </select>
        <input
          v-if="errorStrategy === 'retry'"
          class="retry-input"
          type="number" min="1" max="10"
          :value="retryCount"
          @input="setRetryCount(Number(($event.target as HTMLInputElement).value))"
          @mousedown.stop
        />
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
  background: var(--wf-node-bg);
  border: 2px solid var(--wf-node-border);
  border-radius: var(--radius-lg);
  min-width: 170px;
  max-width: 240px;
  font-size: 12px;
  box-shadow: var(--shadow-md);
  transition: border-color 0.15s, box-shadow 0.15s;
  position: relative;
  overflow: visible;
}

.custom-node.selected {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(var(--wf-accent-rgb), 0.3);
}

.custom-node.status-running { border-color: var(--wf-warn); animation: pulse 1.5s ease-in-out infinite; }
.custom-node.status-success { border-color: var(--wf-success); }
.custom-node.status-failed { border-color: var(--wf-danger); }

@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }

.node-header {
  padding: 6px 12px; border-radius: 7px 7px 0 0;
  color: var(--wf-node-header-text); font-weight: 600; font-size: 12px;
  display: flex; align-items: center; gap: 6px;
}

/* ── Inline params ── */
.node-params {
  padding: 4px 12px; border-bottom: 1px solid var(--wf-node-divider);
  background: var(--wf-node-input-bg);
}

/* ── Badges ── */
.node-badges { position: absolute; top: -8px; right: -8px; display: flex; gap: 2px; z-index: 30; }
.badge { font-size: 12px; line-height: 1; }
.param-row { display: flex; align-items: center; gap: 6px; padding: 1px 0; cursor: pointer; }
.param-row { position: relative; }
.param-row:hover { background: rgba(var(--wf-accent-rgb), .12); border-radius: 3px; }
.param-row:hover .edit-hint { opacity: 1; }
.edit-hint {
  position: absolute; right: 4px; font-size: var(--font-xs); color: var(--color-accent);
  opacity: 0; transition: opacity .15s;
}
.params-hint {
  font-size: var(--font-xs); color: var(--wf-node-hint);
  padding: 0 0 3px; font-style: italic;
}
.param-key { font-size: var(--font-xs); color: var(--wf-node-subtext); flex-shrink: 0; }
.param-key::after { content: ':'; margin-right: 1px; }
.param-val { font-size: 12px; color: var(--wf-node-text); font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.param-input { width: 100%; padding: 2px 6px; border: 1px solid var(--color-accent); border-radius: 3px; background: var(--wf-node-input-bg); color: var(--wf-node-text); font-size: var(--font-xs); font-family: monospace; outline: none; }

/* ── Port rows ── */
.node-ports { padding: 4px 0; }
.port-row {
  display: flex; justify-content: space-between; align-items: center;
  height: 26px; padding: 0 12px; position: relative;
}
.port-label { font-size: var(--font-xs); color: var(--color-text-tertiary); user-select: none; white-space: nowrap; }
.right-label { text-align: right; }

/* ── Port dots on card edges ── */
.port-dot {
  width: 10px !important; height: 10px !important;
  background: var(--wf-accent) !important;
  border: 2px solid var(--wf-port-ring) !important;
  border-radius: 50% !important;
  position: absolute !important;
  top: 50% !important;
  transform: translateY(-50%) !important;
  z-index: 20;
}
.port-dot:hover { transform: scale(1.6); background: var(--wf-accent-hover) !important; }
.port-dot-left { left: -6px; }
.port-dot-right { right: -6px; }

/* ── Error handling (selected only) ── */
.node-error-section {
  padding: 4px 8px; border-top: 1px solid var(--wf-node-divider);
  background: var(--wf-node-input-bg);
}
.error-row { display: flex; align-items: center; gap: 4px; }
.error-select {
  flex: 1; padding: 2px 4px; font-size: var(--font-xs);
  background: var(--color-bg-input); border: 1px solid var(--color-border);
  border-radius: 3px; color: var(--color-text-secondary); outline: none; cursor: pointer;
}
.error-select:focus { border-color: var(--color-accent); }
.retry-input {
  width: 44px; padding: 2px 4px; font-size: var(--font-xs);
  background: var(--color-bg-input); border: 1px solid var(--color-border);
  border-radius: 3px; color: var(--color-text-primary); outline: none; text-align: center;
}
.retry-input:focus { border-color: var(--color-accent); }

/* ── Status ── */
.running-indicator { position: absolute; bottom: 4px; right: 4px; width: 8px; height: 8px; background: var(--wf-warn); border-radius: 50%; }
.success-check { position: absolute; bottom: 2px; right: 6px; color: var(--wf-success); font-size: 14px; font-weight: bold; }
.failed-mark { position: absolute; bottom: 2px; left: 6px; color: var(--wf-danger); font-size: var(--font-xs); max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
