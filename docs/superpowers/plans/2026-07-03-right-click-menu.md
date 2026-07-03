# Implementation Plan: Right-Click Context Menu for Workflow Canvas

## Task 1: ContextMenu.vue component

**File**: `frontend/src/workflow/components/ContextMenu.vue`

Create reusable context menu component.

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

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
```

## Task 2: WorkflowCanvas contextmenu handler

**File**: `frontend/src/workflow/canvas/WorkflowCanvas.vue`

Add context menu import, state, and handlers. Wire up contextmenu events.

**Script additions** (after line 8 — existing imports):
```ts
import ContextMenu from '@/workflow/components/ContextMenu.vue'
```

**Script additions** (after `const workflow = useWorkflowStore()`):
```ts
const ctxMenu = ref<{ x: number; y: number; items: any[] } | null>(null)

function onNodeContextMenu(e: MouseEvent, node: any) {
  ctxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    items: [
      { label: '固定输出', icon: '📌', action: () => workflow.togglePin(node.id) },
      {
        label: '禁用', icon: '⏸', action: () => workflow.toggleDisable(node.id),
        disabled: node.data.mode === 2,
      },
      { label: '编组', icon: '📦', action: () => workflow.groupNodes([node.id]) },
      { label: '', divider: true },
      {
        label: '复制', icon: '📋', shortcut: 'Ctrl+C',
        action: () => workflow.copyNodes([node.id]),
      },
      {
        label: '克隆', icon: '📄',
        action: () => workflow.cloneNode(node.id),
      },
      {
        label: '删除', icon: '🗑', shortcut: 'Del',
        action: () => workflow.removeNode(node.id),
      },
    ],
  }
}

function onPaneContextMenu(e: MouseEvent) {
  ctxMenu.value = {
    x: e.clientX,
    y: e.clientY,
    items: [
      {
        label: '粘贴', icon: '📋', shortcut: 'Ctrl+V',
        disabled: !workflow.clipboard.length,
        action: () => workflow.pasteNodes(),
      },
      {
        label: '全选', icon: '🔲', shortcut: 'Ctrl+A',
        action: () => workflow.selectAllNodes(),
      },
      { label: '', divider: true },
      {
        label: '添加节点...', icon: '➕',
        action: () => { /* open palette — handled externally */ },
      },
    ],
  }
}

function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  const nodeEl = (e.target as HTMLElement).closest('.vue-flow__node')
  if (nodeEl) {
    const nodeId = nodeEl.getAttribute('data-id')
    const node = workflow.nodes.find((n: any) => n.id === nodeId)
    if (node) onNodeContextMenu(e, node)
  } else {
    onPaneContextMenu(e)
  }
}
```

**Template additions** (after `</VueFlow>` closing tag):
```vue
<ContextMenu v-if="ctxMenu" v-bind="ctxMenu" @close="ctxMenu = null" />
```

**Template event binding** — add `@contextmenu="onContextMenu"` to `.workflow-canvas` div:
```vue
<div class="workflow-canvas" @dragover="onDragOver" @drop="onDrop" @contextmenu="onContextMenu">
```

## Task 3: Store additions

**File**: `frontend/src/stores/workflow.ts`

Add group/pin/clipboard/disable state and actions.

**Interface additions** (before store definition):
```ts
export interface NodeGroup {
  id: string
  label: string
  nodes: string[]
  style: { color: string }
}
```

**Ref additions** (after `selectedNodeId`):
```ts
const groups = ref<NodeGroup[]>([])
const pinnedOutputs = ref<Map<string, boolean>>(new Map())
const clipboard = ref<VFNode[]>([])
const disabledNodes = ref<Set<string>>(new Set())
```

**Function additions** (after `selectNode`):
```ts
function togglePin(nodeId: string) {
  const map = pinnedOutputs.value
  if (map.has(nodeId)) map.delete(nodeId)
  else map.set(nodeId, true)
  updateNodeBadge(nodeId, 'pin', map.has(nodeId))
}

function toggleDisable(nodeId: string) {
  const set = disabledNodes.value
  if (set.has(nodeId)) set.delete(nodeId)
  else set.add(nodeId)
  const node = nodes.value.find(n => n.id === nodeId)
  if (node) node.data.mode = set.has(nodeId) ? 2 : 0
}

function groupNodes(nodeIds: string[]) {
  const id = `group-${Date.now()}`
  groups.value.push({ id, label: 'Group', nodes: nodeIds, style: { color: '#30363d' } })
}

function copyNodes(nodeIds: string[]) {
  clipboard.value = nodes.value
    .filter(n => nodeIds.includes(n.id))
    .map(n => JSON.parse(JSON.stringify(n)))
}

function pasteNodes() {
  if (!clipboard.value.length) return
  pushHistory()
  for (const n of clipboard.value) {
    const newId = `${n.data.nodeType}-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`
    nodes.value.push({
      ...JSON.parse(JSON.stringify(n)),
      id: newId,
      position: { x: n.position.x + 30, y: n.position.y + 30 },
      data: { ...n.data, status: 'idle' },
    })
  }
}

function cloneNode(nodeId: string) {
  const orig = nodes.value.find(n => n.id === nodeId)
  if (!orig) return
  pushHistory()
  const newId = `${orig.data.nodeType}-${Date.now()}`
  nodes.value.push({
    ...JSON.parse(JSON.stringify(orig)),
    id: newId,
    position: { x: orig.position.x + 30, y: orig.position.y + 30 },
    data: { ...orig.data, status: 'idle' },
  })
}

function selectAllNodes() {
  // VueFlow handles selection via fitView — placeholder for multi-select
}

function updateNodeBadge(nodeId: string, badge: string, active: boolean) {
  const node = nodes.value.find(n => n.id === nodeId)
  if (node) {
    if (!node.data.badges) node.data.badges = {}
    node.data.badges[badge] = active
  }
}
```

**Return additions** (add to return object):
```ts
groups,
pinnedOutputs,
clipboard,
disabledNodes,
togglePin,
toggleDisable,
groupNodes,
copyNodes,
pasteNodes,
cloneNode,
selectAllNodes,
```

## Task 4: NodeGroup.vue component

**File**: `frontend/src/workflow/canvas/NodeGroup.vue`

Simple visual group overlay.

```vue
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
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  user-select: none;
}
.group-label {
  opacity: 0.8;
}
</style>
```
