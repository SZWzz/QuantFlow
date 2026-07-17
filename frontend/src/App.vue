<script setup lang="ts">
import { watch, onMounted, computed, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useThemeStore } from '@/lib/theme'
import { useUpdateStore } from '@/stores/update'
import { useCrashStore } from '@/stores/crash'
import UpdatePrompt from '@/terminal/components/UpdatePrompt.vue'
import CrashDialog from '@/terminal/components/CrashDialog.vue'
import ToastContainer from '@/terminal/components/ToastContainer.vue'

// Init theme at mount — sets body classes and watches reactive session state
onMounted(() => {
  const t = useThemeStore()
  t.apply()
})

const session = useSessionStore()
const router = useRouter()
const route = useRoute()
const isTearOff = computed(() => route.path.startsWith('/tearoff'))

// --- Update auto-check ---
const updateStore = useUpdateStore()
const showUpdatePrompt = ref(false)

onMounted(() => {
  // Auto-check for updates 30 seconds after mount, so the app is fully loaded
  setTimeout(async () => {
    if (updateStore.shouldCheck()) {
      const info = await updateStore.check()
      if (info?.has_update) {
        showUpdatePrompt.value = true
      }
      updateStore.markChecked()
    }
  }, 30000)
})

async function handleApplyUpdate() {
  await updateStore.apply()
  showUpdatePrompt.value = false
}

// --- Crash recovery dialog (shown on next launch after a crash) ---
const crashStore = useCrashStore()
const showCrashDialog = ref(false)
// Anonymous upload is strictly opt-in for a trading app: the dialog checkbox
// defaults to UNCHECKED and emits `upload` whenever the user toggles it.
const sendCrashReport = ref(false)

onMounted(async () => {
  // Tear-off windows must not show the crash dialog — only the main window.
  if (isTearOff.value) return
  await crashStore.init()
  if (crashStore.pending) {
    showCrashDialog.value = true
  }
})

/** Dismiss the crash dialog: upload if opted in, then acknowledge. */
async function handleCrashDismiss() {
  const pending = crashStore.pending
  if (sendCrashReport.value && pending) {
    const ok = await crashStore.upload(pending.id)
    // No dialog on the dismiss path — the report stays on disk; just warn.
    if (!ok) console.warn('[Crash] report upload failed; report kept locally')
  }
  crashStore.ack()
  showCrashDialog.value = false
}

// Sync theme/density body classes when session changes
watch(() => [session.ui.theme, session.ui.density], () => {
  const t = useThemeStore()
  t.apply()
})

// Keep URL in sync with session mode — this runs in the root component
// so it survives route changes (TerminalMode ↔ WorkflowMode).
// Skip in tear-off windows: they run their own panel, no mode toggle.
if (!route.path.startsWith('/tearoff')) {
  watch(() => session.ui.mode, (mode) => {
    const target = mode === 'workflow' ? '/workflow' : '/'
    if (route.path !== target) router.push(target)
  }, { immediate: true })
}

// Keep session mode in sync with URL (back/forward browser buttons).
// Skip in tear-off windows.
if (!route.path.startsWith('/tearoff')) {
  watch(() => route.path, (path) => {
    const expectedMode = path === '/workflow' ? 'workflow' : 'terminal'
    if (session.ui.mode !== expectedMode) {
      session.ui.mode = expectedMode
    }
  })
}
</script>

<template>
  <div class="app">
    <router-view />
    <UpdatePrompt
      v-if="updateStore.updateInfo"
      :visible="showUpdatePrompt"
      :current-version="updateStore.updateInfo.current_version"
      :latest-version="updateStore.updateInfo.latest_version"
      :changelog="updateStore.updateInfo.changelog"
      @close="showUpdatePrompt = false"
      @update="handleApplyUpdate"
    />
    <CrashDialog
      v-if="crashStore.pending"
      :visible="showCrashDialog"
      :crash-time="crashStore.pending.timestamp"
      :crash-path="crashStore.crashDir"
      :report="crashStore.pending"
      @close="handleCrashDismiss"
      @restart="handleCrashDismiss"
      @upload="(send) => sendCrashReport = send"
    />
    <ToastContainer />
  </div>
</template>

<style>
.app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>
