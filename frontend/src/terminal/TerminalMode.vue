<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useTerminalStore } from '@/stores/terminal'
import CommandBar from './CommandBar.vue'
import DockView from './DockView/DockView.vue'
import PushPinBar from './PushPinBar.vue'
import StatusBar from './StatusBar.vue'
import SymbolBar from './SymbolBar.vue'
import TickerBar from './TickerBar.vue'
import LiveModeBanner from './components/LiveModeBanner.vue'
import { getIcon } from '@/lib/icons'

const session = useSessionStore()
const terminal = useTerminalStore()
const router = useRouter()

const showCommandBar = ref(false)

const SHORTCUT_MAP: Record<string, string> = {
  D: 'dragon-tiger',
  L: 'limit-up-down',
  H: 'hk-connect',
  F: 'funding-rate',
  Q: 'sector-rotation',
  E: 'economic-calendar',
  W: 'watchlist',
}

function onGlobalKeydown(e: KeyboardEvent) {
  if (!e.ctrlKey || !e.shiftKey) return
  const panelId = SHORTCUT_MAP[e.key.toUpperCase()]
  if (panelId && session.ui.mode === 'terminal') {
    e.preventDefault()
    terminal.openPanel(panelId)
  }
}

function onOpenPanel(panelId: string, params?: Record<string, any>) {
  terminal.openPanel(panelId, params)
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown)
  // Preload echarts in idle time — used by 19 panels, 1MB chunk
  if ('requestIdleCallback' in window) {
    requestIdleCallback(() => import('echarts'), { timeout: 5000 })
  }
})
onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))

function onNavigate(path: string) {
  router.push(path)
}

function onSwitchToWorkflow() {
  session.ui.mode = 'workflow'
}
</script>

<template>
  <div class="terminal-mode">
    <header class="terminal-header">
      <div class="header-left">
        <div class="logo">
          <span class="logo-icon" v-html="getIcon('terminal')" />
        </div>
        <span class="title">QuantFlow</span>
      </div>
      <div class="header-actions">
        <button class="header-btn action-btn" @click="showCommandBar = true" title="Command Bar (Ctrl+K)">
          <span class="btn-icon" v-html="getIcon('command')" />
          <span class="btn-key">K</span>
        </button>
        <button class="header-btn" @click="terminal.openPanel('settings')" :title="$t('settings.title')">
          <span class="btn-icon" v-html="getIcon('settings')" />
        </button>
        <button class="mode-switch" @click="onSwitchToWorkflow">
          <span class="mode-icon" v-html="getIcon('workflow')" />
          Workflow
        </button>
      </div>
    </header>

    <LiveModeBanner />
    <SymbolBar />
    <TickerBar />
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
  background: var(--gradient-header);
  border-bottom: 1px solid var(--color-border);
  min-height: 42px;
  -webkit-app-region: drag;
  user-select: none;
  position: relative;
  z-index: 10;
}

.terminal-header::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent 0%, var(--color-accent) 50%, transparent 100%);
  opacity: 0.3;
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
  width: 26px;
  height: 26px;
  background: var(--gradient-accent);
  border: 1px solid var(--color-border-glow);
  border-radius: var(--radius-md);
  box-shadow: 0 0 8px var(--color-accent-glow);
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--color-accent);
}

.logo-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.title {
  font-weight: 700;
  font-size: var(--font-base);
  color: var(--color-text-primary);
  letter-spacing: 0.5px;
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
  gap: 4px;
  padding: 5px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  transition: all var(--transition-fast);
  min-width: 30px;
  height: 30px;
}

.header-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-soft);
  box-shadow: 0 0 8px var(--color-accent-glow);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
}

.btn-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.btn-key {
  font-size: 10px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  padding: 1px 4px;
  background: var(--color-bg-panel);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-family: 'JetBrains Mono', monospace;
}

.mode-switch {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: 1px solid var(--color-brand);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-xs);
  font-weight: 600;
  transition: all var(--transition-fast);
  height: 30px;
}

.mode-switch:hover {
  background: var(--color-brand);
  color: var(--color-text-inverse);
  box-shadow: 0 0 10px var(--color-brand-glow);
}

.mode-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}

.mode-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.mode-switch:hover .mode-icon {
  color: currentColor;
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
