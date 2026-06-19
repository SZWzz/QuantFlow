// internal/market/adapters/gdelt_test.go
package adapters

import (
	"context"
	"testing"
	"time"
)

func TestGDELTAdapter_Name(t *testing.T) {
	a := NewGDELTAdapter()
	if a.Name() != "gdelt" {
		t.Errorf("Name() = %s, want gdelt", a.Name())
	}
}

func TestGDELTAdapter_IsAvailable(t *testing.T) {
	a := NewGDELTAdapter()
	ctx := context.Background()
	available := a.IsAvailable(ctx)
	t.Logf("IsAvailable=%v", available)
	// Test should not panic regardless of network state
}

func TestGDELTAdapter_FetchTopicVolume(t *testing.T) {
	a := NewGDELTAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("GDELT API not reachable")
	}

	points, err := a.FetchTopicVolume(ctx, "taiwan-strait", "7d")
	if err != nil {
		t.Fatalf("FetchTopicVolume error: %v", err)
	}
	t.Logf("got %d volume points", len(points))
	if len(points) > 0 {
		t.Logf("first point: date=%s value=%.2f", points[0].Date, points[0].Value)
	}
}

func TestGDELTAdapter_FetchTopicTone(t *testing.T) {
	a := NewGDELTAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("GDELT API not reachable")
	}

	points, err := a.FetchTopicTone(ctx, "taiwan-strait", "7d")
	if err != nil {
		t.Fatalf("FetchTopicTone error: %v", err)
	}
	t.Logf("got %d tone points", len(points))
	if len(points) > 0 {
		t.Logf("first point: date=%s tone=%.2f", points[0].Date, points[0].Tone)
	}
}

func TestGDELTAdapter_TopicQueries(t *testing.T) {
	a := NewGDELTAdapter()
	if len(a.TopicQueries) != 10 {
		t.Errorf("expected 10 topic queries, got %d", len(a.TopicQueries))
	}

	expectedIDs := []string{
		"middle-east", "taiwan-strait", "ukraine-war", "trade-tariffs",
		"north-korea", "fed-policy", "europe-energy", "terrorism",
		"china-economy", "semiconductors",
	}
	for _, id := range expectedIDs {
		tq, ok := a.TopicQueries[id]
		if !ok {
			t.Errorf("missing topic: %s", id)
			continue
		}
		if tq.ID != id {
			t.Errorf("topic %s: ID mismatch (%s)", id, tq.ID)
		}
		if tq.Query == "" {
			t.Errorf("topic %s: query is empty", id)
		}
		if tq.Title == "" {
			t.Errorf("topic %s: title is empty", id)
		}
	}
	t.Logf("all 10 topics configured correctly")
}

func TestGDELTAdapter_UnknownTopic(t *testing.T) {
	a := NewGDELTAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("GDELT API not reachable")
	}

	// Fetch with a raw query string as topicID (not in pre-defined map)
	points, err := a.FetchTopicVolume(ctx, "nonexistent-topic", "7d")
	if err != nil {
		t.Fatalf("FetchTopicVolume with unknown topic should fall back to raw query: %v", err)
	}
	t.Logf("got %d points for unknown topic (used as raw query)", len(points))
}
