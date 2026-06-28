import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import type { DockLayoutTree, DockTabState } from '@/terminal/DockView/types'
import { getPanelMeta } from '@/terminal/panels/registry'

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
  const recentPanels = ref<string[]>(loadRecentPanels())

  function loadRecentPanels(): string[] {
    try {
      const saved = localStorage.getItem('quantflow-recent-panels')
      return saved ? JSON.parse(saved) : []
    } catch { return [] }
  }

  function persistRecentPanels() {
    try {
      localStorage.setItem('quantflow-recent-panels', JSON.stringify(recentPanels.value))
    } catch {}
  }

  const layout = reactive<DockLayoutTree>(loadLayout())

  function loadLayout(): DockLayoutTree {
    try {
      const saved = localStorage.getItem('quantflow-layout')
      if (saved) return JSON.parse(saved)
    } catch {}
    return {
      id: 'root',
      type: 'tab',
      tabs: [{ id: 'welcome', panelId: 'welcome', label: '欢迎', icon: '🏠' }],
      activeTab: 'welcome',
    }
  }

  function persistLayout() {
    try {
      localStorage.setItem('quantflow-layout', JSON.stringify(layout))
    } catch {}
  }

  /** Replace layout object to force reactivity propagation. */
  function applyLayout(next: DockLayoutTree) {
    Object.assign(layout, next)
    persistLayout()
  }

  function openPanel(panelId: string, params?: Record<string, any>) {
    const instanceId = `${panelId}-${crypto.randomUUID().slice(0, 6)}`
    const panel: PanelState = {
      instanceId,
      panelId,
      label: getPanelMeta(panelId)?.label || panelId,
      icon: '📊',
      params,
    }
    activePanels.value.push(panel)
    // Track recent panels (dedupe, max 20)
    recentPanels.value = [panelId, ...recentPanels.value.filter(p => p !== panelId)].slice(0, 20)
    persistRecentPanels()
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

  function closeTab(leafId: string, tabId: string) {
    const leaf = findLeaf(layout, leafId)
    function removeFrom(n: DockLayoutTree): boolean {
      if (n.id === leafId && n.type === 'tab' && n.tabs) {
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
        return false
      }
      if (n.children) {
        for (const c of n.children) { if (removeFrom(c)) return true }
      }
      return false
    }
    const target = leaf || layout
    function searchFromRoot(n: DockLayoutTree): boolean {
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
        for (const c of n.children) { if (searchFromRoot(c)) return true }
      }
      return false
    }
    if (leaf?.type === 'tab' && leaf.tabs?.some(t => t.id === tabId)) {
      removeFrom(layout)
    } else {
      searchFromRoot(layout)
    }
    persistLayout()
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
    activePanels, commandHistory, pushPins, focusMode, layout, recentPanels,
    openPanel, closePanel, addCommand, toggleFocusMode,
    selectTab, closeTab, moveTab, updateSplitRatios, applyLayout, persistLayout,
  }
})
