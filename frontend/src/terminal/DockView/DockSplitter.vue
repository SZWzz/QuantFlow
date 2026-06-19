<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  direction: 'row' | 'column'
  index: number
  ratios: number[]
}>()

const emit = defineEmits<{
  (e: 'resize', index: number, newRatios: number[]): void
}>()

const isDragging = ref(false)
const startPos = ref(0)
const startRatios = ref<number[]>([])

const isHorizontal = computed(() => props.direction === 'row')

function onMouseDown(e: MouseEvent) {
  isDragging.value = true
  startPos.value = isHorizontal.value ? e.clientX : e.clientY
  startRatios.value = [...props.ratios]
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  e.preventDefault()
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value) return
  const containerEl = (e.target as HTMLElement).parentElement
  if (!containerEl) return

  const rect = containerEl.getBoundingClientRect()
  const containerSize = isHorizontal.value ? rect.width : rect.height
  const currentPos = isHorizontal.value ? e.clientX : e.clientY
  const delta = (currentPos - startPos.value) / containerSize

  const newRatios = [...startRatios.value]
  newRatios[props.index] = Math.max(0.1, Math.min(0.9, startRatios.value[props.index] + delta))
  newRatios[props.index + 1] = Math.max(0.1, Math.min(0.9, startRatios.value[props.index + 1] - delta))

  emit('resize', props.index, newRatios)
}

function onMouseUp() {
  isDragging.value = false
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
}
</script>

<template>
  <div
    class="dock-splitter"
    :class="{ horizontal: isHorizontal, vertical: !isHorizontal, dragging: isDragging }"
    @mousedown="onMouseDown"
  >
    <div class="splitter-handle" />
  </div>
</template>

<style scoped>
.dock-splitter {
  flex-shrink: 0;
  position: relative;
  z-index: 10;
}

.dock-splitter.horizontal {
  width: 4px;
  cursor: col-resize;
}

.dock-splitter.vertical {
  height: 4px;
  cursor: row-resize;
}

.splitter-handle {
  position: absolute;
  inset: -2px;
  transition: background 0.15s;
}

.dock-splitter:hover .splitter-handle,
.dock-splitter.dragging .splitter-handle {
  background: var(--color-accent);
}

.dock-splitter.horizontal .splitter-handle {
  width: 3px;
  left: 50%;
  transform: translateX(-50%);
}

.dock-splitter.vertical .splitter-handle {
  height: 3px;
  top: 50%;
  transform: translateY(-50%);
}
</style>
