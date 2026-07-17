<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useWorkflowStore } from '@/stores/workflow'
import { useTerminalStore } from '@/stores/terminal'
import WorkflowCanvas from './canvas/WorkflowCanvas.vue'
import NodePalette from './NodePalette.vue'
import ExecutionLog from './ExecutionLog.vue'
import WorkflowList from './WorkflowList.vue'
import { getIcon } from '@/lib/icons'
import { logger } from '@/lib/logger'

import { onMounted } from 'vue'

const session = useSessionStore()
const workflow = useWorkflowStore()
const terminal = useTerminalStore()
const router = useRouter()

onMounted(() => { workflow.fetchNodeMeta() })

const showLog = ref(false)
const showWorkflowList = ref(false)

async function onRun() {
  showLog.value = true
  workflow.startAsyncRun()
}

watch(() => workflow.executionStatus, (val) => {
  if (val === 'completed' || val === 'failed') animateExecution()
})

async function animateExecution() {
  const statuses = workflow.nodeStatuses
  if (statuses.size === 0) return

  const completedNodes = workflow.nodes.filter(n =>
    statuses.get(n.id)?.status === 'success'
  )

  // Reset all edges to default
  for (const edge of workflow.edges as any[]) {
    edge.style = { stroke: 'var(--wf-edge)', strokeWidth: 2 }
    edge.animated = false
  }

  for (const node of completedNodes) {
    node.data.status = 'running'
    await delay(60)
    node.data.status = 'success'
    for (const edge of workflow.edges as any[]) {
      if (edge.source === node.id) {
        edge.style = { stroke: 'var(--wf-success)', strokeWidth: 2.5 }
        edge.animated = true
      }
    }
  }
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

function onSave() {
  const wfJSON = workflow.toWorkflowJSON()
  localStorage.setItem('quantflow-current-workflow', JSON.stringify(wfJSON))
  logger.info('Workflow saved')
}

function onLoad() {
  const saved = localStorage.getItem('quantflow-current-workflow')
  if (!saved) return
  try {
    const wf = JSON.parse(saved)
    workflow.fromWorkflowJSON(wf)
  } catch (e) {
    logger.error('Failed to load workflow:', e)
  }
}

function pinToTerminal() {
  if (!workflow.selectedNodeId) return
  const node = workflow.nodes.find((n) => n.id === workflow.selectedNodeId)
  if (!node) return

  // Map node type → panel type
  const panelMap: Record<string, string> = {
    data_loader: 'candlestick',
    sma: 'candlestick',
    cross_signal: 'order-entry',
    log_output: 'system-monitor',
    loop: 'watchlist',
    backtest: 'backtest',
  }
  const panelId = panelMap[node.data.nodeType] || 'system-monitor'
  const instanceId = terminal.openPanel(panelId, {
    symbol: node.data.params?.symbol || '600519',
  })
  // Also add to push pins
  terminal.pushPins.push({
    id: `wf-${node.id}`,
    label: `${node.data.nodeType} output`,
    type: 'workflow',
    payload: { nodeId: node.id, panelId, instanceId },
  })
}

function pinOutputToTerminal() {
  if (!workflow.selectedNodeId) return
  const node = workflow.nodes.find((n) => n.id === workflow.selectedNodeId)
  if (!node) return

  terminal.openPanel('candlestick', { symbol: node.data.params?.symbol || '600519' })
}

// Keyboard shortcuts
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'F5') {
    event.preventDefault()
    onRun()
  }
  if ((event.ctrlKey || event.metaKey) && event.key === 'z') {
    event.preventDefault()
    if (event.shiftKey) {
      workflow.redo()
    } else {
      workflow.undo()
    }
  }
}
</script>

<template>
  <div class="workflow-mode" @keydown="onKeydown" tabindex="0">
    <header class="wf-header">
      <div class="wf-left">
        <div class="logo">
          <span class="logo-icon" v-html="getIcon('workflow')" />
        </div>
        <span class="wf-title">{{ $t('workflow.workflow_mode') }}</span>
      </div>
      <div class="wf-actions">
        <button class="wf-btn btn-run" @click="onRun" :disabled="workflow.executionStatus === 'running'">
          <span class="btn-icon" v-html="getIcon('execution')" />
          {{ $t('workflow.run') }} <kbd class="key-hint">F5</kbd>
        </button>
        <button class="wf-btn btn-secondary" @click="showWorkflowList = true">
          <span class="btn-icon" v-html="getIcon('save')" />
          {{ $t('workflow.my_workflows') }}
        </button>
        <button class="wf-btn btn-secondary" @click="showLog = !showLog">
          <span class="btn-icon" v-html="getIcon('terminal')" />
          {{ showLog ? $t('workflow.hide_log') : $t('workflow.show_log') }}
        </button>
        <button class="wf-btn mode-switch" @click="router.push('/')">
          <span class="mode-icon" v-html="getIcon('terminal')" />
          Terminal
        </button>
      </div>
    </header>

    <div class="wf-main">
      <NodePalette />
      <div class="canvas-area">
        <WorkflowCanvas />
      </div>
    </div>

    <ExecutionLog v-if="showLog" />
    <WorkflowList v-if="showWorkflowList" @close="showWorkflowList = false" />
  </div>
</template>

<style scoped>
.workflow-mode {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-app);
  color: var(--color-text-primary);
  outline: none;
}

.wf-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 var(--space-lg);
  background: var(--gradient-header);
  border-bottom: 1px solid var(--color-border);
  min-height: 42px;
  position: relative;
  z-index: 10;
}

.wf-header::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent 0%, var(--color-accent) 50%, transparent 100%);
  opacity: 0.3;
}

.wf-left {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: var(--gradient-accent);
  border: 1px solid var(--color-border-glow);
  border-radius: var(--radius-md);
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--color-accent);
}

.logo-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.wf-title {
  font-weight: 700;
  font-size: var(--font-base);
  color: var(--color-text-primary);
  letter-spacing: 0.5px;
}

.wf-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.wf-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-subtle);
  color: var(--color-text-primary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: inherit;
  transition: all var(--transition-fast);
  height: 30px;
  width: auto;
}

.wf-btn:hover:not(:disabled) {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-soft);
}

.wf-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: transparent;
  color: var(--color-text-secondary);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
  box-shadow: none;
}

.btn-run {
  background: var(--color-accent-soft);
  border-color: var(--color-accent);
  color: var(--color-accent);
  font-weight: 600;
}

.btn-run:hover:not(:disabled) {
  background: var(--color-accent);
  color: var(--color-text-inverse);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}

.btn-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.key-hint {
  font-size: 9px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  padding: 1px 5px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-family: 'JetBrains Mono', monospace;
  margin-left: 2px;
}

.mode-switch {
  border-color: var(--color-brand);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-weight: 600;
}

.mode-switch:hover {
  background: var(--color-brand);
  color: var(--color-text-inverse);
}

.mode-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}

.mode-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.mode-switch:hover .mode-icon {
  color: currentColor;
}

.wf-main {
  flex: 1;
  display: flex;
  min-height: 0;
}

.canvas-area {
  flex: 1;
  min-width: 0;
}
</style>
