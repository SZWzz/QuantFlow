package nodes

import (
	"context"
	"testing"
)

func TestIndicatorZhuoyaoNode_NoBridge(t *testing.T) {
	node, err := NewIndicatorZhuoyaoNode("zy1", nil)
	if err != nil {
		t.Fatalf("NewIndicatorZhuoyaoNode() error = %v", err)
	}
	if node.NodeType() != "indicator_zhuoyao" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "indicator_zhuoyao")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"ohlcv": []float64{1, 2, 3}}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing python bridge")
	}
	for _, key := range []string{"zhuoyao_20", "zhuoyao_60", "zhuoyao_120"} {
		val, ok := outputs[key].([]float64)
		if !ok || len(val) != 0 {
			t.Errorf("expected empty []float64 for %q", key)
		}
	}
}
