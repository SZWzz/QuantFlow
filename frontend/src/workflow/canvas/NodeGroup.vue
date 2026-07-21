<script setup lang="ts">
defineProps<{
  id: string
  label: string
  nodes: string[]
  style: { color: string }
}>()
</script>

<template>
  <div class="node-group" :style="{ borderColor: style.color }">
    <div class="group-header" :style="{ background: style.color }">
      <span class="group-label">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.node-group {
  position: absolute;
  border: 2px dashed;
  border-radius: 12px;
  padding: 0;
  pointer-events: none;
  min-height: 60px;
  min-width: 120px;
  z-index: 1;
}
.group-header {
  padding: 4px 12px;
  border-radius: 9px 9px 0 0;
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--wf-node-header-text);
  user-select: none;
}
.group-label {
  opacity: 0.8;
}
</style>
