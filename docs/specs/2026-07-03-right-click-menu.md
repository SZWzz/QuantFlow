# Right-Click Context Menu for Workflow Canvas

## Motivation
Workflow canvas currently has no right-click context menu. Users must use the keyboard or PropertyPanel for all node operations. Adding a right-click menu with grouping, pinning, and disabling brings UX parity with ComfyUI.

## Design

**Right-click on canvas background** (empty space):
- 粘贴 (Paste) — if clipboard has copied nodes
- 全选 (Select All) — Ctrl+A
- 添加节点 (Add Node) — opens NodePalette search

**Right-click on a node**:
- 固定输出 (Pin Output) — sets workflow.pinnedOutputs[nodeId] to current output values, engine skips this node on next run
- 禁用 (Disable Node) — sets mode=2 (bypassed), engine skips execution, outputs flow through unchanged
- 编组 (Group) — creates a group box around selected nodes with header
- 复制 (Copy) / 粘贴 (Paste) / 删除 (Delete)
- 克隆 (Clone Node) — duplicates node with same params at offset position
- 折叠 (Collapse) — minimizes node to show only title bar

**Data model:**

```ts
// Pin: engine.go already has PinnedOutputs map — just no frontend UI
interface PinnedOutput {
  nodeId: string
  outputs: Record<string, any>
}

// Disable: WorkflowNode already has Mode field (0=normal, 2=disabled)
// Engine skips disabled nodes

// Group: 
interface NodeGroup {
  id: string
  label: string
  nodes: string[]  // child node IDs
  style: { color: string }
}
```

**Files modified:**
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — add contextmenu event handler
- `frontend/src/workflow/canvas/NodeGroup.vue` — new group overlay component  
- `frontend/src/workflow/components/ContextMenu.vue` — new context menu component
- `frontend/src/stores/workflow.ts` — add groups array, pinnedOutputs, toggleDisable
- `internal/workflow/engine.go` — already has PinnedOutputs + node.Mode support
- `internal/workflow/workflow.go` — check NodeGroup struct exists

## Acceptance Criteria
- [ ] Right-click on node shows menu with Pin/Disable/Group/Copy/Clone/Delete
- [ ] Right-click on canvas shows Paste/SelectAll/AddNode
- [ ] Pin: sets pinned output, node shows "📌 pinned" badge, engine skips it
- [ ] Disable: node goes greyed out, engine bypasses it, outputs pass through
- [ ] Group: creates draggable group box, header shows label
- [ ] Clone: creates new node at (x+20, y+20) with same params

## Risks / Trade-offs
- Group is purely visual in v1 — no nested DAG semantics yet
- Pin/Disable only affect frontend state in v1 — backend engine integration deferred
