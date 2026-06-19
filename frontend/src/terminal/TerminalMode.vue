<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useTerminalStore } from '@/stores/terminal'
import CommandBar from './CommandBar.vue'
import DockView from './DockView/DockView.vue'
import PushPinBar from './PushPinBar.vue'
import StatusBar from './StatusBar.vue'

const session = useSessionStore()
const terminal = useTerminalStore()
const router = useRouter()

const showCommandBar = ref(false)

// Sync route with mode changes
watch(() => session.ui.mode, (mode) => {
  if (mode === 'workflow') router.push('/workflow')
  else router.push('/')
}, { immediate: true })

function onOpenPanel(panelId: string, params?: Record<string, any>) {
  terminal.openPanel(panelId, params)
}

function onNavigate(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="terminal-mode">
    <header class="terminal-header">
      <div class="header-left">
        <span class="logo">QF</span>
        <span class="title">QuantFlow</span>
      </div>
      <div class="header-center">
        <span class="breadcrumb">Terminal Mode</span>
      </div>
      <div class="header-actions">
        <button class="header-btn" @click="showCommandBar = true" title="Command Bar (Ctrl+K)">
          ⌘
        </button>
        <button class="mode-switch" @click="session.toggleMode()">
          Workflow
        </button>
      </div>
    </header>

    <main class="terminal-content">
      <PushPinBar />
      <DockView />
    </main>

    <StatusBar />

    <CommandBar
      v-model="showCommandBar"
      @open-panel="onOpenPanel"
      @navigate="onNavigate"
    />
  </div>
</template>

<style scoped>
.terminal-mode {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-bg-app);
  color: var(--color-text-primary);
}

/* ── Header ──────────────────────────────────────────────────── */
.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 var(--space-lg);
  background: var(--color-bg-panel);
  border-bottom: 1px solid var(--color-border);
  min-height: 40px;
  -webkit-app-region: drag;
  user-select: none;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-md);
}

.logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: var(--color-brand);
  color: #fff;
  border-radius: var(--radius-sm);
  font-weight: 800;
  font-size: var(--font-sm);
  letter-spacing: -0.5px;
}

.title {
  font-weight: 600;
  font-size: var(--font-base);
  color: var(--color-text-primary);
  letter-spacing: 0.3px;
}

.header-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
}

.breadcrumb {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
}

.header-actions {
  display: flex;
  gap: var(--space-sm);
  align-items: center;
  -webkit-app-region: no-drag;
}

.header-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 14px;
  font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}

.header-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-soft);
}

.mode-switch {
  padding: 4px 10px;
  border: 1px solid var(--color-brand);
  background: transparent;
  color: var(--color-brand);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.mode-switch:hover {
  background: var(--color-brand-soft);
}

/* ── Content ─────────────────────────────────────────────────── */
.terminal-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
