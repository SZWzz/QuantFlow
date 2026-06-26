# Workflow Mode Bugfixes — Implementation Plan (9 tasks)

> **Spec:** [docs/specs/2026-06-26-workflow-mode-fixes.md](../../specs/2026-06-26-workflow-mode-fixes.md)

**Priority:** P0 items first (status string → cache key → nr.Status → fromWorkflowJSON), then P1.

---

## P0: Critical (4 tasks)

### P0-1: Fix status string mismatch (`engine.go`)

**Files:**
- `internal/workflow/engine.go`

**Changes:**
```go
// Line 98: result.Status = "success" → "completed"
result.Status = "completed"

// Line 140: nr.Status = "success" → "completed"
nr.Status = "completed"

// Line 156: nr.Status = "success" → "completed"
nr.Status = "completed"
```

**Tests (`engine_test.go` + `integration_test.go`):**
- Expect `result.Status == "completed"` instead of `"success"`

**Commit:** `[Fix] workflow: align Go status "success" → "completed" for frontend`

---

### P0-2: Fix cache key non-determinism (`cache.go`)

**Files:**
- `internal/workflow/cache.go`

**Changes:** Sort map keys before hashing:
```go
import "sort"

func CacheKey(nodeID string, inputs map[string]any) string {
    keys := make([]string, 0, len(inputs))
    for k := range inputs { keys = append(keys, k) }
    sort.Strings(keys)
    var b strings.Builder
    b.WriteString(nodeID)
    for _, k := range keys {
        fmt.Fprintf(&b, "|%s:%v", k, inputs[k])
    }
    hash := sha256.Sum256([]byte(b.String()))
    return fmt.Sprintf("%x", hash[:16])
}
```

**Tests:** Add a test in existing test file that creates two identical input maps in different key order and asserts same cache key.

**Commit:** `[Fix] workflow: deterministic CacheKey via sorted map keys`

---

### P0-3: Fix missing `nr.Status = "failed"` (`engine.go`)

**Files:**
- `internal/workflow/engine.go`

**Changes:**
```go
// Line 112-113: before return, set nr.Status
if nodeInstance == nil {
    nr.Status = "failed"          // ADD THIS LINE
    return fmt.Errorf("node %q not found in workflow", nodeID)
}
```

**Tests:** Add test case in `engine_test.go` that references a non-existent node and checks `nr.Status`.

**Commit:** `[Fix] workflow: set nr.Status=failed when nodeInstance is nil`

---

### P0-4: Fix `fromWorkflowJSON` edge reconnection (`workflow.ts`)

**Files:**
- `frontend/src/stores/workflow.ts`

**Changes:** Rewrite `fromWorkflowJSON` to use an ID map:
```typescript
function fromWorkflowJSON(wf: WorkflowJSON) {
    pushHistory()
    nodes.value = []
    edges.value = []
    resetExecution()

    const nodeIdMap = new Map<string, string>() // oldID → newID
    for (const n of wf.nodes) {
        const newId = addNode(n.node_type, {
            x: 100 + Math.random() * 300,
            y: 100 + Math.random() * 200,
        }, n.params)
        nodeIdMap.set(n.id, newId)
    }

    for (const e of wf.edges) {
        const sourceId = nodeIdMap.get(e.from_node)
        const targetId = nodeIdMap.get(e.to_node)
        if (sourceId && targetId && sourceId !== targetId) {
            edges.value.push({
                id: `e-${sourceId}-${targetId}`,
                source: sourceId,
                target: targetId,
                sourceHandle: e.from_port,
                targetHandle: e.to_port,
                type: 'smoothstep',
                style: { stroke: '#30363d', strokeWidth: 2 },
            })
        }
    }
}
```

**Test:** Use `vue-tsc --noEmit` to check types. Add a test that round-trips a workflow with 2 SMA nodes and verifies edges.

**Commit:** `[Fix] frontend: fromWorkflowJSON use ID map for edge reconnection`

---

## P1: High (4 tasks)

### P1-1: Dynamic port mapping via `GetNodePorts` API

**Files:**
- `app.go` — add `GetNodePorts` export
- `frontend/src/stores/workflow.ts` — `addNode` use dynamic ports
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — `onDrop` fetch ports
- `frontend/src/lib/wails.ts` — add `GetNodePorts` typed wrapper

**Changes:**

```go
// app.go
func (a *App) GetNodePorts(nodeType string) (map[string]any, error) {
    meta := a.wfRegistry.ListAll()
    for _, m := range meta {
        if m.NodeType == nodeType {
            node, err := a.wfRegistry.Create(nodeType, "__dummy__", nil)
            if err != nil {
                return nil, err
            }
            inputs := make([]map[string]any, 0)
            for _, p := range node.InputPorts() {
                inputs = append(inputs, map[string]any{"name": p.Name, "type": p.Type})
            }
            outputs := make([]map[string]any, 0)
            for _, p := range node.OutputPorts() {
                outputs = append(outputs, map[string]any{"name": p.Name, "type": p.Type})
            }
            return map[string]any{"inputs": inputs, "outputs": outputs}, nil
        }
    }
    return nil, fmt.Errorf("node type %q not found", nodeType)
}
```

```typescript
// wails.ts
export async function GetNodePorts(nodeType: string): Promise<{ inputs: Array<{name: string, type: string}>, outputs: Array<{name: string, type: string}> }> {
    return wailsCall('GetNodePorts', nodeType)
}

// workflow.ts - addNode: optional ports parameter, fallback to API call
async function addNode(type, position, params?) {
    pushHistory()
    const id = `${type}-${Date.now()}`
    let ports = portMap[type]
    if (!ports) {
        try {
            const result = await GetNodePorts(type)
            ports = {
                inputs: result.inputs.map(p => p.name),
                outputs: result.outputs.map(p => p.name),
            }
        } catch {
            ports = { inputs: ['input'], outputs: ['output'] }
        }
    }
    nodes.value.push({...})
    return id
}
```

**Note:** `addNode` becomes async. Update callers in `WorkflowCanvas.vue` and `fromWorkflowJSON` with `await`.

**Commit:** `[Feat] workflow: dynamic port mapping via GetNodePorts API`

---

### P1-2: Fix Pin to Terminal symbol

**Files:**
- `frontend/src/workflow/PropertyPanel.vue`
- `frontend/src/workflow/WorkflowMode.vue`

**Changes:**
```typescript
// PropertyPanel.vue:53
terminal.openPanel(panelId, {
    symbol: node.data.params?.symbol || '600519'
})

// WorkflowMode.vue:88,104  
// Already reads params.symbol at line 88, fix line 104:
terminal.openPanel('candlestick', {
    symbol: node.data.params?.symbol || '600519'
})
```

**Commit:** `[Fix] frontend: pin-to-terminal use node param symbol instead of hardcoded AAPL`

---

### P1-3: NodePalette use typed wrapper

**Files:**
- `frontend/src/workflow/NodePalette.vue`

**Changes:**
```typescript
import { ListNodes } from '@/lib/wails'
// ...
const result = await ListNodes()  // instead of (window as any).go.main.App.ListNodes()
```

**Commit:** `[Fix] frontend: NodePalette use typed ListNodes wrapper`

---

### P1-4: Implement WorkflowExecutor for scheduler

**Files:**
- `app.go` — add `ExecuteWorkflowByID` method; wire scheduler with executor

**Changes:**
```go
// app.go — add method
func (a *App) ExecuteWorkflowByID(ctx context.Context, workflowID string) (string, error) {
    repo := storage.NewWorkflowRepo(a.db)
    wf, err := repo.Load(workflowID, nil)
    if err != nil {
        return "", fmt.Errorf("load workflow %q: %w", workflowID, err)
    }
    result, err := a.engine.Execute(ctx, wf)
    if err != nil {
        return "", err
    }
    return result.WorkflowID, nil
}
```

```go
// app.go — line 244, replace nil exec
wfExecutor := workflowExecutorAdapter{a: a}  // adapter implementing WorkflowExecutor
a.sched = schedule.New(a.db, wfExecutor, nil)
a.sched.Start()
```

```go
// app.go — add adapter type
type workflowExecutorAdapter struct { a *App }
func (w workflowExecutorAdapter) Execute(ctx context.Context, wfID string) (string, error) {
    return w.a.ExecuteWorkflowByID(ctx, wfID)
}
```

**Commit:** `[Feat] workflow: wire WorkflowExecutor to scheduler`

---

## Build + Test + CHANGELOG

### Verify

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go vet ./internal/workflow/...
go test ./internal/workflow/... -v -count=1 -run TestEngine
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

### Update CHANGELOG.md

Add entries for all changes.

### Version bump

Check `frontend/package.json` version matches today.

**Commit:** `[Chore] CHANGELOG: workflow mode P0-P1 fixes`
