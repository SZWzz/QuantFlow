<script setup lang="ts">
import { useToast } from '@/lib/composables/useToast'

const { toasts, removeToast } = useToast()

const typeColors: Record<string, { bg: string; border: string; icon: string }> = {
  info: { bg: 'var(--color-info-soft)', border: 'var(--color-info)', icon: 'ℹ️' },
  success: { bg: 'var(--color-success-soft)', border: 'var(--color-success)', icon: '✅' },
  warning: { bg: 'var(--color-warning-soft)', border: 'var(--color-warning)', icon: '⚠️' },
  error: { bg: 'var(--color-danger-soft)', border: 'var(--color-danger)', icon: '❌' },
}
</script>

<template>
  <div class="toast-container">
    <div
      v-for="toast in toasts"
      :key="toast.id"
      class="toast-item"
      data-test="toast"
      :style="{
        background: typeColors[toast.type].bg,
        borderColor: typeColors[toast.type].border,
      }"
    >
      <span class="toast-icon">{{ typeColors[toast.type].icon }}</span>
      <div class="toast-content">
        <div class="toast-title">{{ toast.title }}</div>
        <div class="toast-message">{{ toast.message }}</div>
        <span v-if="toast.action" class="toast-action" @click="toast.action.onClick">
          {{ toast.action.label }}
        </span>
      </div>
      <button
        v-if="toast.duration === 0"
        class="toast-dismiss"
        data-test="toast-dismiss"
        @click="removeToast(toast.id)"
      >✕</button>
    </div>
  </div>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: 12px;
  right: 12px;
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 380px;
  pointer-events: none;
}
.toast-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  border: 1px solid;
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  pointer-events: auto;
  animation: slideIn 0.25s ease-out;
}
.toast-icon { font-size: 18px; flex-shrink: 0; margin-top: 1px; }
.toast-content { flex: 1; min-width: 0; }
.toast-title { font-weight: 600; font-size: 13px; margin-bottom: 2px; }
.toast-message { font-size: 12px; color: var(--color-text-secondary); word-break: break-word; }
.toast-action { font-size: 12px; color: var(--color-accent); cursor: pointer; font-weight: 600; }
.toast-dismiss {
  background: none; border: none; color: var(--color-text-tertiary);
  cursor: pointer; font-size: 14px; padding: 0; line-height: 1;
}
@keyframes slideIn {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
</style>
