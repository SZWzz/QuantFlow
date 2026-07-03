# Implementation Plan: Server-Side Port Type Validation

## Step 1: Add type compatibility check

**File**: `internal/workflow/workflow.go`

Add the `TypesCompatible` helper function:

```go
// TypesCompatible returns true if output type can connect to input type.
func TypesCompatible(output, input PortType) bool {
    if output == PortAny || input == PortAny {
        return true
    }
    if output == input {
        return true
    }
    // ohlcv is compatible with series
    if output == PortOHLCV && input == PortSeries {
        return true
    }
    // signal is compatible with number
    if output == PortSignal && input == PortNumber {
        return true
    }
    return false
}
```

Commit: `feat(workflow): add TypesCompatible helper for port type validation`

## Step 2: Add validation in Validate()

**File**: `internal/workflow/validate.go`

In the `Validate` function, add after cycle detection:

```go
for _, edge := range wf.Edges {
    srcNode := findNode(wf.Nodes, edge.FromNode)
    dstNode := findNode(wf.Nodes, edge.ToNode)
    if srcNode == nil {
        return fmt.Errorf("edge from %q: source node not found", edge.FromNode)
    }
    if dstNode == nil {
        return fmt.Errorf("edge to %q: target node not found", edge.ToNode)
    }
    // Get source output port type
    srcNodeImpl, err := registry.Create(srcNode.NodeType, "_val", nil)
    if err != nil {
        continue
    }
    var srcType PortType
    for _, p := range srcNodeImpl.OutputPorts() {
        if p.Name == edge.FromPort {
            srcType = p.Type
            break
        }
    }
    // Get target input port type
    dstNodeImpl, err := registry.Create(dstNode.NodeType, "_val", nil)
    if err != nil {
        continue
    }
    var dstType PortType
    for _, p := range dstNodeImpl.InputPorts() {
        if p.Name == edge.ToPort {
            dstType = p.Type
            break
        }
    }
    if srcType != "" && dstType != "" && !TypesCompatible(srcType, dstType) {
        return fmt.Errorf("edge %s.%s \u2192 %s.%s: type mismatch %s \u2192 %s",
            edge.FromNode, edge.FromPort, edge.ToNode, edge.ToPort, srcType, dstType)
    }
}
```

Add `findNode` helper at package level:

```go
func findNode(nodes []WorkflowNode, id string) *WorkflowNode {
    for i := range nodes {
        if nodes[i].ID == id {
            return &nodes[i]
        }
    }
    return nil
}
```

Commit: `feat(workflow): add server-side port type validation to Validate()`
