<script setup lang="ts">
defineProps<{
  state: 'loading' | 'loaded' | 'error' | 'empty'
  error?: string
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="panel-shell" role="region" :aria-busy="state === 'loading'">
    <!-- Loading -->
    <div v-if="state === 'loading'" class="panel-shell-loading" data-testid="panel-loading">
      <div class="panel-shell-spinner" aria-label="加载中" />
      <span class="panel-shell-text">加载中…</span>
    </div>

    <!-- Error -->
    <div v-else-if="state === 'error'" class="panel-shell-error" data-testid="panel-error">
      <span class="panel-shell-error-icon" aria-hidden="true">⚠</span>
      <p class="panel-shell-error-message">{{ error }}</p>
      <button
        class="panel-shell-retry-btn"
        data-testid="panel-retry-btn"
        tabindex="0"
        @click="$emit('retry')"
      >
        重试
      </button>
    </div>

    <!-- Empty -->
    <div v-else-if="state === 'empty'" class="panel-shell-empty" data-testid="panel-empty">
      <slot name="empty">
        <span class="panel-shell-empty-default">暂无数据</span>
      </slot>
    </div>

    <!-- Loaded content -->
    <div v-else class="panel-shell-loaded" data-testid="panel-loaded">
      <slot name="loaded" />
    </div>
  </div>
</template>

<style scoped>
.panel-shell {
  display: flex;
  flex-direction: column;
  min-height: 120px;
}

.panel-shell-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--muted, #888);
  font-size: var(--font-sm, 13px);
}

.panel-shell-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border, #e0e0e0);
  border-top-color: var(--accent, #4a90d9);
  border-radius: 50%;
  animation: panel-shell-spin 0.7s linear infinite;
}

@keyframes panel-shell-spin {
  to { transform: rotate(360deg); }
}

.panel-shell-text { user-select: none; }

.panel-shell-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--danger, #d32f2f);
  font-size: var(--font-sm, 13px);
}

.panel-shell-error-icon { font-size: 20px; }

.panel-shell-error-message {
  margin: 0;
  text-align: center;
  word-break: break-word;
  color: var(--muted, #888);
}

.panel-shell-retry-btn {
  padding: 4px 14px;
  border: 1px solid var(--border, #e0e0e0);
  border-radius: var(--radius-sm, 4px);
  background: var(--surface, #f5f5f5);
  color: var(--text, #222);
  cursor: pointer;
  font-size: var(--font-sm, 13px);
  outline: none;
}

.panel-shell-retry-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--accent, #4a90d9);
}

.panel-shell-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  color: var(--muted, #888);
  font-size: var(--font-sm, 13px);
}
</style>
