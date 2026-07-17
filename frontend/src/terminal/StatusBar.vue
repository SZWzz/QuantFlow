<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { getIcon } from '@/lib/icons'
import { GetVersion, GetConnectionStatus } from '@/lib/wails'

const workflow = useWorkflowStore()
const session = useSessionStore()
const terminal = useTerminalStore()
const ctx = useSymbolContext()

const time = ref(new Date().toLocaleTimeString())
const version = ref('...')
let timer: ReturnType<typeof setInterval> | null = null
let statusTimer: ReturnType<typeof setInterval> | null = null

async function refreshConnectionStatus() {
  try {
    const status = await GetConnectionStatus()
    if (status) terminal.updateConnectionStatus(status)
  } catch { /* ignore */ }
}

onMounted(async () => {
  timer = setInterval(() => time.value = new Date().toLocaleTimeString(), 1000)
  try { version.value = await GetVersion() } catch { version.value = '?' }

  // Fetch connection status immediately, then refresh every 30s
  await refreshConnectionStatus()
  statusTimer = setInterval(refreshConnectionStatus, 30000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
  if (statusTimer) clearInterval(statusTimer)
})

const activeGroups = computed(() =>
  Object.values(ctx.linkGroups).filter(g => g.activeSymbol)
)

// ── Connection status detail dialog ──────────────────────────────────

interface StatusDetail {
  title: string
  items: Array<{ label: string; status: string }>
}
const detailDialog = ref<StatusDetail | null>(null)

function showDetail(title: string, items: Array<{ label: string; status: string }>) {
  detailDialog.value = { title, items }
}
function closeDialog() {
  detailDialog.value = null
}

// ── Connection status display ────────────────────────────────────────

interface StatusItem { label: string; status: string }

const connStatus = computed(() => terminal.connectionStatus)
const marketEntries = computed<StatusItem[]>(() =>
  Object.entries(connStatus.value?.markets ?? {}).map(([m, s]) => ({ label: m, status: s }))
)

const brokerEntries = computed<StatusItem[]>(() =>
  Object.entries(connStatus.value?.brokers ?? {}).map(([n, s]) => ({ label: n, status: s }))
)

const pythonItem = computed<StatusItem>(() => ({
  label: 'Python', status: connStatus.value?.python ?? '未知',
}))

function isOK(s: string) {
  return s.includes('已配置') || s.includes('已连接') || s.includes('运行中') || s.includes('适配器')
}

// All healthy items merged into the "已连接" pill
const okItems = computed<StatusItem[]>(() => [
  ...marketEntries.value.filter(i => isOK(i.status)),
  ...brokerEntries.value.filter(i => isOK(i.status)),
  ...(isOK(pythonItem.value.status) ? [pythonItem.value] : []),
])

// Only broken items shown individually
const badItems = computed<StatusItem[]>(() => [
  ...marketEntries.value.filter(i => !isOK(i.status) && i.status !== '初始化中'),
  ...brokerEntries.value.filter(i => !isOK(i.status)),
  ...(!isOK(pythonItem.value.status) ? [pythonItem.value] : []),
])

const allOK = computed(() => badItems.value.length === 0)
</script>

<template>
  <div class="status-bar">
    <div class="status-left">
      <!-- 已连接：合并所有正常项 -->
      <span
        class="status-badge"
        :class="{ offline: okItems.length === 0 }"
        data-test="status-group"
        @click="showDetail('连接状态', [
          ...marketEntries.value,
          ...brokerEntries.value,
          pythonItem.value,
        ])"
      >
        <span class="status-dot" :class="{ pulse: okItems.length > 0, offline: okItems.length === 0 }" />
        <span class="status-text">{{ okItems.length > 0 ? '已连接' : '未连接' }}</span>
      </span>

      <!-- 只展示异常项 -->
      <span
        v-for="item in badItems"
        :key="item.label"
        class="status-badge offline"
        data-test="status-group"
        @click="showDetail(item.label, [{ label: '状态', status: item.status }])"
      >
        <span class="status-dot offline" />
        <span class="status-text">{{ item.label }}</span>
      </span>
    </div>
    <div class="status-groups">
      <span v-for="g in activeGroups" :key="g.id" class="group-badge"
        :style="{ borderColor: g.color, color: g.color, background: g.color + '15' }">
        <span class="group-dot" :style="{ background: g.color }"></span>
        {{ g.activeSymbol }}
      </span>
    </div>
    <div class="status-center">
      <span class="status-item">
        <span class="item-icon" v-html="getIcon('workflow')" />
        {{ workflow.executionStatus }}
      </span>
      <span class="status-item">
        <span class="item-icon" v-html="getIcon('terminal')" />
        {{ terminal.activePanels.length }} panels
      </span>
    </div>
    <div class="status-right">
      <span class="version-badge">v{{ version }}</span>
      <span class="time-display">
        <span class="time-icon" v-html="getIcon('schedule')" />
        {{ time }}
      </span>
    </div>

    <!-- Connection detail dialog -->
    <Teleport to="body">
      <div v-if="detailDialog" class="detail-overlay" @click.self="closeDialog">
        <div class="detail-modal">
          <h3>{{ detailDialog.title }}</h3>
          <div v-for="item in detailDialog.items" :key="item.label" class="detail-row">
            <span class="detail-label">{{ item.label }}</span>
            <span class="detail-value">{{ item.status }}</span>
          </div>
          <button class="btn-close" @click="closeDialog">关闭</button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 3px 12px;
  background: var(--gradient-header);
  border-top: 1px solid var(--color-border);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  min-height: 26px;
  user-select: none;
  position: relative;
  z-index: 10;
  flex-wrap: wrap;
}

.status-bar::before {
  content: '';
  position: absolute;
  top: -1px;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(59, 130, 246, 0.2) 15%,
    rgba(59, 130, 246, 0.5) 50%,
    rgba(59, 130, 246, 0.2) 85%,
    transparent 100%
  );
  opacity: 0.8;
}

.status-left, .status-center, .status-right { display: flex; gap: 10px; align-items: center; }

.status-badge {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  background: var(--color-success-soft);
  border: 1px solid var(--color-success);
  border-radius: var(--radius-lg);
  font-weight: 600;
  color: var(--color-success);
  font-size: 10px;
  transition: all var(--transition-fast);
  cursor: pointer;
}

.status-badge.offline {
  background: var(--color-danger-soft);
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success);
  box-shadow: 0 0 4px var(--color-success);
  flex-shrink: 0;
}

.status-dot.offline {
  background: var(--color-danger);
  box-shadow: 0 0 4px var(--color-danger);
  animation: none;
}

.status-dot.pulse {
  animation: pulse 2s ease infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.85); }
}

.status-text {
  font-variant-numeric: tabular-nums;
}

/* ── Link groups ───────────────────────────────────────────────────── */
.status-groups { display: flex; gap: 6px; }

.group-badge {
  display: flex; align-items: center; gap: 4px;
  padding: 1px 7px; border: 1px solid; border-radius: var(--radius-lg);
  font-size: 10px; font-weight: 600;
  transition: all var(--transition-fast);
}

.group-dot { width: 5px; height: 5px; border-radius: 50%; flex-shrink: 0; box-shadow: 0 0 4px currentColor; }

.status-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-variant-numeric: tabular-nums;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.status-item:hover {
  background: var(--color-bg-hover);
}

.item-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
  opacity: 0.6;
}

.item-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

/* ── Version & time ────────────────────────────────────────────────── */
.version-badge {
  padding: 1px 6px;
  background: var(--color-bg-subtle);
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-weight: 600;
}

.time-display {
  display: flex;
  align-items: center;
  gap: 5px;
  font-weight: 600;
  color: var(--color-text-secondary);
  padding: 2px 8px;
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
}

.time-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 11px;
  height: 11px;
  opacity: 0.5;
}

.time-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

/* ── Detail dialog ─────────────────────────────────────────────────── */
.detail-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center;
  z-index: 10001;
}
.detail-modal {
  background: var(--color-bg-app);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 24px;
  min-width: 300px;
}
.detail-modal h3 { margin-bottom: 16px; font-size: 15px; }
.detail-row {
  display: flex; justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border);
}
.detail-label { font-weight: 600; font-size: 13px; }
.detail-value { font-size: 13px; }
.btn-close {
  margin-top: 16px; padding: 8px 24px;
  background: var(--color-accent); color: #fff;
  border: none; border-radius: 8px; cursor: pointer;
}
</style>
