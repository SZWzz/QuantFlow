package trading

import (
	"context"
	"log/slog"
	"time"
)

// WashSaleEvent represents a flagged wash sale trade pair.
type WashSaleEvent struct {
	Symbol       string  `json:"symbol"`
	LossDate     string  `json:"loss_date"`
	LossAmount   float64 `json:"loss_amount"`
	RepurchaseDate string `json:"repurchase_date"`
	WindowDays   int     `json:"window_days"`
	DisallowedLoss float64 `json:"disallowed_loss"`
	AdjustedBasis  float64 `json:"adjusted_basis"`
}

// TradeRecord is a simplified trade for wash sale detection.
type TradeRecord struct {
	Symbol   string
	Date     time.Time
	Quantity int
	Price    float64
	Side     string // "buy" or "sell"
}

// WashSaleChecker scans trade history for wash sale patterns.
// IRS Rule 1091: loss on a sale is disallowed if the same security
// is purchased within 61 days (30 before + 1 sale day + 30 after).
type WashSaleChecker struct {
	WindowDays int
}

func NewWashSaleChecker() *WashSaleChecker {
	return &WashSaleChecker{WindowDays: 61}
}

func (w *WashSaleChecker) Check(trades []TradeRecord) []WashSaleEvent {
	var events []WashSaleEvent
	bySymbol := make(map[string][]TradeRecord)
	for _, t := range trades {
		bySymbol[t.Symbol] = append(bySymbol[t.Symbol], t)
	}
	for symbol, st := range bySymbol {
		// Track running weighted-average cost basis from prior buy trades.
		// We compute cost basis BEFORE the sell date so look-ahead bias is avoided.
		for i, t := range st {
			if t.Side != "sell" || t.Price <= 0 {
				continue
			}
			// Compute cost basis from all buy trades before this sell
			var totalCost, totalQty float64
			for _, bt := range st {
				if bt.Side != "buy" || !bt.Date.Before(t.Date) {
					continue
				}
				totalCost += float64(bt.Quantity) * bt.Price
				totalQty += float64(bt.Quantity)
			}
			if totalQty <= 0 {
				continue
			}
			costBasis := totalCost / totalQty
			// Loss = cost basis - sell price (positive means sold at a loss)
			lossAmt := float64(t.Quantity) * (costBasis - t.Price)
			if lossAmt <= 0 {
				continue
			}

			// Check for repurchases within 61-day window (30 before + sale day + 30 after)
			lossWindowStart := t.Date.AddDate(0, 0, -30)
			lossWindowEnd := t.Date.AddDate(0, 0, 30)
			for j := 0; j < len(st); j++ {
				if i == j || st[j].Side != "buy" {
					continue
				}
				if (st[j].Date.Equal(lossWindowStart) || st[j].Date.After(lossWindowStart)) &&
					(st[j].Date.Equal(lossWindowEnd) || st[j].Date.Before(lossWindowEnd)) {
					events = append(events, WashSaleEvent{
						Symbol:          symbol,
						LossDate:        t.Date.Format("2006-01-02"),
						LossAmount:      lossAmt,
						RepurchaseDate:  st[j].Date.Format("2006-01-02"),
						WindowDays:      w.WindowDays,
						DisallowedLoss:  lossAmt,
						AdjustedBasis:   st[j].Price + lossAmt/float64(st[j].Quantity),
					})
				}
			}
		}
	}
	return events
}

// CheckSymbol runs wash sale detection for a single symbol using the OMS.
func (w *WashSaleChecker) CheckSymbol(ctx context.Context, symbol string, oms *OMS) ([]WashSaleEvent, error) {
	allTrades := oms.GetTrades()
	if len(allTrades) == 0 {
		return nil, nil
	}
	var trades []TradeRecord
	for _, t := range allTrades {
		if t.Symbol != symbol {
			continue
		}
		side := "sell"
		if string(t.Side) == "buy" {
			side = "buy"
		}
		trades = append(trades, TradeRecord{
			Symbol:   t.Symbol,
			Date:     t.Timestamp,
			Quantity: int(t.Quantity),
			Price:    t.Price,
			Side:     side,
		})
	}
	if len(trades) == 0 {
		slog.Debug("wash_sale: no trades for symbol", "symbol", symbol)
		return nil, nil
	}
	return w.Check(trades), nil
}
