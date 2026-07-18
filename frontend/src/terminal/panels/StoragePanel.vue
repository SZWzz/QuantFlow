<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetStorageStats, ArchiveData, ExportData, CleanupData, confirmDialog, type TableStat } from '@/lib/wails'
import { usePanelCache } from '@/lib/composables/usePanelCache'
import { useDataFetch } from '@/lib/composables/useDataFetch'

defineProps<{ panelId?: string; params?: Record<string, any> }>()

const { fetchWithCache } = usePanelCache()
const statsFetcher = useDataFetch(async () => {
  const { data } = await fetchWithCache<TableStat[]>('storage_stats', () => GetStorageStats(), 30000)
  return data
})

const error = ref('')

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatTableLabel(table: string): string {
  const labels: Record<string, string> = {
    ohlcv_cache: 'OHLCV 缓存',
    minute_cache: '分时缓存',
    data_archive: '数据归档',
  }
  return labels[table] || table
}

async function handleArchive(source: string) {
  const ok = await confirmDialog(`确认归档 ${formatTableLabel(source)} 数据？`)
  if (!ok) return
  try {
    await ArchiveData(source, '', '')
    error.value = '归档完成'
    statsFetcher.execute()
  } catch (e: any) {
    error.value = e.message || '归档失败'
  }
}

async function handleExport(table: string) {
  const ok = await confirmDialog(`确认导出 ${formatTableLabel(table)} 数据？`)
  if (!ok) return
  try {
    const path = await ExportData(table, '', '', 'csv', '', '')
    error.value = `导出完成: ${path}`
  } catch (e: any) {
    error.value = e.message || '导出失败'
  }
}

async function handleCleanup(table: string) {
  const ok = await confirmDialog(`确认清理 ${formatTableLabel(table)} 数据？此操作不可恢复。`, '警告')
  if (!ok) return
  try {
    const result = await CleanupData(table, '', '', false)
    error.value = `已清理 ${result.affected_rows} 行`
    statsFetcher.execute()
  } catch (e: any) {
    error.value = e.message || '清理失败'
  }
}

onMounted(() => statsFetcher.execute())
</script>

<template>
  <div class="storage-panel">
    <div class="panel-header">
      <span class="panel-title">{{ $t('storage.title') }}</span>
      <button class="refresh-btn" @click="statsFetcher.execute()" :disabled="statsFetcher.loading.value">
        ↻
      </button>
    </div>

    <div class="storage-content">
      <div v-if="statsFetcher.loading.value" class="loading-state" data-testid="loading">
        <div class="skeleton-row" v-for="i in 3" :key="i"></div>
      </div>

      <div v-else-if="statsFetcher.error.value" class="error-state">
        {{ statsFetcher.error.value }}
      </div>

      <div v-else-if="error" class="status-msg">{{ error }}</div>

      <table v-if="statsFetcher.data.value" class="storage-table">
        <thead>
          <tr>
            <th>{{ $t('storage.table') }}</th>
            <th class="num">{{ $t('storage.rows') }}</th>
            <th class="num">{{ $t('storage.size') }}</th>
            <th>{{ $t('storage.oldest') }}</th>
            <th>{{ $t('storage.newest') }}</th>
            <th>{{ $t('storage.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in statsFetcher.data.value" :key="s.table">
            <td>{{ formatTableLabel(s.table) }}</td>
            <td class="num">{{ s.rows.toLocaleString() }}</td>
            <td class="num">{{ formatBytes(s.size_bytes) }}</td>
            <td>{{ s.oldest || '-' }}</td>
            <td>{{ s.newest || '-' }}</td>
            <td class="actions">
              <button
                v-if="s.table === 'ohlcv_cache' || s.table === 'minute_cache'"
                @click="handleExport(s.table)"
                :title="$t('storage.export')"
              >⬇</button>
              <button
                v-if="s.table === 'ohlcv_cache' || s.table === 'minute_cache'"
                @click="handleArchive(s.table)"
                :title="$t('storage.archive')"
              >📦</button>
              <button
                v-if="s.table === 'ohlcv_cache' || s.table === 'minute_cache'"
                @click="handleCleanup(s.table)"
                :title="$t('storage.cleanup')"
                class="danger"
              >🗑</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.storage-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  font-size: 13px;
}

.panel-title {
  font-weight: 600;
}
.refresh-btn {
  background: none;
  border: 1px solid var(--border-color, #444);
  border-radius: 4px;
  cursor: pointer;
  padding: 2px 8px;
  font-size: 14px;
}
.storage-content {
  flex: 1;
  overflow: auto;
  padding: 8px 12px;
}
.storage-table {
  width: 100%;
  border-collapse: collapse;
}
.storage-table th,
.storage-table td {
  padding: 6px 8px;
  text-align: left;
  border-bottom: 1px solid var(--border-color, #333);
}
.storage-table th {
  font-weight: 600;
  color: var(--text-secondary, #888);
  position: sticky;
  top: 0;
  background: var(--color-bg-panel);
  white-space: nowrap;
}
.storage-table td.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.storage-table td.actions {
  white-space: nowrap;
}
.storage-table button {
  background: none;
  border: 1px solid var(--border-color, #444);
  border-radius: 4px;
  cursor: pointer;
  padding: 2px 6px;
  margin: 0 2px;
  font-size: 14px;
}
.storage-table button.danger:hover {
  background: rgba(255, 50, 50, 0.2);
  border-color: #f44;
}
.loading-state {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 0;
}
.skeleton-row {
  height: 32px;
  background: var(--border-color, #333);
  border-radius: 4px;
  opacity: 0.3;
}
.error-state,
.status-msg {
  padding: 12px;
  color: var(--text-error, #f44);
}
</style>
