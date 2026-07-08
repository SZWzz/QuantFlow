<script setup lang="ts">
import { ref, onMounted, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import { getPanelComponent } from '@/terminal/panels/registry'

const route = useRoute()
const instanceId = route.params.instanceId as string
const panelId = ref('')
const params = ref<Record<string, any> | undefined>()
const panelComponent = shallowRef<any>(null)

onMounted(async () => {
  try {
    const go = (window as any).go?.main?.App
    if (!go) {
      console.error('[TearOffPanel] Wails bridge not available')
      return
    }
    const info: [string, string, string] = await go.GetTearOffPanelInfo(instanceId)
    panelId.value = info[0]
    const paramsJson = info[2]
    if (paramsJson && paramsJson !== '{}' && paramsJson !== '""') {
      try { params.value = JSON.parse(paramsJson) } catch { params.value = undefined }
    }
    panelComponent.value = getPanelComponent(panelId.value)
  } catch (err) {
    console.error('[TearOffPanel] failed to get panel info:', err)
  }
})
</script>

<template>
  <component
    v-if="panelComponent"
    :is="panelComponent"
    :panel-id="panelId"
    :params="params"
    class="tearoff-panel"
  />
  <div v-else class="tearoff-loading">
    <p>Loading panel...</p>
  </div>
</template>

<style scoped>
.tearoff-panel {
  width: 100vw;
  height: 100vh;
  overflow: auto;
}
.tearoff-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  color: #888;
  font-family: system-ui, sans-serif;
}
</style>
