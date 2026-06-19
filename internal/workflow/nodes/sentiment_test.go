package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestSentimentNode_Interface(t *testing.T) {
	node, err := NewSentimentNode("test-1", map[string]any{"text_type": "news"})
	if err != nil {
		t.Fatalf("NewSentimentNode: %v", err)
	}
	if node.ID() != "test-1" {
		t.Errorf("expected id 'test-1', got %s", node.ID())
	}
	if node.NodeType() != "sentiment" {
		t.Errorf("expected node_type 'sentiment', got %s", node.NodeType())
	}
	if node.Category() != "research" {
		t.Errorf("expected category 'research', got %s", node.Category())
	}
}

func TestSentimentNode_Ports(t *testing.T) {
	node, _ := NewSentimentNode("test-1", nil)

	inputs := node.InputPorts()
	if len(inputs) != 2 {
		t.Errorf("expected 2 input ports, got %d", len(inputs))
	}
	if inputs[0].Name != "symbol" || !inputs[0].Required {
		t.Error("first input must be 'symbol' and required")
	}

	outputs := node.OutputPorts()
	if len(outputs) != 4 {
		t.Errorf("expected 4 output ports, got %d", len(outputs))
	}
}

func TestSentimentNode_Execute_Mock(t *testing.T) {
	oldEngine := sentimentEngine
	sentimentEngine = nil
	defer func() { sentimentEngine = oldEngine }()

	node, _ := NewSentimentNode("test-1", map[string]any{})
	_, _ = node.(workflow.BaseNode)

	result, err := node.Execute(context.Background(),
		map[string]any{"symbol": "AAPL"},
		map[string]any{"text_type": "news", "language": "en"},
	)
	if err != nil {
		t.Fatalf("Execute should not error in mock mode: %v", err)
	}

	if result["sentiment_label"] != "neutral" {
		t.Errorf("expected neutral label in mock mode, got %v", result["sentiment_label"])
	}

	signal, ok := result["signal"].(map[string]any)
	if !ok {
		t.Fatal("signal output must be a map")
	}
	if signal["action"] != "hold" {
		t.Errorf("expected hold signal in mock mode, got %v", signal["action"])
	}
}

func TestSentimentNode_Execute_MissingSymbol(t *testing.T) {
	node, _ := NewSentimentNode("test-1", nil)
	_, err := node.Execute(context.Background(), map[string]any{}, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}

func TestSentimentToSignal_Thresholds(t *testing.T) {
	// Low confidence: always hold
	s := sentimentToSignal(0.8, 0.2)
	if s["action"] != "hold" {
		t.Errorf("low confidence must hold, got %v", s["action"])
	}

	// High confidence + strong positive → buy
	s = sentimentToSignal(0.5, 0.8)
	if s["action"] != "buy" {
		t.Errorf("strong positive must buy, got %v", s["action"])
	}

	// High confidence + strong negative → sell
	s = sentimentToSignal(-0.5, 0.8)
	if s["action"] != "sell" {
		t.Errorf("strong negative must sell, got %v", s["action"])
	}

	// High confidence + near zero → hold
	s = sentimentToSignal(0.05, 0.8)
	if s["action"] != "hold" {
		t.Errorf("near zero must hold, got %v", s["action"])
	}

	// Boundary: exactly at threshold
	s = sentimentToSignal(0.15, 0.4)
	if s["action"] != "hold" {
		t.Errorf("boundary value (> threshold) should be hold? got %v", s["action"])
	}
	s = sentimentToSignal(0.1501, 0.4)
	if s["action"] != "buy" {
		t.Errorf("just above threshold should buy, got %v", s["action"])
	}
}
