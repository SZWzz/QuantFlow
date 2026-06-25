<script setup lang="ts">
defineProps<{ panelId: string; params?: Record<string, any> }>()
</script>

<template>
  <div class="backtest-panel">
    <div class="empty-state">
      <p class="empty-title">{{ $t('workflow.backtest_empty') }}</p>
      <p class="empty-desc">在 Workflow Editor 中构建并运行回测工作流后，结果将在此显示。</p>
    </div>
  </div>
</template>

<style scoped>
.backtest-panel { padding: 16px; height: 100%; display: flex; align-items: center; justify-content: center; color: var(--color-text, #e5e7eb); background: var(--color-bg, var(--color-bg-panel)); }
.empty-state { text-align: center; }
.empty-title { font-size: 16px; font-weight: 600; margin-bottom: 8px; }
.empty-desc { font-size: 13px; color: var(--color-text-tertiary); }
</style>
