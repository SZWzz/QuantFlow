<script setup lang="ts">
/**
 * CrashDialog — recovery dialog shown on next launch after a crash.
 *
 * Displays the crash timestamp, panic message, collapsible stack trace and
 * recent logs, plus an opt-in checkbox for anonymous report upload.
 * Pure presentational: all actions are emitted to the parent (App.vue).
 */
import type { CrashReport } from '@/lib/wails'

const props = defineProps<{
  visible: boolean
  /** Crash timestamp (RFC3339 string) for display. */
  crashTime: string
  /** Path (directory or file) where the crash report was saved. */
  crashPath: string
  /** Full report for panic/stack/logs details (optional). */
  report?: CrashReport | null
}>()

const emit = defineEmits<{
  close: []
  restart: []
  upload: [send: boolean]
}>()

function formatTime(ts: string): string {
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="crash-overlay" @click.self="emit('close')">
      <div class="crash-dialog">
        <div class="crash-header">
          <span class="crash-icon">💥</span>
          <h2>QuantFlow 崩溃了</h2>
        </div>
        <div class="crash-body">
          <p>很抱歉，应用在上次运行时遇到了意外错误。</p>
          <p class="crash-time">崩溃时间: {{ formatTime(crashTime) }}</p>
          <p class="crash-path">
            已保存崩溃报告:<br>
            <code>{{ crashPath }}</code>
          </p>

          <div v-if="report?.panic" class="crash-panic">
            <span class="panic-label">错误信息</span>
            <code>{{ report.panic }}</code>
          </div>

          <details v-if="report?.stack" class="crash-details">
            <summary>堆栈信息</summary>
            <pre>{{ report.stack }}</pre>
          </details>

          <details v-if="report?.logs?.length" class="crash-details">
            <summary>最近日志 ({{ report.logs.length }} 条)</summary>
            <pre>{{ report.logs.join('\n') }}</pre>
          </details>

          <label class="crash-upload-opt">
            <input
              type="checkbox"
              data-test="upload-optin"
              :checked="true"
              @change="emit('upload', ($event.target as HTMLInputElement).checked)"
            />
            <span>发送匿名崩溃报告帮助改进</span>
          </label>
        </div>
        <div class="crash-actions">
          <button class="btn btn-secondary" data-test="dismiss" @click="emit('close')">忽略</button>
          <button class="btn btn-primary" data-test="restart" @click="emit('restart')">继续使用</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.crash-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.crash-dialog {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 480px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}
.crash-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.crash-header h2 {
  margin: 0;
  font-size: 16px;
}
.crash-body {
  padding: 16px 20px;
}
.crash-body > p {
  margin: 0 0 8px;
  font-size: 13px;
}
.crash-time {
  color: var(--text-secondary);
  font-size: 12px;
}
.crash-path {
  font-size: 12px;
}
.crash-path code {
  font-size: 11px;
  word-break: break-all;
}
.crash-panic {
  background: var(--bg-code);
  border-radius: 4px;
  padding: 8px 12px;
  margin: 8px 0;
  font-size: 12px;
}
.panic-label {
  display: block;
  color: var(--text-secondary);
  font-size: 11px;
  margin-bottom: 4px;
}
.crash-panic code {
  word-break: break-all;
}
.crash-details {
  margin: 8px 0;
  font-size: 12px;
}
.crash-details summary {
  cursor: pointer;
  color: var(--text-secondary);
  user-select: none;
}
.crash-details pre {
  background: var(--bg-code);
  border-radius: 4px;
  padding: 8px 12px;
  margin: 6px 0 0;
  max-height: 180px;
  overflow: auto;
  font-size: 11px;
  white-space: pre-wrap;
  line-height: 1.5;
}
.crash-upload-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
  margin-top: 12px;
}
.crash-actions {
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
</style>
