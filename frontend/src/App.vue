<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useThemeStore } from '@/lib/theme'

// Init theme at mount — sets body classes
onMounted(() => {
  useThemeStore()
  // Re-apply if localStorage changed while app was closed
  const t = useThemeStore()
  t.apply()
  // Poll for changes every 2s (SettingsPanel updates localStorage → here picks up)
  setInterval(() => t.apply(), 2000)
})

const session = useSessionStore()
const router = useRouter()
const route = useRoute()

// Keep URL in sync with session mode — this runs in the root component
// so it survives route changes (TerminalMode ↔ WorkflowMode).
watch(() => session.ui.mode, (mode) => {
  const target = mode === 'workflow' ? '/workflow' : '/'
  if (route.path !== target) router.push(target)
}, { immediate: true })

// Keep session mode in sync with URL (back/forward browser buttons)
watch(() => route.path, (path) => {
  const expectedMode = path === '/workflow' ? 'workflow' : 'terminal'
  if (session.ui.mode !== expectedMode) {
    session.ui.mode = expectedMode
  }
})
</script>

<template>
  <div class="app" :class="[`theme-${session.ui.theme}`, `density-${session.ui.density}`]">
    <router-view />
  </div>
</template>

<style>
.app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>
