<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'

const workflow = useWorkflowStore()

interface ExecRecord {
  id: number
  run_id: string
  workflow_name: string
  status: string
  node_count: number
  started_at: string
  finished_at: string
  triggered_by: string
  error: string
  workflow_json: string
  node_results: string
}

const records = ref<ExecRecord[]>([])
const loading = ref(false)
const expanded = ref<string | null>(null)

async function loadHistory() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetExecutionHistory) return
    const result = await app.GetExecutionHistory(20)
    records.value = Array.isArray(result) ? result : []
  } catch { records.value = [] }
  finally { loading.value = false }
}

function replayRecord(rec: ExecRecord) {
  try {
    const wf = JSON.parse(rec.workflow_json)
    workflow.fromWorkflowJSON(wf)
  } catch { /* ignore */ }
}

function toggleExpand(runId: string) {
  expanded.value = expanded.value === runId ? null : runId
}

function formatTime(ts: string): string {
  if (!ts) return '--'
  return ts.replace('T', ' ').substring(0, 19)
}

function statusBadge(s: string): string {
  if (s === 'completed') return '✓'
  if (s === 'failed') return '✗'
  return '⏳'
}

onMounted(loadHistory)
</script>

<template>
  <div class="execution-history">
    <div class="history-header">
      <h3>执行历史</h3>
      <button class="refresh-btn" @click="loadHistory" :disabled="loading">⟳</button>
    </div>
    <div v-if="records.length === 0" class="empty-state">
      暂无执行记录，运行一个 Workflow 后自动保存
    </div>
    <div v-for="rec in records" :key="rec.run_id" class="history-item" @click="toggleExpand(rec.run_id)">
      <div class="item-main">
        <span class="item-status" :class="rec.status">{{ statusBadge(rec.status) }}</span>
        <span class="item-name">{{ rec.workflow_name || rec.run_id }}</span>
        <span class="item-meta">{{ rec.node_count }} 节点 · {{ formatTime(rec.started_at) }}</span>
      </div>
      <div v-if="expanded === rec.run_id" class="item-detail">
        <div class="detail-row"><span>Run ID:</span><code>{{ rec.run_id }}</code></div>
        <div class="detail-row"><span>Status:</span><span :class="rec.status">{{ rec.status }}</span></div>
        <div v-if="rec.error" class="detail-row error-text"><span>Error:</span>{{ rec.error }}</div>
        <button class="replay-btn" @click.stop="replayRecord(rec)">📋 加载到画布</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.execution-history { padding: 8px; }
.history-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.history-header h3 { font-size: 12px; color: var(--color-text-secondary); margin: 0; }
.refresh-btn { padding: 2px 8px; border: 1px solid var(--color-border); border-radius: var(--radius-sm); background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; }
.refresh-btn:hover { border-color: var(--color-accent); }
.empty-state { padding: 24px; text-align: center; color: var(--color-text-tertiary); font-size: 12px; }
.history-item { padding: 6px 8px; border-radius: var(--radius-sm); cursor: pointer; transition: background .1s; margin-bottom: 4px; border: 1px solid var(--color-border); }
.history-item:hover { background: var(--color-bg-hover); }
.item-main { display: flex; align-items: center; gap: 8px; font-size: 11px; }
.item-status { font-weight: bold; width: 16px; text-align: center; }
.item-status.completed { color: #3fb950; }
.item-status.failed { color: #f85149; }
.item-name { color: var(--color-text-primary); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.item-meta { color: var(--color-text-tertiary); font-size: 10px; flex-shrink: 0; }
.item-detail { margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--color-border); display: flex; flex-direction: column; gap: 4px; font-size: 11px; }
.detail-row { display: flex; gap: 8px; color: var(--color-text-secondary); }
.detail-row code { color: var(--color-accent); font-size: 10px; }
.error-text { color: #f85149; }
.replay-btn { margin-top: 6px; padding: 4px 10px; border: 1px solid var(--color-accent); border-radius: var(--radius-sm); background: rgba(88,166,255,.1); color: var(--color-accent); cursor: pointer; font-size: 11px; }
.replay-btn:hover { background: rgba(88,166,255,.2); }
</style>
