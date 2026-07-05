package nodes

import (
	"context"
	"testing"

	"quantflow/internal/trading"
)

func TestPositionQueryNode_NoOMS(t *testing.T) {
	node, err := NewPositionQueryNode("pq1", nil)
	if err != nil {
		t.Fatalf("NewPositionQueryNode() error = %v", err)
	}
	if node.NodeType() != "position_query" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "position_query")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	positions, ok := outputs["positions"].([]*trading.Position)
	if !ok || len(positions) != 0 {
		t.Errorf("expected empty positions slice")
	}
	if outputs["count"] != 0 {
		t.Errorf("count = %v, want 0", outputs["count"])
	}
}
