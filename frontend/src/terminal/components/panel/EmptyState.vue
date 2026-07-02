<script setup lang="ts">
import { getIcon } from '@/lib/icons'
import type { IconName } from '@/lib/icons'

interface EmptyAction {
  label: string
  primary?: boolean
  handler: () => void
}

withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  actions?: EmptyAction[]
}>(), {
  icon: 'inbox',
})
</script>

<template>
  <div class="empty-state">
    <span class="empty-icon" v-html="getIcon(icon as IconName)" />
    <h4 class="empty-title">{{ title }}</h4>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <div v-if="actions && actions.length" class="empty-actions">
      <button
        v-for="(act, idx) in actions"
        :key="idx"
        :class="['btn', act.primary ? 'btn-primary' : 'btn-ghost']"
        @click="act.handler"
      >
        {{ act.label }}
      </button>
    </div>
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
  animation: empty-enter 0.4s ease-out;
}

@keyframes empty-enter {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
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

.empty-actions {
  display: flex;
  gap: var(--space-sm);
  flex-wrap: wrap;
  justify-content: center;
}
</style>
