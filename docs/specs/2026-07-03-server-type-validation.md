# Server-Side Port Type Validation

## Motivation
Currently port type compatibility is checked only on the frontend (`workflow.ts:canConnectPorts()`). The Go backend accepts any connection regardless of type. This leads to runtime errors in node.Execute() that could be caught earlier.

## Design
Add type validation to both `Validate(&wf)` and the engine's execute path.

**Type compatibility matrix:**

| PortType | Compatible types |
|---|---|
| `ohlcv` | `ohlcv` |
| `series` | `series`, `ohlcv` |
| `signal` | `signal` |
| `string` | `string` |
| `number` | `number`, `signal` |
| `boolean` | `boolean` |
| `any` | everything |

**Validation in validate.go:**

```go
func ValidateEdgeTypes(wf *Workflow, registry *NodeRegistry) error {
    for _, edge := range wf.Edges {
        srcNode := findNode(wf, edge.FromNode)
        dstNode := findNode(wf, edge.ToNode)
        // get output port type from src
        // get input port type from dst
        if !typesCompatible(srcType, dstType) {
            return fmt.Errorf("edge %s->%s: type mismatch %s != %s", ...)
        }
    }
    return nil
}
```

**Files modified:**
- `internal/workflow/validate.go` — add type checking to Validate
- `internal/workflow/workflow.go` — add typesCompatible helper
- `frontend/src/stores/workflow.ts` — canConnectPorts stays for frontend guard

## Acceptance Criteria
- [ ] Validate rejects edges with incompatible types
- [ ] Error message includes node IDs and types
- [ ] Existing valid workflows still pass

## Risks / Trade-offs
None. This is a pure safety net — no behavioural change for valid workflows.
