<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'

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

const category = computed(() => {
  const cat = props.data.nodeType
  if (cat.includes('data')) return 'data'
  if (cat.includes('sma') || cat.includes('indicator')) return 'indicator'
  if (cat.includes('signal') || cat.includes('cross')) return 'signal'
  if (cat.includes('log') || cat.includes('output')) return 'output'
  if (cat.includes('loop')) return 'control'
  return 'default'
})

const categoryColor = computed(() => {
  const colors: Record<string, string> = {
    data: '#58a6ff',
    indicator: '#3fb950',
    signal: '#f0883e',
    output: '#a371f7',
    control: '#e94560',
    default: '#5a6380',
  }
  return colors[category.value] || colors.default
})

const statusClass = computed(() => `status-${props.data.status || 'idle'}`)

const paramSummary = computed(() => {
  const params = props.data.params || {}
  const keys = Object.keys(params)
  if (keys.length === 0) return ''
  return keys.map(k => `${k}=${params[k]}`).join(', ')
})
</script>

<template>
  <div class="custom-node" :class="[statusClass, { selected }]">
    <div class="node-header" :style="{ background: categoryColor }">
      <span class="node-type">{{ data.label || data.nodeType }}</span>
    </div>

    <div class="node-body">
      <div v-if="paramSummary" class="node-params">
        {{ paramSummary }}
      </div>

      <!-- Input handles (left side) -->
      <div class="handles inputs">
        <div
          v-for="port in (data.inputs || ['input'])"
          :key="port"
          class="handle-row"
        >
          <Handle :type="'target'" :position="Position.Left" :id="port" class="port-handle" />
          <span class="port-label">{{ port }}</span>
        </div>
      </div>

      <!-- Output handles (right side) -->
      <div class="handles outputs">
        <div
          v-for="port in (data.outputs || ['output'])"
          :key="port"
          class="handle-row output-row"
        >
          <span class="port-label">{{ port }}</span>
          <Handle :type="'source'" :position="Position.Right" :id="port" class="port-handle" />
        </div>
      </div>
    </div>

    <!-- Status indicator -->
    <div v-if="data.status === 'running'" class="running-indicator" />
    <div v-if="data.status === 'success'" class="success-check">✓</div>
    <div v-if="data.status === 'failed'" class="failed-mark">✗ {{ data.error }}</div>
  </div>
</template>

<style scoped>
.custom-node {
  background: #1c2333;
  border: 2px solid #30363d;
  border-radius: 8px;
  min-width: 150px;
  max-width: 220px;
  font-size: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.custom-node.selected {
  border-color: #58a6ff;
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.3);
}

.custom-node.status-running {
  border-color: #f0883e;
  animation: pulse 1.5s ease-in-out infinite;
}

.custom-node.status-success {
  border-color: #3fb950;
}

.custom-node.status-failed {
  border-color: #f85149;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.node-header {
  padding: 6px 12px;
  border-radius: 6px 6px 0 0;
  color: #fff;
  font-weight: 600;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-body {
  padding: 8px 0;
  position: relative;
}

.node-params {
  padding: 2px 12px;
  font-size: 10px;
  color: #5a6380;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.handles {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.handle-row {
  display: flex;
  align-items: center;
  padding: 2px 0;
}

.handle-row.output-row {
  justify-content: flex-end;
}

.port-label {
  font-size: 10px;
  color: #5a6380;
  margin: 0 8px;
}

.port-handle {
  width: 10px !important;
  height: 10px !important;
  background: #30363d !important;
  border: 2px solid #58a6ff !important;
  border-radius: 50% !important;
}

.running-indicator {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 8px;
  height: 8px;
  background: #f0883e;
  border-radius: 50%;
}

.success-check {
  position: absolute;
  bottom: 2px;
  right: 6px;
  color: #3fb950;
  font-size: 14px;
  font-weight: bold;
}

.failed-mark {
  position: absolute;
  bottom: 2px;
  right: 6px;
  color: #f85149;
  font-size: 10px;
}
</style>
