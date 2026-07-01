<script setup lang="ts">
interface Tab {
  key: string
  label: string
}

withDefaults(defineProps<{
  tabs: Tab[]
  active: string
  variant?: 'pill' | 'underline' | 'button'
}>(), {
  variant: 'pill',
})

defineEmits<{
  (e: 'change', key: string): void
}>()
</script>

<template>
  <div :class="['panel-tabs', `variant-${variant}`]">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      :class="['tab', { active: active === tab.key }]"
      @click="$emit('change', tab.key)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<style scoped>
.panel-tabs {
  display: flex;
  gap: var(--space-xs);
}

/* pill variant */
.panel-tabs.variant-pill .tab {
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

.panel-tabs.variant-pill .tab:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.panel-tabs.variant-pill .tab.active {
  color: var(--color-accent);
  border-color: var(--tab-active-border);
  background: var(--tab-active-bg);
}

/* underline variant */
.panel-tabs.variant-underline .tab {
  padding: 4px 0;
  font-size: var(--tab-font-size);
  border: none;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--tab-inactive-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-right: var(--space-md);
}

.panel-tabs.variant-underline .tab:hover {
  color: var(--color-text-primary);
}

.panel-tabs.variant-underline .tab.active {
  color: var(--color-accent);
  border-bottom-color: var(--color-accent);
}

/* button variant */
.panel-tabs.variant-button .tab {
  padding: var(--tab-padding);
  height: var(--tab-height);
  font-size: var(--tab-font-size);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.panel-tabs.variant-button .tab:hover {
  border-color: var(--color-border-strong);
  background: var(--color-bg-hover);
}

.panel-tabs.variant-button .tab.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-text-inverse);
}
</style>
