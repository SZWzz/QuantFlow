import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import type { DockLayoutTree, DockTabState } from '@/terminal/DockView/types'

export interface PanelState {
  instanceId: string
  panelId: string
  label: string
  icon: string
  params?: Record<string, any>
}

export interface PushPin {
  id: string
  label: string
  type: 'symbol' | 'panel' | 'workflow'
  payload: Record<string, any>
}

export const useTerminalStore = defineStore('terminal', () => {
  const activePanels = ref<PanelState[]>([])
  const commandHistory = ref<string[]>([])
  const pushPins = ref<PushPin[]>([])
  const focusMode = ref(false)

  // DockView layout
  const layout = reactive<DockLayoutTree>({
    id: 'root',
    type: 'tab',
    tabs: [],
  })

  function openPanel(panelId: string, params?: Record<string, any>) {
    const instanceId = `${panelId}-${Date.now()}`
    const panel: PanelState = {
      instanceId,
      panelId,
      label: panelId.replace(/-/g, ' ').replace(/\b\w/g, c => c.toUpperCase()),
      icon: '📊',
      params,
    }
    activePanels.value.push(panel)
    return instanceId
  }

  function closePanel(instanceId: string) {
    activePanels.value = activePanels.value.filter(
      (p) => p.instanceId !== instanceId
    )
  }

  function addCommand(cmd: string) {
    commandHistory.value.unshift(cmd)
    if (commandHistory.value.length > 20) {
      commandHistory.value.pop()
    }
  }

  function toggleFocusMode() {
    focusMode.value = !focusMode.value
  }

  // DockView actions
  function selectTab(leafId: string, tabId: string) {
    const leaf = findLeaf(layout, leafId)
    if (leaf && leaf.type === 'tab') {
      leaf.activeTab = tabId
    }
  }

  function closeTab(leafId: string, tabId: string) {
    // Recursively remove tab and clean up empty leaves
    function removeFromTree(node: DockLayoutTree): boolean {
      if (node.type === 'tab' && node.tabs) {
        const idx = node.tabs.findIndex((t) => t.id === tabId)
        if (idx !== -1) {
          node.tabs.splice(idx, 1)
          if (node.activeTab === tabId) {
            node.activeTab = node.tabs.length > 0 ? node.tabs[0].id : ''
          }
          return true
        }
      }
      if (node.children) {
        for (const child of node.children) {
          if (removeFromTree(child)) return true
        }
      }
      return false
    }

    const found = removeFromTree(layout)

    // Clean up activePanels
    const panelIdx = activePanels.value.findIndex((p) => p.instanceId === tabId)
    if (panelIdx !== -1) {
      activePanels.value.splice(panelIdx, 1)
    }

    // Collapse: if container has only one child after removal, promote it
    if (layout.type === 'container' && layout.children) {
      // Remove empty tab leaves
      layout.children = layout.children.filter(c => {
        if (c.type === 'tab' && (!c.tabs || c.tabs.length === 0)) return false
        if (c.type === 'container' && (!c.children || c.children.length === 0)) return false
        return true
      })
      // If only one child left, flatten
      if (layout.children.length === 1) {
        const child = layout.children[0]
        layout.id = child.id
        layout.type = child.type
        layout.direction = child.direction
        layout.splitRatios = child.splitRatios
        layout.children = child.children
        layout.tabs = child.tabs
        layout.activeTab = child.activeTab
      }
    }

    // If no tabs left, restore welcome
    if (layout.type === 'tab' && (!layout.tabs || layout.tabs.length === 0)) {
      layout.tabs = [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }]
      layout.activeTab = 'welcome'
    }
  }

  function moveTab(leafId: string, fromIdx: number, toIdx: number) {
    const leaf = findLeaf(layout, leafId)
    if (leaf && leaf.type === 'tab' && leaf.tabs) {
      const [moved] = leaf.tabs.splice(fromIdx, 1)
      if (moved) {
        leaf.tabs.splice(toIdx, 0, moved)
      }
    }
  }

  function updateSplitRatios(containerId: string, ratios: number[]) {
    const node = findLeaf(layout, containerId)
    if (node && node.type === 'container') {
      node.splitRatios = ratios
    }
  }

  function findLeaf(node: DockLayoutTree, id: string): DockLayoutTree | null {
    if (node.id === id) return node
    if (node.children) {
      for (const child of node.children) {
        const found = findLeaf(child, id)
        if (found) return found
      }
    }
    return null
  }

  return {
    activePanels,
    commandHistory,
    pushPins,
    focusMode,
    layout,
    openPanel,
    closePanel,
    addCommand,
    toggleFocusMode,
    selectTab,
    closeTab,
    moveTab,
    updateSplitRatios,
  }
})
