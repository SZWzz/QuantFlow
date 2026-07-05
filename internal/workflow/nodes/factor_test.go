package nodes

import (
	"context"
	"testing"
)

func TestFactorNode_Execute(t *testing.T) {
	node, err := NewFactorNode("f1", nil)
	if err != nil {
		t.Fatalf("NewFactorNode() error = %v", err)
	}
	if node.NodeType() != "factor" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "factor")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{"factor_name": "momentum_20d", "symbols": "000001.SZ"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	meta, ok := outputs["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected map metadata, got %T", outputs["metadata"])
	}
	if meta["factor_name"] != "momentum_20d" {
		t.Errorf("factor_name = %v, want 'momentum_20d'", meta["factor_name"])
	}
	if meta["status"] != "computed" {
		t.Errorf("status = %v, want 'computed'", meta["status"])
	}
}

func TestFactorNode_MissingFactorName(t *testing.T) {
	node, err := NewFactorNode("f1", map[string]any{"factor_name": ""})
	if err != nil {
		t.Fatalf("NewFactorNode() error = %v", err)
	}
	err = node.Validate()
	if err == nil {
		t.Error("expected error for empty factor_name")
	}
}
