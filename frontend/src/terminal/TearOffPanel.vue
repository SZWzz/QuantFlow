<script setup lang="ts">
import { ref, onMounted, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import { getPanelComponent } from '@/terminal/panels/registry'
import { Call } from '@wailsio/runtime'
import { logger } from '@/lib/logger'

const route = useRoute()
const instanceId = route.params.instanceId as string
const panelId = ref('')
const label = ref('')
const params = ref<Record<string, any> | undefined>()
const panelComponent = shallowRef<any>(null)
const error = ref('')
const loading = ref(true)

async function fetchPanelInfo() {
  try {
    // Try the shim bridge first
    const go = (window as any).go?.main?.App
    let info: any

    if (go?.GetTearOffPanelInfo) {
      info = await go.GetTearOffPanelInfo(instanceId)
    } else {
      // Fallback: direct Wails v3 Call.ByName
      info = await Call.ByName('main.App.GetTearOffPanelInfo', instanceId)
    }

    if (!info || !info[0]) {
      error.value = `Panel not found: ${instanceId}`
      return
    }

    panelId.value = info[0]
    label.value = info[1] || ''

    const paramsJson = info[2]
    if (paramsJson && paramsJson !== '{}' && paramsJson !== '""') {
      try { params.value = JSON.parse(paramsJson) } catch { /* ignore */ }
    }

    const comp = getPanelComponent(panelId.value)
    if (!comp) {
      error.value = `Unknown panel type: ${panelId.value}`
      return
    }

    panelComponent.value = comp
    // Update window title if possible
    if (label.value) {
      try { document.title = label.value } catch { /* ignore */ }
    }
  } catch (err: any) {
    logger.error('[TearOffPanel] failed:', err)
    error.value = err?.message || String(err)
  } finally {
    loading.value = false
  }
}

onMounted(fetchPanelInfo)
</script>

<template>
  <component
    v-if="panelComponent"
    :is="panelComponent"
    :panel-id="panelId"
    :params="params"
    class="tearoff-panel"
  />
  <div v-else class="tearoff-status">
    <template v-if="error">
      <p class="tearoff-error-icon">⚠️</p>
      <p class="tearoff-error-msg">{{ error }}</p>
      <p class="tearoff-error-detail">Instance: {{ instanceId }}</p>
    </template>
    <template v-else-if="loading">
      <p class="tearoff-loading-text">Loading panel...</p>
    </template>
  </div>
</template>

<style scoped>
.tearoff-panel {
  width: 100vw;
  height: 100vh;
  overflow: auto;
}
.tearoff-status {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100vh;
  gap: 8px;
  color: #888;
  font-family: system-ui, sans-serif;
  background: var(--color-bg-panel);
}
.tearoff-error-icon {
  font-size: 32px;
  margin: 0;
}
.tearoff-error-msg {
  font-size: 14px;
  color: #ef4444;
  margin: 0;
}
.tearoff-error-detail {
  font-size: 11px;
  color: #666;
  margin: 0;
  font-family: monospace;
}
.tearoff-loading-text {
  font-size: 14px;
  margin: 0;
}
</style>
