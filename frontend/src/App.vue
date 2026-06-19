<script setup lang="ts">
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
</script>

<template>
  <div class="app" :class="[`theme-${session.ui.theme}`, `density-${session.ui.density}`]">
    <router-view />
  </div>
</template>

<style>
@import '@/assets/themes.css';
</style>
