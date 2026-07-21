<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { PanelHeader } from '@/terminal/components/panel'
import { useAddToWorkflow } from '@/terminal/composables/useAddToWorkflow'
import { useLogger } from '@/lib/composables/useLogger'
import { confirmDialog } from '@/lib/wails'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const { control: addToWfControl } = useAddToWorkflow(props.panelId)

const LEVELS = ['debug', 'info', 'warn', 'error']

const {
  filter, filteredEntries,
  toggleLevel, setSearch, clear, poll, error,
} = useLogger()

const scrollContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const searchInput = ref('')

const displayEntries = computed(() => filteredEntries().slice(-500))

const controls = computed(() => {
  const list: any[] = []
  if (addToWfControl.value) list.push(addToWfControl.value)
  list.push({ icon: 'refresh', title: '刷新', action: () => { clear(); poll() } })
  return list
})

function onScroll() {
  if (!scrollContainer.value) return
  const el = scrollContainer.value
  const threshold = 30
  autoScroll.value = (el.scrollHeight - el.scrollTop - el.clientHeight) < threshold
}

watch(displayEntries, async () => {
  if (autoScroll.value) {
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  }
})

async function handleClear() {
  const ok = await confirmDialog('确定清空所有日志？')
  if (ok) clear()
}

function onSearchInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  searchInput.value = val
  setSearch(val)
}

function formatTime(t: string): string {
  try {
    const d = new Date(t)
    return d.toLocaleTimeString('zh-CN', { hour12: false })
  } catch {
    return t
  }
}

function formatAttrs(attrs?: Record<string, any>): string {
  if (!attrs || Object.keys(attrs).length === 0) return ''
  return Object.entries(attrs)
    .map(([k, v]) => `${k}=${v}`)
    .join(' ')
}

function highlightSearch(text: string): string {
  if (!searchInput.value) return text
  const q = searchInput.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${q})`, 'gi')
  return text.replace(re, '<mark class="log-highlight">$1</mark>')
}
</script>

<template>
  <div class="log-panel">
    <PanelHeader
      title="日志面板"
      :controls="controls"
    />

    <!-- Toolbar -->
    <div class="log-toolbar">
      <div class="log-levels">
        <button
          v-for="lvl in LEVELS"
          :key="lvl"
          class="log-level-btn"
          :class="{ active: filter.levels.has(lvl), [`log-level-${lvl}`]: true }"
          @click="toggleLevel(lvl)"
        >
          {{ lvl.toUpperCase() }}
        </button>
      </div>
      <div class="log-toolbar-right">
        <input
          class="log-search"
          type="text"
          :value="searchInput"
          @input="onSearchInput"
          placeholder="搜索日志..."
        />
        <button class="log-clear-btn" @click="handleClear">清空</button>
      </div>
    </div>

    <!-- Log entries -->
    <div class="log-entries" ref="scrollContainer" @scroll="onScroll">
      <div
        v-for="entry in displayEntries"
        :key="entry.id"
        class="log-line"
        :class="`log-level-${entry.level}`"
      >
        <span class="log-time">{{ formatTime(entry.time) }}</span>
        <span class="log-level-tag">[{{ entry.level.toUpperCase() }}]</span>
        <span class="log-msg" v-html="highlightSearch(entry.message)"></span>
        <span v-if="entry.attrs && Object.keys(entry.attrs).length > 0" class="log-attrs">
          {{ formatAttrs(entry.attrs) }}
        </span>
      </div>
      <div v-if="error" class="log-error">
        连接错误: {{ error }}
      </div>
      <div v-else-if="displayEntries.length === 0" class="log-empty">
        暂无日志
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-panel);
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: var(--font-xs);
  color: var(--color-text-primary);
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-xs) var(--space-sm);
  background: var(--color-bg-subtle);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
  gap: var(--space-sm);
}

.log-levels {
  display: flex;
  gap: var(--space-xs);
}

.log-level-btn {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: inherit;
  transition: all 0.15s;
}

.log-level-btn.active {
  border-color: var(--color-border-strong);
}

.log-level-btn.log-level-debug.active { color: var(--color-text-tertiary); border-color: var(--color-text-tertiary); }
.log-level-btn.log-level-info.active { color: var(--color-text-primary); border-color: var(--color-text-primary); }
.log-level-btn.log-level-warn.active { color: var(--color-warn); border-color: var(--color-warn); }
.log-level-btn.log-level-error.active { color: var(--color-danger); border-color: var(--color-danger); }

.log-toolbar-right {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.log-search {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-input);
  color: var(--color-text-primary);
  font-size: var(--font-xs);
  font-family: inherit;
  width: 160px;
  outline: none;
}

.log-search:focus {
  border-color: var(--color-accent);
}

.log-clear-btn {
  padding: 2px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: inherit;
}

.log-clear-btn:hover {
  color: var(--color-danger);
  border-color: var(--color-danger);
}

.log-entries {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-xs) 0;
}

.log-entries::-webkit-scrollbar {
  width: 6px;
}

.log-entries::-webkit-scrollbar-track {
  background: transparent;
}

.log-entries::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  padding: 1px var(--space-sm);
  line-height: 1.5;
  white-space: nowrap;
  font-size: var(--font-xs);
}

.log-line:hover {
  background: var(--color-bg-hover);
}

.log-time {
  color: var(--color-text-tertiary);
  opacity: 0.7;
  flex-shrink: 0;
  width: 60px;
}

.log-level-tag {
  flex-shrink: 0;
  width: 52px;
  font-weight: 500;
}

.log-line.log-level-debug { color: var(--color-text-tertiary); }
.log-line.log-level-info { color: var(--color-text-primary); }
.log-line.log-level-warn { color: var(--color-warn); }
.log-line.log-level-error { color: var(--color-danger); }

.log-msg {
  flex-shrink: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-attrs {
  color: var(--color-text-tertiary);
  opacity: 0.6;
  flex-shrink: 0;
  margin-left: auto;
}

.log-empty {
  padding: var(--space-lg);
  text-align: center;
  color: var(--color-text-tertiary);
  font-style: italic;
}

.log-error {
  padding: var(--space-md);
  text-align: center;
  color: var(--color-danger);
  font-size: var(--font-xs);
}

:deep(.log-highlight) {
  background: var(--color-accent-soft);
  border-radius: 2px;
  padding: 0 1px;
}
</style>
