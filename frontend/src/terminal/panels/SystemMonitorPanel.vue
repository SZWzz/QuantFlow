<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{
  panelId: string
  params?: Record<string, any>
}>()

const goRoutines = ref(0)
const memAlloc = ref('0 MB')
const memSys = ref('0 MB')
const uptime = ref('--')
const startTime = Date.now()
const dataSources = ref([
  { name: 'Yahoo Finance', status: '已连接' as const },
  { name: 'EastMoney', status: 'dis已连接' as const },
  { name: 'Binance', status: '已连接' as const },
])

let timer: ReturnType<typeof setInterval> | null = null

function update() {
  goRoutines.value = Math.floor(Math.random() * 50) + 10
  const alloc = Math.floor(Math.random() * 500) + 200
  memAlloc.value = `${alloc} MB`
  memSys.value = `${alloc + Math.floor(Math.random() * 300)} MB`

  const elapsed = Math.floor((Date.now() - startTime) / 1000)
  const h = Math.floor(elapsed / 3600)
  const m = Math.floor((elapsed % 3600) / 60)
  const s = elapsed % 60
  uptime.value = `${h}h ${m}m ${s}s`
}

onMounted(() => {
  update()
  timer = setInterval(update, 2000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function statusColor(s: string): string {
  return s === '已连接' ? '#3fb950' : s === 'error' ? '#f85149' : '#5a6380'
}
</script>

<template>
  <div class="sysmon-panel">
    <div class="section">
      <h3 class="section-title">Go 运行时</h3>
      <div class="metric-row">
        <span class="metric-label">协程数</span>
        <span class="metric-value">{{ goRoutines }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">堆内存</span>
        <span class="metric-value">{{ memAlloc }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">系统内存</span>
        <span class="metric-value">{{ memSys }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">运行时间</span>
        <span class="metric-value">{{ uptime }}</span>
      </div>
    </div>

    <div class="section">
      <h3 class="section-title">数据源</h3>
      <div
        v-for="src in dataSources"
        :key="src.name"
        class="source-row"
      >
        <span
          class="status-dot"
          :style="{ color: statusColor(src.status) }"
        >●</span>
        <span class="source-name">{{ src.name }}</span>
        <span class="source-status">{{ src.status }}</span>
      </div>
    </div>

    <div class="section">
      <h3 class="section-title">工作流引擎</h3>
      <div class="metric-row">
        <span class="metric-label">已注册节点</span>
        <span class="metric-value">5</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">缓存大小</span>
        <span class="metric-value">256</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">活跃运行</span>
        <span class="metric-value">0</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sysmon-panel {
  padding: 10px;
  background: #1a1a2e;
  height: 100%;
  overflow-y: auto;
  font-size: 12px;
}

.section {
  margin-bottom: 14px;
}

.section-title {
  font-size: 10px;
  text-transform: uppercase;
  color: #5a6380;
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid #0f3460;
}

.metric-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
}

.metric-label {
  color: #5a6380;
}

.metric-value {
  color: #c9d1d9;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.source-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 0;
}

.status-dot {
  font-size: 8px;
}

.source-name {
  flex: 1;
  color: #c9d1d9;
}

.source-status {
  font-size: 10px;
  color: #5a6380;
}
</style>
