<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const goRoutines = ref(0)
const memAlloc = ref('0 MB')
const memSys = ref('0 MB')
const numGC = ref(0)
const uptime = ref('--')
const goVersion = ref('')

let timer: ReturnType<typeof setInterval> | null = null

async function update() {
  const app = (window as any).go?.main?.App
  if (!app?.GetSystemStats) return
  try {
    const s = await app.GetSystemStats()
    goRoutines.value = s.goroutines || 0
    memAlloc.value = `${s.mem_alloc_mb || 0} MB`
    memSys.value = `${s.mem_sys_mb || 0} MB`
    numGC.value = s.num_gc || 0
    goVersion.value = s.go_version || ''
    const sec = s.uptime_seconds || 0
    const h = Math.floor(sec / 3600), m = Math.floor((sec % 3600) / 60), s2 = sec % 60
    uptime.value = `${h}h ${m}m ${s2}s`
  } catch {}
}

onMounted(() => { update(); timer = setInterval(update, 5000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<template>
  <div class="sysmon-panel">
    <div class="section">
      <h3 class="section-title">{{ $t('monitor.go_runtime') }}</h3>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.goroutines') }}</span>
        <span class="metric-value">{{ goRoutines }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.heap_memory') }}</span>
        <span class="metric-value">{{ memAlloc }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.system_memory') }}</span>
        <span class="metric-value">{{ memSys }}</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.uptime') }}</span>
        <span class="metric-value">{{ uptime }}</span>
      </div>
    </div>

    <div class="section">
      <h3 class="section-title">{{ $t('monitor.workflow_engine') }}</h3>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.registered_nodes') }}</span>
        <span class="metric-value">5</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.cache_size') }}</span>
        <span class="metric-value">256</span>
      </div>
      <div class="metric-row">
        <span class="metric-label">{{ $t('monitor.active_runs') }}</span>
        <span class="metric-value">0</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sysmon-panel {
  padding: 10px;
  background: var(--color-bg-panel);
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
  color: var(--color-text-tertiary);
  letter-spacing: 0.5px;
  margin-bottom: 6px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--color-accent-soft);
}

.metric-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
}

.metric-label {
  color: var(--color-text-tertiary);
}

.metric-value {
  color: var(--color-text-primary);
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
  color: var(--color-text-primary);
}

.source-status {
  font-size: 10px;
  color: var(--color-text-tertiary);
}
</style>
