# Sub-Workflow Double-Click to Expand

## Motivation
SubWorkflowNode exists (`internal/workflow/nodes/sub_workflow.go`) and the runner is wired in `app.go:372-398`, but there is no UI for double-clicking a sub-workflow node to expand/edit its nested workflow. Users must edit the sub-workflow separately. ComfyUI allows double-clicking a subgraph node to navigate into it.

## Design

**Frontend**: Double-click on a sub_workflow node navigates the canvas into the child workflow. A breadcrumb bar appears at the top showing the path. An "Up" button returns to parent.

**Backend**: The `LoadWorkflow` API returns child workflows. `SubWorkflowNode` stores the child workflow ID.

**Data model:**
```ts
interface SubWorkflowState {
  stack: { parentId: string; wfId: string; viewport: ViewportTransform }[]
  currentWfId: string | null
}
```

**Files modified:**
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — double-click handler + breadcrumb
- `frontend/src/stores/workflow.ts` — subworkflow stack state
- `internal/workflow/nodes/sub_workflow.go` — store child workflow ID in params

## Acceptance Criteria
- [ ] Double-click sub_workflow node navigates canvas into child workflow
- [ ] Breadcrumb bar shows parent → child path
- [ ] Up button returns to parent with original viewport restored
- [ ] Changes in child workflow are saved on exit

## Risks / Trade-offs
- Sub-workflow editing could lead to infinite recursion (circular refs). Guard with depth limit.
