<script setup lang="ts">
import { getIcon } from '@/lib/icons'

withDefaults(defineProps<{
  title?: string
  description?: string
  retryLabel?: string
}>(), {
  title: '加载失败',
  retryLabel: '重试',
})

defineEmits<{
  (e: 'retry'): void
}>()
</script>

<template>
  <div class="error-state">
    <span class="error-icon" v-html="getIcon('warning')" />
    <h4 class="error-title">{{ title }}</h4>
    <p v-if="description" class="error-desc">{{ description }}</p>
    <button class="btn error-retry" @click="$emit('retry')">{{ retryLabel }}</button>
  </div>
</template>

<style scoped>
.error-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  text-align: center;
}

.error-icon {
  display: inline-flex;
  width: 32px;
  height: 32px;
  color: var(--color-danger);
}

.error-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.error-title {
  font-size: var(--font-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  margin: 0;
}

.error-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin: 0;
  max-width: 280px;
  line-height: 1.6;
}
</style>
