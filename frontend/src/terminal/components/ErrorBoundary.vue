<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'

const props = defineProps<{ panelId: string }>()

const hasError = ref(false)
const errorMessage = ref('')

onErrorCaptured((err: Error) => {
  hasError.value = true
  errorMessage.value = err.message || 'Unknown error'
  return false
})

function retry() {
  hasError.value = false
  errorMessage.value = ''
}
</script>

<template>
  <div v-if="hasError" class="error-boundary">
    <div class="error-icon">⚠</div>
    <div class="error-title">{{ $t('common.panel_error') }}</div>
    <div class="error-message">{{ errorMessage }}</div>
    <div class="error-actions">
      <button class="error-btn" @click="retry">{{ $t('common.retry') }}</button>
    </div>
  </div>
  <slot v-else />
</template>

<style scoped>
.error-boundary {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 32px;
  color: var(--color-text-secondary, #9ca3af);
  background: var(--color-bg-panel, #1a1a2e);
  gap: 8px;
}
.error-icon {
  font-size: 32px;
  color: var(--color-warning, #f59e0b);
}
.error-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary, #e5e7eb);
}
.error-message {
  font-size: 12px;
  text-align: center;
  max-width: 300px;
  word-break: break-all;
  color: var(--color-text-tertiary, #6b7280);
}
.error-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}
.error-btn {
  padding: 6px 16px;
  border: 1px solid var(--color-border-strong, #374151);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated, #2a2a3e);
  color: var(--color-text-primary, #e5e7eb);
  cursor: pointer;
  font-size: 12px;
}
.error-btn:hover {
  background: var(--color-bg-hover, #3a3a4e);
}
</style>
