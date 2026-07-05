package research

import (
	"context"
	"testing"
)

func TestAnalystEstimatesService_GetEstimates_MockFallback(t *testing.T) {
	svc := NewAnalystEstimatesService(nil, nil)
	estimates, err := svc.GetEstimates(context.Background(), "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if len(estimates) == 0 {
		t.Fatal("expected non-empty mock estimates")
	}
	if estimates[0].Analyst == "" {
		t.Error("expected analyst name in mock data")
	}
	if estimates[0].Firm == "" {
		t.Error("expected firm name in mock data")
	}
}

func TestAnalystEstimatesService_GetEstimates_ReturnsAllFive(t *testing.T) {
	svc := NewAnalystEstimatesService(nil, nil)
	estimates, err := svc.GetEstimates(context.Background(), "MSFT")
	if err != nil {
		t.Fatal(err)
	}
	if len(estimates) != 5 {
		t.Errorf("expected 5 mock estimates, got %d", len(estimates))
	}
}

func TestAnalystEstimatesService_GetConsensusEPS_NilAdapters(t *testing.T) {
	svc := NewAnalystEstimatesService(nil, nil)
	consensus, err := svc.GetConsensusEPS(context.Background(), "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if consensus != nil {
		t.Error("expected nil consensus when both adapters are nil")
	}
}
