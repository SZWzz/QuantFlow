<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, provide, reactive } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import DockContainer from './DockContainer.vue'
import {
  type DockLayoutTree,
  type DockTabState,
  createTabLeaf,
  createContainer,
} from './types'

const terminal = useTerminalStore()

// 跨组件共享的分时数据缓存：key = "symbol:date"
const minuteDataCache = reactive(new Map<string, any[]>())
provide('minuteDataCache', minuteDataCache)

// Track which leaf was last clicked — new panels open there, not always leftmost.
const activeLeafId = ref<string>('')

function onSelectTab(leafId: string, tabId: string) {
  activeLeafId.value = leafId
  terminal.selectTab(leafId, tabId)
}

const layout = computed(() => terminal.layout)

function initDefaultLayout() {
  const root = terminal.layout
  if (!root) return
  if (root.type === 'tab' && (!root.tabs || root.tabs.length === 0)) {
    root.tabs = [{ id: 'welcome', panelId: 'welcome', label: '欢迎', icon: '🏠' }]
    root.activeTab = 'welcome'
  }
}

// Add panel to layout — split single tab, or add to active/last-clicked leaf.
function addPanel(tab: DockTabState) {
  const root = terminal.layout

  if (root.type === 'tab') {
    if (!root.tabs || root.tabs.length === 0) {
      root.tabs = [tab]
      root.activeTab = tab.id
    } else if (root.tabs.length === 1) {
      const existingLeaf = createTabLeaf('leaf-existing', root.tabs[0])
      const newLeaf = createTabLeaf(`leaf-${tab.id}`, tab)
      Object.assign(terminal.layout, createContainer('root', 'row', [existingLeaf, newLeaf]))
    } else {
      root.tabs.push(tab)
      root.activeTab = tab.id
    }
  } else if (root.type === 'container') {
    // Try active leaf first, fall back to first leaf
    if (activeLeafId.value && addToLeafById(root, activeLeafId.value, tab)) return
    addToFirstLeaf(root, tab)
  }
}

function addToFirstLeaf(node: DockLayoutTree, tab: DockTabState): boolean {
  if (node.type === 'tab') {
    if (!node.tabs) node.tabs = []
    if (!node.tabs.find((t) => t.id === tab.id)) node.tabs.push(tab)
    node.activeTab = tab.id
    return true
  }
  if (node.children) {
    for (const child of node.children) {
      if (addToFirstLeaf(child, tab)) return true
    }
  }
  return false
}

function addToLeafById(node: DockLayoutTree, leafId: string, tab: DockTabState): boolean {
  if (node.id === leafId && node.type === 'tab') {
    if (!node.tabs) node.tabs = []
    if (!node.tabs.find((t) => t.id === tab.id)) node.tabs.push(tab)
    node.activeTab = tab.id
    return true
  }
  if (node.children) {
    for (const child of node.children) {
      if (addToLeafById(child, leafId, tab)) return true
    }
  }
  return false
}

function onCloseTab(leafId: string, tabId: string) {
  terminal.closeTab(leafId, tabId)
}

function onSplitRatio(containerId: string, index: number, ratios: number[]) {
  terminal.updateSplitRatios(containerId, ratios)
}

// Tab reorder: reorder tabs within the same leaf.
function onTabReorder(leafId: string, tabId: string, toIndex: number) {
  const leaf = findLeafById(terminal.layout, leafId)
  if (!leaf || leaf.type !== 'tab' || !leaf.tabs) return
  const fromIndex = leaf.tabs.findIndex(t => t.id === tabId)
  if (fromIndex === -1 || fromIndex === toIndex) return
  const [tab] = leaf.tabs.splice(fromIndex, 1)
  leaf.tabs.splice(toIndex, 0, tab)
}

// Tab drag: move a tab from one leaf to another.
function onTabDrag(fromLeafId: string, tabId: string, toLeafId: string) {
  // Find source leaf and remove tab
  const fromLeaf = findLeafById(terminal.layout, fromLeafId)
  const toLeaf = findLeafById(terminal.layout, toLeafId)
  if (!fromLeaf || !toLeaf || fromLeaf.type !== 'tab' || toLeaf.type !== 'tab') return
  if (!fromLeaf.tabs || !toLeaf.tabs) return

  const tabIdx = fromLeaf.tabs.findIndex((t) => t.id === tabId)
  if (tabIdx === -1) return

  const [tab] = fromLeaf.tabs.splice(tabIdx, 1)
  if (fromLeaf.activeTab === tabId && fromLeaf.tabs.length > 0) {
    fromLeaf.activeTab = fromLeaf.tabs[0].id
  }
  toLeaf.tabs.push(tab)
  toLeaf.activeTab = tab.id
}

function findLeafById(node: DockLayoutTree, id: string): DockLayoutTree | null {
  if (node.id === id) return node
  if (node.children) {
    for (const child of node.children) {
      const found = findLeafById(child, id)
      if (found) return found
    }
  }
  return null
}

// Watch for new panels from terminalStore
const unwatch = terminal.$subscribe((_mutation, state) => {
  const panels = state.activePanels as { instanceId: string; panelId: string; label: string; icon: string; params?: Record<string, any> }[]
  for (const panel of panels) {
    const existing = findTabInTree(layout.value, panel.instanceId)
    if (!existing) {
      addPanel({
        id: panel.instanceId, panelId: panel.panelId,
        label: panel.label, icon: panel.icon, params: panel.params,
      })
    }
  }
})

function findTabInTree(node: DockLayoutTree, tabId: string): DockTabState | null {
  if (node.type === 'tab' && node.tabs) return node.tabs.find((t) => t.id === tabId) || null
  if (node.children) {
    for (const child of node.children) {
      const found = findTabInTree(child, tabId)
      if (found) return found
    }
  }
  return null
}

function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key >= '1' && e.key <= '4') {
    e.preventDefault(); applyPreset(parseInt(e.key))
  }
}

function applyPreset(preset: number) {
  const tabs = getExistingTabs()
  if (tabs.length === 0) return
  switch (preset) {
    case 1: Object.assign(terminal.layout, createTabLeaf('root', tabs[0])); break
    case 2: if (tabs.length >= 2) Object.assign(terminal.layout, createContainer('root', 'row', [createTabLeaf('left', tabs[0]), createTabLeaf('right', tabs[1])])); break
    case 3: if (tabs.length >= 4) {
      Object.assign(terminal.layout, createContainer('root', 'column', [
        createContainer('top', 'row', [createTabLeaf('tl', tabs[0]), createTabLeaf('tr', tabs[1])]),
        createContainer('bottom', 'row', [createTabLeaf('bl', tabs[2]), createTabLeaf('br', tabs[3])])
      ]))
    }; break
    case 4: if (tabs.length >= 2) {
      Object.assign(terminal.layout, createContainer('root', 'row', [createTabLeaf('sidebar', tabs[0]), createTabLeaf('main', tabs[1])]))
      terminal.layout.splitRatios = [0.25, 0.75]
    }; break
  }
}

function getExistingTabs(): DockTabState[] { const t: DockTabState[] = []; collectTabs(terminal.layout, t); return t }
function collectTabs(node: DockLayoutTree, out: DockTabState[]) {
  if (node.type === 'tab' && node.tabs) out.push(...node.tabs)
  if (node.children) for (const child of node.children) collectTabs(child, out)
}

onMounted(() => { initDefaultLayout(); window.addEventListener('keydown', onKeydown) })
onUnmounted(() => { unwatch(); window.removeEventListener('keydown', onKeydown) })
</script>

<template>
  <div class="dock-view">
    <div class="dock-view-toolbar">
      <div class="preset-buttons">
        <button title="Single (Ctrl+1)" @click="applyPreset(1)">□</button>
        <button title="Split H (Ctrl+2)" @click="applyPreset(2)">◫</button>
        <button title="2×2 (Ctrl+3)" @click="applyPreset(3)">⊞</button>
        <button title="Sidebar (Ctrl+4)" @click="applyPreset(4)">⊟</button>
      </div>
    </div>
    <div class="dock-view-content">
      <DockContainer
        :node="layout"
        :active-leaf-id="activeLeafId"
        @select-tab="onSelectTab"
        @close-tab="onCloseTab"
        @split-ratio="onSplitRatio"
        @tab-drag="(a, b, c) => onTabDrag(a, b, c)"
        @tab-reorder="(a, b, c) => onTabReorder(a, b, c)"
      />
    </div>
  </div>
</template>

<style scoped>
.dock-view { display: flex; flex-direction: column; height: 100%; background: var(--color-bg-app); }
.dock-view-toolbar { display: flex; align-items: center; padding: 2px 8px; background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border); }
.preset-buttons { display: flex; gap: 4px; }
.preset-buttons button { padding: 2px 8px; border: 1px solid var(--color-border); background: transparent; color: var(--color-text-tertiary); border-radius: var(--radius-sm); cursor: pointer; font-size: var(--font-xs); transition: all var(--transition-fast); }
.preset-buttons button:hover { border-color: var(--color-accent); color: var(--color-accent); }
.dock-view-content { flex: 1; min-height: 0; overflow: hidden; }
</style>
