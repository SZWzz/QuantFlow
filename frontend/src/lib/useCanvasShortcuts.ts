import { onMounted, onUnmounted } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'

export function useCanvasShortcuts() {
  const workflow = useWorkflowStore()

  function handler(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement)?.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || (e.target as HTMLElement)?.isContentEditable) return

    const ctrl = e.ctrlKey || e.metaKey

    if ((e.key === 'Delete' || e.key === 'Backspace') && workflow.selectedNodeId) {
      workflow.removeNode(workflow.selectedNodeId)
      e.preventDefault()
    } else if (ctrl && e.key === 'c' && workflow.selectedNodeId) {
      const n = (workflow.nodes as any[]).find((x: any) => x.id === workflow.selectedNodeId)
      if (n) workflow.copyNodes([n.id])
      e.preventDefault()
    } else if (ctrl && e.key === 'v' && workflow.clipboard.length) {
      workflow.pasteNodes()
      e.preventDefault()
    } else if (ctrl && e.key === 'a') {
      workflow.selectAllNodes()
      e.preventDefault()
    } else if (ctrl && e.key === 'z' && !e.shiftKey) {
      workflow.undo()
      e.preventDefault()
    } else if (ctrl && e.key === 'z' && e.shiftKey) {
      workflow.redo()
      e.preventDefault()
    } else if (ctrl && e.key === 'd' && workflow.selectedNodeId) {
      workflow.cloneNode(workflow.selectedNodeId)
      e.preventDefault()
    } else if (ctrl && e.key === 'Enter') {
      // Run workflow — triggers parent event
      window.dispatchEvent(new CustomEvent('workflow:run'))
      e.preventDefault()
    } else if (e.key === 'Escape') {
      workflow.selectedNodeId = null
      e.preventDefault()
    }
  }

  onMounted(() => window.addEventListener('keydown', handler))
  onUnmounted(() => window.removeEventListener('keydown', handler))
}
