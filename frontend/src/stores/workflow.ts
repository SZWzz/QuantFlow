import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Node, Edge, ViewportTransform } from '@vue-flow/core'

export type ExecutionStatus = 'idle' | 'running' | 'completed' | 'failed'

export interface NodeExecStatus {
  nodeId: string
  status: 'success' | 'failed' | 'skipped' | 'running'
  duration?: number
  error?: string
}

export interface WorkflowJSON {
  id: string
  name: string
  description?: string
  nodes: { id: string; node_type: string; params: Record<string, any> }[]
  edges: { from_node: string; from_port: string; to_node: string; to_port: string }[]
}

export const useWorkflowStore = defineStore('workflow', () => {
  const nodes = ref<Node[]>([])
  const edges = ref<Edge[]>([])
  const viewport = ref<ViewportTransform>({ x: 0, y: 0, zoom: 1 })
  const executionStatus = ref<ExecutionStatus>('idle')
  const nodeStatuses = ref<Map<string, NodeExecStatus>>(new Map())
  const runId = ref<string | null>(null)
  const selectedNodeId = ref<string | null>(null)

  // Undo/redo history
  const history = ref<{ nodes: Node[]; edges: Edge[] }[]>([])
  const historyIndex = ref(-1)

  function pushHistory() {
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push({
      nodes: JSON.parse(JSON.stringify(nodes.value)),
      edges: JSON.parse(JSON.stringify(edges.value)),
    })
    historyIndex.value = history.value.length - 1
    // Keep max 50 history entries
    if (history.value.length > 50) {
      history.value.shift()
      historyIndex.value--
    }
  }

  function undo() {
    if (historyIndex.value > 0) {
      historyIndex.value--
      const entry = history.value[historyIndex.value]
      nodes.value = JSON.parse(JSON.stringify(entry.nodes))
      edges.value = JSON.parse(JSON.stringify(entry.edges))
    }
  }

  function redo() {
    if (historyIndex.value < history.value.length - 1) {
      historyIndex.value++
      const entry = history.value[historyIndex.value]
      nodes.value = JSON.parse(JSON.stringify(entry.nodes))
      edges.value = JSON.parse(JSON.stringify(entry.edges))
    }
  }

  function addNode(type: string, position: { x: number; y: number }, params?: Record<string, any>) {
    pushHistory()
    const id = `${type}-${Date.now()}`
    const portMap: Record<string, { inputs: string[]; outputs: string[] }> = {
      data_loader: { inputs: [], outputs: ['ohlcv'] },
      sma: { inputs: ['input'], outputs: ['output'] },
      cross_signal: { inputs: ['fast', 'slow'], outputs: ['signal'] },
      log_output: { inputs: ['input'], outputs: ['output'] },
      loop: { inputs: ['items'], outputs: ['batched'] },
    }
    const ports = portMap[type] || { inputs: ['input'], outputs: ['output'] }

    nodes.value.push({
      id,
      type: 'custom',
      position,
      data: {
        nodeType: type,
        label: type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase()),
        params: params || {},
        inputs: ports.inputs,
        outputs: ports.outputs,
        status: 'idle',
      },
    })
    return id
  }

  function removeNode(id: string) {
    pushHistory()
    nodes.value = nodes.value.filter((n) => n.id !== id)
    edges.value = edges.value.filter(
      (e) => e.source !== id && e.target !== id
    )
    if (selectedNodeId.value === id) {
      selectedNodeId.value = null
    }
  }

  function addEdge(edge: Edge) {
    pushHistory()
    edges.value.push(edge)
  }

  function removeEdge(id: string) {
    pushHistory()
    edges.value = edges.value.filter((e) => e.id !== id)
  }

  function selectNode(id: string | null) {
    selectedNodeId.value = id
  }

  function resetExecution() {
    executionStatus.value = 'idle'
    nodeStatuses.value = new Map()
    runId.value = null
    // Reset node status visuals
    for (const node of nodes.value) {
      node.data.status = 'idle'
      node.data.error = undefined
    }
  }

  // Serialize to Phase 1 JSON format
  function toWorkflowJSON(name = 'Canvas Workflow'): WorkflowJSON {
    return {
      id: `canvas-${Date.now()}`,
      name,
      nodes: nodes.value.map((n) => ({
        id: n.id,
        node_type: n.data.nodeType,
        params: n.data.params || {},
      })),
      edges: edges.value.map((e) => ({
        from_node: e.source,
        from_port: e.sourceHandle || 'output',
        to_node: e.target,
        to_port: e.targetHandle || 'input',
      })),
    }
  }

  // Load from Phase 1 JSON format
  function fromWorkflowJSON(wf: WorkflowJSON) {
    pushHistory()
    nodes.value = []
    edges.value = []
    resetExecution()

    // Create nodes
    for (const n of wf.nodes) {
      addNode(n.node_type, {
        x: 100 + Math.random() * 300,
        y: 100 + Math.random() * 200,
      }, n.params)
    }

    // Create edges (match by index since IDs might change)
    const createdNodes = nodes.value
    for (const e of wf.edges) {
      const sourceNode = createdNodes.find((n) => n.data.nodeType ===
        wf.nodes.find((wn) => wn.id === e.from_node)?.node_type &&
        !edges.value.some((edge) => edge.source === n.id && edge.target ===
          createdNodes.find((n2) => n2.data.nodeType ===
            wf.nodes.find((wn) => wn.id === e.to_node)?.node_type)?.id)
      )
      const targetNode = createdNodes.find((n) => n.data.nodeType ===
        wf.nodes.find((wn) => wn.id === e.to_node)?.node_type &&
        !edges.value.some((edge) => edge.target === n.id)
      )
      if (sourceNode && targetNode && sourceNode.id !== targetNode.id) {
        edges.value.push({
          id: `e-${sourceNode.id}-${targetNode.id}`,
          source: sourceNode.id,
          target: targetNode.id,
          sourceHandle: e.from_port,
          targetHandle: e.to_port,
          type: 'smoothstep',
          style: { stroke: '#30363d', strokeWidth: 2 },
        })
      }
    }
  }

  return {
    nodes,
    edges,
    viewport,
    executionStatus,
    nodeStatuses,
    runId,
    selectedNodeId,
    history,
    historyIndex,
    addNode,
    removeNode,
    addEdge,
    removeEdge,
    selectNode,
    resetExecution,
    toWorkflowJSON,
    fromWorkflowJSON,
    undo,
    redo,
  }
})
