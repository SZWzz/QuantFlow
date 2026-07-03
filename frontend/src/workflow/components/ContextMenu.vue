<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  x: number
  y: number
  items: {
    label: string
    icon?: string
    shortcut?: string
    action?: () => void
    divider?: boolean
    disabled?: boolean
  }[]
}>()
const emit = defineEmits<{ (e: 'close'): void }>()

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div class="ctx-menu" :style="{ left: x + 'px', top: y + 'px' }" @click.stop>
    <template v-for="(item, i) in items" :key="i">
      <div v-if="item.divider" class="ctx-divider" />
      <button
        v-else
        class="ctx-item"
        :disabled="item.disabled"
        @click="item.action?.(); emit('close')"
      >
        <span v-if="item.icon" class="ctx-icon">{{ item.icon }}</span>
        <span class="ctx-label">{{ item.label }}</span>
        <span v-if="item.shortcut" class="ctx-shortcut">{{ item.shortcut }}</span>
      </button>
    </template>
  </div>
</template>

<style scoped>
.ctx-menu {
  position: fixed;
  z-index: 9999;
  min-width: 180px;
  background: #1c2333;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 4px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}
.ctx-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 12px;
  border: none;
  background: none;
  color: var(--color-text-primary);
  font-size: 12px;
  cursor: pointer;
  border-radius: 4px;
  text-align: left;
}
.ctx-item:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.1);
}
.ctx-item:disabled {
  opacity: 0.4;
  cursor: default;
}
.ctx-divider {
  height: 1px;
  background: var(--color-border);
  margin: 4px 8px;
}
.ctx-icon {
  width: 18px;
  text-align: center;
}
.ctx-shortcut {
  margin-left: auto;
  color: var(--color-text-tertiary);
  font-size: 10px;
}
</style>
