package research

import (
	"context"
	"testing"
)

func TestInsiderTradingService_GetInsiderTrades_NilBridge(t *testing.T) {
	svc := NewInsiderTradingService(nil)
	trades, err := svc.GetInsiderTrades(context.Background(), "AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if trades != nil {
		t.Error("expected nil when bridge is nil")
	}
}
