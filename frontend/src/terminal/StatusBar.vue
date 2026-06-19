<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'

const data = useDataStore()
const workflow = useWorkflowStore()
const session = useSessionStore()
const terminal = useTerminalStore()
const ctx = useSymbolContext()

const time = ref(new Date().toLocaleTimeString())
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => { timer = setInterval(() => time.value = new Date().toLocaleTimeString(), 1000) })
onUnmounted(() => { if (timer) clearInterval(timer) })

const activeGroups = computed(() =>
  Object.values(ctx.linkGroups).filter(g => g.activeSymbol)
)
</script>

<template>
  <div class="status-bar">
    <div class="status-left">
      <span class="status-item connected" :class="{ offline: data.isOffline }">
        ● {{ data.isOffline ? 'Offline' : 'Connected' }}
      </span>
    </div>
    <div class="status-groups">
      <span v-for="g in activeGroups" :key="g.id" class="group-badge"
        :style="{ borderColor: g.color }">
        <span class="group-dot" :style="{ background: g.color }"></span>
        {{ g.activeSymbol }}
      </span>
    </div>
    <div class="status-center">
      <span class="status-item">WF: {{ workflow.executionStatus }}</span>
      <span class="status-item">Mode: {{ session.ui.mode }}</span>
      <span class="status-item">{{ terminal.activePanels.length }} panels</span>
    </div>
    <div class="status-right">
      <span class="status-item">{{ time }}</span>
    </div>
  </div>
</template>

<style scoped>
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 2px 10px;
  background: var(--color-bg-panel);
  border-top: 1px solid var(--color-border);
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  min-height: 24px;
  user-select: none;
}
.status-left, .status-center, .status-right { display: flex; gap: 12px; align-items: center; }
.status-item { font-variant-numeric: tabular-nums; }
.status-item.connected { color: var(--color-up); }
.status-item.connected.offline { color: var(--color-down); }
.status-groups { display: flex; gap: 8px; }
.group-badge {
  display: flex; align-items: center; gap: 4px;
  padding: 0 6px; border: 1px solid; border-radius: var(--radius-sm);
  font-size: var(--font-xs); font-weight: 600;
}
.group-dot { width: 6px; height: 6px; border-radius: 50%; }
</style>
