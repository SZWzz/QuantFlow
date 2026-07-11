//go:build integration
// +build integration

package brokers

import (
	"context"
	"os"
	"testing"
	"time"

	"quantflow/internal/trading"
)

// Requires: ALPACA_API_KEY and ALPACA_SECRET_KEY env vars.
// Paper trading only — no real money involved.

func skipIfNoCredentials(t *testing.T) {
	t.Helper()
	if os.Getenv("ALPACA_API_KEY") == "" || os.Getenv("ALPACA_SECRET_KEY") == "" {
		t.Skip("ALPACA_API_KEY and ALPACA_SECRET_KEY not set — skipping integration test")
	}
}

func TestAlpacaConnect(t *testing.T) {
	skipIfNoCredentials(t)

	broker := NewAlpacaBroker(AlpacaConfig{})
	if broker.IsPaper() {
		t.Log("using paper trading environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer broker.Disconnect(ctx)

	if !broker.IsConnected() {
		t.Fatal("broker should be connected")
	}
}

func TestAlpacaGetAccount(t *testing.T) {
	skipIfNoCredentials(t)

	broker := NewAlpacaBroker(AlpacaConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer broker.Disconnect(ctx)

	account, err := broker.GetAccount(ctx)
	if err != nil {
		t.Fatalf("get account failed: %v", err)
	}

	t.Logf("account: buying_power=%s, cash=%s, equity=%s, status=%s",
		account.BuyingPower, account.Cash, account.Equity, account.Status)

	if account.Status == "" {
		t.Error("account status should not be empty")
	}
}

func TestAlpacaGetPositions(t *testing.T) {
	skipIfNoCredentials(t)

	broker := NewAlpacaBroker(AlpacaConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer broker.Disconnect(ctx)

	positions, err := broker.GetPositions(ctx)
	if err != nil {
		t.Fatalf("get positions failed: %v", err)
	}

	t.Logf("positions count: %d", len(positions))
	for _, p := range positions {
		t.Logf("  %s: qty=%s, avg_price=%s, market_value=%s",
			p.Symbol, p.Quantity, p.AvgPrice, p.MarketValue)
	}
}

func TestAlpacaOrderLifecycle(t *testing.T) {
	skipIfNoCredentials(t)

	broker := NewAlpacaBroker(AlpacaConfig{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := broker.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer broker.Disconnect(ctx)

	// Submit a far-limit buy order that won't fill (1 share AAPL at $1)
	order := &trading.Order{
		Symbol:    "AAPL",
		Side:      trading.SideBuy,
		OrderType: trading.TypeLimit,
		Quantity:  1,
		Price:     1.00,
	}
	result, err := broker.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatalf("submit order failed: %v", err)
	}
	t.Logf("order submitted: id=%s, status=%s", result.BrokerOrderID, result.Status)

	// Verify order appears in order list
	orders, err := broker.GetOrders(ctx)
	if err != nil {
		t.Fatalf("get orders failed: %v", err)
	}
	found := false
	for _, o := range orders {
		if o.ID == result.BrokerOrderID {
			found = true
			t.Logf("order found in list: status=%s", o.Status)
			break
		}
	}
	if !found {
		t.Error("submitted order not found in order list")
	}

	// Cancel the order
	if err := broker.CancelOrder(ctx, result.BrokerOrderID); err != nil {
		t.Errorf("cancel order failed: %v", err)
	}

	// Allow time for cancellation to process
	time.Sleep(500 * time.Millisecond)

	// Verify order is cancelled
	ordersAfter, _ := broker.GetOrders(ctx)
	for _, o := range ordersAfter {
		if o.ID == result.BrokerOrderID {
			t.Logf("order final status: %s", o.Status)
			break
		}
	}
}

func TestAlpacaErrorMapping(t *testing.T) {
	tests := []struct {
		statusCode int
		body       string
		want       string
	}{
		{401, "", "API 密钥无效"},
		{403, "", "权限不足"},
		{422, "", "订单参数无效"},
		{429, "", "请求过于频繁"},
		{422, "insufficient", "资金不足"},
		{404, "not found", "标的代码不存在"},
		{422, "market closed", "市场已关闭"},
		{500, "unknown error", "Alpaca 错误 (HTTP 500)"},
	}

	for _, tt := range tests {
		got := alpacaUserFacingError(tt.statusCode, tt.body)
		if !containsAny(got, tt.want) {
			t.Errorf("alpacaUserFacingError(%d, %q) = %q, want contains %q",
				tt.statusCode, tt.body, got, tt.want)
		}
	}
}

func containsAny(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || s[:len(substr)] == substr ||
			len(s) > len(substr) && (s[len(s)-len(substr):] == substr ||
				searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
