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
import { getIcon } from '@/lib/icons'
import { SwitchToLive, SwitchToPaper, EmergencyClose } from '@/lib/wails'
import { confirmDialog, alertDialog } from '@/lib/wails'
import { useToast } from '@/lib/composables/useToast'

const session = useSessionStore()
const terminal = useTerminalStore()
const router = useRouter()
const toast = useToast()

const showCommandBar = ref(false)

// ── Trading mode ────────────────────────────────────────────────────

const isLive = ref(false)
const modeChecked = ref(false)
const showSafetyDialog = ref(false)
const safetyReport = ref<any>(null)
const emergencyLoading = ref(false)

onMounted(async () => {
  window.addEventListener('keydown', onGlobalKeydown)
  if ('requestIdleCallback' in window) {
    requestIdleCallback(() => import('echarts'), { timeout: 5000 })
  }
  // Check trading mode
  try {
    const mode = await (window as any).go?.main?.App?.GetTradingMode()
    isLive.value = mode === 'live'
    modeChecked.value = true
    terminal.tradingMode = mode || 'paper'
  } catch { modeChecked.value = true }
})

onUnmounted(() => window.removeEventListener('keydown', onGlobalKeydown))

async function handleGoLive() {
  showSafetyDialog.value = true
  try {
    const result = await SwitchToLive(false)
    safetyReport.value = result
    if (result?.all_clear) {
      isLive.value = true
      terminal.tradingMode = 'live'
      toast.success('已切换到实盘模式')
      showSafetyDialog.value = false
    }
  } catch { safetyReport.value = { checks: [], all_clear: false } }
}

async function handleForceLive() {
  const confirmed = await confirmDialog('⚠️ 强制切换到实盘模式（跳过安全检查），是否继续？')
  if (!confirmed) return
  try {
    await SwitchToLive(true)
    isLive.value = true
    terminal.tradingMode = 'live'
    toast.warning('已强制切换到实盘模式')
    showSafetyDialog.value = false
  } catch (e) { await alertDialog('切换失败: ' + String(e)) }
}

async function handleSwitchToPaper() {
  const confirmed = await confirmDialog('确定切换回模拟模式？')
  if (!confirmed) return
  try {
    await SwitchToPaper()
    isLive.value = false
    terminal.tradingMode = 'paper'
    toast.info('已切换回模拟模式')
  } catch (e) { await alertDialog('切换失败: ' + String(e)) }
}

async function handleEmergencyClose() {
  const confirmed = await confirmDialog('⚠️⚠️⚠️ 紧急平仓：确定平掉所有持仓并撤销所有委托？此操作不可撤销！')
  if (!confirmed) return
  emergencyLoading.value = true
  try {
    const result = await EmergencyClose()
    isLive.value = false
    terminal.tradingMode = 'paper'
    await alertDialog(`紧急平仓完成：已撤销 ${result.cancelled} 笔委托`)
  } catch (e) { await alertDialog('紧急平仓失败: ' + String(e)) }
  finally { emergencyLoading.value = false }
}

// ── Keyboard shortcuts ──────────────────────────────────────────────

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

function onNavigate(path: string) { router.push(path) }
function onSwitchToWorkflow() { session.ui.mode = 'workflow' }
</script>

<template>
  <div class="terminal-mode">
    <header class="terminal-header">
      <div class="header-left">
        <div class="logo">
          <span class="logo-icon" v-html="getIcon('terminal')" />
        </div>
        <span class="title">QuantFlow</span>

        <!-- Trading mode indicator — integrated into header -->
        <div v-if="modeChecked" class="mode-indicator">
          <!-- Paper mode -->
          <button v-if="!isLive" class="mode-tag paper" title="切换实盘" @click="handleGoLive">
            <span class="mode-dot" />
            <span class="mode-label">模拟</span>
          </button>
          <!-- Live mode -->
          <div v-else class="live-group">
            <span class="mode-tag live">
              <span class="mode-dot live-pulse" />
              <span class="mode-label">实盘</span>
            </span>
            <button class="live-emergency" :disabled="emergencyLoading" title="紧急平仓" @click="handleEmergencyClose">
              {{ emergencyLoading ? '...' : '🛑' }}
            </button>
          </div>
        </div>
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

    <!-- Safety check dialog -->
    <Teleport to="body">
      <div v-if="showSafetyDialog" class="safety-overlay" @click.self="showSafetyDialog = false">
        <div class="safety-modal">
          <h3>⚠️ 安全检查 — 切换到实盘</h3>
          <div v-if="safetyReport" class="check-list">
            <div v-for="check in safetyReport.checks" :key="check.name"
              class="check-item" :class="{ pass: check.ok, fail: !check.ok }">
              <span class="check-icon">{{ check.ok ? '✅' : check.blocking ? '❌' : '⚠️' }}</span>
              <div class="check-info">
                <span class="check-name">{{ check.name }}</span>
                <span class="check-message">{{ check.message }}</span>
              </div>
            </div>
          </div>
          <div class="safety-actions">
            <button class="btn-cancel" @click="showSafetyDialog = false">取消</button>
            <button v-if="safetyReport?.all_clear" class="btn-switch" @click="handleGoLive">确认切换</button>
            <button class="btn-force" @click="handleForceLive">强制切换</button>
          </div>
        </div>
      </div>
    </Teleport>
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
  z-index: var(--z-sticky);
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
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--color-accent);
}

.logo-icon :deep(svg) { width: 100%; height: 100%; }

.title {
  font-weight: 700;
  font-size: var(--font-base);
  color: var(--color-text-primary);
  letter-spacing: 0.5px;
}

/* ── Trading mode indicator ───────────────────────────────────── */
.mode-indicator {
  display: flex;
  align-items: center;
  -webkit-app-region: no-drag;
}

.mode-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  transition: all var(--transition-fast);
  font-family: inherit;
}

.mode-tag.paper {
  background: var(--color-success-soft);
  border: 1px solid var(--color-success);
  color: var(--color-success);
  cursor: pointer;
}
.mode-tag.paper:hover {
  background: var(--color-success);
  color: #fff;
}

.mode-tag.live {
  background: var(--color-danger);
  border: 1px solid var(--color-danger);
  color: #fff;
  cursor: default;
}

.mode-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: currentColor;
}
.mode-dot.live-pulse {
  animation: livePulse 1.5s ease infinite;
}
@keyframes livePulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.mode-label { letter-spacing: 0.3px; }

.live-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.live-emergency {
  padding: 2px 6px;
  font-size: 11px;
  background: var(--color-danger-soft);
  border: 1px solid var(--color-danger);
  color: var(--color-danger);
  border-radius: 6px;
  cursor: pointer;
  font-weight: 700;
  transition: all var(--transition-fast);
}
.live-emergency:hover { background: var(--color-danger); color: #fff; }
.live-emergency:disabled { opacity: 0.5; cursor: not-allowed; }

/* ── Header actions ────────────────────────────────────────────── */
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
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
}
.btn-icon :deep(svg) { width: 100%; height: 100%; }

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
}

.mode-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}
.mode-icon :deep(svg) { width: 100%; height: 100%; }
.mode-switch:hover .mode-icon { color: currentColor; }

/* ── Content ─────────────────────────────────────────────────── */
.terminal-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ── Safety dialog ─────────────────────────────────────────────── */
.safety-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center;
  z-index: var(--z-modal);
}
.safety-modal {
  background: var(--color-bg-app);
  border: 1px solid var(--color-border);
  border-radius: 12px; padding: 24px;
  min-width: 380px; max-width: 500px;
}
.safety-modal h3 { margin-bottom: 16px; font-size: 15px; }
.check-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.check-item {
  display: flex; gap: 10px; align-items: flex-start;
  padding: 8px 12px; border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
}
.check-item.pass { background: var(--color-success-soft); border-color: var(--color-success); }
.check-item.fail { background: var(--color-danger-soft); border-color: var(--color-danger); }
.check-icon { font-size: 14px; flex-shrink: 0; margin-top: 1px; }
.check-info { display: flex; flex-direction: column; gap: 2px; }
.check-name { font-weight: 600; font-size: 13px; }
.check-message { font-size: 11px; color: var(--color-text-tertiary); }
.safety-actions { display: flex; gap: 8px; justify-content: flex-end; }
.btn-cancel {
  padding: 6px 16px; font-size: 12px;
  background: var(--color-bg-subtle); color: var(--color-text-secondary);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm); cursor: pointer;
}
.btn-switch {
  padding: 6px 16px; font-size: 12px; font-weight: 600;
  background: var(--color-accent); color: #fff;
  border: none; border-radius: var(--radius-sm); cursor: pointer;
}
.btn-force {
  padding: 6px 16px; font-size: 12px;
  background: var(--color-danger); color: #fff;
  border: none; border-radius: var(--radius-sm); cursor: pointer;
}
</style>
