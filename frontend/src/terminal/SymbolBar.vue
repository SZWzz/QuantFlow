<script setup lang="ts">
import { useSymbolContext } from '@/stores/symbolContext'
import SymbolSearch from './SymbolSearch.vue'
import type { StockEntry } from '@/lib/symbolSearch'
import { getIcon } from '@/lib/icons'

const ctx = useSymbolContext()

function onSelect(entry: StockEntry) {
  ctx.setGroupSymbol(ctx.activeGroupId, entry.code)
}

const groups = Object.entries(ctx.linkGroups)
</script>

<template>
  <div class="symbol-bar">
    <div class="group-tabs">
      <button
        v-for="[id, g] in groups"
        :key="id"
        :class="{ active: ctx.activeGroupId === id }"
        :style="{ '--gcolor': g.color }"
        @click="ctx.setActiveGroup(id)"
      >
        <span class="dot" :style="{ background: g.color }"></span>
        <span class="group-symbol">{{ g.activeSymbol || '--' }}</span>
      </button>
    </div>
    <div class="symbol-divider" />
    <div class="symbol-input-area">
      <span class="search-icon" v-html="getIcon('search')" />
      <SymbolSearch
        :placeholder="$t('common.search') + '...'"
        @select="onSelect"
      />
    </div>
  </div>
</template>

<style scoped>
.symbol-bar {
  display: flex;
  align-items: center;
  padding: 5px 12px;
  background: var(--color-bg-subtle);
  background-image: var(--gradient-card);
  border-bottom: 1px solid var(--color-border);
  min-height: 34px;
  gap: 10px;
}

.group-tabs {
  display: flex;
  gap: 4px;
}

.group-tabs button {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-panel);
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  font-size: var(--font-xs);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  overflow: hidden;
}

.group-tabs button::before {
  content: '';
  position: absolute;
  bottom: 0;
  left: 4px;
  right: 4px;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--gcolor);
  opacity: 0;
  transition: opacity var(--transition-fast);
}

.group-tabs button:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.group-tabs button.active {
  border-color: var(--gcolor);
  color: var(--color-text-primary);
  background: var(--color-bg-panel);
}

.group-tabs button.active::before {
  opacity: 1;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.group-symbol {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
}

.symbol-divider {
  width: 1px;
  height: 18px;
  background: var(--color-border);
  margin: 0 4px;
}

.symbol-input-area {
  flex: 1;
  max-width: 280px;
  display: flex;
  align-items: center;
  gap: 8px;
  position: relative;
}

.search-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
  position: absolute;
  left: 10px;
  z-index: 1;
  pointer-events: none;
}

.search-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.symbol-input-area :deep(input) {
  padding-left: 30px;
  width: 100%;
}
</style>
