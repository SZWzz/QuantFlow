package research

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/market/adapters"
)

// CongressTradingService monitors US Congress trading activity
// via the free telep.io Capitol Trades API.
type CongressTradingService struct {
	adapter *adapters.CongressTradesAdapter
}

// NewCongressTradingService creates a new CongressTradingService.
// adapter may be nil; if nil, GetCongressTrades returns empty results.
func NewCongressTradingService(adapter *adapters.CongressTradesAdapter) *CongressTradingService {
	return &CongressTradingService{adapter: adapter}
}

// GetCongressTrades returns recent congress trading records.
// Uses the telep.io API for real data; falls back to empty results.
func (s *CongressTradingService) GetCongressTrades(ctx context.Context) ([]CongressTrade, error) {
	if s.adapter == nil {
		slog.Warn("congress trades: adapter not configured")
		return nil, nil
	}

	items, err := s.adapter.FetchRecentTrades(ctx, 100)
	if err != nil {
		slog.Warn("congress trades: fetch failed", "error", err)
		return nil, fmt.Errorf("congress_trades: %w", err)
	}

	trades := make([]CongressTrade, 0, len(items))
	for _, item := range items {
		// Only include stock trades (skip crypto, ETFs, mutual funds, etc.)
		if item.AssetType != "Stock" {
			continue
		}
		// Normalize chamber
		chamber := "House"
		if item.Chamber == "senate" {
			chamber = "Senate"
		}
		trades = append(trades, CongressTrade{
			Name:    item.PoliticianName,
			Chamber: chamber,
			Party:   item.Party,
			Symbol:  item.Ticker,
			Type:    normalizeTransactionType(item.TransactionType),
			Amount:  normalizeAmount(item.AmountText),
			Date:    item.TransactionDate,
		})
	}

	slog.Info("congress trades: fetched real data",
		"total_from_api", len(items),
		"stock_trades", len(trades))
	return trades, nil
}

// normalizeTransactionType maps API transaction types to simple buy/sell.
func normalizeTransactionType(t string) string {
	switch t {
	case "purchase":
		return "buy"
	case "sale (full)", "sale (partial)":
		return "sell"
	case "exchange":
		return "exchange"
	default:
		return t
	}
}

// normalizeAmount trims long amount strings for display.
func normalizeAmount(a string) string {
	if len(a) <= 20 {
		return a
	}
	return a[:20]
}
