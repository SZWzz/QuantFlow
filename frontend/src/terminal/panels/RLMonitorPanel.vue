<script setup lang="ts">
defineProps<{ panelId: string; params?: Record<string, any> }>()
</script>

<template>
  <div class="rl-monitor-panel">
    <div class="placeholder-content">
      <div class="placeholder-title">{{ $t('ml.rl_monitor') }}</div>
      <div class="placeholder-message">
        需要 Python sidecar 支持当前功能
      </div>
      <div class="placeholder-detail">
        强化学习训练监控需要启动 Python 训练服务，当前 Go 后端不支持直接 RL 训练。
        请通过 Python sidecar 连接训练实验。
      </div>
    </div>
  </div>
</template>

<style scoped>
.rl-monitor-panel {
  padding: 12px;
  background: var(--bg);
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder-content {
  text-align: center;
  max-width: 320px;
}

.placeholder-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 12px;
}

.placeholder-message {
  font-size: 14px;
  color: var(--color-accent);
  margin-bottom: 8px;
  padding: 8px 16px;
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
  background: rgba(240, 136, 62, 0.1);
}

.placeholder-detail {
  font-size: 12px;
  color: var(--muted);
  line-height: 1.6;
}
</style>
