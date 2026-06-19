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
  background: #1a1a2e;
  border-radius: 4px;
  overflow: hidden;
}

.tab-header {
  background: #16213e;
  border-bottom: 1px solid #0f3460;
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
  color: #5a6380;
  font-size: 11px;
  cursor: pointer;
  white-space: nowrap;
  border-right: 1px solid #0f3460;
  min-width: 0;
}

.tab-btn.active {
  background: #1a1a2e;
  color: #c9d1d9;
}

.tab-btn:hover {
  color: #c9d1d9;
}

.tab-icon {
  font-size: 12px;
}

.tab-close {
  font-size: 10px;
  opacity: 0;
  padding: 2px;
  border-radius: 2px;
}

.tab-btn:hover .tab-close {
  opacity: 0.6;
}

.tab-close:hover {
  opacity: 1;
  color: #e94560;
}

.tab-content {
  flex: 1;
  overflow: auto;
}

.empty-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #2a3a5c;
  font-size: 13px;
}

.panel-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #3a4a6c;
  font-size: 13px;
}
</style>
