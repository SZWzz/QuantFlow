<script setup lang="ts">
import { ref } from 'vue'
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
      <span class="logo">QuantFlow Terminal</span>
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
  background: #1a1a2e;
  color: #e0e0e0;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 12px;
  background: #16213e;
  border-bottom: 1px solid #0f3460;
  min-height: 36px;
}

.logo {
  font-weight: bold;
  font-size: 13px;
  color: #e94560;
  letter-spacing: 0.5px;
}

.header-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.header-btn {
  padding: 2px 10px;
  border: 1px solid #5a6380;
  background: transparent;
  color: #5a6380;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  font-family: monospace;
}

.header-btn:hover {
  border-color: #e94560;
  color: #e94560;
}

.mode-switch {
  padding: 2px 10px;
  border: 1px solid #e94560;
  background: transparent;
  color: #e94560;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
}

.mode-switch:hover {
  background: rgba(233, 69, 96, 0.1);
}

.terminal-content {
  flex: 1;
  overflow: hidden;
}

.terminal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 12px;
  background: #16213e;
  border-top: 1px solid #0f3460;
  font-size: 11px;
  color: #5a6380;
  min-height: 24px;
}
</style>
