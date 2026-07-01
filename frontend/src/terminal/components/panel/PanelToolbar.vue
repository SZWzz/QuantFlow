<script setup lang="ts">
import { ref } from 'vue'
import { getIcon } from '@/lib/icons'
import type { IconName } from '@/lib/icons'

interface Filter {
  key: string
  label: string
}

interface Action {
  label: string
  icon?: string
  handler: () => void
}

withDefaults(defineProps<{
  searchPlaceholder?: string
  filters?: Filter[]
  activeFilter?: string
  actions?: Action[]
}>(), {
  filters: () => [],
  actions: () => [],
})

defineEmits<{
  (e: 'search', val: string): void
  (e: 'filterChange', key: string): void
}>()

const searchVal = ref('')
</script>

<template>
  <div class="panel-toolbar">
    <div v-if="searchPlaceholder" class="toolbar-search">
      <input
        type="text"
        :placeholder="searchPlaceholder"
        v-model="searchVal"
        @input="$emit('search', searchVal)"
        class="toolbar-input"
      />
    </div>
    <div v-if="filters?.length" class="toolbar-filters">
      <button
        v-for="f in filters"
        :key="f.key"
        :class="['filter-btn', { active: activeFilter === f.key }]"
        @click="$emit('filterChange', f.key)"
      >
        {{ f.label }}
      </button>
    </div>
    <div v-if="actions?.length" class="toolbar-actions">
      <button
        v-for="a in actions"
        :key="a.label"
        class="btn btn-ghost"
        @click="a.handler"
      >
        <span v-if="a.icon" class="icon" v-html="getIcon(a.icon as IconName)" />
        {{ a.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.panel-toolbar {
  display: flex;
  align-items: center;
  gap: var(--toolbar-gap);
  padding: var(--toolbar-padding);
  min-height: var(--toolbar-height);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
}

.toolbar-input {
  width: 180px;
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-sm);
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  transition: all var(--transition-fast);
}

.toolbar-input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.toolbar-input::placeholder {
  color: var(--color-text-tertiary);
}

.toolbar-filters {
  display: flex;
  gap: var(--space-xs);
}

.filter-btn {
  padding: 2px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.filter-btn:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.filter-btn.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}

.toolbar-actions {
  display: flex;
  gap: var(--space-xs);
  margin-left: auto;
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
