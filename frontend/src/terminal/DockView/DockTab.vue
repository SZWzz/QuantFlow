<script setup lang="ts">
import { computed, type Component } from 'vue'
import type { DockTabState } from './types'
import { getPanelComponent } from '@/terminal/panels/registry'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'

const props = defineProps<{
  tabs: DockTabState[]
  activeTab: string
  leafId: string
}>()

const emit = defineEmits<{
  (e: 'select-tab', tabId: string): void
  (e: 'close-tab', tabId: string): void
  (e: 'tab-drag', fromLeafId: string, tabId: string, toLeafId: string): void
}>()

function onDragStart(e: DragEvent, tabId: string) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', JSON.stringify({ leafId: props.leafId, tabId }))
}
function onDragOver(e: DragEvent) { e.preventDefault(); if (e.dataTransfer) e.dataTransfer.dropEffect = 'move' }
function onDrop(e: DragEvent) {
  e.preventDefault(); if (!e.dataTransfer) return
  try {
    const data = JSON.parse(e.dataTransfer.getData('text/plain'))
    if (data.leafId !== props.leafId) emit('tab-drag', data.leafId, data.tabId, props.leafId)
  } catch { /* ignore */ }
}

const terminal = useTerminalStore()
const ctx = useSymbolContext()

function tabGroupDot(panelId: string): { show: boolean; color: string } {
  const symbol = ctx.getActiveSymbolForPanel(panelId)
  if (!symbol) return { show: false, color: '' }
  const groupId = ctx.getPanelGroupId(panelId)
  const group = ctx.linkGroups[groupId]
  return { show: true, color: group?.color || '' }
}

const activePanel = computed(() => props.tabs.find((t) => t.id === props.activeTab))

const activeComponent = computed<Component | undefined>(() => {
  if (!activePanel.value) return undefined
  return getPanelComponent(activePanel.value.panelId)
})

const activeParams = computed(() => activePanel.value?.params)

// Close directly via store — bypasses fragile event propagation chain
function closeTab(tabId: string) {
  terminal.closeTab(props.leafId, tabId)
}
</script>

<template>
  <div class="dock-tab">
    <div class="tab-header" @dragover="onDragOver" @drop="onDrop">
      <div class="tab-list">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: tab.id === activeTab }"
          @click="emit('select-tab', tab.id)"
        >
          <span class="tab-icon">{{ tab.icon }}</span>
          <span
            v-if="tabGroupDot(tab.panelId).show"
            class="tab-group-dot"
            :style="{ background: tabGroupDot(tab.panelId).color }"
          ></span>
          <span class="tab-label">{{ tab.label }}</span>
          <span
            class="tab-close"
            @click.stop="closeTab(tab.id)"
            title="Close"
          >✕</span>
        </button>
      </div>
    </div>
    <div class="tab-content" @dragover="onDragOver" @drop="onDrop">
      <div v-if="tabs.length === 0" class="empty-content">
        Drop panels here
      </div>
      <component
        v-else-if="activeComponent"
        :is="activeComponent"
        :panel-id="activePanel?.panelId"
        :params="activeParams"
        class="panel-instance"
      />
      <div v-else class="empty-content">
        Panel "{{ activePanel?.panelId }}" not registered
      </div>
    </div>
  </div>
</template>

<style scoped>
.dock-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  overflow: hidden;
}

.tab-header {
  display: flex;
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border);
  min-height: 32px;
}

.tab-list {
  display: flex;
  overflow-x: auto;
  flex: 1;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  border: none;
  border-right: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  font-size: var(--font-xs);
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  min-width: 0;
  transition: background var(--transition-fast);
  user-select: none;
}

.tab-btn.active {
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
}

.tab-btn:hover {
  color: var(--color-text-primary);
}

.tab-icon { font-size: 12px; flex-shrink: 0; }

.tab-group-dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
}

.tab-label {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  font-size: 10px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  opacity: 0.5;
  transition: all var(--transition-fast);
  cursor: pointer;
}

.tab-close:hover {
  opacity: 1;
  background: var(--color-down-soft);
  color: var(--color-down);
}

.tab-content {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.panel-instance {
  height: 100%;
}

.empty-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-tertiary);
  font-size: var(--font-base);
}
</style>
