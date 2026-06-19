<script setup lang="ts">
import { computed, type Component } from 'vue'
import type { DockTabState } from './types'
import { getPanelComponent } from '@/terminal/panels/registry'

const props = defineProps<{
  tabs: DockTabState[]
  activeTab: string
}>()

const emit = defineEmits<{
  (e: 'select-tab', tabId: string): void
  (e: 'close-tab', tabId: string): void
}>()

const activePanel = computed(() => props.tabs.find((t) => t.id === props.activeTab))

const activeComponent = computed<Component | undefined>(() => {
  if (!activePanel.value) return undefined
  return getPanelComponent(activePanel.value.panelId)
})

const activeParams = computed(() => activePanel.value?.params)
</script>

<template>
  <div class="dock-tab">
    <div class="tab-header">
      <div class="tab-list">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: tab.id === activeTab }"
          @click="emit('select-tab', tab.id)"
        >
          <span class="tab-icon">{{ tab.icon }}</span>
          <span class="tab-label">{{ tab.label }}</span>
          <span
            class="tab-close"
            @click.stop="emit('close-tab', tab.id)"
          >✕</span>
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
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border);
}

.tab-list {
  display: flex;
  overflow-x: auto;
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
}

.tab-btn.active {
  background: var(--color-bg-panel);
  color: var(--color-text-primary);
}

.tab-btn:hover {
  color: var(--color-text-primary);
}

.tab-icon {
  font-size: 12px;
}

.tab-close {
  font-size: 10px;
  opacity: 0;
  padding: 2px;
  border-radius: 2px;
  transition: opacity var(--transition-fast);
}

.tab-btn:hover .tab-close {
  opacity: 0.6;
}

.tab-close:hover {
  opacity: 1;
  color: var(--color-brand);
}

.tab-content {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.empty-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-tertiary);
  font-size: var(--font-base);
}

.panel-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-tertiary);
  font-size: var(--font-base);
}
</style>
