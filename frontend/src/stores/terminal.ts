import { defineStore } from 'pinia'
import { ref, triggerRef } from 'vue'
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

  // DockView layout — ref for reliable reactivity with deep trees
  const layout = ref<DockLayoutTree>({
    id: 'root',
    type: 'tab',
    tabs: [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }],
    activeTab: 'welcome',
  })

  /** Force re-render after mutation by triggering ref. */
  function notifyLayout() {
    triggerRef(layout)
  }

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
    if (commandHistory.value.length > 20) commandHistory.value.pop()
  }

  function toggleFocusMode() {
    focusMode.value = !focusMode.value
  }

  // ── Layout manipulation ──────────────────────────────────────

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

  function selectTab(leafId: string, tabId: string) {
    const leaf = findLeaf(layout.value, leafId)
    if (leaf && leaf.type === 'tab') {
      leaf.activeTab = tabId
      notifyLayout()
    }
  }

  function closeTab(_leafId: string, tabId: string) {
    const root = layout.value

    function removeFrom(node: DockLayoutTree): boolean {
      if (node.type === 'tab' && node.tabs) {
        const idx = node.tabs.findIndex((t) => t.id === tabId)
        if (idx !== -1) {
          node.tabs.splice(idx, 1)
          if (node.activeTab === tabId) {
            node.activeTab = node.tabs.length > 0 ? node.tabs[0].id : ''
          }
          if (node.tabs.length === 0) {
            node.tabs = [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }]
            node.activeTab = 'welcome'
          }
          return true
        }
      }
      if (node.children) {
        for (const child of node.children) {
          if (removeFrom(child)) return true
        }
      }
      return false
    }

    removeFrom(root)

    // Clean up activePanels
    const panelIdx = activePanels.value.findIndex((p) => p.instanceId === tabId)
    if (panelIdx !== -1) {
      activePanels.value.splice(panelIdx, 1)
    }

    notifyLayout()
  }

  function moveTab(leafId: string, fromIdx: number, toIdx: number) {
    const leaf = findLeaf(layout.value, leafId)
    if (leaf && leaf.type === 'tab' && leaf.tabs) {
      const [moved] = leaf.tabs.splice(fromIdx, 1)
      if (moved) {
        leaf.tabs.splice(toIdx, 0, moved)
        notifyLayout()
      }
    }
  }

  function updateSplitRatios(containerId: string, ratios: number[]) {
    const node = findLeaf(layout.value, containerId)
    if (node && node.type === 'container') {
      node.splitRatios = ratios
      notifyLayout()
    }
  }

  return {
    activePanels, commandHistory, pushPins, focusMode, layout,
    openPanel, closePanel, addCommand, toggleFocusMode,
    selectTab, closeTab, moveTab, updateSplitRatios,
  }
})
