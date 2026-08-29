package trading

import (
	"context"
	"testing"
)

// mockBroker implements Broker for testing OMS broker integration.
type mockBroker struct {
	name         string
	connected    bool
	orders       map[string]*Order
	orderUpdates []func(*Order)
	tradeUpdates []func(*Trade)
}

func newMockBroker(name string) *mockBroker {
	return &mockBroker{name: name, orders: make(map[string]*Order)}
}

func (m *mockBroker) Connect(ctx context.Context) error    { m.connected = true; return nil }
func (m *mockBroker) Disconnect(ctx context.Context) error { m.connected = false; return nil }
func (m *mockBroker) IsConnected() bool                    { return m.connected }
func (m *mockBroker) Name() string                         { return m.name }

func (m *mockBroker) SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error) {
	orderID := "B-" + order.ID
	m.orders[orderID] = order
	return &BrokerOrderResult{BrokerOrderID: orderID, Status: StatusPending, Message: "submitted"}, nil
}

func (m *mockBroker) CancelOrder(ctx context.Context, orderID string) error {
	if o, ok := m.orders[orderID]; ok {
		o.Status = StatusCancelled
	}
	return nil
}

func (m *mockBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	if o, ok := m.orders[orderID]; ok {
		o.Price = newPrice
		o.Quantity = newQty
	}
	return nil
}

func (m *mockBroker) GetOrders(ctx context.Context) ([]*Order, error) {
	result := make([]*Order, 0, len(m.orders))
	for _, o := range m.orders {
		result = append(result, o)
	}
	return result, nil
}

func (m *mockBroker) GetPositions(ctx context.Context) ([]*Position, error) {
	return []*Position{{Symbol: "AAPL", Quantity: 100, AvgPrice: 150.0, MarketPrice: 155.0, PnL: 500.0, PnLPct: 3.33}}, nil
}

func (m *mockBroker) GetAccount(ctx context.Context) (*AccountInfo, error) {
	return &AccountInfo{BrokerName: m.name, TotalValue: 100000.0, CashBalance: 50000.0, BuyingPower: 100000.0, Currency: "USD"}, nil
}

func (m *mockBroker) OnOrderUpdate(fn func(*Order)) { m.orderUpdates = append(m.orderUpdates, fn) }
func (m *mockBroker) OnTradeUpdate(fn func(*Trade)) { m.tradeUpdates = append(m.tradeUpdates, fn) }

func TestOMS_WithBroker_PlaceOrderLive(t *testing.T) {
	oms := NewOMS()
	mb := newMockBroker("test-broker")
	oms.SetBroker(mb)

	ctx := context.Background()
	order, err := oms.PlaceOrderLive(ctx, "AAPL", SideBuy, TypeLimit, 100, 150.0, 0)
	if err != nil {
		t.Fatalf("PlaceOrderLive error: %v", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}
}

func TestOMS_HasBroker(t *testing.T) {
	oms := NewOMS()
	if oms.HasBroker() {
		t.Error("expected no broker initially")
	}
	oms.SetBroker(newMockBroker("test"))
	if !oms.HasBroker() {
		t.Error("expected broker after SetBroker")
	}
}

func TestOMS_WithoutBroker_UsesPaper(t *testing.T) {
	oms := NewOMS()
	order, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}
}
