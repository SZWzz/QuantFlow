<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import type { DockTabState } from './types'
import { getPanelComponent } from '@/terminal/panels/registry'
import ErrorBoundary from '@/terminal/components/ErrorBoundary.vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { getIcon } from '@/lib/icons'

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

const dragTabId = ref<string | null>(null)
const dragOverTabId = ref<string | null>(null)

function onDragStart(e: DragEvent, tabId: string) {
  if (!e.dataTransfer) return
  dragTabId.value = tabId
  dragOverTabId.value = null
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', JSON.stringify({ leafId: props.leafId, tabId }))
}
function onDragEnd() {
  dragTabId.value = null
  dragOverTabId.value = null
}
function onDragOver(e: DragEvent) { e.preventDefault(); if (e.dataTransfer) e.dataTransfer.dropEffect = 'move' }
function onTabDragOver(e: DragEvent, tabId?: string) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  if (tabId && dragTabId.value !== tabId) {
    dragOverTabId.value = tabId
  }
}
function onTabDrop(e: DragEvent, toIdx: number) {
  e.preventDefault()
  if (!e.dataTransfer) return
  try {
    const data = JSON.parse(e.dataTransfer.getData('text/plain'))
    if (data.leafId === props.leafId) {
      const fromIdx = props.tabs.findIndex((t) => t.id === data.tabId)
      if (fromIdx !== -1 && fromIdx !== toIdx) {
        terminal.moveTab(props.leafId, fromIdx, toIdx)
      }
    } else {
      emit('tab-drag', data.leafId, data.tabId, props.leafId)
    }
  } catch { /* ignore */ }
  dragOverTabId.value = null
}
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
  <div :class="['dock-tab', { active: activeTab }]">
    <div class="tab-header" @dragover="onDragOver" @drop="onDrop">
      <div class="tab-list">
        <button
          v-for="(tab, idx) in tabs"
          :key="tab.id"
          :draggable="true"
          class="tab-btn"
          :class="{ active: tab.id === activeTab, dragging: dragTabId === tab.id, 'drag-over': dragOverTabId === tab.id }"
          @click="emit('select-tab', tab.id)"
          @dragstart="onDragStart($event, tab.id)"
          @dragend="onDragEnd"
          @dragover="onTabDragOver($event, tab.id)"
          @drop="onTabDrop($event, idx)"
        >
          <span class="tab-icon" v-html="tab.icon" />
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
            v-html="getIcon('close')"
          />
        </button>
      </div>
    </div>
    <div class="tab-content" @dragover="onDragOver" @drop="onDrop">
      <div v-if="tabs.length === 0" class="empty-content">
        <span class="empty-icon" v-html="getIcon('plus')" />
        Drop panels here
      </div>
      <ErrorBoundary v-else-if="activeComponent" :panel-id="activePanel?.panelId || ''">
        <component
          :is="activeComponent"
          :panel-id="activePanel?.panelId"
          :params="activeParams"
          class="panel-instance"
        />
      </ErrorBoundary>
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
  position: relative;
}

.tab-header {
  display: flex;
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border);
  min-height: 34px;
  position: relative;
}

.tab-list {
  display: flex;
  overflow-x: auto;
  flex: 1;
  padding: 4px 4px 0 4px;
  gap: 2px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px 5px 10px;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: var(--radius-md) var(--radius-md) 0 0;
  background: transparent;
  color: var(--color-text-tertiary);
  font-size: var(--font-xs);
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  min-width: 0;
  transition: all var(--transition-fast);
  user-select: none;
  position: relative;
}

/* Active tab indicator */
.tab-btn::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 8px;
  right: 8px;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--color-accent);
  opacity: 0;
  transition: opacity var(--transition-fast);
  box-shadow: 0 0 6px var(--color-accent-glow);
}

.tab-btn.active {
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
  border-color: var(--color-border);
}

.tab-btn.active::after {
  opacity: 1;
}

.tab-btn:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.tab-btn:hover:not(.active)::after {
  opacity: 0.3;
}

.tab-btn.dragging {
  opacity: 0.25;
  transform: scale(0.92);
  cursor: grabbing;
  filter: grayscale(0.5);
}

.tab-btn.drag-over {
  background: var(--color-bg-selected);
  border-color: var(--color-accent);
}

/* Drop indicator line between tabs */
.drop-indicator {
  width: 2px;
  height: 20px;
  background: var(--color-accent);
  border-radius: 1px;
  box-shadow: 0 0 6px var(--color-accent-glow);
  animation: drop-pulse 1.2s ease infinite;
  flex-shrink: 0;
  align-self: center;
}

@keyframes drop-pulse {
  0%, 100% { opacity: 0.6; transform: scaleY(1); }
  50% { opacity: 1; transform: scaleY(1.2); }
}

.tab-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.7;
  transition: opacity var(--transition-fast);
}

.tab-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.tab-btn.active .tab-icon {
  opacity: 1;
  color: var(--color-accent);
}

.tab-group-dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
  box-shadow: 0 0 4px currentColor;
}

.tab-label {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
  font-weight: 500;
}

.tab-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  opacity: 0;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
  cursor: pointer;
  margin-left: 2px;
}

.tab-close :deep(svg) {
  width: 10px;
  height: 10px;
}

.tab-btn:hover .tab-close {
  opacity: 0.5;
}

.tab-close:hover {
  opacity: 1 !important;
  background: var(--color-down-soft);
  color: var(--color-down) !important;
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
  gap: 8px;
  height: 100%;
  color: var(--color-text-tertiary);
  font-size: var(--font-base);
}

.empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  opacity: 0.5;
}

.empty-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.dock-tab.active {
  box-shadow: inset 0 0 0 1.5px var(--color-accent),
              0 0 12px var(--color-accent-glow);
  z-index: 5;
}

.dock-tab.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--color-accent), transparent);
  opacity: 0.6;
  pointer-events: none;
}
</style>
