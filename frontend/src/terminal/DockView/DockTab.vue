<script setup lang="ts">
import { ref, computed, type Component } from 'vue'
import type { DockTabState } from './types'
import { getPanelComponent } from '@/terminal/panels/registry'
import { useTerminalStore } from '@/stores/terminal'

const props = defineProps<{
  tabs: DockTabState[]
  activeTab: string
  leafId: string
}>()

const emit = defineEmits<{
  (e: 'select-tab', tabId: string): void
  (e: 'close-tab', tabId: string): void
  (e: 'move-tab', fromIdx: number, toIdx: number): void
}>()

const terminal = useTerminalStore()
const dragIdx = ref<number | null>(null)
const dragOverIdx = ref<number | null>(null)

const activePanel = computed(() => props.tabs.find((t) => t.id === props.activeTab))

const activeComponent = computed<Component | undefined>(() => {
  if (!activePanel.value) return undefined
  return getPanelComponent(activePanel.value.panelId)
})

const activeParams = computed(() => activePanel.value?.params)

function onCloseTab(tabId: string, event: MouseEvent) {
  event.stopPropagation()
  emit('close-tab', tabId)
}

function onDragStart(idx: number) {
  dragIdx.value = idx
}

function onDragOver(idx: number, event: DragEvent) {
  event.preventDefault()
  dragOverIdx.value = idx
}

function onDragLeave() {
  dragOverIdx.value = null
}

function onDrop(idx: number) {
  if (dragIdx.value !== null && dragIdx.value !== idx) {
    emit('move-tab', dragIdx.value, idx)
  }
  dragIdx.value = null
  dragOverIdx.value = null
}

function onDragEnd() {
  dragIdx.value = null
  dragOverIdx.value = null
}
</script>

<template>
  <div class="dock-tab">
    <div class="tab-header">
      <div class="tab-list">
        <button
          v-for="(tab, idx) in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{
            active: tab.id === activeTab,
            dragging: dragIdx === idx,
            'drag-over': dragOverIdx === idx,
          }"
          :draggable="true"
          @click="emit('select-tab', tab.id)"
          @dragstart="onDragStart(idx)"
          @dragover="onDragOver(idx, $event)"
          @dragleave="onDragLeave"
          @drop="onDrop(idx)"
          @dragend="onDragEnd"
        >
          <span class="tab-icon">{{ tab.icon }}</span>
          <span class="tab-label">{{ tab.label }}</span>
          <span
            class="tab-close"
            @mousedown="onCloseTab(tab.id, $event)"
            title="Close"
          >✕</span>
        </button>
      </div>
      <div class="tab-actions">
        <button
          v-if="tabs.length > 0"
          class="close-active-btn"
          @click="emit('close-tab', activeTab)"
          title="Close active panel"
        >
          ✕
        </button>
      </div>
    </div>
    <div class="tab-content">
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
  border-radius: var(--radius-sm);
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
  background: transparent;
  color: var(--color-text-tertiary);
  font-size: var(--font-xs);
  cursor: pointer;
  white-space: nowrap;
  border-right: 1px solid var(--color-border);
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

.tab-btn.dragging {
  opacity: 0.4;
}

.tab-btn.drag-over {
  border-left: 2px solid var(--color-accent);
}

.tab-icon { font-size: 12px; flex-shrink: 0; }

.tab-label {
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

.tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  font-size: 10px;
  border-radius: 3px;
  flex-shrink: 0;
  opacity: 0.4;
  transition: all var(--transition-fast);
}

.tab-close:hover {
  opacity: 1;
  background: var(--color-down-soft);
  color: var(--color-down);
}

.tab-actions {
  display: flex;
  align-items: center;
  padding: 0 4px;
}

.close-active-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  border-radius: var(--radius-sm);
  font-size: 12px;
  transition: all var(--transition-fast);
}

.close-active-btn:hover {
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
