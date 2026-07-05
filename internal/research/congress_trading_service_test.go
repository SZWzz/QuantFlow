package research

import (
	"context"
	"testing"
)

func TestCongressTradingService_GetCongressTrades_NilAdapter(t *testing.T) {
	svc := NewCongressTradingService(nil)
	trades, err := svc.GetCongressTrades(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if trades != nil {
		t.Error("expected nil when adapter is nil")
	}
}
