package nodes

import (
	"context"
	"quantflow/internal/workflow"
	"testing"
)

func TestDataNormalizeNode_OHLCV(t *testing.T) {
	node, err := NewDataNormalizeNode("test", map[string]any{})
	if err != nil {
		t.Fatalf("NewDataNormalizeNode() error: %v", err)
	}

	params := map[string]any{
		"source":  "eastmoney",
		"target":  "ohlcv",
		"mapping": `{"symbol":"code","date":"date","open":"open","high":"high","low":"low","close":"close","volume":"volume"}`,
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"raw": map[string]any{
			"code": "000001", "date": "2026-01-02",
			"open": 10.0, "high": 11.0, "low": 9.0, "close": 10.5, "volume": 100.0,
		},
	}, params, &workflow.NodeContext{})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	bar, ok := output["ohlcv"]
	if !ok {
		t.Fatal("expected 'ohlcv' in output")
	}
	_ = bar
}

func TestDataNormalizeNode_OrderStatus(t *testing.T) {
	node, err := NewDataNormalizeNode("test", map[string]any{})
	if err != nil {
		t.Fatalf("NewDataNormalizeNode() error: %v", err)
	}

	params := map[string]any{
		"source": "ibkr",
		"target": "order_status",
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"status": "Filled",
	}, params, &workflow.NodeContext{})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	status, ok := output["normalized_status"]
	if !ok {
		t.Fatal("expected 'normalized_status' in output")
	}
	if status != "filled" {
		t.Errorf("normalized_status = %q, want 'filled'", status)
	}
}

func TestDataNormalizeNode_OrderType(t *testing.T) {
	node, err := NewDataNormalizeNode("test", map[string]any{})
	if err != nil {
		t.Fatalf("NewDataNormalizeNode() error: %v", err)
	}

	params := map[string]any{
		"source": "binance",
		"target": "order_type",
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"order_type": "LIMIT",
	}, params, &workflow.NodeContext{})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	orderType, ok := output["normalized_type"]
	if !ok {
		t.Fatal("expected 'normalized_type' in output")
	}
	if orderType != "limit" {
		t.Errorf("normalized_type = %q, want 'limit'", orderType)
	}
}

func TestDataNormalizeNode_MissingInput(t *testing.T) {
	node, err := NewDataNormalizeNode("test", map[string]any{})
	if err != nil {
		t.Fatalf("NewDataNormalizeNode() error: %v", err)
	}

	params := map[string]any{
		"target": "ohlcv",
	}

	_, err = node.Execute(context.Background(), map[string]any{}, params, &workflow.NodeContext{})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func TestDataNormalizeNode_NodeType(t *testing.T) {
	node, _ := NewDataNormalizeNode("test", nil)
	if node.NodeType() != "data_normalize" {
		t.Errorf("NodeType() = %q, want 'data_normalize'", node.NodeType())
	}
	if node.Category() != "data" {
		t.Errorf("Category() = %q, want 'data'", node.Category())
	}
	if node.ID() != "test" {
		t.Errorf("ID() = %q, want 'test'", node.ID())
	}
}

func TestParseJSONMapping(t *testing.T) {
	m, err := parseJSONMapping(`{"symbol":"code","open":"opn"}`)
	if err != nil {
		t.Fatalf("parseJSONMapping() error: %v", err)
	}
	if m["symbol"] != "code" {
		t.Errorf("symbol = %q, want 'code'", m["symbol"])
	}
	if m["open"] != "opn" {
		t.Errorf("open = %q, want 'opn'", m["open"])
	}

	// empty
	m2, err := parseJSONMapping("")
	if err != nil {
		t.Fatalf("parseJSONMapping('') error: %v", err)
	}
	if len(m2) != 0 {
		t.Errorf("expected empty map, got %v", m2)
	}
}
