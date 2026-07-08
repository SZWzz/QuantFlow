package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quantflow/internal/trading"
)

const mockAccountID = "U1234567"

func setupIBKRTestServer() (*httptest.Server, *IBKRBroker) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		method := r.Method

		switch {
		case path == "/sso/validate":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"authenticated": true,
				"token":         "sess-tok",
				"expires":       1800,
			})

		case path == "/logout":
			w.WriteHeader(http.StatusOK)

		case path == fmt.Sprintf("/iserver/account/%s/orders", mockAccountID):
			switch method {
			case http.MethodGet:
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"orderId": 1001, "symbol": "AAPL", "side": "BUY", "orderType": "MKT",
						"quantity": 100.0, "filledQuantity": 100.0, "avgPrice": 198.50,
						"status": "Filled", "placedTime": "2026-07-08T10:00:00Z"},
					{"orderId": 1002, "symbol": "TSLA", "side": "SELL", "orderType": "LMT",
						"quantity": 50.0, "limitPrice": 250.00, "filledQuantity": 0,
						"status": "Submitted", "placedTime": "2026-07-08T11:00:00Z"},
				})
			case http.MethodPost:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id": "ord-ibkr-001", "order_status": "Submitted",
				})
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}

		case strings.HasPrefix(path, fmt.Sprintf("/iserver/account/%s/order/", mockAccountID)):
			if method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
			} else if method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "Modified"})
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}

		case path == fmt.Sprintf("/portfolio/%s/positions/0", mockAccountID):
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"symbol": "AAPL", "position": 100.0, "avgCost": 190.25,
					"marketPrice": 198.50, "unrealizedPnl": 825.00, "unrealizedPnlPerc": 4.34},
				{"symbol": "MSFT", "position": 50.0, "avgCost": 420.00,
					"marketPrice": 435.75, "unrealizedPnl": 787.50, "unrealizedPnlPerc": 3.75},
			})

		case path == fmt.Sprintf("/portfolio/%s/summary", mockAccountID):
			json.NewEncoder(w).Encode(map[string]ibkrAccountSummary{
				"TotalCashValue": {Value: 25000.50, Currency: "USD"},
				"CashBalance":    {Value: 25000.50, Currency: "USD"},
				"BuyingPower":    {Value: 100000.00, Currency: "USD"},
				"Currency":       {ValueString: "USD", Currency: "USD"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))

	broker := NewIBKRBroker(IBKRConfig{
		Host:      "localhost",
		Port:      0,
		AccountID: mockAccountID,
	})
	broker.baseURL = server.URL
	broker.client = server.Client()
	return server, broker
}

func TestIBKRBroker_Name(t *testing.T) {
	b := NewIBKRBroker(IBKRConfig{})
	if b.Name() != "ibkr" {
		t.Errorf("Name() = %q, want %q", b.Name(), "ibkr")
	}
}

func TestIBKRBroker_Connect(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	if err := broker.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if !broker.IsConnected() {
		t.Error("expected connected after Connect()")
	}
}

func TestIBKRBroker_Connect_NoAccountID(t *testing.T) {
	b := NewIBKRBroker(IBKRConfig{})
	if err := b.Connect(context.Background()); err == nil {
		t.Error("expected error when AccountID not configured")
	}
}

func TestIBKRBroker_Disconnect(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	broker.Connect(context.Background())
	broker.Disconnect(context.Background())
	if broker.IsConnected() {
		t.Error("expected disconnected after Disconnect()")
	}
}

func TestIBKRBroker_SubmitOrder_Market(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeMarket, Quantity: 100,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
	if result.Status != trading.StatusPending {
		t.Errorf("Status = %q, want Pending", result.Status)
	}
}

func TestIBKRBroker_SubmitOrder_Limit(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeLimit,
		Quantity: 100, Price: 195.50,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
}

func TestIBKRBroker_SubmitOrder_Stop(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeStop,
		Quantity: 100, StopPrice: 190.00,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
}

func TestIBKRBroker_CancelOrder(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	if err := broker.CancelOrder(ctx, "1001"); err != nil {
		t.Errorf("CancelOrder() error: %v", err)
	}
}

func TestIBKRBroker_ModifyOrder(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	if err := broker.ModifyOrder(ctx, "1001", 200.00, 150); err != nil {
		t.Errorf("ModifyOrder() error: %v", err)
	}
}

func TestIBKRBroker_GetOrders(t *testing.T) {
	server, broker := setupIBKRTestServer()
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
		t.Errorf("orders[0].Status = %q, want Filled", orders[0].Status)
	}
}

func TestIBKRBroker_GetPositions(t *testing.T) {
	server, broker := setupIBKRTestServer()
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

func TestIBKRBroker_GetAccount(t *testing.T) {
	server, broker := setupIBKRTestServer()
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
	if acc.BuyingPower != 100000.00 {
		t.Errorf("BuyingPower = %v, want 100000.00", acc.BuyingPower)
	}
}

func TestIBKRBroker_NotConnected_ReturnsError(t *testing.T) {
	broker := NewIBKRBroker(IBKRConfig{AccountID: "test"})

	if _, err := broker.SubmitOrder(context.Background(), &trading.Order{}); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetOrders(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetPositions(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetAccount(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
}

func TestIBKRBroker_Callbacks(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()

	var called int
	broker.OnOrderUpdate(func(o *trading.Order) { called++ })
	broker.OnTradeUpdate(func(tr *trading.Trade) { called++ })
	if called != 0 {
		t.Error("callbacks should not fire on registration")
	}
}

func TestIBKROrderStatus_Mapping(t *testing.T) {
	tests := []struct {
		ibkr string
		want trading.OrderStatus
	}{
		{"Submitted", trading.StatusPending},
		{"PreSubmitted", trading.StatusPending},
		{"Filled", trading.StatusFilled},
		{"Cancelled", trading.StatusCancelled},
		{"ApiCancelled", trading.StatusCancelled},
		{"Inactive", trading.StatusPending},
		{"Unknown", trading.StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.ibkr, func(t *testing.T) {
			if got := ibkrOrderStatus(tt.ibkr); got != tt.want {
				t.Errorf("ibkrOrderStatus(%q) = %q, want %q", tt.ibkr, got, tt.want)
			}
		})
	}
}

func TestIBKROrderType_Mapping(t *testing.T) {
	tests := []struct {
		input trading.OrderType
		want  string
	}{
		{trading.TypeMarket, "MKT"},
		{trading.TypeLimit, "LMT"},
		{trading.TypeStop, "STP"},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if got := ibkrOrderType(tt.input); got != tt.want {
				t.Errorf("ibkrOrderType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
