package trading

import (
	"context"
	"testing"
	"time"
)

func TestEngine_FullPipeline(t *testing.T) {
	engine := NewEngine(100000.0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go engine.Start(ctx)
	time.Sleep(10 * time.Millisecond) // Let the engine start

	// Submit a signal
	engine.SubmitSignal(Signal{
		Symbol:    "AAPL",
		Direction: "buy",
		Quantity:  100,
		Timestamp: time.Now(),
	})

	// Submit a few bars
	bars := []OHLCVBar{
		{Date: "2024-01-01", Symbol: "AAPL", Open: 195.0, High: 196.0, Low: 194.5, Close: 195.5},
		{Date: "2024-01-02", Symbol: "AAPL", Open: 195.5, High: 198.0, Low: 195.0, Close: 197.5},
		{Date: "2024-01-03", Symbol: "AAPL", Open: 197.5, High: 200.0, Low: 197.0, Close: 199.0},
	}

	for _, bar := range bars {
		engine.SubmitBar(bar)
	}
	time.Sleep(10 * time.Millisecond) // Let the engine process

	// Check that the order was filled
	oms := engine.GetPaperEngine().GetOMS()
	orders := oms.GetOrders()
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}

	order := orders[0]
	if order.Status != StatusFilled {
		t.Errorf("expected order filled, got status %q", order.Status)
	}
	if order.FilledQty != 100 {
		t.Errorf("expected filled 100, got %f", order.FilledQty)
	}

	// Check position
	pos := oms.GetPosition("AAPL")
	if pos == nil {
		t.Fatal("expected position for AAPL")
	}
	if pos.Quantity != 100 {
		t.Errorf("expected position qty 100, got %f", pos.Quantity)
	}

	// Check trades
	trades := oms.GetTrades()
	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
}

func TestOMS_PlaceAndFill(t *testing.T) {
	oms := NewOMS()

	// Place a limit order
	order, err := oms.PlaceOrder("AAPL", SideBuy, TypeLimit, "", 100, 195.0)
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}

	// Fill it
	trade, err := oms.FillOrder(order.ID, 100, 194.5)
	if err != nil {
		t.Fatalf("FillOrder error: %v", err)
	}
	if trade.Price != 194.5 {
		t.Errorf("trade price = %f, want 194.5", trade.Price)
	}

	// Verify order filled
	filledOrder, ok := oms.GetOrder(order.ID)
	if !ok {
		t.Fatal("order not found after fill")
	}
	if filledOrder.Status != StatusFilled {
		t.Errorf("expected Filled, got %q", filledOrder.Status)
	}

	// Verify position
	pos := oms.GetPosition("AAPL")
	if pos == nil || pos.Quantity != 100 {
		t.Errorf("position qty = %f, want 100", pos.Quantity)
	}
	if pos.AvgPrice != 194.5 {
		t.Errorf("avg price = %f, want 194.5", pos.AvgPrice)
	}
}

func TestOMS_PartialFill(t *testing.T) {
	oms := NewOMS()

	order, _ := oms.PlaceOrder("AAPL", SideBuy, TypeLimit, "", 100, 195.0)

	// Partial fill
	_, err := oms.FillOrder(order.ID, 40, 194.0)
	if err != nil {
		t.Fatalf("FillOrder error: %v", err)
	}

	order, _ = oms.GetOrder(order.ID)
	if order.Status != StatusPartial {
		t.Errorf("expected Partial, got %q", order.Status)
	}
	if order.FilledQty != 40 {
		t.Errorf("filled qty = %f, want 40", order.FilledQty)
	}

	// Complete fill
	_, err = oms.FillOrder(order.ID, 60, 196.0)
	if err != nil {
		t.Fatalf("FillOrder error: %v", err)
	}

	order, _ = oms.GetOrder(order.ID)
	if order.Status != StatusFilled {
		t.Errorf("expected Filled, got %q", order.Status)
	}
	// Average price: (40*194 + 60*196) / 100 = 195.2
	expectedAvg := (40.0*194.0 + 60.0*196.0) / 100.0
	if order.FilledAvgPrice != expectedAvg {
		t.Errorf("avg price = %f, want %f", order.FilledAvgPrice, expectedAvg)
	}
}

func TestOrderMatcher_MarketBuy(t *testing.T) {
	m := NewOrderMatcher()
	order := &Order{
		Side:      SideBuy,
		OrderType: TypeMarket,
		Quantity:  100,
	}
	bar := OHLCVBar{Open: 195.0, High: 196.0, Low: 194.0, Close: 195.5}

	result := m.Match(order, bar)
	if !result.IsFillable {
		t.Error("market order should always be fillable")
	}
	if result.FillPrice != 195.0 {
		t.Errorf("market fill price = %f, want 195.0 (Open)", result.FillPrice)
	}
}

func TestOrderMatcher_LimitBuyFillable(t *testing.T) {
	m := NewOrderMatcher()
	order := &Order{
		Side:      SideBuy,
		OrderType: TypeLimit,
		Quantity:  100,
		Price:     195.0,
	}
	bar := OHLCVBar{Open: 196.0, High: 197.0, Low: 194.5, Close: 195.5}

	result := m.Match(order, bar)
	if !result.IsFillable {
		t.Error("limit buy should fill when bar.Low <= limit price")
	}
}

func TestOrderMatcher_LimitBuyNotFillable(t *testing.T) {
	m := NewOrderMatcher()
	order := &Order{
		Side:      SideBuy,
		OrderType: TypeLimit,
		Quantity:  100,
		Price:     190.0,
	}
	bar := OHLCVBar{Open: 195.0, High: 196.0, Low: 194.5, Close: 195.5}

	result := m.Match(order, bar)
	if result.IsFillable {
		t.Error("limit buy should not fill when bar.Low > limit price")
	}
}

func TestOrderMatcher_StopBuy(t *testing.T) {
	m := NewOrderMatcher()
	order := &Order{
		Side:      SideBuy,
		OrderType: TypeStop,
		Quantity:  100,
		StopPrice: 196.0,
	}
	bar := OHLCVBar{Open: 195.0, High: 197.0, Low: 194.0, Close: 196.5}

	result := m.Match(order, bar)
	if !result.IsFillable {
		t.Error("stop buy should trigger when bar.High >= stop price")
	}
}

func TestRiskPipeline_StopLoss(t *testing.T) {
	rp := NewRiskPipeline(DefaultRiskConfig())
	pos := &Position{
		Symbol:   "AAPL",
		Quantity: 100,
		AvgPrice: 200.0,
	}

	// Price drops 6% → should trigger 5% stop loss
	if !rp.CheckStopLoss(pos, 188.0) {
		t.Error("stop loss should trigger at 6% drop")
	}

	// Price drops 3% → should not trigger
	if rp.CheckStopLoss(pos, 194.0) {
		t.Error("stop loss should not trigger at 3% drop")
	}
}

func TestRiskPipeline_TakeProfit(t *testing.T) {
	rp := NewRiskPipeline(DefaultRiskConfig())
	pos := &Position{
		Symbol:   "AAPL",
		Quantity: 100,
		AvgPrice: 200.0,
	}

	// Price up 16% → should trigger 15% take profit
	if !rp.CheckTakeProfit(pos, 232.0) {
		t.Error("take profit should trigger at 16% gain")
	}
}
