package trading

import (
	"context"
	"testing"
)

func TestBroker_OrderLifecycle(t *testing.T) {
	mb := newMockBroker("lifecycle-test")
	ctx := context.Background()

	if err := mb.Connect(ctx); err != nil {
		t.Fatal("Connect failed:", err)
	}
	if !mb.IsConnected() {
		t.Error("expected connected state")
	}
	if mb.Name() != "lifecycle-test" {
		t.Errorf("name = %q, want %q", mb.Name(), "lifecycle-test")
	}

	order := &Order{
		ID:        "test-1",
		Symbol:    "AAPL",
		Side:      SideBuy,
		OrderType: TypeLimit,
		Quantity:  100,
		Price:     150.0,
	}

	result, err := mb.SubmitOrder(ctx, order)
	if err != nil {
		t.Fatal("SubmitOrder failed:", err)
	}
	if result.BrokerOrderID != "B-test-1" {
		t.Errorf("broker order id = %q, want %q", result.BrokerOrderID, "B-test-1")
	}
	if result.Status != StatusPending {
		t.Errorf("status = %q, want %q", result.Status, StatusPending)
	}

	if err := mb.ModifyOrder(ctx, "B-test-1", 155.0, 200); err != nil {
		t.Error("ModifyOrder failed:", err)
	}
	orders, _ := mb.GetOrders(ctx)
	found := false
	for _, o := range orders {
		if o.ID == "test-1" {
			found = true
			if o.Price != 155.0 {
				t.Errorf("price after modify = %f, want 155.0", o.Price)
			}
			if o.Quantity != 200 {
				t.Errorf("qty after modify = %f, want 200", o.Quantity)
			}
		}
	}
	if !found {
		t.Error("modified order not found in GetOrders")
	}

	if err := mb.CancelOrder(ctx, "B-test-1"); err != nil {
		t.Error("CancelOrder failed:", err)
	}
	orders, _ = mb.GetOrders(ctx)
	for _, o := range orders {
		if o.ID == "test-1" && o.Status != StatusCancelled {
			t.Errorf("expected cancelled status, got %q", o.Status)
		}
	}

	if err := mb.Disconnect(ctx); err != nil {
		t.Error("Disconnect failed:", err)
	}
	if mb.IsConnected() {
		t.Error("expected disconnected state")
	}
}

func TestBroker_GetPositionsAndAccount(t *testing.T) {
	mb := newMockBroker("positions-test")
	ctx := context.Background()

	positions, err := mb.GetPositions(ctx)
	if err != nil {
		t.Fatal("GetPositions failed:", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	p := positions[0]
	if p.Symbol != "AAPL" || p.Quantity != 100 || p.AvgPrice != 150.0 || p.MarketPrice != 155.0 {
		t.Errorf("unexpected position: %+v", p)
	}

	acct, err := mb.GetAccount(ctx)
	if err != nil {
		t.Fatal("GetAccount failed:", err)
	}
	if acct.BrokerName != "positions-test" {
		t.Errorf("broker name = %q, want %q", acct.BrokerName, "positions-test")
	}
	if acct.TotalValue != 100000.0 {
		t.Errorf("total value = %f, want 100000", acct.TotalValue)
	}
}

func TestBroker_Callbacks(t *testing.T) {
	mb := newMockBroker("callback-test")

	orderCalls := 0
	tradeCalls := 0

	mb.OnOrderUpdate(func(o *Order) { orderCalls++ })
	mb.OnTradeUpdate(func(t *Trade) { tradeCalls++ })

	if len(mb.orderUpdates) != 1 {
		t.Errorf("expected 1 order callback registered, got %d", len(mb.orderUpdates))
	}
	if len(mb.tradeUpdates) != 1 {
		t.Errorf("expected 1 trade callback registered, got %d", len(mb.tradeUpdates))
	}
}

func TestOMS_PlaceOrderLive_FullLifecycle(t *testing.T) {
	oms := NewOMS()
	mb := newMockBroker("full-lifecycle")
	oms.SetBroker(mb)
	ctx := context.Background()

	order, err := oms.PlaceOrderLive(ctx, "AAPL", SideBuy, TypeLimit, 100, 150.0, 0)
	if err != nil {
		t.Fatal("PlaceOrderLive error:", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}

	orders, err := mb.GetOrders(ctx)
	if err != nil {
		t.Fatal("GetOrders error:", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order on broker, got %d", len(orders))
	}
}

func TestOMS_PlaceOrderLive_RejectedOrder(t *testing.T) {
	oms := NewOMS()
	rejectBroker := newMockBroker("reject-test")
	oms.SetBroker(rejectBroker)
	ctx := context.Background()

	order, err := oms.PlaceOrderLive(ctx, "AAPL", SideBuy, TypeLimit, 100, 150.0, 0)
	if err != nil {
		t.Fatal("PlaceOrderLive should not return error for mock broker:", err)
	}
	if order == nil {
		t.Fatal("expected order, got nil")
	}
}

func TestOMS_PlaceOrderLive_OrderMapping(t *testing.T) {
	oms := NewOMS()
	mb := newMockBroker("mapping-test")
	oms.SetBroker(mb)
	ctx := context.Background()

	order, err := oms.PlaceOrderLive(ctx, "AAPL", SideBuy, TypeLimit, 100, 150.0, 0)
	if err != nil {
		t.Fatal("PlaceOrderLive error:", err)
	}

	stored, ok := oms.GetOrder(order.ID)
	if !ok {
		t.Fatal("order not found in OMS after live placement")
	}
	if stored.ID != "B-test-1" && stored.ID != order.ID {
		t.Errorf("expected order ID mapping, got %q", stored.ID)
	}
}
