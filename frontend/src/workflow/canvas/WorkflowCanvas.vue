<script setup lang="ts">
import { ref, watch, markRaw } from 'vue'
import { VueFlow, useVueFlow, type Node, type Edge, type Connection } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import { useWorkflowStore } from '@/stores/workflow'
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

function onDrop(event: DragEvent) {
  event.preventDefault()
  const nodeType = event.dataTransfer?.getData('application/node-type')
  if (!nodeType) return

  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const position = {
    x: event.clientX - bounds.left - 75,
    y: event.clientY - bounds.top - 20,
  }

  const nodeId = workflow.addNode(nodeType, position)
  workflow.selectNode(nodeId)
}

// Keyboard shortcuts
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Delete' && workflow.selectedNodeId) {
    workflow.removeNode(workflow.selectedNodeId)
  }
  if ((event.ctrlKey || event.metaKey) && event.key === 'a') {
    // Select all — skip
  }
}
</script>

<template>
  <div class="workflow-canvas" @dragover="onDragOver" @drop="onDrop" @keydown="onKeydown" tabindex="0">
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
  border-radius: 3px;
  font-size: 12px;
  font-family: monospace;
  color: var(--color-text-tertiary);
}
</style>
