package trading

import "testing"

// TestFillOrder_SellOverPosition_ClipsBeforeBookUpdate 验证：当卖出量超过持仓时，
// fillQty 在更新订单账本之前裁剪，保证 order.FilledQty == 持仓实际减少量 == Trade.Quantity。
// Regression for P0-1: 修复前 order.FilledQty 会大于持仓变动。
func TestFillOrder_SellOverPosition_ClipsBeforeBookUpdate(t *testing.T) {
	oms := NewOMS()

	// 先买入 100 股建立持仓
	buyOrder, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if _, err := oms.FillOrder(buyOrder.ID, 100, 150.0); err != nil {
		t.Fatalf("FillOrder buy: %v", err)
	}

	// 下卖单 200 股（超过持仓 100）
	sellOrder, err := oms.PlaceOrder("AAPL", SideSell, TypeMarket, 200, 0)
	if err != nil {
		t.Fatalf("PlaceOrder sell: %v", err)
	}

	trade, err := oms.FillOrder(sellOrder.ID, 200, 160.0)
	if err != nil {
		t.Fatalf("FillOrder sell: %v", err)
	}

	// 1. Trade.Quantity 必须是裁剪后的 100
	if trade.Quantity != 100 {
		t.Errorf("trade.Quantity = %f, want 100 (clipped to position)", trade.Quantity)
	}

	// 2. order.FilledQty 必须是 100，不是 200
	filledSell, _ := oms.GetOrder(sellOrder.ID)
	if filledSell.FilledQty != 100 {
		t.Errorf("order.FilledQty = %f, want 100 (clipped before book update)", filledSell.FilledQty)
	}

	// 3. 持仓应清零
	pos := oms.GetPosition("AAPL")
	if pos == nil || pos.Quantity != 0 {
		t.Errorf("position qty = %v, want 0", pos)
	}

	// 4. order.FilledAvgPrice 应基于 100 股，不是 200
	if filledSell.FilledAvgPrice != 160.0 {
		t.Errorf("order.FilledAvgPrice = %f, want 160.0", filledSell.FilledAvgPrice)
	}
}
