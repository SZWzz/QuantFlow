<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useDataStore } from '@/stores/data'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'

const data = useDataStore()
const workflow = useWorkflowStore()
const session = useSessionStore()

const time = ref(new Date().toLocaleTimeString())
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => { timer = setInterval(() => time.value = new Date().toLocaleTimeString(), 1000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <div class="status-bar">
    <div class="status-left">
      <span class="status-item connected" :class="{ offline: data.isOffline }">
        ● {{ data.isOffline ? 'Offline' : 'Connected' }}
      </span>
      <span class="status-item">Mode: {{ session.ui.mode }}</span>
    </div>
    <div class="status-center">
      <span class="status-item">WF: {{ workflow.executionStatus }}</span>
    </div>
    <div class="status-right">
      <span class="status-item">{{ time }}</span>
      <span class="status-item">Phase 2</span>
    </div>
  </div>
</template>

<style scoped>
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 10px;
  background: #16213e;
  border-top: 1px solid #0f3460;
  font-size: 10px;
  color: #5a6380;
  min-height: 22px;
  user-select: none;
}
.status-left, .status-center, .status-right { display: flex; gap: 12px; align-items: center; }
.status-item { font-variant-numeric: tabular-nums; }
.status-item.connected { color: #3fb950; }
.status-item.connected.offline { color: #f85149; }
</style>
