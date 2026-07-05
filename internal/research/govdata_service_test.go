package research

import (
	"context"
	"testing"
	"time"
)

func TestGovDataService_GetIndicatorList(t *testing.T) {
	svc := NewGovDataService(nil)
	list := svc.GetIndicatorList()
	if len(list) == 0 {
		t.Fatal("expected non-empty indicator list")
	}
	if list[0].ID == "" {
		t.Error("expected indicator ID")
	}
}

func TestGovDataService_GetIndicator_MockFallback(t *testing.T) {
	svc := NewGovDataService(nil)
	points, err := svc.GetIndicator(context.Background(), "GDP", 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) == 0 {
		t.Fatal("expected non-empty mock indicator points")
	}
	if points[0].Date == "" {
		t.Error("expected date in indicator point")
	}
}

func TestGovDataService_GetIndicator_CacheHit(t *testing.T) {
	svc := NewGovDataService(nil)
	// First call populates cache
	first, err := svc.GetIndicator(context.Background(), "GDP", 12)
	if err != nil {
		t.Fatal(err)
	}

	// Second call — should hit cache
	second, err := svc.GetIndicator(context.Background(), "GDP", 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Errorf("cached data length %d != first %d", len(second), len(first))
	}
}

func TestGovDataService_GetIndicator_ExpiredCache(t *testing.T) {
	svc := NewGovDataService(nil)
	// Inject expired cache entry
	svc.mu.Lock()
	svc.cache["GDP"] = &govCacheEntry{
		points:    nil,
		expiresAt: time.Now().Add(-time.Hour),
	}
	svc.mu.Unlock()

	// Should fetch fresh mock data
	points, err := svc.GetIndicator(context.Background(), "GDP", 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) == 0 {
		t.Fatal("expected fresh mock data despite expired cache")
	}
}

func TestGovDataService_GetAllSignals_MockFallback(t *testing.T) {
	svc := NewGovDataService(nil)
	signals, err := svc.GetAllSignals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) == 0 {
		t.Fatal("expected non-empty macro signals")
	}
	if signals[0].IndicatorID == "" {
		t.Error("expected indicator ID in signal")
	}
	if signals[0].Signal == "" {
		t.Error("expected signal in macro signal")
	}
}
