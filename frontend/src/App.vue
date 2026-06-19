<script setup lang="ts">
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
</script>

<template>
  <div class="app" :class="[`theme-${session.ui.theme}`]">
    <router-view />
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Microsoft YaHei', sans-serif;
  overflow: hidden;
}

.app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}

.app.theme-dark {
  background: #0d1117;
  color: #c9d1d9;
}

.app.theme-light {
  background: #ffffff;
  color: #24292f;
}
</style>
