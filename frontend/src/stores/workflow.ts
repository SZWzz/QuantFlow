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

  // Node metadata cache — populated by fetchNodeMeta()
  interface PortInfo { name: string; type: string }
  interface NodeMetaInfo { category: string; inputs: PortInfo[]; outputs: PortInfo[]; params: any[] }
  const nodeMetaCache = ref<Map<string, NodeMetaInfo>>(new Map())

  async function fetchNodeMeta() {
    try {
      const app = (window as any).go?.main?.App
      if (!app?.ListNodes) return
      const list = await app.ListNodes()
      if (!Array.isArray(list)) return
      const m = new Map<string, NodeMetaInfo>()
      for (const n of list) {
        m.set(n.node_type, {
          category: n.category || 'utility',
          inputs: (n.input_ports || []).map((p: any) => ({ name: p.name, type: p.type || 'any' })),
          outputs: (n.output_ports || []).map((p: any) => ({ name: p.name, type: p.type || 'any' })),
          params: n.params || [],
        })
      }
      nodeMetaCache.value = m
    } catch { /* graceful */ }
  }

  function getNodePorts(type: string): { category: string; inputs: string[]; outputs: string[] } {
    const meta = nodeMetaCache.value.get(type)
    if (meta) return {
      category: meta.category,
      inputs: meta.inputs.map(p => p.name),
      outputs: meta.outputs.map(p => p.name),
    }
    return { category: 'utility', inputs: ['input'], outputs: ['output'] }
  }

  // Port compatibility check: returns true if output can connect to input
  function canConnectPorts(outputType: string, inputType: string): boolean {
    if (!outputType || !inputType) return true // unknown types allowed
    if (outputType === inputType) return true
    if (outputType === 'any' || inputType === 'any') return true
    // ohlcv is compatible with series
    if (outputType === 'ohlcv' && inputType === 'series') return true
    // signal is compatible with number
    if (outputType === 'signal' && inputType === 'number') return true
    return false
  }

  function getPortType(nodeType: string, portName: string, direction: 'input' | 'output'): string | null {
    const meta = nodeMetaCache.value.get(nodeType)
    if (!meta) return null
    const ports = direction === 'input' ? meta.inputs : meta.outputs
    return ports.find(p => p.name === portName)?.type || null
  }

  function addNode(type: string, position: { x: number; y: number }, params?: Record<string, any>) {
    pushHistory()
    const id = `${type}-${Date.now()}`
    const ports = getNodePorts(type)

    nodes.value = [...nodes.value as VFNode[], {
      id,
      type: 'custom',
      position,
      data: {
        nodeType: type,
        category: ports.category,
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
      const ports = getNodePorts(n.node_type)
      ;(nodes.value as VFNode[]).push({
        id: newId,
        type: 'custom',
        position: { x: 100 + Math.random() * 300, y: 100 + Math.random() * 200 },
        data: {
          nodeType: n.node_type,
          category: ports.category,
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

  // ── Multi-workflow management ──────────────────────────────────

  const WF_STORAGE_KEY = 'quantflow-workflows'

  interface SavedWorkflow {
    id: string
    name: string
    createdAt: string
    updatedAt: string
    nodeCount: number
    json: WorkflowJSON
  }

  const workflowList = ref<SavedWorkflow[]>(loadWorkflowList())

  function loadWorkflowList(): SavedWorkflow[] {
    try {
      const raw = localStorage.getItem(WF_STORAGE_KEY)
      return raw ? JSON.parse(raw) : []
    } catch { return [] }
  }

  function persistWorkflowList() {
    try { localStorage.setItem(WF_STORAGE_KEY, JSON.stringify(workflowList.value)) } catch {}
  }

  function saveWorkflow(name: string) {
    const wf = toWorkflowJSON(name)
    const now = new Date().toISOString()
    const existing = workflowList.value.find(w => w.name === name)
    if (existing) {
      existing.json = wf
      existing.updatedAt = now
      existing.nodeCount = wf.nodes.length
    } else {
      workflowList.value.push({
        id: `wf-${Date.now()}`,
        name,
        createdAt: now,
        updatedAt: now,
        nodeCount: wf.nodes.length,
        json: wf,
      })
    }
    persistWorkflowList()
  }

  function loadWorkflow(id: string) {
    const saved = workflowList.value.find(w => w.id === id)
    if (saved) fromWorkflowJSON(saved.json)
  }

  function deleteWorkflow(id: string) {
    workflowList.value = workflowList.value.filter(w => w.id !== id)
    persistWorkflowList()
  }

  function renameWorkflow(id: string, newName: string) {
    const wf = workflowList.value.find(w => w.id === id)
    if (wf) { wf.name = newName; wf.updatedAt = new Date().toISOString(); persistWorkflowList() }
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
    nodeMetaCache, fetchNodeMeta, getNodePorts, getPortType, canConnectPorts,
    addNode,
    removeNode,
    addEdge,
    removeEdge,
    selectNode,
    resetExecution,
    toWorkflowJSON,
    fromWorkflowJSON,
    workflowList, saveWorkflow, loadWorkflow, deleteWorkflow, renameWorkflow,
    undo,
    redo,
  }
})
