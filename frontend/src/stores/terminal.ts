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

  // ── Symbol Context (Bloomberg-style cross-panel linkage) ─────
  // When a panel publishes a symbol, all linked subscriber panels
  // automatically update to show data for that symbol.
  const activeSymbol = ref<string | null>(null)
  const lastSymbolUpdate = ref(0) // timestamp to force watchers

  function setActiveSymbol(symbol: string) {
    if (!symbol) return
    const s = symbol.trim().toUpperCase()
    if (s !== activeSymbol.value) {
      activeSymbol.value = s
      lastSymbolUpdate.value = Date.now()
    }
  }

  const layout = reactive<DockLayoutTree>({
    id: 'root',
    type: 'tab',
    tabs: [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }],
    activeTab: 'welcome',
  })

  /** Replace layout object to force reactivity propagation. */
  function applyLayout(next: DockLayoutTree) {
    Object.assign(layout, next)
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
    activePanels.value = activePanels.value.filter(p => p.instanceId !== instanceId)
  }

  function addCommand(cmd: string) {
    commandHistory.value.unshift(cmd)
    if (commandHistory.value.length > 20) commandHistory.value.pop()
  }

  function toggleFocusMode() { focusMode.value = !focusMode.value }

  // ── Layout helpers ────────────────────────────────────────────

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
    const leaf = findLeaf(layout, leafId)
    if (leaf && leaf.type === 'tab') {
      leaf.activeTab = tabId
    }
  }

  function closeTab(_leafId: string, tabId: string) {
    function removeFrom(n: DockLayoutTree): boolean {
      if (n.type === 'tab' && n.tabs) {
        const idx = n.tabs.findIndex(t => t.id === tabId)
        if (idx !== -1) {
          n.tabs.splice(idx, 1)
          if (n.activeTab === tabId) n.activeTab = n.tabs[0]?.id ?? ''
          if (n.tabs.length === 0) {
            n.tabs = [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }]
            n.activeTab = 'welcome'
          }
          return true
        }
      }
      if (n.children) {
        for (const c of n.children) { if (removeFrom(c)) return true }
      }
      return false
    }
    removeFrom(layout)
    activePanels.value = activePanels.value.filter(p => p.instanceId !== tabId)
  }

  function moveTab(leafId: string, fromIdx: number, toIdx: number) {
    const leaf = findLeaf(layout, leafId)
    if (leaf?.type === 'tab' && leaf.tabs) {
      const [moved] = leaf.tabs.splice(fromIdx, 1)
      if (moved) leaf.tabs.splice(toIdx, 0, moved)
    }
  }

  function updateSplitRatios(containerId: string, ratios: number[]) {
    const n = findLeaf(layout, containerId)
    if (n?.type === 'container') n.splitRatios = ratios
  }

  return {
    activePanels, commandHistory, pushPins, focusMode, activeSymbol, lastSymbolUpdate, layout,
    openPanel, closePanel, addCommand, toggleFocusMode, setActiveSymbol,
    selectTab, closeTab, moveTab, updateSplitRatios, applyLayout,
  }
})
