<script setup lang="ts">
import { useUpdateStore } from '@/stores/update'

const props = defineProps<{
  visible: boolean
  currentVersion: string
  latestVersion: string
  changelog: string
}>()

const emit = defineEmits<{
  close: []
  update: []
}>()

const store = useUpdateStore()
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="update-overlay" @click.self="emit('close')">
      <div class="update-dialog">
        <div class="update-header">
          <span class="update-icon">📦</span>
          <h2>新版本 {{ latestVersion }} 可用</h2>
        </div>
        <div class="update-body">
          <p class="update-version-info">
            当前版本: <strong>{{ currentVersion }}</strong>
            → <strong class="latest">{{ latestVersion }}</strong>
          </p>
          <div v-if="changelog" class="update-changelog">
            <h3>更新内容</h3>
            <pre>{{ changelog }}</pre>
          </div>
          <div v-if="store.downloading" class="update-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: store.downloadProgress + '%' }" />
            </div>
            <span>{{ store.downloadProgress }}%</span>
          </div>
          <div v-if="store.error" class="update-error">
            {{ store.error }}
          </div>
        </div>
        <div class="update-actions">
          <button class="btn btn-secondary" data-test="remind-later" @click="emit('close')">
            稍后提醒
          </button>
          <button class="btn btn-secondary" @click="emit('close')">
            查看更新
          </button>
          <button class="btn btn-primary" :disabled="store.downloading" @click="emit('update')">
            {{ store.downloading ? '下载中...' : '更新' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.update-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.update-dialog {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 480px;
  max-width: 90vw;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}
.update-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.update-header h2 {
  margin: 0;
  font-size: 16px;
}
.update-body {
  padding: 16px 20px;
}
.update-version-info {
  margin: 0 0 12px;
}
.latest { color: var(--accent); }
.update-changelog {
  background: var(--bg-code);
  border-radius: 4px;
  padding: 8px 12px;
  max-height: 200px;
  overflow-y: auto;
  margin-bottom: 12px;
}
.update-changelog h3 {
  font-size: 12px;
  margin: 0 0 8px;
  color: var(--text-secondary);
}
.update-changelog pre {
  margin: 0;
  font-size: 12px;
  white-space: pre-wrap;
  line-height: 1.5;
}
.update-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--bg-muted);
  border-radius: 3px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s;
}
.update-error {
  color: var(--error);
  font-size: 12px;
  margin-bottom: 8px;
}
.update-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
}
.btn {
  padding: 6px 16px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid transparent;
}
.btn-secondary {
  background: var(--bg-muted);
  color: var(--text);
  border-color: var(--border);
}
.btn-primary {
  background: var(--accent);
  color: #fff;
}
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
