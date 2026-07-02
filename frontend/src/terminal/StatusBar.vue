<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useDataStore } from '@/stores/data'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { useTerminalStore } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { getIcon } from '@/lib/icons'

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
      <span class="status-badge" :class="{ offline: data.isOffline }">
        <span class="status-dot" :class="{ pulse: !data.isOffline, offline: data.isOffline }" />
        <span class="status-text">{{ data.isOffline ? $t('common.disconnected') : $t('common.connected') }}</span>
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
      <span class="time-display">
        <span class="time-icon" v-html="getIcon('schedule')" />
        {{ time }}
      </span>
    </div>
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
  border-radius: 10px;
  font-weight: 600;
  color: var(--color-success);
  font-size: 10px;
  transition: all var(--transition-fast);
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

.status-groups { display: flex; gap: 6px; }

.group-badge {
  display: flex; align-items: center; gap: 4px;
  padding: 1px 7px; border: 1px solid; border-radius: 10px;
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
  border-radius: 4px;
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

.time-display {
  display: flex;
  align-items: center;
  gap: 5px;
  font-weight: 600;
  color: var(--color-text-secondary);
  padding: 2px 8px;
  background: var(--color-bg-subtle);
  border: 1px solid var(--color-border);
  border-radius: 6px;
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
</style>
