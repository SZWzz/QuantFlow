package research

import (
	"context"
	"log/slog"
	"quantflow/internal/market/adapters"
)

// CapitalService provides capital-side market data: margin trading, block trades,
// shareholder changes, and dividend history. Degrades gracefully to empty slices
// when the adapter is nil or API calls fail.
type CapitalService struct {
	adapter *adapters.EastMoneyCapitalAdapter
}

// NewCapitalService creates a new CapitalService. adapter may be nil for mock mode.
func NewCapitalService(adapter *adapters.EastMoneyCapitalAdapter) *CapitalService {
	return &CapitalService{adapter: adapter}
}

// GetMarginTrading returns recent margin trading records for a stock.
func (s *CapitalService) GetMarginTrading(ctx context.Context, symbol string, pageSize int) ([]adapters.MarginTrading, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchMarginTrading(ctx, symbol, pageSize)
	if err != nil {
		slog.Warn("capital: margin trading fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}

// GetBlockTrades returns recent block trade records for a stock.
func (s *CapitalService) GetBlockTrades(ctx context.Context, symbol string, pageSize int) ([]adapters.BlockTrade, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchBlockTrades(ctx, symbol, pageSize)
	if err != nil {
		slog.Warn("capital: block trades fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}

// GetHolderChanges returns historical shareholder count changes for a stock.
func (s *CapitalService) GetHolderChanges(ctx context.Context, symbol string, pageSize int) ([]adapters.HolderChange, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchHolderChanges(ctx, symbol, pageSize)
	if err != nil {
		slog.Warn("capital: holder changes fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}

// GetDividendHistory returns historical dividend records for a stock.
func (s *CapitalService) GetDividendHistory(ctx context.Context, symbol string, pageSize int) ([]adapters.DividendRecord, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchDividendHistory(ctx, symbol, pageSize)
	if err != nil {
		slog.Warn("capital: dividend history fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}
