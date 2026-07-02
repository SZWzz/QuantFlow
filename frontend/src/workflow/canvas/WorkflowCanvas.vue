<script setup lang="ts">
import { ref, watch, markRaw, onMounted, onUnmounted } from 'vue'
import { VueFlow, useVueFlow, type Node, type Edge, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useWorkflowStore } from '@/stores/workflow'
import { GetNodePorts } from '@/lib/wails'
import CustomNode from './CustomNode.vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

const workflow = useWorkflowStore()

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

// Handle pane click (deselect)
onPaneClick(() => {
  workflow.selectNode(null)
})

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
    type: 'smoothstep',
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

// Keyboard shortcuts
function handleKeyboardShortcut(e: KeyboardEvent) {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
  if (e.key === 'Delete' && workflow.selectedNodeId) {
    workflow.removeNode(workflow.selectedNodeId)
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    // Select all — skip
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyboardShortcut)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyboardShortcut)
})
</script>

<template>
  <div class="workflow-canvas" @dragover="onDragOver" @drop="onDrop">
    <VueFlow
      v-model:nodes="nodes"
      v-model:edges="edges"
      :node-types="nodeTypes"
      :default-viewport="{ x: 0, y: 0, zoom: 1 }"
      :snap-to-grid="true"
      :snap-grid="[20, 20]"
      :connection-line-style="{ stroke: 'var(--color-accent)', strokeWidth: 2 }"
      fit-view-on-init
    >
      <Background :gap="20" :size="1" pattern-color="#1a2a3e" />
      <Controls position="bottom-right" />
      <MiniMap
        position="bottom-left"
        :style="{ background: 'var(--color-bg-input)' }"
        :mask-color="'rgba(0,0,0,0.5)'"
      />
    </VueFlow>

    <div v-if="nodes.length === 0" class="empty-hint">
      <p>Drag nodes from the palette or press <kbd>Ctrl+K</kbd></p>
    </div>
  </div>
</template>

<style scoped>
.workflow-canvas {
  width: 100%;
  height: 100%;
  background: var(--color-bg-input);
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
  color: #3a4a6c;
}

.empty-hint kbd {
  padding: 2px 6px;
  background: #1c2333;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-family: monospace;
  color: var(--color-text-tertiary);
}
</style>
