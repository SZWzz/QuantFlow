<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useWorkflowStore } from '@/stores/workflow'
import { RunWorkflow } from '@/lib/wails'
import { useTerminalStore } from '@/stores/terminal'
import WorkflowCanvas from './canvas/WorkflowCanvas.vue'
import NodePalette from './NodePalette.vue'
import PropertyPanel from './PropertyPanel.vue'
import ExecutionLog from './ExecutionLog.vue'

const session = useSessionStore()
const workflow = useWorkflowStore()
const terminal = useTerminalStore()
const router = useRouter()

const showLog = ref(false)

async function onRun() {
  workflow.resetExecution()
  workflow.executionStatus = 'running'

  const wfJSON = workflow.toWorkflowJSON()

  try {
    const result = await RunWorkflow(JSON.stringify(wfJSON))
    workflow.executionStatus = result.status

    // Update node statuses
    if (result.node_results) {
      for (const nr of result.node_results) {
        workflow.nodeStatuses.set(nr.node_id, {
          nodeId: nr.node_id,
          status: nr.status,
          duration: nr.duration,
          error: nr.error,
        })

        // Update node data in canvas
        const node = workflow.nodes.find((n) => n.id === nr.node_id)
        if (node) {
          node.data.status = nr.status
          if (nr.error) node.data.error = nr.error
        }
      }
    }

    showLog.value = true
  } catch (err: any) {
    workflow.executionStatus = 'failed'
    console.error('Workflow execution failed:', err)
  }
}

function onSave() {
  const wfJSON = workflow.toWorkflowJSON()
  localStorage.setItem('quantflow-current-workflow', JSON.stringify(wfJSON))
  console.log('Workflow saved')
}

function onLoad() {
  const saved = localStorage.getItem('quantflow-current-workflow')
  if (!saved) return
  try {
    const wf = JSON.parse(saved)
    workflow.fromWorkflowJSON(wf)
  } catch (e) {
    console.error('Failed to load workflow:', e)
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
  }
  const panelId = panelMap[node.data.nodeType] || 'system-monitor'
  const instanceId = terminal.openPanel(panelId, {
    symbol: node.data.params?.symbol || 'AAPL',
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

  terminal.openPanel('candlestick', { symbol: 'AAPL' })
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
      <span class="wf-logo">QuantFlow — Workflow Mode</span>
      <div class="wf-actions">
        <button class="wf-btn" @click="onRun" :disabled="workflow.executionStatus === 'running'">
          ▶ Run (F5)
        </button>
        <button class="wf-btn secondary" @click="onSave">Save</button>
        <button class="wf-btn secondary" @click="onLoad">Load</button>
        <button class="wf-btn secondary" @click="showLog = !showLog">
          {{ showLog ? 'Hide Log' : 'Show Log' }}
        </button>
        <button class="wf-btn mode-switch" @click="router.push('/')">
          Terminal
        </button>
      </div>
    </header>

    <div class="wf-main">
      <NodePalette />
      <div class="canvas-area">
        <WorkflowCanvas />
      </div>
      <PropertyPanel />
    </div>

    <ExecutionLog v-if="showLog" />
  </div>
</template>

<style scoped>
.workflow-mode {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #0d1117;
  color: #c9d1d9;
  outline: none;
}

.wf-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 12px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  min-height: 36px;
}

.wf-logo {
  font-weight: bold;
  font-size: 13px;
  color: #58a6ff;
  letter-spacing: 0.5px;
}

.wf-actions {
  display: flex;
  gap: 6px;
}

.wf-btn {
  padding: 4px 12px;
  border: 1px solid #30363d;
  background: #21262d;
  color: #c9d1d9;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  transition: all 0.15s;
}

.wf-btn:hover:not(:disabled) {
  border-color: #58a6ff;
  color: #58a6ff;
}

.wf-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.wf-btn.secondary {
  background: transparent;
  color: #5a6380;
}

.wf-btn.mode-switch {
  border-color: #3fb950;
  color: #3fb950;
  background: transparent;
}

.wf-btn.mode-switch:hover {
  background: rgba(63, 185, 80, 0.1);
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
