import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWorkflowStore } from '../workflow'
import type { WorkflowJSON } from '../workflow'

describe('useWorkflowStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty nodes and edges', () => {
    const store = useWorkflowStore()
    expect(store.nodes).toHaveLength(0)
    expect(store.edges).toHaveLength(0)
    expect(store.selectedNodeId).toBeNull()
    expect(store.executionStatus).toBe('idle')
  })

  it('should add and remove nodes', () => {
    const store = useWorkflowStore()
    const id = store.addNode('sma', { x: 100, y: 200 })
    expect(store.nodes).toHaveLength(1)
    expect(store.nodes[0].data.nodeType).toBe('sma')
    expect(store.nodes[0].data.label).toBe('Sma')
    expect(store.nodes[0].data.inputs).toEqual(['input'])
    expect(store.nodes[0].data.outputs).toEqual(['output'])

    store.removeNode(id)
    expect(store.nodes).toHaveLength(0)
  })

  it('should add and remove edges', () => {
    const store = useWorkflowStore()
    const id1 = store.addNode('sma', { x: 0, y: 0 })
    const id2 = store.addNode('log_output', { x: 200, y: 0 })
    const edge = { id: 'e-1', source: id1, target: id2, type: 'default', style: { stroke: '#30363d', strokeWidth: 2 } }
    store.addEdge(edge)
    expect(store.edges).toHaveLength(1)
    store.removeEdge(edge.id)
    expect(store.edges).toHaveLength(0)
  })

  it('should handle node selection', () => {
    const store = useWorkflowStore()
    const id = store.addNode('data_loader', { x: 50, y: 50 })
    store.selectNode(id)
    expect(store.selectedNodeId).toBe(id)
    store.selectNode(null)
    expect(store.selectedNodeId).toBeNull()
  })

  it('should support undo and redo', () => {
    const store = useWorkflowStore()
    store.addNode('sma', { x: 0, y: 0 })
    store.addNode('data_loader', { x: 200, y: 0 })
    store.addNode('log_output', { x: 400, y: 0 })
    expect(store.nodes).toHaveLength(3)

    store.undo()
    expect(store.nodes).toHaveLength(1)

    store.redo()
    expect(store.nodes).toHaveLength(2)
  })

  it('should serialize and deserialize workflow JSON', () => {
    const store = useWorkflowStore()
    store.addNode('data_loader', { x: 100, y: 100 })
    store.addNode('sma', { x: 300, y: 100 })
    const fromNode = store.nodes[0]
    const toNode = store.nodes[1]
    store.addEdge({
      id: `e-${fromNode.id}-${toNode.id}`,
      source: fromNode.id,
      target: toNode.id,
      sourceHandle: 'ohlcv',
      targetHandle: 'input',
      type: 'default',
      style: { stroke: '#30363d', strokeWidth: 2 },
    })

    const json = store.toWorkflowJSON('Test Workflow')
    expect(json.nodes).toHaveLength(2)
    expect(json.edges).toHaveLength(1)
    expect(json.name).toBe('Test Workflow')

    // Load from JSON into a fresh store
    setActivePinia(createPinia())
    const store2 = useWorkflowStore()
    store2.fromWorkflowJSON(json as WorkflowJSON)
    expect(store2.nodes).toHaveLength(2)
    expect(store2.edges).toHaveLength(1)
  })

  it('should reset execution status', () => {
    const store = useWorkflowStore()
    store.executionStatus = 'running'
    store.resetExecution()
    expect(store.executionStatus).toBe('idle')
    expect(store.runId).toBeNull()
  })
})
