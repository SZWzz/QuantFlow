import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Node, Edge, ViewportTransform } from '@vue-flow/core'
import { LoadWorkflow } from '@/lib/wails'

export type ExecutionStatus = 'idle' | 'running' | 'completed' | 'failed'

export interface NodeExecStatus {
  nodeId: string
  status: 'success' | 'failed' | 'skipped' | 'running'
  duration?: number
  error?: string
}

export interface NodeGroup {
  id: string
  label: string
  nodes: string[]
  style: { color: string }
}

export interface WorkflowJSON {
  id: string
  name: string
  description?: string
  nodes: { id: string; node_type: string; params: Record<string, any>; position?: { x: number; y: number } }[]
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
  const nodeOutputs = ref<Map<string, any>>(new Map())
  const runId = ref<string | null>(null)
  const selectedNodeId = ref<string | null>(null)

  const groups = ref<NodeGroup[]>([])
  const pinnedOutputs = ref<Record<string, boolean>>({})
  const clipboard = ref<VFNode[]>([])
  const disabledNodes = ref<Record<string, boolean>>({})

  // Sub-workflow navigation stack
  interface SubWFStackEntry {
    parentId: string
    parentNodes: any[]
    parentEdges: any[]
    viewport: ViewportTransform
  }
  const subWFStack = ref<SubWFStackEntry[]>([])
  const currentSubWFId = ref<string | null>(null)

  async function navigateIntoSubWF(nodeId: string, childWfId: string) {
    subWFStack.value.push({
      parentId: nodeId,
      parentNodes: JSON.parse(JSON.stringify(nodes.value)),
      parentEdges: JSON.parse(JSON.stringify(edges.value)),
      viewport: { ...viewport.value },
    })
    currentSubWFId.value = childWfId
    try {
      const wf = await LoadWorkflow(childWfId)
      if (wf) fromWorkflowJSON(wf)
    } catch { /* child workflow not found */ }
  }

  function navigateUpFromSubWF() {
    const entry = subWFStack.value.pop()
    if (!entry) return
    currentSubWFId.value = subWFStack.value.length > 0 ? subWFStack.value[subWFStack.value.length - 1].parentId : null
    nodes.value = entry.parentNodes
    edges.value = entry.parentEdges
    viewport.value = entry.viewport
  }

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

  // Recent & favorite nodes
  const recentTypes = ref<string[]>(loadRecentTypes())
  const favoriteTypes = ref<Set<string>>(loadFavoriteTypes())

  function loadRecentTypes(): string[] {
    try { const raw = localStorage.getItem('qf-recent-nodes'); return raw ? JSON.parse(raw) : [] } catch { return [] }
  }
  function persistRecentTypes() { localStorage.setItem('qf-recent-nodes', JSON.stringify(recentTypes.value)) }
  function loadFavoriteTypes(): Set<string> {
    try { const raw = localStorage.getItem('qf-favorite-nodes'); return new Set(raw ? JSON.parse(raw) : []) } catch { return new Set() }
  }
  function persistFavoriteTypes() { localStorage.setItem('qf-favorite-nodes', JSON.stringify([...favoriteTypes.value])) }

  function addNode(type: string, position: { x: number; y: number }, params?: Record<string, any>, portOverrides?: { category?: string; inputs: string[]; outputs: string[] }) {
    recentTypes.value = [type, ...recentTypes.value.filter(t => t !== type)].slice(0, 10)
    persistRecentTypes()
    pushHistory()
    const id = `${type}-${Date.now()}`
    const cached = getNodePorts(type)
    const ports = portOverrides ? { ...cached, ...portOverrides } : cached

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
        mode: 0,
        badges: {},
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

  function togglePin(nodeId: string) {
    const rec = pinnedOutputs.value
    if (rec[nodeId]) { delete rec[nodeId] }
    else { rec[nodeId] = true }
    updateNodeBadge(nodeId, 'pin', !!rec[nodeId])
  }

  function toggleDisable(nodeId: string) {
    const rec = disabledNodes.value
    if (rec[nodeId]) { delete rec[nodeId] }
    else { rec[nodeId] = true }
    const node = (nodes.value as any[]).find((n: any) => n.id === nodeId)
    if (node) node.data.mode = rec[nodeId] ? 2 : 0
  }

  function groupNodes(nodeIds: string[]) {
    const id = `group-${Date.now()}`
    groups.value.push({ id, label: 'Group', nodes: nodeIds, style: { color: '#30363d' } })
  }

  function copyNodes(nodeIds: string[]) {
    clipboard.value = (nodes.value as any[])
      .filter((n: any) => nodeIds.includes(n.id))
      .map((n: any) => JSON.parse(JSON.stringify(n)))
  }

  function pasteNodes() {
    if (!clipboard.value.length) return
    pushHistory()
    for (const n of clipboard.value) {
      const newId = `${n.data.nodeType}-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
      ;(nodes.value as any[]).push({
        ...JSON.parse(JSON.stringify(n)),
        id: newId,
        position: { x: n.position.x + 30, y: n.position.y + 30 },
        data: { ...n.data, status: 'idle' },
      })
    }
  }

  function cloneNode(nodeId: string) {
    const orig = (nodes.value as any[]).find((n: any) => n.id === nodeId)
    if (!orig) return
    pushHistory()
    const newId = `${orig.data.nodeType}-${Date.now()}`
    ;(nodes.value as any[]).push({
      ...JSON.parse(JSON.stringify(orig)),
      id: newId,
      position: { x: orig.position.x + 30, y: orig.position.y + 30 },
      data: { ...orig.data, status: 'idle' },
    })
  }

  function selectAllNodes() {
    for (const n of nodes.value as any[]) {
      n.selected = true
    }
  }

  function updateNodeBadge(nodeId: string, badge: string, active: boolean) {
    const node = (nodes.value as any[]).find((n: any) => n.id === nodeId)
    if (node) {
      if (!node.data.badges) node.data.badges = {}
      node.data.badges[badge] = active
    }
  }

  function resetExecution() {
    executionStatus.value = 'idle'
    nodeStatuses.value = new Map()
    nodeOutputs.value = new Map()
    runId.value = null
    stopPolling()
    for (const node of nodes.value as VFNode[]) {
      node.data.status = 'idle'
      node.data.error = undefined
    }
  }

  // ── Async execution with polling ──
  const queuePosition = ref<number>(0)
  let pollTimer: ReturnType<typeof setTimeout> | null = null

  function stopPolling() {
    if (pollTimer !== null) {
      clearTimeout(pollTimer)
      pollTimer = null
    }
  }

  function applyNodeResults(results: any[]) {
    if (!results) return
    for (const nr of results) {
      if (!nr.node_id) continue
      // Store outputs for terminal/display nodes
      if (nr.outputs && Object.keys(nr.outputs).length > 0) {
        nodeOutputs.value.set(nr.node_id, nr.outputs)
      }
      const existing = nodeStatuses.value.get(nr.node_id)
      if (existing?.status !== nr.status) {
        nodeStatuses.value.set(nr.node_id, {
          nodeId: nr.node_id,
          status: nr.status || 'success',
          duration: nr.duration_ms || nr.duration,
          error: nr.error,
        })
        const node = (nodes.value as any[]).find((n: any) => n.id === nr.node_id)
        if (node) {
          node.data.status = nr.status === 'completed' || nr.status === 'success' ? 'success'
            : nr.status === 'failed' ? 'failed'
            : nr.status === 'running' ? 'running'
            : 'idle'
          if (nr.error) node.data.error = nr.error
        }
      }
    }
  }

  async function startAsyncRun() {
    const wfJSON = toWorkflowJSON('Async Run')
    resetExecution()
    executionStatus.value = 'running'

    const app = (window as any).go?.main?.App
    if (!app?.QueueWorkflow) {
      // Fallback to sync
      try {
        const result = await app.RunWorkflow(JSON.stringify(wfJSON))
        executionStatus.value = result?.status || 'completed'
        if (result?.node_results) applyNodeResults(result.node_results)
      } catch (e: any) {
        executionStatus.value = 'failed'
      }
      return
    }

    try {
      const id = await app.QueueWorkflow(JSON.stringify(wfJSON))
      if (!id) { executionStatus.value = 'failed'; return }
      runId.value = id
      pollExecution(id)
    } catch {
      executionStatus.value = 'failed'
    }
  }

  function pollExecution(execId: string) {
    const app = (window as any).go?.main?.App
    if (!app?.GetExecutionStatus) return
    const id = execId

    const poll = async () => {
      try {
        const status = await app.GetExecutionStatus(id)
        if (!status) { stopPolling(); executionStatus.value = 'failed'; return }

        queuePosition.value = status.queue_position ?? 0
        if (status.node_results) applyNodeResults(Object.values(status.node_results))

        if (status.status === 'completed' || status.status === 'failed') {
          executionStatus.value = status.status
          runId.value = null
          if (status.error) console.error('workflow failed:', status.error)
          stopPolling()
          return
        }
        pollTimer = setTimeout(poll, 300)
      } catch {
        stopPolling()
        executionStatus.value = 'failed'
      }
    }
    poll()
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
        position: n.position ? { x: n.position.x, y: n.position.y } : undefined,
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
        position: n.position || { x: 100 + Math.random() * 300, y: 100 + Math.random() * 200 },
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
    nodeOutputs,
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
    startAsyncRun,
    toWorkflowJSON,
    fromWorkflowJSON,
    workflowList, saveWorkflow, loadWorkflow, deleteWorkflow, renameWorkflow,
    recentTypes, favoriteTypes,
    groups, pinnedOutputs, clipboard, disabledNodes,
    togglePin, toggleDisable, groupNodes, copyNodes, pasteNodes, cloneNode, selectAllNodes,
    subWFStack, currentSubWFId, navigateIntoSubWF, navigateUpFromSubWF,
    undo,
    redo,
  }
})
