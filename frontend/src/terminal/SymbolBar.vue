<script setup lang="ts">
import { ref } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'

const ctx = useSymbolContext()
const inputVal = ref('')

function submit() {
  if (inputVal.value.trim()) {
    ctx.setGroupSymbol(ctx.activeGroupId, inputVal.value.trim())
    inputVal.value = ''
  }
}

const groups = Object.entries(ctx.linkGroups)
</script>

<template>
  <div class="symbol-bar">
    <div class="group-tabs">
      <button v-for="[id, g] in groups" :key="id"
        :class="{ active: ctx.activeGroupId === id }"
        :style="{ '--gcolor': g.color }"
        @click="ctx.setActiveGroup(id)">
        <span class="dot" :style="{ background: g.color }"></span>
        {{ g.activeSymbol || '--' }}
      </button>
    </div>
    <form class="symbol-input-area" @submit.prevent="submit">
      <input v-model="inputVal" placeholder="Enter symbol..." />
    </form>
  </div>
</template>

<style scoped>
.symbol-bar {
  display: flex; align-items: center; padding: 4px 10px;
  background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border);
  min-height: 30px; gap: 8px;
}
.group-tabs { display: flex; gap: 4px; }
.group-tabs button {
  display: flex; align-items: center; gap: 4px;
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-secondary);
  border-radius: var(--radius-sm); font-size: var(--font-xs); cursor: pointer;
}
.group-tabs button.active { border-color: var(--gcolor); color: var(--color-text-primary); }
.dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.symbol-input-area { flex: 1; }
.symbol-input-area input {
  width: 100%; padding: 2px 8px; background: var(--color-bg-input);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: var(--font-xs); outline: none;
}
.symbol-input-area input:focus { border-color: var(--color-accent); }
</style>
