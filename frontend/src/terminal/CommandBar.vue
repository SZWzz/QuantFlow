<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { useSessionStore } from '@/stores/session'
import { getAllPanelMeta, type PanelMeta } from '@/terminal/panels/registry'
import { getIcon, PANEL_ICONS } from '@/lib/icons'

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
  category: string
  icon: string
  action: () => void
}

// Dynamic panel list from registry
const allPanels = getAllPanelMeta().filter(p => p.id !== 'welcome')

const commands: { id: string; label: string; description: string; shortcut?: string; icon: string }[] = [
  { id: 'toggle-mode', label: 'Toggle Workflow/Terminal', description: '切换工作流/终端模式', shortcut: 'Ctrl+W', icon: getIcon('workflow') },
  { id: 'toggle-focus', label: 'Toggle Focus Mode', description: '专注模式', shortcut: 'Ctrl+Shift+F', icon: getIcon('terminal') },
  { id: 'clear-history', label: 'Clear Command History', description: '清除命令历史', icon: getIcon('delete') },
]

const navigations: { id: string; label: string; description: string; path: string; icon: string }[] = [
  { id: 'nav-workflow', label: '/workflow', description: '切换到 Workflow Mode', path: '/workflow', icon: getIcon('workflow') },
  { id: 'nav-terminal', label: '/terminal', description: '切换到 Terminal Mode', path: '/', icon: getIcon('terminal') },
]

const results = computed<CommandItem[]>(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) {
    return terminal.commandHistory.slice(0, 5).map((cmd) => ({
      id: `history-${cmd}`,
      label: cmd,
      description: 'Recent',
      category: 'Recent',
      icon: getIcon('terminal'),
      action: () => {
        query.value = cmd
        selectedIndex.value = 0
      },
    }))
  }

  const items: CommandItem[] = []

  // Match panels from registry
  for (const p of allPanels) {
    if (p.label.toLowerCase().includes(q) || p.description.toLowerCase().includes(q) || p.id.includes(q)) {
      const iconName = PANEL_ICONS[p.id]
      items.push({
        id: `panel-${p.id}`,
        label: p.label,
        description: p.description,
        category: p.category,
        icon: iconName ? getIcon(iconName) : getIcon('terminal'),
        action: () => {
          emit('open-panel', p.id)
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
        category: 'Commands',
        icon: c.icon,
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
        category: 'Navigation',
        icon: n.icon,
        action: () => {
          emit('navigate', n.path)
          close()
        },
      })
    }
  }

  // If query looks like a symbol, add quick open
  if (/^[a-zA-Z0-9.]+$/.test(q)) {
    items.unshift({
      id: `symbol-${q}`,
      label: q.toUpperCase(),
      description: '快速查看行情',
      category: 'Quick',
      icon: getIcon('quote'),
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
          <span class="search-icon" v-html="getIcon('search')" />
          <input
            ref="inputRef"
            v-model="query"
            type="text"
            class="search-input"
            :placeholder="$t('common.search') + '...'"
            autocomplete="off"
          />
          <kbd class="shortcut-hint">Esc</kbd>
        </div>
        <div v-if="results.length > 0" class="results-list">
          <template v-for="(item, idx) in results" :key="item.id">
            <div
              v-if="idx === 0 || results[idx - 1].category !== item.category"
              class="category-header"
            >
              {{ item.category }}
            </div>
            <div
              class="result-item"
              :class="{ selected: idx === selectedIndex }"
              @click="item.action()"
              @mouseenter="selectedIndex = idx"
            >
              <span class="item-icon" v-html="item.icon" />
              <span class="item-label">{{ item.label }}</span>
              <span class="item-desc">{{ item.description }}</span>
            </div>
          </template>
        </div>
        <div v-else-if="query" class="no-results">
          <span class="no-results-icon" v-html="getIcon('search')" />
          {{ $t('common.no_data') }}
        </div>
        <div class="command-footer">
          <div class="footer-hints">
            <span class="hint"><kbd>↑</kbd><kbd>↓</kbd> Navigate</span>
            <span class="hint"><kbd>Enter</kbd> Select</span>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.command-bar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  padding-top: 12vh;
  z-index: 9999;
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  animation: fadeInOverlay 0.2s ease;
}

@keyframes fadeInOverlay {
  from { opacity: 0; }
  to { opacity: 1; }
}

.command-bar {
  width: 600px;
  max-height: 520px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border-strong);
  border-radius: 16px;
  box-shadow: var(--shadow-lg), 0 0 40px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.25s ease;
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: translateY(0); }
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border);
  gap: 12px;
}

.search-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  color: var(--color-accent);
  flex-shrink: 0;
}

.search-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--color-text-primary);
  font-size: 16px;
  outline: none;
  font-family: inherit;
  padding: 0;
}

.search-input::placeholder { color: var(--color-text-tertiary); }

.shortcut-hint {
  color: var(--color-text-tertiary);
  font-size: 11px;
  padding: 3px 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-family: inherit;
  background: var(--color-bg-subtle);
  flex-shrink: 0;
}

.results-list { overflow-y: auto; flex: 1; padding: 6px; }

.category-header {
  padding: 8px 12px 4px;
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-text-tertiary);
  letter-spacing: 1px;
  font-weight: 600;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  cursor: pointer;
  transition: all var(--transition-fast);
  border-radius: var(--radius-md);
  margin: 1px 0;
}

.result-item.selected {
  background: var(--color-accent-soft);
  border: 1px solid var(--color-border-glow);
  box-shadow: 0 0 8px var(--color-accent-glow);
}

.result-item:hover:not(.selected) {
  background: var(--color-bg-hover);
}

.item-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}

.item-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.result-item.selected .item-icon {
  color: var(--color-accent);
}

.item-label { font-size: 13px; font-weight: 500; color: var(--color-text-primary); flex-shrink: 0; }
.item-desc { font-size: 12px; color: var(--color-text-tertiary); margin-left: auto; }

.result-item.selected .item-label { color: var(--color-accent); }

.no-results {
  padding: 32px 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--color-text-tertiary);
  font-size: 13px;
  flex-direction: column;
}

.no-results-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  opacity: 0.3;
}

.no-results-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.command-footer {
  border-top: 1px solid var(--color-border);
  padding: 10px 16px;
  background: var(--color-bg-subtle);
}

.footer-hints {
  display: flex;
  gap: 16px;
  align-items: center;
}

.hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.hint kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px 6px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 10px;
  font-family: inherit;
  min-width: 20px;
  text-align: center;
}
</style>
