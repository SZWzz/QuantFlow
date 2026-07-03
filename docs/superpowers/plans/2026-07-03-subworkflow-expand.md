# Sub-Workflow Double-Click to Expand — Implementation Plan

## Task 1: Store navigation state

**File**: `frontend/src/stores/workflow.ts`

```ts
interface SubWFStackEntry {
  parentId: string
  parentNodes: VFNode[]
  parentEdges: Edge[]
  viewport: ViewportTransform
}

const subWFStack = ref<SubWFStackEntry[]>([])
const currentSubWFId = ref<string | null>(null)

function navigateIntoSubWF(nodeId: string, childWfId: string) {
  subWFStack.value.push({
    parentId: nodeId,
    parentNodes: JSON.parse(JSON.stringify(nodes.value)),
    parentEdges: JSON.parse(JSON.stringify(edges.value)),
    viewport: { ...viewport.value },
  })
  currentSubWFId.value = childWfId
  loadWorkflow(childWfId)  // existing method
}

function navigateUpFromSubWF() {
  const entry = subWFStack.value.pop()
  if (!entry) return
  currentSubWFId.value = subWFStack.value.length > 0 ? subWFStack.value[subWFStack.value.length - 1].parentId : null
  nodes.value = entry.parentNodes
  edges.value = entry.parentEdges
  viewport.value = entry.viewport
}
```

## Task 2: Canvas double-click + breadcrumb

**File**: `frontend/src/workflow/canvas/WorkflowCanvas.vue`

```ts
function onNodeDoubleClick(e: MouseEvent, node: any) {
  if (node.data?.nodeType === 'sub_workflow') {
    const childWfId = node.data.params?.workflow_id
    if (childWfId) workflow.navigateIntoSubWF(node.id, childWfId)
  }
}
```

Template addition above VueFlow:
```vue
<div v-if="workflow.currentSubWFId" class="subwf-breadcrumb">
  <button class="breadcrumb-back" @click="workflow.navigateUpFromSubWF()">← 返回</button>
  <span class="breadcrumb-path">{{ workflow.currentSubWFId }}</span>
</div>
```
```css
.subwf-breadcrumb { display: flex; align-items: center; gap: 8px; padding: 6px 12px; background: #161b22; border-bottom: 1px solid var(--color-border); font-size: 12px; }
.breadcrumb-back { padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 4px; background: none; color: var(--color-accent); cursor: pointer; }
```

Commit: `feat(workflow): add sub-workflow double-click expand with breadcrumb navigation`
