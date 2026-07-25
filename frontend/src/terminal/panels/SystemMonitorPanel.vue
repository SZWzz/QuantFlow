<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useDataFetch } from '@/lib/composables/useDataFetch'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader, LoadingState, ErrorState, EmptyState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { fetchWithCache } = usePanelCache()
const app = useWailsApp()
const statsFetcher = useDataFetch(async () => {
  const { data } = await fetchWithCache<any>('system_stats', () => app?.GetSystemStats(), 5000)
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
    <PanelHeader title="系统监控" />

    <LoadingState v-if="statsFetcher.loading.value" type="card" :rows="4" />
    <ErrorState v-else-if="statsFetcher.error.value" :description="statsFetcher.error.value" @retry="statsFetcher.execute" />
    <EmptyState v-else-if="!statsFetcher.data.value" title="暂无数据" />
    <div v-else class="sysmon-content">
      <div class="section">
        <h4 class="section-title">{{ $t('monitor.go_runtime') }}</h4>

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
      </div>

      <div class="section">
        <h4 class="section-title">{{ $t('monitor.workflow_engine') }}</h4>
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
  </div>
</template>

<style scoped>
.sysmon-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.sysmon-content { flex: 1; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }
.section { margin-bottom: var(--space-lg); }
.section > .section-title { display: block; margin-bottom: var(--space-sm); text-transform: uppercase; letter-spacing: 0.5px; }
.metric-row { display: flex; justify-content: space-between; padding: var(--space-xs) 0; }
.metric-label { color: var(--color-text-tertiary); font-size: var(--font-xs); }
.metric-value { color: var(--color-text-primary); font-weight: 500; font-variant-numeric: tabular-nums; font-size: var(--font-xs); }
</style>
