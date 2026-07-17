<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { SwitchToLive, SwitchToPaper, EmergencyClose } from '@/lib/wails'
import { confirmDialog, alertDialog } from '@/lib/wails'
import { useToast } from '@/lib/composables/useToast'

const terminal = useTerminalStore()
const toast = useToast()
const showSafetyDialog = ref(false)
const safetyReport = ref<any>(null)
const skipChecks = ref(false)
const emergencyLoading = ref(false)

const isLive = ref(false)
const liveChecked = ref(false)

onMounted(async () => {
  // Check initial mode
  await refreshMode()
})

async function refreshMode() {
  try {
    const mode = await (window as any).go?.main?.App?.GetTradingMode()
    isLive.value = mode === 'live'
    liveChecked.value = true
    terminal.tradingMode = mode || 'paper'
  } catch {
    liveChecked.value = true
  }
}

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
  } catch (e) {
    safetyReport.value = { checks: [], all_clear: false }
  }
}

async function handleForceLive() {
  const confirmed = await confirmDialog('⚠️ 强制切换到实盘模式（跳过安全检查），是否继续？')
  if (!confirmed) return
  try {
    await SwitchToLive(true)
    isLive.value = true
    terminal.tradingMode = 'live'
    toast.warning('已强制切换到实盘模式（跳过安全检查）')
    showSafetyDialog.value = false
  } catch (e) {
    await alertDialog('切换失败: ' + String(e))
  }
}

async function handleSwitchToPaper() {
  const confirmed = await confirmDialog('确定切换回模拟模式？')
  if (!confirmed) return
  try {
    await SwitchToPaper()
    isLive.value = false
    terminal.tradingMode = 'paper'
    toast.info('已切换回模拟模式')
  } catch (e) {
    await alertDialog('切换失败: ' + String(e))
  }
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
  } catch (e) {
    await alertDialog('紧急平仓失败: ' + String(e))
  } finally {
    emergencyLoading.value = false
  }
}
</script>

<template>
  <div v-if="liveChecked" class="live-mode-container">
    <!-- Paper mode: subtle indicator -->
    <div v-if="!isLive" class="paper-indicator">
      <span class="paper-badge">📄 Paper Mode</span>
      <button class="go-live-btn" @click="handleGoLive">切换实盘</button>
    </div>

    <!-- Live mode: red banner -->
    <div v-else class="live-banner">
      <span class="live-pulse" />
      <span class="live-text">🔴 LIVE MODE — 真实资金交易中</span>
      <div class="live-actions">
        <button class="emergency-btn" :disabled="emergencyLoading" @click="handleEmergencyClose">
          🛑 {{ emergencyLoading ? '处理中...' : '紧急平仓' }}
        </button>
        <button class="paper-switch-btn" @click="handleSwitchToPaper">
          切换模拟
        </button>
      </div>
    </div>

    <!-- Safety check dialog -->
    <Teleport to="body">
      <div v-if="showSafetyDialog" class="safety-overlay" @click.self="showSafetyDialog = false">
        <div class="safety-modal">
          <h3>⚠️ 安全检查 — 切换到实盘</h3>
          <div v-if="safetyReport" class="check-list">
            <div
              v-for="check in safetyReport.checks"
              :key="check.name"
              class="check-item"
              :class="{ pass: check.ok, fail: !check.ok, blocking: check.blocking }"
            >
              <span class="check-icon">{{ check.ok ? '✅' : check.blocking ? '❌' : '⚠️' }}</span>
              <div class="check-info">
                <span class="check-name">{{ check.name }}</span>
                <span class="check-message">{{ check.message }}</span>
              </div>
            </div>
          </div>
          <div class="safety-actions">
            <button class="btn-cancel" @click="showSafetyDialog = false">取消</button>
            <button v-if="safetyReport?.all_clear" class="btn-switch" @click="handleGoLive">
              确认切换
            </button>
            <button class="btn-force" @click="handleForceLive">
              强制切换
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.live-mode-container { flex-shrink: 0; }

.paper-indicator {
  display: flex; align-items: center; justify-content: center; gap: 12px;
  padding: 3px 12px;
  background: var(--color-success-soft);
  border-bottom: 1px solid var(--color-success);
  font-size: 11px;
}
.paper-badge { font-weight: 600; color: var(--color-success); }
.go-live-btn {
  padding: 2px 10px; font-size: 10px; font-weight: 600;
  background: var(--color-accent); color: #fff;
  border: none; border-radius: var(--radius-sm); cursor: pointer;
}

.live-banner {
  display: flex; align-items: center; justify-content: center; gap: 16px;
  padding: 4px 12px;
  background: linear-gradient(90deg, var(--color-danger), #991b1b);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  border-bottom: 2px solid #7f1d1d;
}
.live-pulse {
  width: 8px; height: 8px; border-radius: 50%;
  background: #fff;
  animation: livePulse 1.5s ease infinite;
}
@keyframes livePulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
.live-text { letter-spacing: 0.5px; }
.live-actions { display: flex; gap: 8px; }
.emergency-btn {
  padding: 2px 12px; font-size: 10px; font-weight: 700;
  background: #fff; color: var(--color-danger);
  border: none; border-radius: var(--radius-sm); cursor: pointer;
}
.emergency-btn:hover { background: #fecaca; }
.emergency-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.paper-switch-btn {
  padding: 2px 10px; font-size: 10px;
  background: transparent; color: #fff;
  border: 1px solid rgba(255,255,255,0.5); border-radius: var(--radius-sm); cursor: pointer;
}

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
