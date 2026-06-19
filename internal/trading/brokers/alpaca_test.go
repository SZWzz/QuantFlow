package brokers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quantflow/internal/trading"
)

func setupAlpacaTestServer() (*httptest.Server, *AlpacaBroker) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case path == "/v2/clock":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"timestamp": "2026-06-19T12:00:00Z", "is_open": true,
				"next_open": "2026-06-20T09:30:00-04:00", "next_close": "2026-06-19T16:00:00-04:00",
			})
		case path == "/v2/account":
			json.NewEncoder(w).Encode(map[string]string{
				"cash": "25000.50", "portfolio_value": "125000.75",
				"buying_power": "100000.00", "long_market_value": "100000.25", "currency": "USD",
			})
		case path == "/v2/orders":
			switch r.Method {
			case http.MethodGet:
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "ord-001", "client_order_id": "cli-001", "symbol": "AAPL", "side": "buy", "type": "limit",
						"qty": "100", "limit_price": "195.50", "filled_qty": "100", "filled_avg_price": "195.45",
						"status": "filled", "created_at": "2026-06-19T10:00:00Z", "filled_at": "2026-06-19T10:05:00Z"},
					{"id": "ord-002", "symbol": "TSLA", "side": "sell", "type": "market",
						"qty": "50", "filled_qty": "0", "status": "pending_new", "created_at": "2026-06-19T11:00:00Z"},
				})
			case http.MethodPost:
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"id": "ord-new", "status": "accepted"})
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case strings.HasPrefix(path, "/v2/orders/"):
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case path == "/v2/positions":
			json.NewEncoder(w).Encode([]map[string]string{
				{"symbol": "AAPL", "qty": "100", "avg_entry_price": "190.25",
					"current_price": "198.50", "unrealized_pl": "825.00", "unrealized_plpc": "0.0434"},
				{"symbol": "MSFT", "qty": "50", "avg_entry_price": "420.00",
					"current_price": "435.75", "unrealized_pl": "787.50", "unrealized_plpc": "0.0375"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	broker := NewAlpacaBroker(AlpacaConfig{
		APIKey:    "test-key",
		SecretKey: "test-secret",
		BaseURL:   server.URL,
	})
	return server, broker
}

func TestAlpacaBroker_Name(t *testing.T) {
	b := NewAlpacaBroker(AlpacaConfig{})
	if b.Name() != "alpaca" {
		t.Errorf("Name() = %q, want %q", b.Name(), "alpaca")
	}
}

func TestAlpacaBroker_Connect(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	if err := broker.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if !broker.IsConnected() {
		t.Error("expected connected after Connect()")
	}
}

func TestAlpacaBroker_ConnectNoKey(t *testing.T) {
	b := NewAlpacaBroker(AlpacaConfig{})
	if err := b.Connect(context.Background()); err == nil {
		t.Error("expected error when API key not configured")
	}
}

func TestAlpacaBroker_GetAccount(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	acc, err := broker.GetAccount(ctx)
	if err != nil {
		t.Fatalf("GetAccount() error: %v", err)
	}
	if acc.CashBalance != 25000.50 {
		t.Errorf("CashBalance = %v, want 25000.50", acc.CashBalance)
	}
	if acc.TotalValue != 125000.75 {
		t.Errorf("TotalValue = %v, want 125000.75", acc.TotalValue)
	}
}

func TestAlpacaBroker_GetOrders(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	orders, err := broker.GetOrders(ctx)
	if err != nil {
		t.Fatalf("GetOrders() error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	if orders[0].Symbol != "AAPL" {
		t.Errorf("orders[0].Symbol = %q, want AAPL", orders[0].Symbol)
	}
	if orders[0].Status != trading.StatusFilled {
		t.Errorf("orders[0].Status = %q, want filled", orders[0].Status)
	}
}

func TestAlpacaBroker_SubmitOrder(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeLimit, Quantity: 100, Price: 195.50,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-new" {
		t.Errorf("BrokerOrderID = %q, want ord-new", result.BrokerOrderID)
	}
}

func TestAlpacaBroker_CancelOrder(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	if err := broker.CancelOrder(ctx, "ord-001"); err != nil {
		t.Errorf("CancelOrder() error: %v", err)
	}
}

func TestAlpacaBroker_GetPositions(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	positions, err := broker.GetPositions(ctx)
	if err != nil {
		t.Fatalf("GetPositions() error: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}
	if positions[0].Symbol != "AAPL" {
		t.Errorf("positions[0].Symbol = %q, want AAPL", positions[0].Symbol)
	}
}

func TestAlpacaBroker_Disconnect(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)
	broker.Disconnect(ctx)
	if broker.IsConnected() {
		t.Error("expected disconnected after Disconnect()")
	}
}

func TestAlpacaBroker_Callbacks(t *testing.T) {
	server, broker := setupAlpacaTestServer()
	defer server.Close()
	var called int
	broker.OnOrderUpdate(func(o *trading.Order) { called++ })
	broker.OnTradeUpdate(func(tr *trading.Trade) { called++ })
	if called != 0 {
		t.Error("callbacks should not fire on registration")
	}
}

func TestAlpacaStatus_Mapping(t *testing.T) {
	tests := []struct {
		alpaca string
		want   trading.OrderStatus
	}{
		{"new", trading.StatusPending}, {"accepted", trading.StatusPending},
		{"pending_new", trading.StatusPending}, {"partially_filled", trading.StatusPartial},
		{"filled", trading.StatusFilled}, {"canceled", trading.StatusCancelled},
		{"expired", trading.StatusCancelled}, {"rejected", trading.StatusRejected},
		{"stopped", trading.StatusRejected}, {"unknown", trading.StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.alpaca, func(t *testing.T) {
			if got := alpacaStatus(tt.alpaca); got != tt.want {
				t.Errorf("alpacaStatus(%q) = %q, want %q", tt.alpaca, got, tt.want)
			}
		})
	}
}
