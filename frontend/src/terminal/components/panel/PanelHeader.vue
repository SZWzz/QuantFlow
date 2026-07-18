<script setup lang="ts">
import PanelTabs from './PanelTabs.vue'
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
    <PanelTabs
      v-if="tabs?.length"
      :tabs="tabs"
      :active="activeTab ?? ''"
      variant="underline"
      @change="(key: string) => $emit('tabChange', key)"
    />
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
  border-bottom: 1px solid var(--color-border-subtle);
  min-height: var(--panel-header-height);
  flex-wrap: wrap;
  flex-shrink: 0;
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
