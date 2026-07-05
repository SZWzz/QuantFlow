package nodes

import (
	"context"
	"testing"
)

func TestJSONParseNode_ValidJSON(t *testing.T) {
	node, err := NewJSONParseNode("jp1", nil)
	if err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", err)
	}
	if node.NodeType() != "json_parse" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "json_parse")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"json_str": `{"name": "test", "value": 42}`}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	parsed := outputs["parsed"]
	m, ok := parsed.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", parsed)
	}
	if m["name"] != "test" {
		t.Errorf("name = %v, want 'test'", m["name"])
	}
}

func TestJSONParseNode_WithPath(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"json_str": `{"name": "test", "value": 42}`}, map[string]any{"path": "name"}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	val := outputs["value"]
	if val != "test" {
		t.Errorf("value = %v, want 'test'", val)
	}
}

func TestJSONParseNode_InvalidJSON(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"json_str": `{invalid}`}, nil, nil)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONParseNode_MissingInput(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing input")
	}
}

func TestJSONParseNode_EmptyString(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"json_str": ""}, nil, nil)
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestJSONParseNode_MissingKey(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"json_str": `{"a": 1}`}, map[string]any{"path": "b"}, nil)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestJSONParseNode_NonObject(t *testing.T) {
	node, _err := NewJSONParseNode("jp1", nil)
	if _err != nil {
		t.Fatalf("NewJSONParseNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{"json_str": `[1,2,3]`}, map[string]any{"path": "a"}, nil)
	if err == nil {
		t.Error("expected error for non-object with path")
	}
}
