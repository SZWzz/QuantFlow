<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import DockContainer from './DockContainer.vue'
import {
  type DockLayoutTree,
  type DockTabState,
  createTabLeaf,
  createContainer,
} from './types'

const terminal = useTerminalStore()

// Initialize layout from store or create default
const layout = computed(() => {
  return terminal.layout
})

function initDefaultLayout() {
  if (terminal.layout.type !== 'tab' && terminal.layout.type !== 'container') {
    terminal.layout = createTabLeaf('root', {
      id: 'welcome',
      panelId: 'welcome',
      label: 'Welcome',
      icon: '🏠',
    })
  }
}

// Add a panel to the layout — if single tab, split; otherwise add to active leaf
function addPanel(tab: DockTabState) {
  const root = terminal.layout

  if (root.type === 'tab') {
    if (!root.tabs || root.tabs.length === 0) {
      // Replace empty welcome tab
      root.tabs = [tab]
      root.activeTab = tab.id
    } else if (root.tabs.length === 1) {
      // Single tab → split into 2 horizontally
      const existingLeaf = createTabLeaf('leaf-existing', root.tabs[0])
      const newLeaf = createTabLeaf(`leaf-${tab.id}`, tab)
      terminal.layout = createContainer('root', 'row', [existingLeaf, newLeaf])
    } else {
      // Multiple tabs already — add to this tab group
      root.tabs.push(tab)
      root.activeTab = tab.id
    }
  } else if (root.type === 'container') {
    // Find first tab leaf and add tab there
    addToFirstLeaf(root, tab)
  }
}

function addToFirstLeaf(node: DockLayoutTree, tab: DockTabState): boolean {
  if (node.type === 'tab') {
    if (!node.tabs) node.tabs = []
    // Avoid duplicate
    if (!node.tabs.find((t) => t.id === tab.id)) {
      node.tabs.push(tab)
    }
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

function onSelectTab(leafId: string, tabId: string) {
  terminal.selectTab(leafId, tabId)
}

function onCloseTab(leafId: string, tabId: string) {
  terminal.closeTab(leafId, tabId)
}

function onSplitRatio(containerId: string, index: number, ratios: number[]) {
  terminal.updateSplitRatios(containerId, ratios)
}

// Watch for new panels from terminalStore
const unwatch = terminal.$subscribe((_mutation, state) => {
  const panels = state.activePanels as { instanceId: string; panelId: string; label: string; icon: string; params?: Record<string, any> }[]
  for (const panel of panels) {
    const existing = findTabInTree(layout.value, panel.instanceId)
    if (!existing) {
      addPanel({
        id: panel.instanceId,
        panelId: panel.panelId,
        label: panel.label,
        icon: panel.icon,
        params: panel.params,
      })
    }
  }
})

function findTabInTree(node: DockLayoutTree, tabId: string): DockTabState | null {
  if (node.type === 'tab' && node.tabs) {
    return node.tabs.find((t) => t.id === tabId) || null
  }
  if (node.children) {
    for (const child of node.children) {
      const found = findTabInTree(child, tabId)
      if (found) return found
    }
  }
  return null
}

// Keyboard shortcuts for layout presets
function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key >= '1' && e.key <= '4') {
    e.preventDefault()
    const preset = parseInt(e.key)
    applyPreset(preset)
  }
}

function applyPreset(preset: number) {
  const tabs = getExistingTabs()
  if (tabs.length === 0) return

  switch (preset) {
    case 1: // Single
      terminal.layout = createTabLeaf('root', tabs[0])
      break
    case 2: // Split horizontal
      if (tabs.length >= 2) {
        terminal.layout = createContainer('root', 'row', [
          createTabLeaf('left', tabs[0]),
          createTabLeaf('right', tabs[1]),
        ])
      }
      break
    case 3: // 2x2 grid
      if (tabs.length >= 4) {
        const top = createContainer('top', 'row', [
          createTabLeaf('tl', tabs[0]),
          createTabLeaf('tr', tabs[1]),
        ])
        const bottom = createContainer('bottom', 'row', [
          createTabLeaf('bl', tabs[2]),
          createTabLeaf('br', tabs[3]),
        ])
        terminal.layout = createContainer('root', 'column', [top, bottom])
      }
      break
    case 4: // Classic: sidebar + main
      if (tabs.length >= 2) {
        terminal.layout = createContainer('root', 'row', [
          createTabLeaf('sidebar', tabs[0]),
          createTabLeaf('main', tabs[1]),
        ])
        terminal.layout.splitRatios = [0.25, 0.75]
      }
      break
  }
}

function getExistingTabs(): DockTabState[] {
  const tabs: DockTabState[] = []
  collectTabs(terminal.layout, tabs)
  return tabs
}

function collectTabs(node: DockLayoutTree, out: DockTabState[]) {
  if (node.type === 'tab' && node.tabs) {
    out.push(...node.tabs)
  }
  if (node.children) {
    for (const child of node.children) {
      collectTabs(child, out)
    }
  }
}

onMounted(() => {
  initDefaultLayout()
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  unwatch()
  window.removeEventListener('keydown', onKeydown)
})
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
        @select-tab="onSelectTab"
        @close-tab="onCloseTab"
        @split-ratio="onSplitRatio"
      />
    </div>
  </div>
</template>

<style scoped>
.dock-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-app);
}

.dock-view-toolbar {
  display: flex;
  align-items: center;
  padding: 2px 8px;
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border);
}

.preset-buttons {
  display: flex;
  gap: 4px;
}

.preset-buttons button {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  transition: all var(--transition-fast);
}

.preset-buttons button:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.dock-view-content {
  flex: 1;
  overflow: hidden;
}
</style>
