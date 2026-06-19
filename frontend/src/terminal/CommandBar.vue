<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'open-panel', panelId: string, params?: Record<string, any>): void
  (e: 'navigate', path: string): void
}>()

const terminal = useTerminalStore()
const session = useSessionStore()

const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

interface CommandItem {
  id: string
  label: string
  description: string
  category: 'panel' | 'command' | 'navigation'
  action: () => void
}

const panels: { id: string; label: string; description: string; params?: Record<string, any> }[] = [
  { id: 'watchlist', label: 'Watchlist', description: '自选列表 + 实时报价' },
  { id: 'quote-detail', label: 'Quote Detail', description: '单股详细报价', params: { symbol: '' } },
  { id: 'candlestick', label: 'Candlestick Chart', description: 'K 线图', params: { symbol: '', interval: '1d' } },
  { id: 'order-entry', label: 'Order Entry', description: '下单面板' },
  { id: 'position', label: 'Positions', description: '当前持仓' },
  { id: 'news', label: 'News Feed', description: '新闻流' },
  { id: 'ai-chat', label: 'AI Chat', description: 'AI 对话' },
  { id: 'system-monitor', label: 'System Monitor', description: '系统资源监控' },
]

const commands: { id: string; label: string; description: string; shortcut?: string }[] = [
  { id: 'toggle-mode', label: 'Toggle Workflow/Terminal', description: '切换工作流/终端模式', shortcut: 'Ctrl+W' },
  { id: 'toggle-focus', label: 'Toggle Focus Mode', description: '专注模式', shortcut: 'Ctrl+Shift+F' },
  { id: 'clear-history', label: 'Clear Command History', description: '清除命令历史' },
]

const navigations: { id: string; label: string; description: string; path: string }[] = [
  { id: 'nav-workflow', label: '/workflow', description: '切换到 Workflow Mode', path: '/workflow' },
  { id: 'nav-terminal', label: '/terminal', description: '切换到 Terminal Mode', path: '/' },
]

const results = computed<CommandItem[]>(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) {
    // Show recent history when no query
    return terminal.commandHistory.slice(0, 5).map((cmd) => ({
      id: `history-${cmd}`,
      label: cmd,
      description: 'Recent',
      category: 'command' as const,
      action: () => {
        query.value = cmd
        selectedIndex.value = 0
      },
    }))
  }

  const items: CommandItem[] = []

  // Match panels
  for (const p of panels) {
    if (p.label.toLowerCase().includes(q) || p.description.toLowerCase().includes(q)) {
      items.push({
        id: `panel-${p.id}`,
        label: p.label,
        description: p.description,
        category: 'panel',
        action: () => {
          emit('open-panel', p.id, p.params)
          close()
        },
      })
    }
  }

  // Match commands
  for (const c of commands) {
    if (c.label.toLowerCase().includes(q)) {
      items.push({
        id: `cmd-${c.id}`,
        label: c.label,
        description: c.shortcut ? `${c.description} (${c.shortcut})` : c.description,
        category: 'command',
        action: () => executeCommand(c.id),
      })
    }
  }

  // Match navigations
  for (const n of navigations) {
    if (n.label.toLowerCase().includes(q)) {
      items.push({
        id: `nav-${n.id}`,
        label: n.label,
        description: n.description,
        category: 'navigation',
        action: () => {
          emit('navigate', n.path)
          close()
        },
      })
    }
  }

  // If query looks like a symbol (alphanumeric, no spaces)
  if (/^[a-zA-Z0-9.]+$/.test(q)) {
    items.unshift({
      id: `symbol-${q}`,
      label: q.toUpperCase(),
      description: 'Open quote detail',
      category: 'panel',
      action: () => {
        emit('open-panel', 'quote-detail', { symbol: q.toUpperCase() })
        close()
      },
    })
  }

  return items
})

function close() {
  emit('update:modelValue', false)
  query.value = ''
  selectedIndex.value = 0
}

function executeCommand(cmdId: string) {
  switch (cmdId) {
    case 'toggle-mode':
      session.toggleMode()
      break
    case 'toggle-focus':
      terminal.toggleFocusMode()
      break
    case 'clear-history':
      terminal.commandHistory.splice(0)
      break
  }
  terminal.addCommand(cmdId)
  close()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, results.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (results.value[selectedIndex.value]) {
      results.value[selectedIndex.value].action()
    }
  }
}

function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault()
    emit('update:modelValue', true)
  }
}

const categoryLabels: Record<string, string> = {
  panel: 'Panels',
  command: 'Commands',
  navigation: 'Navigation',
}

watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      selectedIndex.value = 0
      query.value = ''
      setTimeout(() => inputRef.value?.focus(), 50)
    }
  }
)

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue" class="command-bar-overlay" @click.self="close">
      <div class="command-bar" @keydown="onKeydown">
        <div class="search-input-wrapper">
          <span class="search-icon">></span>
          <input
            ref="inputRef"
            v-model="query"
            type="text"
            class="search-input"
            placeholder="Search panels, commands, symbols..."
            autocomplete="off"
          />
          <span class="shortcut-hint">Esc to close</span>
        </div>
        <div v-if="results.length > 0" class="results-list">
          <template v-for="(item, idx) in results" :key="item.id">
            <div
              v-if="idx === 0 || results[idx - 1].category !== item.category"
              class="category-header"
            >
              {{ categoryLabels[item.category] || item.category }}
            </div>
            <div
              class="result-item"
              :class="{ selected: idx === selectedIndex }"
              @click="item.action()"
              @mouseenter="selectedIndex = idx"
            >
              <span class="item-label">{{ item.label }}</span>
              <span class="item-desc">{{ item.description }}</span>
            </div>
          </template>
        </div>
        <div v-else-if="query" class="no-results">
          No results for "{{ query }}"
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.command-bar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  padding-top: 15vh;
  z-index: 9999;
  backdrop-filter: blur(4px);
}

.command-bar {
  width: 560px;
  max-height: 480px;
  background: #1c2333;
  border: 1px solid #30363d;
  border-radius: 12px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #30363d;
  gap: 10px;
}

.search-icon {
  color: #e94560;
  font-weight: bold;
  font-size: 16px;
  font-family: monospace;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: #e0e0e0;
  font-size: 15px;
  outline: none;
  font-family: inherit;
}

.search-input::placeholder {
  color: #5a6380;
}

.shortcut-hint {
  color: #5a6380;
  font-size: 11px;
  padding: 2px 8px;
  border: 1px solid #30363d;
  border-radius: 4px;
}

.results-list {
  overflow-y: auto;
  flex: 1;
}

.category-header {
  padding: 6px 16px 2px;
  font-size: 10px;
  text-transform: uppercase;
  color: #5a6380;
  letter-spacing: 0.5px;
}

.result-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
  transition: background 0.1s;
}

.result-item.selected {
  background: rgba(88, 166, 255, 0.15);
}

.item-label {
  font-size: 13px;
  font-weight: 500;
  color: #c9d1d9;
}

.item-desc {
  font-size: 11px;
  color: #5a6380;
}

.no-results {
  padding: 24px 16px;
  text-align: center;
  color: #5a6380;
  font-size: 13px;
}
</style>
