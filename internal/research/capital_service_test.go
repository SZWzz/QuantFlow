package research

import (
	"context"
	"testing"
)

func TestCapitalService_GetMarginTrading_NilAdapter(t *testing.T) {
	svc := NewCapitalService(nil)
	data, err := svc.GetMarginTrading(context.Background(), "000001", 10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestCapitalService_GetBlockTrades_NilAdapter(t *testing.T) {
	svc := NewCapitalService(nil)
	data, err := svc.GetBlockTrades(context.Background(), "000001", 10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestCapitalService_GetHolderChanges_NilAdapter(t *testing.T) {
	svc := NewCapitalService(nil)
	data, err := svc.GetHolderChanges(context.Background(), "000001", 10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}

func TestCapitalService_GetDividendHistory_NilAdapter(t *testing.T) {
	svc := NewCapitalService(nil)
	data, err := svc.GetDividendHistory(context.Background(), "000001", 10)
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Error("expected nil when adapter is nil")
	}
}
