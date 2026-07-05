package research

import (
	"context"
	"testing"
)

func TestPeerComparisonService_GetPeers_NilAdapter(t *testing.T) {
	svc := NewPeerComparisonService(nil, nil, nil, nil)
	peers, err := svc.GetPeers(context.Background(), "000001")
	if err != nil {
		t.Fatal(err)
	}
	if peers != nil {
		t.Error("expected nil when concept adapter is nil")
	}
}

func TestPeerComparisonService_GetIndustryRanks_NilAdapter(t *testing.T) {
	svc := NewPeerComparisonService(nil, nil, nil, nil)
	ranks, err := svc.GetIndustryRanks(context.Background(), 10)
	if err == nil {
		t.Error("expected error when signals adapter is nil")
	}
	if ranks != nil {
		t.Error("expected nil ranks on error")
	}
}
