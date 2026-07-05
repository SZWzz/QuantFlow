package nodes

import (
	"context"
	"testing"
)

func TestPeerCompareNode_MockData(t *testing.T) {
	node, err := NewPeerCompareNode("pc1", nil)
	if err != nil {
		t.Fatalf("NewPeerCompareNode() error = %v", err)
	}
	if node.NodeType() != "peer_compare" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "peer_compare")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"symbol": "AAPL"}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	peers, ok := outputs["peers"].([]any)
	if !ok || len(peers) != 0 {
		t.Errorf("expected empty peers, got len=%d", len(peers))
	}
	metrics, ok := outputs["comparison_metrics"].(map[string]any)
	if !ok {
		t.Errorf("expected comparison_metrics map, got %T", outputs["comparison_metrics"])
	} else if len(metrics) > 0 {
		t.Errorf("expected empty metrics for nil service, got %v", metrics)
	}
}

func TestPeerCompareNode_MissingSymbol(t *testing.T) {
	node, _err := NewPeerCompareNode("pc1", nil)
	if _err != nil {
		t.Fatalf("NewPeerCompareNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}
