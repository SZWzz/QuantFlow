package research

import (
	"context"
	"log/slog"

	"quantflow/internal/market/adapters"
)

// FundFlowService provides stock capital flow data (intraday minute + daily history).
// Degrades gracefully to empty slices when the adapter is nil or API calls fail.
type FundFlowService struct {
	adapter *adapters.EastMoneyFundFlowAdapter
}

// NewFundFlowService creates a new FundFlowService. adapter may be nil for mock mode.
func NewFundFlowService(adapter *adapters.EastMoneyFundFlowAdapter) *FundFlowService {
	return &FundFlowService{adapter: adapter}
}

// GetMinuteFlow returns today's intraday per-minute capital flow for a stock.
func (s *FundFlowService) GetMinuteFlow(ctx context.Context, symbol string) ([]adapters.FundFlowMinute, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchMinuteFlow(ctx, symbol)
	if err != nil {
		slog.Warn("fundflow: minute flow fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}

// GetDailyFlow returns the last 120 trading days of daily capital flow for a stock.
func (s *FundFlowService) GetDailyFlow(ctx context.Context, symbol string) ([]adapters.FundFlowDaily, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchDailyFlow(ctx, symbol)
	if err != nil {
		slog.Warn("fundflow: daily flow fetch failed", "symbol", symbol, "error", err)
		return nil, nil
	}
	return data, nil
}
