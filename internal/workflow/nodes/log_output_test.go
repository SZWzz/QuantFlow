package nodes

import (
	"context"
	"testing"
)

func TestLogOutputNode_PassThrough(t *testing.T) {
	node, _ := NewLogOutputNode("log1", nil)
	inputs := map[string]any{"input": "hello"}
	outputs, err := node.Execute(context.Background(), inputs, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["input"] != "hello" {
		t.Errorf("pass-through failed: %v", outputs["input"])
	}
}
