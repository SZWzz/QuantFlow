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
    const leaf = findLeaf(layout, leafId)
    if (leaf && leaf.type === 'tab' && leaf.tabs) {
      leaf.tabs = leaf.tabs.filter((t) => t.id !== tabId)
      if (leaf.activeTab === tabId && leaf.tabs.length > 0) {
        leaf.activeTab = leaf.tabs[0].id
      }
      // Also remove from activePanels
      activePanels.value = activePanels.value.filter(
        (p) => p.instanceId !== tabId
      )
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
    updateSplitRatios,
  }
})
