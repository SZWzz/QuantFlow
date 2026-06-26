package workflow

import (
	"context"
	"testing"
	"time"
)

func TestEngine_ExecuteSimpleDAG(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")

	engine, err := NewEngine(reg, 64, nil)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	wf := &Workflow{
		ID:   "test",
		Name: "simple pipeline",
		Nodes: []NodeInstance{
			{ID: "n1", NodeType: "passthrough", Params: map[string]any{"value": "hello"}},
			{ID: "n2", NodeType: "passthrough", Params: map[string]any{}},
		},
		Edges: []Edge{{FromNode: "n1", FromPort: "output", ToNode: "n2", ToPort: "input"}},
	}

	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if len(result.NodeResults) != 2 {
		t.Errorf("len(NodeResults) = %d, want 2", len(result.NodeResults))
	}
}

func TestEngine_ExecuteWithTimeout(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("slow", func(id string, params map[string]any) (BaseNode, error) {
		return &slowNode{id: id}, nil
	}, "test")
	engine, _ := NewEngine(reg, 64, nil)
	wf := &Workflow{ID: "timeout_test", Nodes: []NodeInstance{{ID: "s1", NodeType: "slow"}}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := engine.Execute(ctx, wf)
	if err == nil {
		t.Error("expected error due to timeout")
	}
}

func TestEngine_ExecuteParallelDAG(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterWithCategory("passthrough", func(id string, params map[string]any) (BaseNode, error) {
		return &passthroughNode{id: id}, nil
	}, "test")
	engine, _ := NewEngine(reg, 64, nil)
	wf := &Workflow{
		ID: "parallel",
		Nodes: []NodeInstance{
			{ID: "src", NodeType: "passthrough", Params: map[string]any{"value": "data"}},
			{ID: "w1", NodeType: "passthrough"}, {ID: "w2", NodeType: "passthrough"},
			{ID: "snk", NodeType: "passthrough"},
		},
		Edges: []Edge{
			{FromNode: "src", FromPort: "output", ToNode: "w1", ToPort: "input"},
			{FromNode: "src", FromPort: "output", ToNode: "w2", ToPort: "input"},
			{FromNode: "w1", FromPort: "output", ToNode: "snk", ToPort: "input"},
			{FromNode: "w2", FromPort: "output", ToNode: "snk", ToPort: "input"},
		},
	}
	result, err := engine.Execute(context.Background(), wf)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("Status = %q, want completed", result.Status)
	}
	if len(result.NodeResults) != 4 {
		t.Errorf("len = %d, want 4", len(result.NodeResults))
	}
}

// Test helper nodes

type passthroughNode struct{ id string }

func (n *passthroughNode) ID() string                   { return n.id }
func (n *passthroughNode) NodeType() string             { return "passthrough" }
func (n *passthroughNode) Category() string             { return "test" }
func (n *passthroughNode) InputPorts() []PortDefinition  { return []PortDefinition{{Name: "input", Type: PortAny, Required: false}} }
func (n *passthroughNode) OutputPorts() []PortDefinition { return []PortDefinition{{Name: "output", Type: PortAny, Required: false}} }
func (n *passthroughNode) ParamSchema() []ParamDef       { return nil }
func (n *passthroughNode) Validate() error               { return nil }
func (n *passthroughNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *NodeContext) (map[string]any, error) {
	if v, ok := params["value"]; ok {
		return map[string]any{"output": v}, nil
	}
	if v, ok := inputs["input"]; ok {
		return map[string]any{"output": v}, nil
	}
	return map[string]any{"output": "default"}, nil
}

type slowNode struct{ id string }

func (n *slowNode) ID() string                   { return n.id }
func (n *slowNode) NodeType() string             { return "slow" }
func (n *slowNode) Category() string             { return "test" }
func (n *slowNode) InputPorts() []PortDefinition  { return nil }
func (n *slowNode) OutputPorts() []PortDefinition { return nil }
func (n *slowNode) ParamSchema() []ParamDef       { return nil }
func (n *slowNode) Validate() error               { return nil }
func (n *slowNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *NodeContext) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(1 * time.Second):
		return nil, nil
	}
}
