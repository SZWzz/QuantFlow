// internal/market/adapters/polymarket_test.go
package adapters

import (
	"context"
	"testing"
	"time"
)

func TestPolymarketAdapter_Name(t *testing.T) {
	a := NewPolymarketAdapter()
	if a.Name() != "polymarket" {
		t.Errorf("Name() = %s, want polymarket", a.Name())
	}
}

func TestPolymarketAdapter_IsAvailable(t *testing.T) {
	a := NewPolymarketAdapter()
	ctx := context.Background()
	// Should be available (or gracefully return false on network error)
	available := a.IsAvailable(ctx)
	t.Logf("IsAvailable=%v", available)
	// Test should not panic
}

func TestPolymarketAdapter_FetchEvents_HappyPath(t *testing.T) {
	a := NewPolymarketAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("polymarket API not reachable")
	}

	events, err := a.FetchEvents(ctx, "economics", 5)
	if err != nil {
		t.Fatalf("FetchEvents error: %v", err)
	}
	if len(events) == 0 {
		t.Error("FetchEvents returned empty slice")
	}
	t.Logf("got %d events, first: %s (volume=$%.0f)", len(events), events[0].Title, events[0].Volume)
}

func TestPolymarketAdapter_FetchEvent(t *testing.T) {
	a := NewPolymarketAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("polymarket API not reachable")
	}

	// First get an event ID from the list
	events, err := a.FetchEvents(ctx, "", 1)
	if err != nil || len(events) == 0 {
		t.Skip("no events available for detail test")
	}

	event, err := a.FetchEvent(ctx, events[0].ID)
	if err != nil {
		t.Fatalf("FetchEvent error: %v", err)
	}
	if event.Title == "" {
		t.Error("event title should not be empty")
	}
	if len(event.Outcomes) == 0 {
		t.Error("event should have outcomes")
	}
	t.Logf("event: %s, outcomes: %d", event.Title, len(event.Outcomes))
}

func TestPolymarketAdapter_FetchPriceHistory(t *testing.T) {
	a := NewPolymarketAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("polymarket API not reachable")
	}

	// Get an event, take first outcome's price history
	events, err := a.FetchEvents(ctx, "", 1)
	if err != nil || len(events) == 0 || len(events[0].Outcomes) == 0 {
		t.Skip("no events with outcomes available")
	}

	prices, err := a.FetchPriceHistory(ctx, events[0].Outcomes[0].ID, "1d", 30)
	if err != nil {
		t.Fatalf("FetchPriceHistory error: %v", err)
	}
	t.Logf("got %d price points", len(prices))
}

func TestPolymarketAdapter_RequiresAuth(t *testing.T) {
	a := NewPolymarketAdapter()
	if a.RequiresAuth() {
		t.Error("Polymarket should not require auth")
	}
}
