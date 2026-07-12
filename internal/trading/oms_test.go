package trading

import (
	"strings"
	"testing"
)

// TestFillOrder_SellOverPosition_ClipsBeforeBookUpdate 验证：当卖出量超过持仓时，
// fillQty 在更新订单账本之前裁剪，保证 order.FilledQty == 持仓实际减少量 == Trade.Quantity。
// Regression for P0-1: 修复前 order.FilledQty 会大于持仓变动。
func TestFillOrder_SellOverPosition_ClipsBeforeBookUpdate(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(50000)

	// 先买入 100 股建立持仓
	buyOrder, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if _, err := oms.FillOrder(buyOrder.ID, 100, 150.0); err != nil {
		t.Fatalf("FillOrder buy: %v", err)
	}

	// Simulate next day for T+1 lock
	oms.ClearT1Lock()

	// 下卖单 200 股（超过持仓 100）
	sellOrder, err := oms.PlaceOrder("AAPL", SideSell, TypeMarket, "", 200, 0)
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

// ---- P0-1: T+1 Lock Tests ----

func TestT1Lock_SameDaySellFails(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(100000)

	// Buy
	buy, err := oms.PlaceOrder("000001.SZ", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if _, err := oms.FillOrder(buy.ID, 100, 50.0); err != nil {
		t.Fatalf("FillOrder buy: %v", err)
	}

	// Try to sell same day — must fail
	sell, err := oms.PlaceOrder("000001.SZ", SideSell, TypeMarket, "", 50, 0)
	if err != nil {
		t.Fatalf("PlaceOrder sell: %v", err)
	}
	_, err = oms.FillOrder(sell.ID, 50, 55.0)
	if err == nil {
		t.Fatal("expected T+1 error, got nil")
	}
	if !strings.Contains(err.Error(), "T+1 lock") {
		t.Errorf("error should mention T+1 lock, got: %v", err)
	}
}

func TestT1Lock_NextDaySellSucceeds(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(100000)

	// Buy
	buy, err := oms.PlaceOrder("000001.SZ", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder buy: %v", err)
	}
	if _, err := oms.FillOrder(buy.ID, 100, 50.0); err != nil {
		t.Fatalf("FillOrder buy: %v", err)
	}

	// Simulate next day
	oms.ClearT1Lock()

	// Sell should succeed
	sell, err := oms.PlaceOrder("000001.SZ", SideSell, TypeMarket, "", 50, 0)
	if err != nil {
		t.Fatalf("PlaceOrder sell: %v", err)
	}
	_, err = oms.FillOrder(sell.ID, 50, 55.0)
	if err != nil {
		t.Fatalf("expected sell to succeed next day, got: %v", err)
	}
}

// ---- P0-2: Price Limit Tests ----

func TestPriceLimit_WithinLimitSucceeds(t *testing.T) {
	oms := NewOMS()

	// Set price limit: prevClose=100, maxPct=0.10 → [90, 110]
	oms.SetPriceLimit("600519.SH", 100.0, 0.10)

	order, err := oms.PlaceOrder("600519.SH", SideBuy, TypeLimit, "", 100, 105.0)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	_, err = oms.FillOrder(order.ID, 100, 105.0)
	if err != nil {
		t.Fatalf("price within limit should succeed, got: %v", err)
	}
}

func TestPriceLimit_OutsideLimitFails(t *testing.T) {
	oms := NewOMS()

	// Set price limit: prevClose=100, maxPct=0.10 → [90, 110]
	oms.SetPriceLimit("600519.SH", 100.0, 0.10)

	order, err := oms.PlaceOrder("600519.SH", SideBuy, TypeLimit, "", 100, 115.0)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	// Fill at 115 (above 110 limit)
	_, err = oms.FillOrder(order.ID, 100, 115.0)
	if err == nil {
		t.Fatal("price above limit should fail")
	}
	if !strings.Contains(err.Error(), "price limit") {
		t.Errorf("error should mention price limit, got: %v", err)
	}
}

func TestPriceLimit_NoLimitConfigured(t *testing.T) {
	oms := NewOMS()

	// No price limit set — should succeed at any price
	order, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	_, err = oms.FillOrder(order.ID, 100, 9999.0)
	if err != nil {
		t.Fatalf("no limit configured should succeed, got: %v", err)
	}
}

// ---- P0-3: Trading Costs Tests ----

func TestTradingCosts_BuyOnlyCommission(t *testing.T) {
	oms := NewOMS()

	order, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	trade, err := oms.FillOrder(order.ID, 100, 150.0)
	if err != nil {
		t.Fatalf("FillOrder: %v", err)
	}

	// commission = max(150*100*0.00025, 5) = 5
	if trade.Commission != 5.0 {
		t.Errorf("commission = %f, want 5.0", trade.Commission)
	}
	if trade.StampTax != 0 {
		t.Errorf("stampTax = %f, want 0 (buy has no stamp tax)", trade.StampTax)
	}
}

func TestTradingCosts_SellCommissionAndStampTax(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(100000)

	// Buy first to establish position
	buy, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	oms.FillOrder(buy.ID, 100, 150.0)

	// Simulate next day for T+1
	oms.ClearT1Lock()

	// Sell
	sell, _ := oms.PlaceOrder("AAPL", SideSell, TypeMarket, "", 100, 0)
	trade, err := oms.FillOrder(sell.ID, 100, 160.0)
	if err != nil {
		t.Fatalf("FillOrder sell: %v", err)
	}

	// commission = max(160*100*0.00025, 5) = 5
	// stampTax = 160*100*0.0005 = 8.0
	if trade.Commission != 5.0 {
		t.Errorf("commission = %f, want 5.0", trade.Commission)
	}
	if trade.StampTax != 8.0 {
		t.Errorf("stampTax = %f, want 8.0", trade.StampTax)
	}
}

// ---- P0-4: CashLedger Tests ----

func TestCashLedger_DepositAndWithdraw(t *testing.T) {
	cl := NewCashLedger()

	if err := cl.Deposit(10000); err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if cl.GetBalance() != 10000 {
		t.Errorf("balance = %f, want 10000", cl.GetBalance())
	}

	if err := cl.Withdraw(3000); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if cl.GetBalance() != 7000 {
		t.Errorf("balance = %f, want 7000", cl.GetBalance())
	}

	// Overdraft should fail
	if err := cl.Withdraw(8000); err == nil {
		t.Fatal("expected overdraft error, got nil")
	}
}

func TestCashLedger_DepositNegativeFails(t *testing.T) {
	cl := NewCashLedger()
	if err := cl.Deposit(-100); err == nil {
		t.Fatal("negative deposit should fail")
	}
}

func TestCashLedger_BuyReducesBalance(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(50000)

	order, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	oms.FillOrder(order.ID, 100, 150.0)

	// Cash = 50000 - 150*100 - 5(commission) = 50000 - 15005 = 34995
	balance := oms.GetCashBalance()
	if balance != 34995.0 {
		t.Errorf("balance = %f, want 34995", balance)
	}
}

func TestCashLedger_SellIncreasesBalance(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(50000)

	// Buy
	buy, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 0)
	oms.FillOrder(buy.ID, 100, 150.0)
	// Cash after buy: 50000 - 15005 = 34995

	// Simulate next day for T+1
	oms.ClearT1Lock()

	// Sell
	sell, _ := oms.PlaceOrder("AAPL", SideSell, TypeMarket, "", 100, 0)
	oms.FillOrder(sell.ID, 100, 160.0)
	// Cash after sell: 34995 + (16000 - 5 - 8) = 34995 + 15987 = 50982

	balance := oms.GetCashBalance()
	if balance != 50982.0 {
		t.Errorf("balance = %f, want 50982", balance)
	}
}

func TestCashLedger_EntriesAuditTrail(t *testing.T) {
	cl := NewCashLedger()

	cl.Deposit(10000)
	cl.Withdraw(2000)

	entries := cl.GetEntries()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}

	if entries[0].Type != "deposit" || entries[0].Amount != 10000.0 {
		t.Errorf("entry[0] = %+v, want deposit 10000", entries[0])
	}
	if entries[0].Balance != 10000.0 {
		t.Errorf("entry[0].Balance = %f, want 10000", entries[0].Balance)
	}

	if entries[1].Type != "withdraw" || entries[1].Amount != -2000.0 {
		t.Errorf("entry[1] = %+v, want withdraw -2000", entries[1])
	}
	if entries[1].Balance != 8000.0 {
		t.Errorf("entry[1].Balance = %f, want 8000", entries[1].Balance)
	}
}

func TestOMS_GetCashBalance_DelegatesToLedger(t *testing.T) {
	oms := NewOMS()
	oms.GetCashLedger().Deposit(12345.67)

	if oms.GetCashBalance() != 12345.67 {
		t.Errorf("GetCashBalance = %f, want 12345.67", oms.GetCashBalance())
	}
}
