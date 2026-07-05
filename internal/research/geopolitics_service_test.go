package research

import (
	"context"
	"testing"
	"time"
)

func TestGeopoliticsService_GetTopicRisks_MockFallback(t *testing.T) {
	svc := NewGeopoliticsService(nil)
	risks, err := svc.GetTopicRisks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(risks) == 0 {
		t.Fatal("expected non-empty mock risks")
	}
	if risks[0].ID == "" {
		t.Error("expected topic ID in mock data")
	}
	if risks[0].RiskLevel == "" {
		t.Error("expected risk level in mock data")
	}
}

func TestGeopoliticsService_GetTopicRisks_CacheHit(t *testing.T) {
	svc := NewGeopoliticsService(nil)
	// First call populates cache
	first, err := svc.GetTopicRisks(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Second call — should hit cache
	second, err := svc.GetTopicRisks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Errorf("cached data length %d != first %d", len(second), len(first))
	}
}

func TestGeopoliticsService_GetTopicRisks_ExpiredCache(t *testing.T) {
	svc := NewGeopoliticsService(nil)
	// Inject expired cache entry
	svc.mu.Lock()
	svc.cache["all_risks"] = &geoCachedResult{
		risks:     []TopicRisk{{ID: "stale", Title: "Stale Data", RiskLevel: "low", Tone: 0, ToneChange: 0, VolChange: 0}},
		expiresAt: time.Now().Add(-time.Hour),
	}
	svc.mu.Unlock()

	// Should fetch fresh mock data
	risks, err := svc.GetTopicRisks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(risks) == 0 {
		t.Fatal("expected fresh mock data despite expired cache")
	}
	if len(risks) <= 1 {
		t.Error("expected more than 1 fresh topic risk (not just stale entry)")
	}
}

func TestGeopoliticsService_GetTopicDetail_MockFallback(t *testing.T) {
	svc := NewGeopoliticsService(nil)
	detail, err := svc.GetTopicDetail(context.Background(), "middle-east", "7d")
	if err != nil {
		t.Fatal(err)
	}
	if detail == nil {
		t.Fatal("expected non-nil topic detail")
	}
	if _, ok := detail["volumes"]; !ok {
		t.Error("expected volumes in topic detail")
	}
	if _, ok := detail["tones"]; !ok {
		t.Error("expected tones in topic detail")
	}
}

func TestGeopoliticsService_ExtractRiskSignals_MockFallback(t *testing.T) {
	svc := NewGeopoliticsService(nil)
	signals, err := svc.ExtractRiskSignals(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) == 0 {
		t.Fatal("expected at least one risk signal")
	}
	if signals[0].ID == "" {
		t.Error("expected topic ID in signal")
	}
}
