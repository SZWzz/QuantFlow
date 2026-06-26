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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type VFNode = Node<any, any, any>
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type VFEdge = Edge<any, any, any>

export const useWorkflowStore = defineStore('workflow', () => {
  const nodes = ref<VFNode[]>([])
  const edges = ref<Edge[]>([])
  const viewport = ref<ViewportTransform>({ x: 0, y: 0, zoom: 1 })
  const executionStatus = ref<ExecutionStatus>('idle')
  const nodeStatuses = ref<Map<string, NodeExecStatus>>(new Map())
  const runId = ref<string | null>(null)
  const selectedNodeId = ref<string | null>(null)

  // Undo/redo history
  const history = ref<{ nodes: VFNode[]; edges: Edge[] }[]>([])
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

  function addNode(type: string, position: { x: number; y: number }, params?: Record<string, any>, portOverrides?: { inputs: string[]; outputs: string[] }) {
    pushHistory()
    const id = `${type}-${Date.now()}`
    const pmap: Record<string, { inputs: string[]; outputs: string[] }> = {
      data_loader: { inputs: [], outputs: ['ohlcv'] },
      sma: { inputs: ['input'], outputs: ['output'] },
      cross_signal: { inputs: ['fast', 'slow'], outputs: ['signal'] },
      log_output: { inputs: ['input'], outputs: ['output'] },
      loop: { inputs: ['items'], outputs: ['batched'] },
    }
    const ports = portOverrides || pmap[type] || { inputs: ['input'], outputs: ['output'] }

    nodes.value = [...nodes.value as VFNode[], {
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
    } as VFNode]
    return id
  }

  function removeNode(id: string) {
    pushHistory()
    /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
    nodes.value = (nodes.value as any).filter((n: any) => n.id !== id)
    edges.value = (edges.value as any).filter(
      (e: any) => e.source !== id && e.target !== id
    )
    if (selectedNodeId.value === id) {
      selectedNodeId.value = null
    }
  }

  function addEdge(edge: VFEdge) {
    pushHistory()
    ;(edges.value as VFEdge[]).push(edge)
  }

  function removeEdge(id: string) {
    pushHistory()
    const list = edges.value as VFEdge[]
    edges.value = list.filter((e) => e.id !== id) as VFEdge[]
  }

  function selectNode(id: string | null) {
    selectedNodeId.value = id
  }

  function resetExecution() {
    executionStatus.value = 'idle'
    nodeStatuses.value = new Map()
    runId.value = null
    // Reset node status visuals
    for (const node of nodes.value as VFNode[]) {
      node.data.status = 'idle'
      node.data.error = undefined
    }
  }

  // Serialize to Phase 1 JSON format
  function toWorkflowJSON(name = 'Canvas Workflow'): WorkflowJSON {
    return {
      id: `canvas-${Date.now()}`,
      name,
      nodes: (nodes.value as VFNode[]).map((n) => ({
        id: n.id,
        node_type: n.data.nodeType,
        params: n.data.params || {},
      })),
      edges: (edges.value as VFEdge[]).map((e) => ({
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
    nodes.value = [] as VFNode[]
    edges.value = [] as VFEdge[]
    resetExecution()

    // Build ID mapping: oldID → newID
    const nodeIdMap = new Map<string, string>()
    for (const n of wf.nodes) {
      const newId = `${n.node_type}-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
      nodeIdMap.set(n.id, newId)
      const portMap: Record<string, { inputs: string[]; outputs: string[] }> = {
        data_loader: { inputs: [], outputs: ['ohlcv'] },
        sma: { inputs: ['input'], outputs: ['output'] },
        cross_signal: { inputs: ['fast', 'slow'], outputs: ['signal'] },
        log_output: { inputs: ['input'], outputs: ['output'] },
        loop: { inputs: ['items'], outputs: ['batched'] },
      }
      const ports = portMap[n.node_type] || { inputs: ['input'], outputs: ['output'] }
      ;(nodes.value as VFNode[]).push({
        id: newId,
        type: 'custom',
        position: { x: 100 + Math.random() * 300, y: 100 + Math.random() * 200 },
        data: {
          nodeType: n.node_type,
          label: n.node_type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase()),
          params: n.params || {},
          inputs: ports.inputs,
          outputs: ports.outputs,
          status: 'idle',
        },
      } as VFNode)
    }

    // Create edges using ID map
    for (const e of wf.edges) {
      const sourceId = nodeIdMap.get(e.from_node)
      const targetId = nodeIdMap.get(e.to_node)
      if (sourceId && targetId && sourceId !== targetId) {
        ;(edges.value as VFEdge[]).push({
          id: `e-${sourceId}-${targetId}`,
          source: sourceId,
          target: targetId,
          sourceHandle: e.from_port,
          targetHandle: e.to_port,
          type: 'smoothstep',
          style: { stroke: '#30363d', strokeWidth: 2 },
        } as VFEdge)
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
