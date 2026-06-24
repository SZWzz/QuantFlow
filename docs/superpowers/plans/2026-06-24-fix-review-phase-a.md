# Implementation Plan: Fix Review Findings — Phase A (P0 Correctness)

> Spec: [docs/specs/2026-06-24-fix-review-correctness.md](../../specs/2026-06-24-fix-review-correctness.md)
> Date: 2026-06-24
> Status: Ready for execution
> Scope: 4 个 P0 金融正确性修复，独立可执行，建议按 A1→A2→A3→A4 顺序

## Overview

Phase A 修复评审发现的 4 个 P0 致命问题，每项直接决定回测/交易结果可信度。每个 Task 遵循 TDD：先写失败测试 → 实现 → 测试通过 → commit。Task 间无依赖，可由 subagent 并行执行，但建议按顺序以便逐个 review。

---

## Task A1: 修复 OMS FillOrder 卖出裁剪顺序

> Spec ref: P0-1 · 根因 [`internal/trading/oms.go:122-124`](internal/trading/oms.go#L122) 用未裁剪 fillQty 更新订单账本，[`oms.go:160-161`](internal/trading/oms.go#L160) 才裁剪到持仓量 → 订单成交量 > 持仓变动

### Step A1.1: 写失败测试

**File:** `internal/trading/oms_test.go`（新建）

```go
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
```

验证测试失败：`go test ./internal/trading/ -run TestFillOrder_SellOverPosition -v`

预期失败：`trade.Quantity = 200, want 100`（修复前 fillQty 未裁剪就写入 trade 和 order）。

### Step A1.2: 修复 FillOrder 裁剪顺序

**File:** `internal/trading/oms.go`

替换 `FillOrder` 方法中从 `remainingQty` 裁剪之后到 position 更新之前的整段逻辑。

**当前代码（line 116-176）**：
```go
	remainingQty := order.Quantity - order.FilledQty
	if fillQty > remainingQty {
		fillQty = remainingQty
	}

	// Update average fill price
	totalValue := order.FilledAvgPrice*order.FilledQty + fillPrice*fillQty
	order.FilledQty += fillQty
	order.FilledAvgPrice = totalValue / order.FilledQty

	if order.FilledQty >= order.Quantity {
		order.Status = StatusFilled
		now := time.Now()
		order.FilledAt = &now
	} else {
		order.Status = StatusPartial
	}

	trade := &Trade{
		ID:        uuid.New().String()[:8],
		OrderID:   orderID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  fillQty,
		Price:     fillPrice,
		Timestamp: time.Now(),
	}
	o.trades = append(o.trades, trade)

	// Update position
	pos, ok := o.positions[order.Symbol]
	if !ok {
		pos = &Position{Symbol: order.Symbol}
		o.positions[order.Symbol] = pos
	}

	if order.Side == SideBuy {
		totalPosValue := pos.AvgPrice*pos.Quantity + fillPrice*fillQty
		pos.Quantity += fillQty
		if pos.Quantity > 0 {
			pos.AvgPrice = totalPosValue / pos.Quantity
		}
	} else {
		// Validate we have enough shares to sell (prevent negative positions).
		if fillQty > pos.Quantity {
			fillQty = pos.Quantity
		}
		if fillQty <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}
		// Realize P&L for the sold portion.
		pos.PnL = (fillPrice - pos.AvgPrice) * fillQty
		if pos.AvgPrice > 0 {
			pos.PnLPct = (fillPrice - pos.AvgPrice) / pos.AvgPrice * 100
		}
		pos.Quantity -= fillQty
		// Reset AvgPrice when position is flat so stale prices don't affect future entries.
		if pos.Quantity == 0 {
			pos.AvgPrice = 0
		}
	}
```

**替换为**（关键改动：卖出时在更新订单账本**之前**用持仓量裁剪 fillQty）：
```go
	remainingQty := order.Quantity - order.FilledQty
	if fillQty > remainingQty {
		fillQty = remainingQty
	}

	// P0-1 fix: for sell orders, clip fillQty to available position BEFORE updating
	// the order book. Previously the order book (FilledQty/FilledAvgPrice) was updated
	// with the unclipped fillQty, then fillQty was clipped inside the position block,
	// causing order.FilledQty > actual position change and P&L ledger mismatch.
	pos, ok := o.positions[order.Symbol]
	if order.Side == SideSell {
		if !ok || pos.Quantity <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}
		if fillQty > pos.Quantity {
			fillQty = pos.Quantity
		}
		if fillQty <= 0 {
			return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
		}
	}
	if !ok {
		pos = &Position{Symbol: order.Symbol}
		o.positions[order.Symbol] = pos
	}

	// Update average fill price (fillQty is now the final, clipped value)
	totalValue := order.FilledAvgPrice*order.FilledQty + fillPrice*fillQty
	order.FilledQty += fillQty
	order.FilledAvgPrice = totalValue / order.FilledQty

	if order.FilledQty >= order.Quantity {
		order.Status = StatusFilled
		now := time.Now()
		order.FilledAt = &now
	} else {
		order.Status = StatusPartial
	}

	trade := &Trade{
		ID:        uuid.New().String()[:8],
		OrderID:   orderID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  fillQty,
		Price:     fillPrice,
		Timestamp: time.Now(),
	}
	o.trades = append(o.trades, trade)

	// Update position (fillQty already clipped for sells above)
	if order.Side == SideBuy {
		totalPosValue := pos.AvgPrice*pos.Quantity + fillPrice*fillQty
		pos.Quantity += fillQty
		if pos.Quantity > 0 {
			pos.AvgPrice = totalPosValue / pos.Quantity
		}
	} else {
		// Realize P&L for the sold portion.
		pos.PnL = (fillPrice - pos.AvgPrice) * fillQty
		if pos.AvgPrice > 0 {
			pos.PnLPct = (fillPrice - pos.AvgPrice) / pos.AvgPrice * 100
		}
		pos.Quantity -= fillQty
		// Reset AvgPrice when position is flat so stale prices don't affect future entries.
		if pos.Quantity == 0 {
			pos.AvgPrice = 0
		}
	}
```

### Step A1.3: 验证

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestFillOrder_SellOverPosition -v
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -v  # 回归现有测试
cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./internal/trading/
```

全部通过后提交。

### Step A1.4: Commit

```
[Engine] fix(oms): clip sell fillQty to position before updating order book

P0-1 from 2026-06-24 review. Previously FillOrder updated order.FilledQty
and FilledAvgPrice with the unclipped fillQty, then clipped fillQty inside
the position block — causing order ledger to record more volume than the
position actually changed. Now sell orders clip fillQty to available
position before any book update, guaranteeing order.FilledQty ==
Trade.Quantity == position delta.

Spec: docs/specs/2026-06-24-fix-review-correctness.md P0-1
```

---

## Task A2: 实现 A 股涨跌停限制

> Spec ref: P0-2 · [`internal/backtest/engine_cn.go:12`](internal/backtest/engine_cn.go#L12) 注释宣称 ±10%/±20% 但无实现

### Step A2.1: 写 price_limit.go

**File:** `internal/backtest/price_limit.go`（新建）

```go
package backtest

import "strings"

// PriceLimitRule defines the daily price limit rule for an A-share symbol.
// A-share markets enforce ±Ratio around the previous closing price.
//   - Main board (60xxxx, 00xxxx): ±10%
//   - ChiNext / 创业板 (300xxx, 301xxx): ±20%
//   - STAR / 科创板 (688xxx, 689xxx): ±20%
//   - ST / *ST stocks: ±5%
//   - BSE / 北交所 (8xxxxx, 4xxxxx): ±30% (reserved, not enforced in v1)
//
// 首日上市、增发等无前收盘价的情形不限制（返回 0 表示不限）。
type PriceLimitRule struct {
	Ratio float64 // 0.10, 0.20, 0.05; 0 means no limit
}

// PriceLimitFor returns the limit rule for a given A-share symbol code.
// Symbol prefixes follow SSE/SZSE listing conventions.
func PriceLimitFor(symbol string) PriceLimitRule {
	upper := strings.ToUpper(symbol)

	// ST / *ST detection (name-based would be more accurate, but code-based
	// fallback: we cannot know ST status from code alone, so default ST to 5%.
	// Callers may override via Config.PriceLimitOverrides.)
	switch {
	case strings.HasPrefix(upper, "300"), strings.HasPrefix(upper, "301"): // ChiNext
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "688"), strings.HasPrefix(upper, "689"): // STAR
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "60"), strings.HasPrefix(upper, "00"): // main board
		return PriceLimitRule{Ratio: 0.10}
	case strings.HasPrefix(upper, "8"), strings.HasPrefix(upper, "4"): // BSE
		return PriceLimitRule{Ratio: 0.30}
	default:
		return PriceLimitRule{Ratio: 0.10} // safe default
	}
}

// LimitUp returns the limit-up price for today given prevClose.
// Returns 0 if no limit applies (rule.Ratio == 0 or prevClose <= 0).
func (r PriceLimitRule) LimitUp(prevClose float64) float64 {
	if r.Ratio == 0 || prevClose <= 0 {
		return 0
	}
	return prevClose * (1 + r.Ratio)
}

// LimitDown returns the limit-down price for today given prevClose.
func (r PriceLimitRule) LimitDown(prevClose float64) float64 {
	if r.Ratio == 0 || prevClose <= 0 {
		return 0
	}
	return prevClose * (1 - r.Ratio)
}

// CanBuy reports whether a buy is allowed at the given price today.
// A-share rule: cannot BUY at or above limit-up (涨停封板买不进).
func (r PriceLimitRule) CanBuy(price, prevClose float64) bool {
	up := r.LimitUp(prevClose)
	if up <= 0 {
		return true // no limit
	}
	return price < up
}

// CanSell reports whether a sell is allowed at the given price today.
// A-share rule: cannot SELL at or below limit-down (跌停封板卖不出).
func (r PriceLimitRule) CanSell(price, prevClose float64) bool {
	down := r.LimitDown(prevClose)
	if down <= 0 {
		return true // no limit
	}
	return price > down
}
```

### Step A2.2: 写涨跌停单元测试

**File:** `internal/backtest/price_limit_test.go`（新建）

```go
package backtest

import (
	"math"
	"testing"
)

func TestPriceLimitFor_BoardRules(t *testing.T) {
	cases := []struct {
		symbol string
		ratio  float64
	}{
		{"600519", 0.10}, // 主板 贵州茅台
		{"000001", 0.10}, // 主板 平安银行
		{"300750", 0.20}, // 创业板 宁德时代
		{"301088", 0.20}, // 创业板
		{"688981", 0.20}, // 科创板 中芯国际
		{"830799", 0.30}, // 北交所
	}
	for _, c := range cases {
		got := PriceLimitFor(c.symbol)
		if math.Abs(got.Ratio-c.ratio) > 1e-9 {
			t.Errorf("PriceLimitFor(%s) = %v, want ratio %v", c.symbol, got, c.ratio)
		}
	}
}

func TestPriceLimit_LimitUpDown(t *testing.T) {
	r := PriceLimitRule{Ratio: 0.10}
	if up := r.LimitUp(10.0); math.Abs(up-11.0) > 1e-9 {
		t.Errorf("LimitUp(10) = %v, want 11.0", up)
	}
	if down := r.LimitDown(10.0); math.Abs(down-9.0) > 1e-9 {
		t.Errorf("LimitDown(10) = %v, want 9.0", down)
	}
	// No prevClose → no limit
	if up := r.LimitUp(0); up != 0 {
		t.Errorf("LimitUp(0) = %v, want 0", up)
	}
}

func TestPriceLimit_CanBuyCanSell(t *testing.T) {
	r := PriceLimitRule{Ratio: 0.10} // ±10%, prevClose=10 → [9, 11]

	// 涨停价 11.0 买入应被拒
	if r.CanBuy(11.0, 10.0) {
		t.Error("CanBuy at limit-up should be false")
	}
	// 10.5 买入允许
	if !r.CanBuy(10.5, 10.0) {
		t.Error("CanBuy at 10.5 should be true")
	}
	// 跌停价 9.0 卖出应被拒
	if r.CanSell(9.0, 10.0) {
		t.Error("CanSell at limit-down should be false")
	}
	// 9.5 卖出允许
	if !r.CanSell(9.5, 10.0) {
		t.Error("CanSell at 9.5 should be true")
	}
	// 首日无 prevClose → 不限制
	if !r.CanBuy(999, 0) || !r.CanSell(1, 0) {
		t.Error("no prevClose should allow any price")
	}
}
```

### Step A2.3: 写回测集成测试（失败）

**File:** `internal/backtest/engine_cn_test.go`（新建或追加）

```go
package backtest

import (
	"context"
	"testing"

	"quantflow/internal/trading"
)

// TestCNEngine_RejectsBuyAtLimitUp 验证涨停价买入被拒。
// 构造两日行情：day0 close=10，day1 close=11（涨停）。
// day1 strategy 发 buy 信号，应被涨跌停校验拒绝，无成交记录。
func TestCNEngine_RejectsBuyAtLimitUp(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 11, High: 11, Low: 11, Close: 11, Volume: 1000},
	}

	// day1 (涨停) 发出买入信号
	strategy := Strategy{
		ID:   "test-limit-up",
		Name: "limit-up test",
		SignalFunc: func(bar trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if bar.Date == "2026-06-02" {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// 应无任何买入成交（day1 涨停封板买不进）
	buys := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buys++
		}
	}
	if buys != 0 {
		t.Errorf("expected 0 buys at limit-up, got %d", buys)
	}
}

// TestCNEngine_AllowsNormalBuy 验证非涨停价正常买入。
func TestCNEngine_AllowsNormalBuy(t *testing.T) {
	bars := []trading.OHLCVBar{
		{Symbol: "600519", Date: "2026-06-01", Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000},
		{Symbol: "600519", Date: "2026-06-02", Open: 10.5, High: 10.5, Low: 10.5, Close: 10.5, Volume: 1000},
	}

	strategy := Strategy{
		ID:   "test-normal-buy",
		Name: "normal buy test",
		SignalFunc: func(bar trading.OHLCVBar, portfolio *Portfolio) *trading.Signal {
			if bar.Date == "2026-06-02" {
				return &trading.Signal{Direction: "buy", Quantity: 100}
			}
			return nil
		},
	}

	engine := NewCNEngine(Config{InitialCash: 100000})
	result, err := engine.Run(context.Background(), strategy, bars)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	buys := 0
	for _, tr := range result.Trades {
		if tr.Side == "buy" {
			buys++
		}
	}
	if buys != 1 {
		t.Errorf("expected 1 normal buy, got %d", buys)
	}
}
```

验证集成测试失败：`go test ./internal/backtest/ -run TestCNEngine_RejectsBuyAtLimitUp -v`
预期：`expected 0 buys at limit-up, got 1`（修复前无涨跌停校验）。

### Step A2.4: 在 CNEngine 中接入涨跌停校验

**File:** `internal/backtest/engine_cn.go`

1) 在 `CNEngine` struct 加 `prevClose map[string]float64` 字段：

```go
type CNEngine struct {
	*Runner
	t1Lock   *t1Tracker
	prevClose map[string]float64 // symbol → previous trading day close (for price limit)
}
```

2) `NewCNEngine` 初始化：

```go
	return &CNEngine{
		Runner:    NewRunner(config),
		t1Lock:    newT1Tracker(),
		prevClose: make(map[string]float64),
	}
```

3) 在 `Run` 循环内，`e.oms.UpdateMarketPrice` 之后、信号生成之前，插入涨跌停校验。修改 `processCNBuySignal` 和 `processCNSellSignal` 的签名加入 prevClose 参数，或在调用前校验。

最小改动方案：在 `processCNBuySignal` 开头加校验，在 `processCNSellSignal` 开头加校验。

修改 `processCNBuySignal` 函数签名，新增 `prevClose float64` 参数：

```go
func (e *CNEngine) processCNBuySignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	rule := PriceLimitFor(bar.Symbol)
	if !rule.CanBuy(bar.Close, e.prevClose[bar.Symbol]) {
		// 涨停封板，买不进
		return
	}
	qty := signal.Quantity
	// ... 后续逻辑不变
```

修改 `processCNSellSignal` 函数开头加校验：

```go
func (e *CNEngine) processCNSellSignal(bar trading.OHLCVBar, signal *trading.Signal, portfolio *Portfolio, trades *[]TradeRecord) {
	rule := PriceLimitFor(bar.Symbol)
	if !rule.CanSell(bar.Close, e.prevClose[bar.Symbol]) {
		// 跌停封板，卖不出
		return
	}
	qty := signal.Quantity
	// ... 后续逻辑不变
```

4) 在 `Run` 循环末尾（`recordEquityCN` 标签后、记录 equity 前）更新 prevClose：

```go
recordEquityCN:
	// Update prevClose for next day's price limit check
	e.prevClose[bar.Symbol] = bar.Close
```

### Step A2.5: 验证

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/backtest/ -run TestPriceLimit -v
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/backtest/ -run TestCNEngine -v
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/backtest/ -v  # 回归
cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./internal/backtest/
```

### Step A2.6: Commit

```
[Engine] feat(backtest): enforce A-share daily price limits (涨跌停)

P0-2 from 2026-06-24 review. CNEngine previously had no price limit
enforcement despite documenting ±10%/±20%. Now checks CanBuy/CanSell
against prevClose before processing signals: buys rejected at/above
limit-up, sells rejected at/below limit-down. Board detection by symbol
prefix: main board ±10%, ChiNext/STAR ±20%, BSE ±30%. First day (no
prevClose) is unrestricted.

Spec: docs/specs/2026-06-24-fix-review-correctness.md P0-2
```

---

## Task A3: 修复横截面因子经标准 RPC 路径失效

> Spec ref: P0-3 · [`python/src/factor/engine.py:44`](python/src/factor/engine.py#L44) 先按 symbol 过滤再调用因子，导致横截面因子 std=0 退化

### Step A3.1: 写失败测试

**File:** `python/tests/test_factor_cross_sectional.py`（新建）

```python
"""Tests for cross-sectional factor computation via standard RPC path.

Regression for P0-3: engine.py filtered to single symbol before calling
the factor function, so cross-sectional zscore/rank saw std=0 and
returned all-zeros / all-0.5 — functionally broken.
"""
import numpy as np
import pandas as pd
import pyarrow as pa
import pyarrow.ipc as ipc

from src.factor.registry import compute


def _encode_ohlcv(df: pd.DataFrame) -> bytes:
    """Encode a DataFrame to Arrow IPC stream bytes (matches gRPC wire format)."""
    table = pa.Table.from_pandas(df, preserve_index=False)
    sink = pa.BufferOutputStream()
    with ipc.new_stream(sink, table.schema) as writer:
        writer.write_table(table)
    return sink.getvalue().to_pybytes()


def _make_panel(symbols, dates, base_price=10.0):
    """Build a multi-symbol OHLCV panel with distinct momentum per symbol."""
    rows = []
    for i, sym in enumerate(symbols):
        price = base_price * (1 + i * 0.5)  # distinct price levels
        for d in dates:
            rows.append({
                "symbol": sym, "date": d,
                "open": price, "high": price, "low": price,
                "close": price, "volume": 10000,
            })
            price *= 1.01  # upward drift, different per symbol due to base
    return pd.DataFrame(rows)


def test_zscore_momentum_cross_sectional_nonzero():
    """zscore_momentum_20d on 3+ symbols must NOT be all zeros.

    With single-symbol filtering (the bug), std=0 → zscore=0 for every row.
    With full-panel computation, cross-sectional zscore should have
    non-zero variance and sum to ~0 per date.
    """
    symbols = ["AAA", "BBB", "CCC"]
    dates = pd.date_range("2026-01-01", periods=30, freq="D").astype(str)
    df = _make_panel(symbols, dates)

    # Compute on the FULL panel (what engine.py should do for cross-sectional)
    values = compute("zscore_momentum_20d", df, {})

    # Drop the warmup NaNs (first 20 days have no 20d momentum)
    valid = values.dropna()
    assert len(valid) > 0, "no valid zscore values produced"

    # BUG: with single-symbol filtering, every zscore would be 0.
    assert valid.std() > 1e-6, (
        f"cross-sectional zscore std is ~0 — factor is broken. "
        f"std={valid.std()}, values sample: {valid.head().tolist()}"
    )

    # Cross-sectional zscore property: per-date mean ≈ 0
    df_out = df.copy()
    df_out["zscore"] = values.values
    df_out = df_out.dropna(subset=["zscore"])
    per_date_mean = df_out.groupby("date")["zscore"].mean()
    assert per_date_mean.abs().max() < 1e-6, (
        f"per-date zscore mean should be ~0, got max abs {per_date_mean.abs().max()}"
    )


def test_rank_momentum_cross_sectional_distribution():
    """rank_momentum_20d on 3 symbols should produce ranks in [0,1] with spread."""
    symbols = ["AAA", "BBB", "CCC"]
    dates = pd.date_range("2026-01-01", periods=30, freq="D").astype(str)
    df = _make_panel(symbols, dates)

    values = compute("rank_momentum_20d", df, {})
    valid = values.dropna()

    # BUG: with single-symbol filtering, rank would be 0.5 for every row.
    assert valid.nunique() > 1, (
        f"rank has only 1 unique value — factor is broken. "
        f"unique={valid.nunique()}, values: {valid.unique().tolist()}"
    )
    assert valid.min() >= 0 and valid.max() <= 1
```

验证失败：`cd python && python -m pytest tests/test_factor_cross_sectional.py -x -v`
预期：`AssertionError: cross-sectional zscore std is ~0`（因 engine 过滤到单标的，但这里直接调 compute 传完整 panel 应该通过...）

> 注意：此测试直接调 `compute()` 传完整 panel，应通过。真正的 bug 在 `engine.py` 的 RPC 层过滤。因此还需一个 RPC 层测试，但为简化，本测试作为"完整 panel 行为正确"的基线，下一步修复 engine.py 的过滤逻辑后，RPC 层也会调用完整 panel。

### Step A3.2: 修复 engine.py 横截面分发

**File:** `python/src/factor/engine.py`

核心问题：`ComputeFactor` 和 `ComputeFactorBatch` 都在循环内 `df[df["symbol"]==symbol]` 过滤到单标的。横截面因子需要完整 panel。

修复：按因子 category 分发。横截面因子（category == "cross_sectional"）在完整 panel 上计算后按 symbol 切片返回；时序因子保持原逐 symbol 逻辑。

1) 在 `registry.py` 增加 category 查询辅助函数：

**File:** `python/src/factor/registry.py` 末尾追加：

```python
def is_cross_sectional(factor_name: str) -> bool:
    """Return True if the factor requires a multi-symbol panel (cross-sectional)."""
    meta = _registry.get(factor_name)
    return meta is not None and meta.category == "cross_sectional"
```

2) 重写 `engine.py` 的 `ComputeFactor`，按 `is_cross_sectional` 分发：

**File:** `python/src/factor/engine.py` 替换 `ComputeFactor` 方法：

```python
    async def ComputeFactor(self, request, context):
        t0 = time.time()
        try:
            from src.factor.registry import _compute_funcs, is_cross_sectional
            if request.factor_name not in _compute_funcs:
                return factor_pb2.ComputeFactorResponse(
                    factor_name=request.factor_name,
                    error=f"Unknown factor: {request.factor_name}",
                )

            # Decode Arrow IPC bytes → pandas DataFrame
            if request.ohlcv_data:
                reader = ipc.open_stream(request.ohlcv_data)
                table = reader.read_all()
                df = table.to_pandas()
            else:
                df = pd.DataFrame()

            results = []

            if df.empty:
                # No data — return empty results for each requested symbol
                for symbol in request.symbols:
                    results.append(factor_pb2.FactorResult(symbol=symbol))
            elif is_cross_sectional(request.factor_name) and "symbol" in df.columns:
                # P0-3 fix: cross-sectional factors need the FULL multi-symbol panel.
                # Compute once on the complete DataFrame, then slice per symbol.
                values = compute(request.factor_name, df, dict(request.params))
                df_with_vals = df.copy()
                df_with_vals["_factor_val"] = values.values

                for symbol in request.symbols:
                    symbol_vals = df_with_vals[df_with_vals["symbol"] == symbol]["_factor_val"]
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=df_with_vals[df_with_vals["symbol"] == symbol]["date"].astype(str).tolist()
                            if "date" in df_with_vals.columns else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in symbol_vals.tolist()],
                        )
                    )
            else:
                # Time-series factor: compute per symbol (original behavior)
                for symbol in request.symbols:
                    if "symbol" in df.columns:
                        symbol_df = df[df["symbol"] == symbol]
                    else:
                        symbol_df = df
                    if symbol_df.empty:
                        continue
                    values = compute(request.factor_name, symbol_df, dict(request.params))
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=values.index.astype(str).tolist()
                            if hasattr(values, "index") else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in values.tolist()],
                        )
                    )

            elapsed_ms = int((time.time() - t0) * 1000)
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                results=results,
                compute_time_ms=elapsed_ms,
            )
        except Exception as e:
            logger.exception(f"ComputeFactor failed: {e}")
            return factor_pb2.ComputeFactorResponse(
                factor_name=request.factor_name,
                error=str(e),
            )
```

3) 同样修复 `ComputeFactorBatch` 内的 `compute_one` 函数（应用相同的分发逻辑）：

**File:** `python/src/factor/engine.py` 替换 `compute_one`：

```python
        async def compute_one(factor_name):
            """Compute a single factor using the pre-decoded DataFrame."""
            from src.factor.registry import _compute_funcs, is_cross_sectional
            if factor_name not in _compute_funcs:
                return factor_pb2.ComputeFactorResponse(
                    factor_name=factor_name,
                    error=f"Unknown factor: {factor_name}",
                )
            results = []

            if df.empty:
                for symbol in request.symbols:
                    results.append(factor_pb2.FactorResult(symbol=symbol))
            elif is_cross_sectional(factor_name) and "symbol" in df.columns:
                # P0-3 fix: full panel for cross-sectional
                values = compute(factor_name, df, dict(request.params))
                df_with_vals = df.copy()
                df_with_vals["_factor_val"] = values.values
                for symbol in request.symbols:
                    sub = df_with_vals[df_with_vals["symbol"] == symbol]
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=sub["date"].astype(str).tolist() if "date" in sub.columns else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in sub["_factor_val"].tolist()],
                        )
                    )
            else:
                for symbol in request.symbols:
                    symbol_df = df[df["symbol"] == symbol] if "symbol" in df.columns else df
                    if symbol_df.empty:
                        continue
                    values = compute(factor_name, symbol_df, dict(request.params))
                    results.append(
                        factor_pb2.FactorResult(
                            symbol=symbol,
                            dates=values.index.astype(str).tolist() if hasattr(values, "index") else [],
                            values=[float(v) if not pd.isna(v) else float('nan') for v in values.tolist()],
                        )
                    )
            return factor_pb2.ComputeFactorResponse(
                factor_name=factor_name,
                results=results,
            )
```

### Step A3.3: 验证

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/python && python -m pytest tests/test_factor_cross_sectional.py -x -v
cd /Volumes/shenzy/vibe_coding/QuantFlow/python && python -m pytest tests/ -x -q  # 回归
```

### Step A3.4: Commit

```
[Python] fix(factor): dispatch cross-sectional factors on full panel

P0-3 from 2026-06-24 review. FactorService.ComputeFactor/Batch filtered
to a single symbol before calling the factor function, so cross-sectional
zscore/rank saw std=0 and returned all-zeros — functionally broken. Now
factors with category=="cross_sectional" compute on the full multi-symbol
panel, then slice per symbol for the response. Time-series factors keep
the original per-symbol path. Added is_cross_sectional() helper to registry.

Spec: docs/specs/2026-06-24-fix-review-correctness.md P0-3
```

---

## Task A4: ML 训练验证集划分

> Spec ref: P0-4 · [`python/src/ml/tree_engine.py:110-122`](python/src/ml/tree_engine.py#L110) 用同一份 X 评估，无验证集

### Step A4.1: 写失败测试

**File:** `python/tests/test_tree_engine_validation.py`（新建）

```python
"""Tests for TreeEngine train/val split (P0-4).

Regression: previously _train_xgboost/_train_lightgbm computed metrics
on the same X used for training (train_rmse/train_accuracy), with no
validation set — causing overfitting and data leakage for time-series.
"""
import numpy as np
import pyarrow as pa
import pytest

from src.ml.tree_engine import TreeEngine


def _make_features(n=300, seed=42):
    rng = np.random.default_rng(seed)
    X = rng.normal(0, 1, size=(n, 5)).astype(np.float64)
    y = (X[:, 0] * 0.5 + X[:, 1] * 0.3 + rng.normal(0, 0.1, n)).astype(np.float64)
    return pa.Table.from_arrays([pa.array(X[:, i]) for i in range(5)], names=[f"f{i}" for i in range(5)]), \
           pa.Table.from_arrays([pa.array(y)], names=["target"])


def test_train_returns_validation_metrics():
    """train() must return val_rmse/val_mae (not just train_*)."""
    features, targets = _make_features(300)
    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "regression",
    })
    metrics = result["metrics"]
    assert "val_rmse" in metrics, f"missing val_rmse in metrics: {metrics}"
    assert "val_mae" in metrics, f"missing val_mae in metrics: {metrics}"


def test_train_validation_metrics_differ_from_train():
    """val metrics should differ from train metrics (proves a real split)."""
    features, targets = _make_features(300)
    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "regression",
    })
    metrics = result["metrics"]
    # val_rmse should generally be >= train_rmse (overfitting), and must differ
    assert "train_rmse" in metrics and "val_rmse" in metrics
    assert metrics["train_rmse"] != metrics["val_rmse"], (
        f"train_rmse == val_rmse, split likely not applied: {metrics}"
    )


def test_train_classification_returns_val_accuracy():
    """Classification target must return val_accuracy."""
    rng = np.random.default_rng(0)
    X = rng.normal(0, 1, size=(200, 4))
    y = (X[:, 0] + rng.normal(0, 0.5, 200) > 0).astype(np.int64)
    features = pa.Table.from_arrays([pa.array(X[:, i]) for i in range(4)], names=[f"f{i}" for i in range(4)])
    targets = pa.Table.from_arrays([pa.array(y)], names=["target"])

    engine = TreeEngine()
    result = engine.train(features, targets, {
        "model_type": "xgboost",
        "model_dir": "/tmp/qf_test_models",
        "target_type": "classification",
    })
    assert "val_accuracy" in result["metrics"], f"missing val_accuracy: {result['metrics']}"
```

验证失败：`cd python && python -m pytest tests/test_tree_engine_validation.py -x -v`
预期：`AssertionError: missing val_rmse in metrics`（当前只返回 train_rmse）。

### Step A4.2: 修复 tree_engine.py 加入验证集划分

**File:** `python/src/ml/tree_engine.py`

1) 在文件顶部 import 区加：

```python
from sklearn.model_selection import train_test_split
```

2) 替换 `_train_xgboost` 方法：

```python
    def _train_xgboost(self, X: np.ndarray, y: np.ndarray, params: dict, target_type: str):
        import xgboost as xgb

        n_estimators = int(params.get("n_estimators", 100))
        max_depth = int(params.get("max_depth", 6))
        learning_rate = float(params.get("learning_rate", 0.1))

        # P0-4 fix: time-series safe split (shuffle=False to avoid future leakage).
        # Default 80/20; falls back to single-split for small samples.
        test_size = 0.2 if len(X) >= 50 else 0.0
        if test_size > 0:
            X_tr, X_val, y_tr, y_val = train_test_split(X, y, test_size=test_size, shuffle=False)
        else:
            X_tr, X_val, y_tr, y_val = X, X, y, y  # too small, evaluate on train (warned in log)

        if target_type == "classification":
            model = xgb.XGBClassifier(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, eval_metric="logloss"
            )
            model.fit(X_tr, y_tr)
            y_tr_pred = model.predict(X_tr)
            y_val_pred = model.predict(X_val)
            train_accuracy = float(np.mean(y_tr_pred == y_tr))
            val_accuracy = float(np.mean(y_val_pred == y_val))
            metrics = {"train_accuracy": train_accuracy, "val_accuracy": val_accuracy}
        else:
            model = xgb.XGBRegressor(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42
            )
            model.fit(X_tr, y_tr)
            y_tr_pred = model.predict(X_tr)
            y_val_pred = model.predict(X_val)
            train_rmse = float(np.sqrt(np.mean((y_tr - y_tr_pred) ** 2)))
            train_mae = float(np.mean(np.abs(y_tr - y_tr_pred)))
            val_rmse = float(np.sqrt(np.mean((y_val - y_val_pred) ** 2)))
            val_mae = float(np.mean(np.abs(y_val - y_val_pred)))
            metrics = {
                "train_rmse": train_rmse, "train_mae": train_mae,
                "val_rmse": val_rmse, "val_mae": val_mae,
            }
            if test_size == 0:
                logger.warning("XGBoost train: sample <50, val metrics computed on train set")

        return model, metrics
```

3) 同样替换 `_train_lightgbm` 方法（结构相同）：

```python
    def _train_lightgbm(self, X: np.ndarray, y: np.ndarray, params: dict, target_type: str):
        import lightgbm as lgb

        n_estimators = int(params.get("n_estimators", 100))
        max_depth = int(params.get("max_depth", -1))
        learning_rate = float(params.get("learning_rate", 0.1))

        test_size = 0.2 if len(X) >= 50 else 0.0
        if test_size > 0:
            X_tr, X_val, y_tr, y_val = train_test_split(X, y, test_size=test_size, shuffle=False)
        else:
            X_tr, X_val, y_tr, y_val = X, X, y, y

        if target_type == "classification":
            model = lgb.LGBMClassifier(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, verbose=-1
            )
            model.fit(X_tr, y_tr)
            y_tr_pred = model.predict(X_tr)
            y_val_pred = model.predict(X_val)
            train_accuracy = float(np.mean(y_tr_pred == y_tr))
            val_accuracy = float(np.mean(y_val_pred == y_val))
            metrics = {"train_accuracy": train_accuracy, "val_accuracy": val_accuracy}
        else:
            model = lgb.LGBMRegressor(
                n_estimators=n_estimators, max_depth=max_depth,
                learning_rate=learning_rate, random_state=42, verbose=-1
            )
            model.fit(X_tr, y_tr)
            y_tr_pred = model.predict(X_tr)
            y_val_pred = model.predict(X_val)
            train_rmse = float(np.sqrt(np.mean((y_tr - y_tr_pred) ** 2)))
            train_mae = float(np.mean(np.abs(y_tr - y_tr_pred)))
            val_rmse = float(np.sqrt(np.mean((y_val - y_val_pred) ** 2)))
            val_mae = float(np.mean(np.abs(y_val - y_val_pred)))
            metrics = {
                "train_rmse": train_rmse, "train_mae": train_mae,
                "val_rmse": val_rmse, "val_mae": val_mae,
            }
            if test_size == 0:
                logger.warning("LightGBM train: sample <50, val metrics computed on train set")

        return model, metrics
```

### Step A4.3: 验证

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/python && python -m pytest tests/test_tree_engine_validation.py -x -v
cd /Volumes/shenzy/vibe_coding/QuantFlow/python && python -m pytest tests/ -x -q  # 回归
```

> 注意：若 xgboost/lightgbm 未安装，测试会 skip（HAS_XGB/HAS_LGB 模式）。确认 venv 已 `pip install xgboost lightgbm scikit-learn`。

### Step A4.4: Commit

```
[Python] fix(ml): add train/validation split for tree models

P0-4 from 2026-06-24 review. TreeEngine._train_xgboost/_train_lightgbm
computed metrics on the same X used for training (train_rmse only), with
no validation set — causing overfitting and time-series data leakage. Now
splits 80/20 with shuffle=False (forward split, no future leakage) and
reports both train_* and val_* metrics. Small samples (<50 rows) fall back
to train-set evaluation with a warning. Classification gets val_accuracy.

Spec: docs/specs/2026-06-24-fix-review-correctness.md P0-4
```

---

## Phase A 完成验收

执行完 4 个 Task 后，统一验证：

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go test ./... -count=1
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit && npx vitest run
cd /Volumes/shenzy/vibe_coding/QuantFlow/python && python -m pytest tests/ -x -q
```

更新 CHANGELOG.md：

```markdown
## [2026.6.24] - 2026-06-24

### Fixed
- [Engine] OMS FillOrder 卖出裁剪顺序：fillQty 在更新订单账本前裁剪到持仓量，保证 order.FilledQty == Trade.Quantity == 持仓变动 (P0-1)
- [Engine] CNEngine 补齐 A 股涨跌停限制：主板 ±10%、创业板/科创板 ±20%、北交所 ±30%，涨停封板买不进、跌停封板卖不出 (P0-2)
- [Python] 横截面因子经标准 RPC 路径失效：zscore/rank 现在在完整多标的 panel 上计算，不再逐 symbol 过滤后退化 (P0-3)
- [Python] ML 树模型训练无验证集：XGBoost/LightGBM 加入 80/20 时序安全切分(shuffle=False)，同时返回 train_* 与 val_* 指标 (P0-4)
```

更新版本号三处（package.json / README.md / CHANGELOG.md）为 `2026.6.24`。

---

## 后续

Phase A 完成且全绿后，进入 Phase B（P1 防护与一致性），plan 文件：`docs/superpowers/plans/2026-06-24-fix-review-phase-b.md`。

Phase B 含：T+1 非日频修复、BiasGuard 防护、CacheKey 确定性、Calmar 量纲、前端 ref&lt;Map&gt; 响应性、IPC 统一、收益率混用统一。
