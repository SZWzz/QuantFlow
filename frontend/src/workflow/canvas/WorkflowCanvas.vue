<script setup lang="ts">
import { ref, watch, markRaw, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueFlow, useVueFlow, type Node, type Edge, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useWorkflowStore } from '@/stores/workflow'
import { GetNodePorts } from '@/lib/wails'
import { useCanvasShortcuts } from '@/lib/useCanvasShortcuts'
import { cssVar } from '@/lib/cssVar'
import CustomNode from './CustomNode.vue'
import StickyNote from './StickyNote.vue'
import ContextMenu from '@/workflow/components/ContextMenu.vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

const { t } = useI18n()
const workflow = useWorkflowStore()
useCanvasShortcuts()

const ctxMenu = ref<{ x: number; y: number; items: any[] } | null>(null)

function onNodeContextMenu(e: MouseEvent, node: any) {
  ctxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    items: [
      { label: t('workflow.pin_output'), icon: '📌', action: () => workflow.togglePin(node.id) },
      {
        label: t('workflow.disable_node'), icon: '⏸', action: () => workflow.toggleDisable(node.id),
        disabled: node.data.mode === 2,
      },
      { label: t('workflow.group_nodes'), icon: '📦', action: () => workflow.groupNodes([node.id]) },
      { label: '', divider: true },
      {
        label: t('workflow.copy_nodes'), icon: '📋', shortcut: 'Ctrl+C',
        action: () => workflow.copyNodes([node.id]),
      },
      {
        label: t('workflow.clone_node'), icon: '📄',
        action: () => workflow.cloneNode(node.id),
      },
      {
        label: t('workflow.delete_node'), icon: '🗑', shortcut: 'Del',
        action: () => workflow.removeNode(node.id),
      },
    ],
  }
}

function onPaneContextMenu(e: MouseEvent) {
  ctxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    items: [
      {
        label: t('workflow.paste'), icon: '📋', shortcut: 'Ctrl+V',
        disabled: !workflow.clipboard.length,
        action: () => workflow.pasteNodes(),
      },
      {
        label: t('workflow.select_all'), icon: '🔲', shortcut: 'Ctrl+A',
        action: () => workflow.selectAllNodes(),
      },
      { label: '', divider: true },
      {
        label: t('workflow.add_node'), icon: '➕',
        action: () => { /* open palette — handled externally */ },
      },
    ],
  }
}

function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  const nodeEl = (e.target as HTMLElement).closest('.vue-flow__node')
  if (nodeEl) {
    const nodeId = nodeEl.getAttribute('data-id')
    const node = workflow.nodes.find((n: any) => n.id === nodeId)
    if (node) onNodeContextMenu(e, node)
  } else {
    onPaneContextMenu(e)
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const nodeTypes: Record<string, any> = {
  custom: markRaw(CustomNode),
}

const { onNodeClick, onConnect, onPaneClick, addNodes, addEdges, removeNodes, removeEdges, fitView } = useVueFlow()

// Sync store → vue-flow
const nodes = ref<Node[]>([])
const edges = ref<Edge[]>([])

// Watch store and update local refs
watch(
  () => [workflow.nodes, workflow.edges],
  () => {
    nodes.value = [...workflow.nodes]
    edges.value = [...workflow.edges]
  },
  { deep: true, immediate: true }
)

// Handle node click
onNodeClick(({ node }) => {
  workflow.selectNode(node.id)
})

// Handle double-click on sub_workflow node — navigate into child
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function onNodeDoubleClick(ev: any) {
  const node = ev?.node
  if (node?.data?.nodeType === 'sub_workflow') {
    const childWfId = node.data?.params?.workflow_id
    if (childWfId) workflow.navigateIntoSubWF(node.id, childWfId)
  }
}

// Handle pane click (deselect)
onPaneClick(() => {
  workflow.selectNode(null)
})

// Edge styles for execution flow animation
const edgeStyles = computed(() => {
  const s: Record<string, any> = {}
  const status = workflow.executionStatus
  for (const edge of workflow.edges) {
    const eid = edge.id || ''
    if (status === 'running') {
      const tgt = (workflow.nodeStatuses as any).get?.(edge.target)
      if (tgt?.status === 'running' || tgt?.status === 'success') {
        s[eid] = { stroke: 'var(--wf-accent)', strokeDasharray: '5 3', strokeWidth: 2, animation: 'edge-flow 0.5s linear infinite' }
        continue
      }
    }
    const tgtDone = (workflow.nodeStatuses as any).get?.(edge.target)
    if (tgtDone?.status === 'success') {
      s[eid] = { stroke: 'var(--wf-success)', strokeWidth: 2 }
    } else if (tgtDone?.status === 'failed') {
      s[eid] = { stroke: 'var(--wf-danger)', strokeWidth: 2 }
    }
  }
  return s
})

// Canvas colors that must be concrete values (SVG attributes, not CSS) —
// read once from theme tokens; correct after reload on theme switch.
const canvasPattern = cssVar('--wf-canvas-pattern', '#d6dce3')
const minimapMask = cssVar('--wf-minimap-mask', 'rgba(255, 255, 255, 0.7)')

// Handle new connection
onConnect((connection: Connection) => {
  if (connection.source === connection.target) return
  const existing = edges.value.some(
    e => e.source === connection.source && e.target === connection.target
  )
  if (existing) return

  // Port type validation — block incompatible connections
  const srcNode = workflow.nodes.find((n: any) => n.id === connection.source)
  const tgtNode = workflow.nodes.find((n: any) => n.id === connection.target)
  if (srcNode && tgtNode) {
    const srcType = workflow.getPortType(srcNode.data?.nodeType, connection.sourceHandle || 'output', 'output')
    const tgtType = workflow.getPortType(tgtNode.data?.nodeType, connection.targetHandle || 'input', 'input')
    if (srcType && tgtType && !workflow.canConnectPorts(srcType, tgtType)) return
  }

  const edge: Edge = {
    id: `e-${connection.source}-${connection.sourceHandle}-${connection.target}-${connection.targetHandle}`,
    source: connection.source,
    target: connection.target,
    sourceHandle: connection.sourceHandle,
    targetHandle: connection.targetHandle,
    type: 'default',
    animated: false,
    style: { stroke: 'var(--color-border)', strokeWidth: 2 },
  }
  workflow.addEdge(edge)
})

// Handle drag-and-drop from NodePalette
function onDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
}

async function onDrop(event: DragEvent) {
  event.preventDefault()
  const nodeType = event.dataTransfer?.getData('application/node-type')
  if (!nodeType) return

  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const position = {
    x: event.clientX - bounds.left - 75,
    y: event.clientY - bounds.top - 20,
  }

  // Fetch dynamic port definitions from backend
  let portOverrides: { inputs: string[]; outputs: string[] } | undefined
  try {
    const result = await GetNodePorts(nodeType)
    portOverrides = {
      inputs: result.inputs.map(p => p.name),
      outputs: result.outputs.map(p => p.name),
    }
  } catch {
    // Fallback to local port map
  }

  const nodeId = workflow.addNode(nodeType, position, undefined, portOverrides)
  workflow.selectNode(nodeId)
}

// Edge flow keyframe — injected once into document
;(function injectEdgeKeyframes() {
  if (document.getElementById('qf-edge-flow')) return
  const style = document.createElement('style')
  style.id = 'qf-edge-flow'
  style.textContent = '@keyframes edge-flow { 0% { stroke-dashoffset: 12; } 100% { stroke-dashoffset: 0; } }'
  document.head.appendChild(style)
})()
</script>

<template>
    <div class="workflow-canvas" @dragover="onDragOver" @drop="onDrop" @contextmenu="onContextMenu">
    <div v-if="workflow.currentSubWFId" class="subwf-breadcrumb">
      <button class="breadcrumb-back" @click="workflow.navigateUpFromSubWF()">← {{ t('workflow.back') }}</button>
      <span class="breadcrumb-path">{{ t('workflow.sub_workflow') }}: {{ workflow.currentSubWFId }}</span>
    </div>

    <VueFlow
      v-model:nodes="nodes"
      v-model:edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ x: 0, y: 0, zoom: 1 }"
      :snap-to-grid="true"
      :snap-grid="[20, 20]"
      :connection-line-style="{ stroke: 'var(--color-accent)', strokeWidth: 2 }"
      :edge-styles="edgeStyles"
      @node-double-click="onNodeDoubleClick"
      fit-view-on-init
    >
      <Background :gap="20" :size="1" :pattern-color="canvasPattern" />
      <Controls position="bottom-right" />
      <MiniMap
        position="bottom-left"
        :style="{ background: 'var(--wf-minimap-bg)' }"
        :mask-color="minimapMask"
      />
    </VueFlow>

    <ContextMenu v-if="ctxMenu" v-bind="ctxMenu" @close="ctxMenu = null" />

    <div v-if="nodes.length === 0" class="empty-hint">
      <p>{{ t('workflow.drag_hint') }}</p>
    </div>
  </div>
</template>

<style scoped>
.workflow-canvas {
  width: 100%;
  height: 100%;
  background: var(--wf-canvas-bg);
  position: relative;
  outline: none;
}

.empty-hint {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
  text-align: center;
}

.empty-hint p {
  font-size: 14px;
  color: var(--wf-canvas-hint);
}

.empty-hint kbd {
  padding: 2px 6px;
  background: var(--wf-node-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-family: monospace;
  color: var(--color-text-tertiary);
}

.subwf-breadcrumb {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 12px; background: var(--wf-panel-bg);
  border-bottom: 1px solid var(--color-border); font-size: 12px;
  position: absolute; top: 0; left: 0; right: 0; z-index: 100;
}
.breadcrumb-back {
  padding: 2px 8px; border: 1px solid var(--color-border);
  border-radius: 4px; background: none; color: var(--color-accent); cursor: pointer; font-size: 11px;
}
.breadcrumb-back:hover { background: rgba(var(--wf-accent-rgb), 0.1); }
.breadcrumb-path { color: var(--color-text-tertiary); font-size: 11px; }
</style>
