<script setup lang="ts">
import { getIcon } from '@/lib/icons'
import type { IconName } from '@/lib/icons'

withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  action?: { label: string; handler: () => void }
}>(), {
  icon: 'inbox',
})
</script>

<template>
  <div class="empty-state">
    <span class="empty-icon" v-html="getIcon(icon as IconName)" />
    <h4 class="empty-title">{{ title }}</h4>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <button v-if="action" class="btn btn-primary" @click="action.handler">
      {{ action.label }}
    </button>
  </div>
</template>

<style scoped>
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  text-align: center;
}

.empty-icon {
  display: inline-flex;
  width: 48px;
  height: 48px;
  color: var(--color-text-tertiary);
  opacity: 0.5;
}

.empty-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.empty-title {
  font-size: var(--font-lg);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0;
}

.empty-desc {
  font-size: var(--font-sm);
  color: var(--color-text-tertiary);
  margin: 0;
  max-width: 280px;
  line-height: 1.5;
}
</style>
