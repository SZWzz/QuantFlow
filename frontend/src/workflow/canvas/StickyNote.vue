<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  id: string
  data: { text?: string; color?: string }
  selected?: boolean
}>()

const colors = ['#fef3c7', '#dbeafe', '#fce7f3', '#d1fae5', '#e0e7ff']
const color = ref(props.data.color || colors[Math.floor(Math.random() * colors.length)])
const editing = ref(false)
const text = ref(props.data.text || '双击编辑...')

function startEdit() { editing.value = true }
function finishEdit() { editing.value = false }
</script>

<template>
  <div class="sticky-note" :class="{ selected }" :style="{ background: color }" @dblclick.stop="startEdit">
    <textarea
      v-if="editing"
      v-model="text"
      class="sticky-textarea"
      @blur="finishEdit"
      @keyup.escape="finishEdit"
      autofocus
      @click.stop
    />
    <div v-else class="sticky-text">{{ text }}</div>
  </div>
</template>

<style scoped>
.sticky-note {
  width: 180px;
  min-height: 60px;
  padding: 12px;
  border-radius: 4px;
  box-shadow: 2px 3px 8px rgba(0,0,0,.4);
  font-size: 12px;
  line-height: 1.5;
  cursor: grab;
  transform: rotate(-1deg);
  transition: box-shadow .15s, transform .15s;
}
.sticky-note.selected {
  box-shadow: 0 0 0 2px var(--color-accent);
  transform: rotate(0deg);
}
.sticky-text {
  color: #1e293b;
  white-space: pre-wrap;
  word-break: break-word;
}
.sticky-textarea {
  width: 100%;
  min-height: 50px;
  background: transparent;
  border: none;
  outline: none;
  resize: vertical;
  font-family: inherit;
  font-size: 12px;
  color: #1e293b;
  line-height: 1.5;
}
</style>
