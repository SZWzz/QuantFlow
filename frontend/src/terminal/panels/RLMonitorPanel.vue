<script setup lang="ts">
import { PanelHeader, EmptyState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()
</script>

<template>
  <div class="rl-monitor-panel">
    <PanelHeader :title="$t('ml.rl_monitor')" />
    <EmptyState
      icon="warning"
      title="需要 Python sidecar 支持当前功能"
      description="强化学习训练监控需要启动 Python 训练服务，当前 Go 后端不支持直接 RL 训练。请通过 Python sidecar 连接训练实验。"
    />
  </div>
</template>

<style scoped>
.rl-monitor-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
