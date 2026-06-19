package research

import (
	"context"
	"testing"
)

func TestSentimentEngine_MockFallback(t *testing.T) {
	engine := NewSentimentEngine(nil, nil, nil) // No bridge, no repo

	output, err := engine.AnalyzeSentiment(context.Background(), "AAPL", "", "news", "en")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if output.Label != "neutral" {
		t.Errorf("expected neutral, got %s", output.Label)
	}
	if output.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", output.Score)
	}
	if len(output.Keywords) == 0 {
		t.Error("expected keywords in mock output")
	}
}

func TestSentimentEngine_IsBridgeAvailable(t *testing.T) {
	engine := NewSentimentEngine(nil, nil, nil)
	if engine.IsBridgeAvailable() {
		t.Error("expected bridge unavailable when nil")
	}
}

func TestMockSentiment_ReturnsNeutral(t *testing.T) {
	engine := NewSentimentEngine(nil, nil, nil)
	output, err := engine.AnalyzeSentiment(context.Background(), "TEST", "", "social", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Label != "neutral" {
		t.Errorf("expected neutral label, got %s", output.Label)
	}
	if output.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", output.Score)
	}
}

func TestSentimentEngine_BatchAnalyze(t *testing.T) {
	engine := NewSentimentEngine(nil, nil, nil)
	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	results, err := engine.BatchAnalyze(context.Background(), symbols, "news", "en")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != len(symbols) {
		t.Errorf("expected %d results, got %d", len(symbols), len(results))
	}
	for i, r := range results {
		if r.Symbol != symbols[i] {
			t.Errorf("result %d: expected symbol %s, got %s", i, symbols[i], r.Symbol)
		}
		if r.Label != "neutral" {
			t.Errorf("result %d: expected neutral, got %s", i, r.Label)
		}
	}
}
