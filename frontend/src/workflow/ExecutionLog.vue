<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useWorkflowStore, type NodeExecStatus } from '@/stores/workflow'

const workflow = useWorkflowStore()
const logContainer = ref<HTMLElement | null>(null)

const entries = computed(() => {
  const result: { id: string; type: string; status: string; duration?: number; error?: string }[] = []

  // Group by status
  for (const [nodeId, status] of workflow.nodeStatuses) {
    result.push({
      id: nodeId,
      type: 'node',
      status: status.status,
      duration: status.duration,
      error: status.error,
    })
  }

  return result
})

function clearLog() {
  workflow.resetExecution()
}

// Auto-scroll to bottom when new entries appear
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
    case 'success': return '#3fb950'
    case 'failed': return '#f85149'
    case 'running': return '#f0883e'
    case 'skipped': return '#5a6380'
  }
  return '#5a6380'
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

      <div v-for="entry in entries" :key="entry.id" class="log-entry">
        <span class="entry-icon" :style="{ color: statusColor(entry.status) }">
          {{ statusIcon(entry.status) }}
        </span>
        <span class="entry-node">{{ entry.id }}</span>
        <span class="entry-status" :style="{ color: statusColor(entry.status) }">{{ entry.status }}</span>
        <span v-if="entry.duration" class="entry-time">{{ (entry.duration / 1000).toFixed(2) }}µs</span>
        <span v-if="entry.error" class="entry-error">{{ entry.error }}</span>
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
  max-height: 180px;
  flex-shrink: 0;
}

.log-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; background: var(--color-bg-input); border-bottom: 1px solid var(--color-border);
}

.log-title { font-size: 11px; font-weight: 600; color: var(--color-text-primary); text-transform: uppercase; letter-spacing: 0.5px; }
.log-actions { display: flex; align-items: center; gap: 8px; }

.running-badge { font-size: 10px; color: #f0883e; animation: pulse 1s ease-in-out infinite; }
.done-badge { font-size: 10px; color: #3fb950; }
.fail-badge { font-size: 10px; color: #f85149; }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.clear-btn {
  padding: 2px 8px; background: transparent; border: 1px solid var(--color-border);
  color: var(--color-text-tertiary); border-radius: var(--radius-sm); font-size: 10px; cursor: pointer;
}
.clear-btn:hover { color: var(--color-text-primary); border-color: var(--color-border-strong); }

.log-content { flex: 1; overflow-y: auto; padding: 6px; font-family: monospace; }

.empty-log { padding: 16px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; font-family: system-ui; }

.log-entry { display: flex; align-items: center; gap: 6px; padding: 2px 4px; font-size: 11px; }
.entry-icon { font-size: 10px; width: 12px; text-align: center; }
.entry-node { color: var(--color-text-primary); min-width: 80px; }
.entry-status { font-weight: 500; }
.entry-time { color: var(--color-text-tertiary); margin-left: auto; }
.entry-error { color: #f85149; font-size: 10px; }
</style>
