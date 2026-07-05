package research

import (
	"context"
	"testing"
	"time"
)

func TestPredictionMarketService_GetEvents_MockFallback(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	events, err := svc.GetEvents(context.Background(), "economics", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected non-empty mock prediction events")
	}
	if events[0].ID == "" {
		t.Error("expected event ID in mock data")
	}
}

func TestPredictionMarketService_GetEvents_CacheHit(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	// First call populates cache
	first, err := svc.GetEvents(context.Background(), "crypto", 10)
	if err != nil {
		t.Fatal(err)
	}

	// Second call — should hit cache
	second, err := svc.GetEvents(context.Background(), "crypto", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Errorf("cached data length %d != first %d", len(second), len(first))
	}
}

func TestPredictionMarketService_GetEvents_ExpiredCache(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	// Inject expired cache entry
	svc.mu.Lock()
	svc.cache["economics"] = &cacheEntry{
		events:    nil,
		expiresAt: time.Now().Add(-time.Hour),
	}
	svc.mu.Unlock()

	// Should fetch fresh mock data
	events, err := svc.GetEvents(context.Background(), "economics", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected fresh mock data despite expired cache")
	}
}

func TestPredictionMarketService_GetEvents_EmptyCategoryDefaultsToAll(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	events, err := svc.GetEvents(context.Background(), "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 {
		t.Fatal("expected events for empty category (maps to 'all')")
	}
}

func TestPredictionMarketService_GetEventDetail_MockFallback(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	event, err := svc.GetEventDetail(context.Background(), "fed-rate-cut-july-2026")
	if err != nil {
		t.Fatal(err)
	}
	if event == nil {
		t.Fatal("expected non-nil event detail")
	}
	if event.ID != "fed-rate-cut-july-2026" {
		t.Errorf("expected event ID fed-rate-cut-july-2026, got %s", event.ID)
	}
}

func TestPredictionMarketService_GetEventDetail_UnknownIDReturnsGeneric(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	event, err := svc.GetEventDetail(context.Background(), "unknown-event-id")
	if err != nil {
		t.Fatal(err)
	}
	if event == nil {
		t.Fatal("expected non-nil event for unknown ID")
	}
	if event.ID != "unknown-event-id" {
		t.Errorf("expected ID to match requested, got %s", event.ID)
	}
}

func TestPredictionMarketService_GetPriceHistory_MockFallback(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	prices, err := svc.GetPriceHistory(context.Background(), "fed-rate-cut-july-2026", "1d", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(prices) == 0 {
		t.Fatal("expected non-empty mock price history")
	}
}

func TestPredictionMarketService_ExtractSignals_MockFallback(t *testing.T) {
	svc := NewPredictionMarketService(nil)
	output, err := svc.ExtractSignals(context.Background(), "economics", 0)
	if err != nil {
		t.Fatal(err)
	}
	if output == nil {
		t.Fatal("expected non-nil signal output")
	}
	if output.Category != "economics" {
		t.Errorf("expected category economics, got %s", output.Category)
	}
	if output.Signal.Action == "" {
		t.Error("expected signal action")
	}
}
