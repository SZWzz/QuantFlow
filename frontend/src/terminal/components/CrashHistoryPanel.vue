<script setup lang="ts">
/**
 * CrashHistoryPanel — lists past crash reports (Settings → 崩溃报告).
 *
 * Allows expanding a report to view its panic/stack/logs, opt-in uploading,
 * and deleting old reports. Data comes from the crash Pinia store backed by
 * the Go crash reporter IPC (ListCrashReports / DeleteCrashReport /
 * UploadCrashReport).
 */
import { ref, onMounted } from 'vue'
import { useCrashStore, type CrashReport } from '@/stores/crash'
import { confirmDialog, alertDialog } from '@/lib/wails'

const store = useCrashStore()
const expandedId = ref<string | null>(null)
const uploadingId = ref<string | null>(null)

onMounted(() => {
  store.list()
})

function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function shortPanic(report: CrashReport): string {
  const p = report.panic || '(unknown)'
  return p.length > 80 ? p.slice(0, 80) + '…' : p
}

async function onDelete(report: CrashReport) {
  // window.confirm is disabled in the Wails v3 webview — must await confirmDialog.
  const ok = await confirmDialog(`删除 ${formatTime(report.timestamp)} 的崩溃报告？`)
  if (!ok) return
  try {
    await store.remove(report.id)
  } catch {
    await alertDialog('删除崩溃报告失败')
  }
}

async function onUpload(report: CrashReport) {
  const ok = await confirmDialog('上传匿名崩溃报告帮助改进？报告不含 API 密钥或持仓明细。')
  if (!ok) return
  uploadingId.value = report.id
  try {
    await store.upload(report.id)
    await alertDialog('崩溃报告已上传')
  } finally {
    uploadingId.value = null
  }
}
</script>

<template>
  <div class="crash-history">
    <div class="crash-history-header">
      <span class="crash-history-title">💥 崩溃历史 ({{ store.reports.length }} 次)</span>
      <button class="btn btn-small" data-test="refresh" :disabled="store.loading" @click="store.list()">
        {{ store.loading ? '加载中…' : '刷新' }}
      </button>
    </div>

    <div v-if="!store.loading && store.reports.length === 0" class="crash-empty">
      暂无崩溃记录
    </div>

    <div v-for="report in store.reports" :key="report.id" class="crash-item">
      <div class="crash-item-row" @click="toggleExpand(report.id)">
        <div class="crash-item-info">
          <span class="crash-item-time">{{ formatTime(report.timestamp) }}</span>
          <span class="crash-item-panic">{{ shortPanic(report) }}</span>
          <span class="crash-item-meta">v{{ report.version }} · {{ report.os }}/{{ report.arch }}</span>
        </div>
        <div class="crash-item-actions" @click.stop>
          <button class="btn btn-small" data-test="view" @click="toggleExpand(report.id)">
            {{ expandedId === report.id ? '收起' : '查看' }}
          </button>
          <button
            class="btn btn-small"
            data-test="upload"
            :disabled="uploadingId === report.id"
            @click="onUpload(report)"
          >
            {{ uploadingId === report.id ? '上传中…' : '上传' }}
          </button>
          <button class="btn btn-small btn-danger" data-test="delete" @click="onDelete(report)">
            删除
          </button>
        </div>
      </div>

      <div v-if="expandedId === report.id" class="crash-item-detail">
        <div v-if="report.panic" class="detail-block">
          <span class="detail-label">Panic</span>
          <code>{{ report.panic }}</code>
        </div>
        <div class="detail-block detail-state">
          <span class="detail-label">状态</span>
          <span>
            模式 {{ report.app_state?.trading_mode || '-' }} ·
            券商 {{ report.app_state?.active_brokers?.join(', ') || '-' }} ·
            运行 {{ report.app_state?.uptime_seconds ?? 0 }}s
          </span>
        </div>
        <details v-if="report.stack">
          <summary>堆栈信息</summary>
          <pre>{{ report.stack }}</pre>
        </details>
        <details v-if="report.logs?.length">
          <summary>最近日志 ({{ report.logs.length }} 条)</summary>
          <pre>{{ report.logs.join('\n') }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>

<style scoped>
.crash-history {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.crash-history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.crash-history-title {
  font-size: 13px;
  font-weight: 600;
}
.crash-empty {
  color: var(--text-secondary);
  font-size: 12px;
  padding: 12px 0;
}
.crash-item {
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}
.crash-item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
}
.crash-item-row:hover {
  background: var(--bg-muted);
}
.crash-item-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.crash-item-time {
  font-size: 12px;
  font-weight: 600;
}
.crash-item-panic {
  font-size: 12px;
  color: var(--error, #ef4444);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crash-item-meta {
  font-size: 11px;
  color: var(--text-secondary);
}
.crash-item-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}
.crash-item-detail {
  border-top: 1px solid var(--border);
  padding: 8px 12px;
  font-size: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.detail-block code {
  display: block;
  background: var(--bg-code);
  border-radius: 4px;
  padding: 6px 8px;
  margin-top: 4px;
  font-size: 11px;
  word-break: break-all;
}
.detail-label {
  color: var(--text-secondary);
  font-size: 11px;
}
.detail-state {
  display: flex;
  gap: 8px;
  align-items: baseline;
}
.crash-item-detail details summary {
  cursor: pointer;
  color: var(--text-secondary);
  user-select: none;
}
.crash-item-detail details pre {
  background: var(--bg-code);
  border-radius: 4px;
  padding: 8px;
  margin: 6px 0 0;
  max-height: 200px;
  overflow: auto;
  font-size: 11px;
  white-space: pre-wrap;
  line-height: 1.5;
}
.btn {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  background: var(--bg-muted);
  color: var(--text);
  border: 1px solid var(--border);
}
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.btn-danger {
  color: var(--error, #ef4444);
}
</style>
