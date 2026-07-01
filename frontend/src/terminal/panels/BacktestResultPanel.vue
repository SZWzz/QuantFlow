<script setup lang="ts">
import { EmptyState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()
</script>

<template>
  <div class="backtest-panel">
    <EmptyState
      icon="activity"
      :title="$t('workflow.backtest_empty')"
      :description="$t('workflow.backtest_desc') || '在 Workflow Editor 中构建并运行回测工作流后，结果将在此显示。'"
    />
  </div>
</template>

<style scoped>
.backtest-panel {
  padding: var(--panel-padding);
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-panel);
}
</style>
