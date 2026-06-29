<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useDataFetch } from '@/lib/composables/useDataFetch'
import { usePanelCache } from '@/lib/composables/usePanelCache'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { fetchWithCache } = usePanelCache()
const statsFetcher = useDataFetch(async () => {
  const { data } = await fetchWithCache<any>('system_stats', () => (window as any).go.main.App.GetSystemStats(), 5000)
  return data
})

let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  statsFetcher.execute()
  timer = setInterval(() => statsFetcher.execute(), 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="sysmon-panel">
    <div class="section">
      <h3 class="section-title">{{ $t('monitor.go_runtime') }}</h3>

      <div v-if="statsFetcher.loading.value" class="stat-loading">加载中...</div>
      <div v-else-if="statsFetcher.error.value" class="stat-error">错误: {{ statsFetcher.error.value }}</div>
      <div v-else-if="!statsFetcher.data.value" class="stat-empty">--</div>
      <template v-else>
        <div class="metric-row">
          <span class="metric-label">{{ $t('monitor.goroutines') }}</span>
          <span class="metric-value">{{ statsFetcher.data.value.goroutines || 0 }}</span>
        </div>
        <div class="metric-row">
          <span class="metric-label">{{ $t('monitor.heap_memory') }}</span>
          <span class="metric-value">{{ statsFetcher.data.value.mem_alloc_mb || 0 }} MB</span>
        </div>
        <div class="metric-row">
          <span class="metric-label">{{ $t('monitor.system_memory') }}</span>
          <span class="metric-value">{{ statsFetcher.data.value.mem_sys_mb || 0 }} MB</span>
        </div>
        <div class="metric-row">
          <span class="metric-label">{{ $t('monitor.uptime') }}</span>
          <span class="metric-value">{{ statsFetcher.data.value.uptime_seconds || 0 }}s</span>
        </div>
      </template>
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

.stat-loading {
  color: var(--color-text-tertiary);
  padding: 8px 0;
  font-style: italic;
}

.stat-error {
  color: var(--color-error);
  padding: 8px 0;
  font-size: 11px;
}

.stat-empty {
  color: var(--color-text-tertiary);
  padding: 8px 0;
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
