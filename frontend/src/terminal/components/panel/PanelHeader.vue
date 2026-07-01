<script setup lang="ts">
import { getIcon } from '@/lib/icons'
import type { IconName } from '@/lib/icons'

interface Tab {
  key: string
  label: string
}

interface Control {
  icon?: string
  label?: string
  title?: string
  action: () => void
  loading?: boolean
}

withDefaults(defineProps<{
  title?: string
  subtitle?: string
  tabs?: Tab[]
  activeTab?: string
  controls?: Control[]
}>(), {
  tabs: () => [],
  controls: () => [],
})

defineEmits<{
  (e: 'tabChange', key: string): void
}>()
</script>

<template>
  <div class="panel-header">
    <div class="header-left">
      <h3 v-if="title" class="panel-title">{{ title }}</h3>
      <span v-if="subtitle" class="panel-subtitle">{{ subtitle }}</span>
    </div>
    <div v-if="tabs?.length" class="header-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tab-btn', { active: activeTab === tab.key }]"
        @click="$emit('tabChange', tab.key)"
      >
        {{ tab.label }}
      </button>
    </div>
    <div v-if="controls?.length" class="header-controls">
      <button
        v-for="ctrl in controls"
        :key="ctrl.icon || ctrl.label"
        :class="['btn btn-ghost', { loading: ctrl.loading }]"
        @click="ctrl.action"
        :title="ctrl.title"
      >
        <span v-if="ctrl.icon" class="icon" v-html="getIcon(ctrl.icon as IconName)" />
        <span v-if="ctrl.label">{{ ctrl.label }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.panel-header {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border);
  min-height: var(--toolbar-height);
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: var(--space-sm);
  flex: 1;
  min-width: 0;
}

.panel-title {
  margin: 0;
  font-size: var(--panel-title-size);
  font-weight: var(--panel-title-weight);
  color: var(--color-text-primary);
  white-space: nowrap;
}

.panel-subtitle {
  font-size: var(--panel-subtitle-size);
  color: var(--panel-subtitle-color);
  white-space: nowrap;
}

.header-tabs {
  display: flex;
  gap: var(--space-xs);
  flex-shrink: 0;
}

.tab-btn {
  padding: var(--tab-padding);
  height: var(--tab-height);
  font-size: var(--tab-font-size);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--tab-inactive-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.tab-btn:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.tab-btn.active {
  color: var(--color-accent);
  border-color: var(--tab-active-border);
  background: var(--tab-active-bg);
}

.header-controls {
  display: flex;
  gap: var(--space-xs);
  align-items: center;
  flex-shrink: 0;
}

.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
}

.icon :deep(svg) {
  width: 100%;
  height: 100%;
}
</style>
