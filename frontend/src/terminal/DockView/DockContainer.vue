<script setup lang="ts">
import { computed } from 'vue'
import type { DockLayoutTree } from './types'
import DockTab from './DockTab.vue'
import DockSplitter from './DockSplitter.vue'

const props = defineProps<{
  node: DockLayoutTree
}>()

const emit = defineEmits<{
  (e: 'update-layout', layout: DockLayoutTree): void
  (e: 'select-tab', leafId: string, tabId: string): void
  (e: 'close-tab', leafId: string, tabId: string): void
  (e: 'tab-drag', fromLeafId: string, tabId: string, toLeafId: string): void
  (e: 'split-ratio', containerId: string, index: number, ratios: number[]): void
}>()

const isHorizontal = computed(() => props.node.direction === 'row')

function onSplitResize(index: number, newRatios: number[]) {
  emit('split-ratio', props.node.id, index, newRatios)
}

function onSelectTab(leafId: string, tabId: string) {
  emit('select-tab', leafId, tabId)
}

function onCloseTab(leafId: string, tabId: string) {
  emit('close-tab', leafId, tabId)
}

function onChildSplitRatio(containerId: string, index: number, ratios: number[]) {
  emit('split-ratio', containerId, index, ratios)
}
</script>

<template>
  <div
    v-if="node.type === 'container'"
    class="dock-container"
    :class="{ 'direction-row': isHorizontal, 'direction-column': !isHorizontal }"
  >
    <template v-for="(child, idx) in node.children" :key="child.id">
      <DockSplitter
        v-if="idx > 0"
        :direction="node.direction || 'row'"
        :index="idx - 1"
        :ratios="node.splitRatios || []"
        @resize="onSplitResize"
      />
      <div
        class="child-wrapper"
        :style="{ flex: (node.splitRatios?.[idx] || 1) * 1000 }"
      >
        <DockContainer
          :node="child"
          @update-layout="emit('update-layout', $event)"
          @select-tab="onSelectTab"
          @close-tab="onCloseTab"
          @split-ratio="onChildSplitRatio"
        />
      </div>
    </template>
  </div>

  <DockTab
    v-else-if="node.type === 'tab'"
    :tabs="node.tabs || []"
    :active-tab="node.activeTab || ''"
    :leaf-id="node.id"
    @select-tab="emit('select-tab', node.id, $event)" @close-tab="emit('close-tab', node.id, $event)" @tab-drag="emit('tab-drag', $event)"
  />
</template>

<style scoped>
.dock-container {
  display: flex;
  height: 100%;
  width: 100%;
  min-height: 0;
}

.dock-container.direction-row {
  flex-direction: row;
}

.dock-container.direction-column {
  flex-direction: column;
}

.child-wrapper {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
</style>
