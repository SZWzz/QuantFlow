package workflow

import (
	"context"
	"testing"
)

// stubNode is a minimal BaseNode for testing the registry.
type stubNode struct {
	id       string
	nodeType string
	cat      string
	params   map[string]any
}

func (s *stubNode) ID() string                   { return s.id }
func (s *stubNode) NodeType() string              { return s.nodeType }
func (s *stubNode) Category() string              { return s.cat }
func (s *stubNode) InputPorts() []PortDefinition  { return nil }
func (s *stubNode) OutputPorts() []PortDefinition { return nil }
func (s *stubNode) ParamSchema() []ParamDef       { return nil }
func (s *stubNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	return nil, nil
}
func (s *stubNode) Validate() error { return nil }

func newStubNode(id string, params map[string]any) (BaseNode, error) {
	return &stubNode{id: id, nodeType: "stub", cat: "test", params: params}, nil
}

func TestRegistry_RegisterAndCreate(t *testing.T) {
	r := NewRegistry()
	r.Register("stub", newStubNode)

	node, err := r.Create("stub", "node1", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if node.ID() != "node1" {
		t.Errorf("ID() = %q, want %q", node.ID(), "node1")
	}
	if node.NodeType() != "stub" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "stub")
	}
}

func TestRegistry_CreateUnknownType(t *testing.T) {
	r := NewRegistry()
	_, err := r.Create("nonexistent", "n1", nil)
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register("stub_a", newStubNode)
	r.Register("stub_b", newStubNode)

	all := r.ListAll()
	if len(all) != 2 {
		t.Errorf("ListAll() len = %d, want 2", len(all))
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	r := NewRegistry()
	r.Register("dup", newStubNode)
	r.Register("dup", newStubNode)
	node, err := r.Create("dup", "n", nil)
	if err != nil {
		t.Fatalf("Create() after re-register error = %v", err)
	}
	if node == nil {
		t.Error("node should not be nil")
	}
}
