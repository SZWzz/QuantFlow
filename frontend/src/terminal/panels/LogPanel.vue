<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useLogger } from '@/lib/composables/useLogger'
import { confirmDialog } from '@/lib/wails'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const LEVELS = ['debug', 'info', 'warn', 'error']

const {
  entries, filter, filteredEntries,
  toggleLevel, setSearch, clear, poll,
} = useLogger()

const scrollContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

const displayEntries = computed(() => filteredEntries().slice(-500))

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
  if (!filter.value.search) return text
  const q = filter.value.search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${q})`, 'gi')
  return text.replace(re, '<mark class="log-highlight">$1</mark>')
}
</script>

<template>
  <div class="log-panel">
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
          :value="filter.search"
          @input="setSearch(($event.target as HTMLInputElement).value)"
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
      <div v-if="displayEntries.length === 0" class="log-empty">
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
  background: #1a1a2e;
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  color: #e0e0e0;
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  background: #16162a;
  border-bottom: 1px solid #2a2a4a;
  flex-shrink: 0;
  gap: 8px;
}

.log-levels {
  display: flex;
  gap: 4px;
}

.log-level-btn {
  padding: 2px 8px;
  border: 1px solid #333;
  border-radius: 3px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 10px;
  font-family: inherit;
  transition: all 0.15s;
}

.log-level-btn.active {
  border-color: #555;
}

.log-level-btn.log-level-debug.active { color: #888; border-color: #888; }
.log-level-btn.log-level-info.active { color: #e0e0e0; border-color: #e0e0e0; }
.log-level-btn.log-level-warn.active { color: #f0ad4e; border-color: #f0ad4e; }
.log-level-btn.log-level-error.active { color: #ef4444; border-color: #ef4444; }

.log-toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.log-search {
  padding: 2px 8px;
  border: 1px solid #333;
  border-radius: 3px;
  background: #0f0f1e;
  color: #e0e0e0;
  font-size: 11px;
  font-family: inherit;
  width: 160px;
  outline: none;
}

.log-search:focus {
  border-color: #555;
}

.log-clear-btn {
  padding: 2px 10px;
  border: 1px solid #333;
  border-radius: 3px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 10px;
  font-family: inherit;
}

.log-clear-btn:hover {
  color: #ef4444;
  border-color: #ef4444;
}

.log-entries {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.log-entries::-webkit-scrollbar {
  width: 6px;
}

.log-entries::-webkit-scrollbar-track {
  background: transparent;
}

.log-entries::-webkit-scrollbar-thumb {
  background: #333;
  border-radius: 3px;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 1px 8px;
  line-height: 1.5;
  white-space: nowrap;
  font-size: 11px;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.03);
}

.log-time {
  color: #666;
  flex-shrink: 0;
  width: 60px;
}

.log-level-tag {
  flex-shrink: 0;
  width: 52px;
  font-weight: 500;
}

.log-line.log-level-debug { color: #888; }
.log-line.log-level-info { color: #e0e0e0; }
.log-line.log-level-warn { color: #f0ad4e; }
.log-line.log-level-error { color: #ef4444; }

.log-msg {
  flex-shrink: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-attrs {
  color: #666;
  flex-shrink: 0;
  margin-left: auto;
}

.log-empty {
  padding: 20px;
  text-align: center;
  color: #555;
  font-style: italic;
}

:deep(.log-highlight) {
  background: rgba(240, 173, 78, 0.3);
  border-radius: 2px;
  padding: 0 1px;
}
</style>
