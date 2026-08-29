package research

import (
	"context"
	"log/slog"
	"quantflow/internal/market/adapters"
)

// NorthboundService provides 沪深股通 (northbound) capital flow data.
// Uses THS API for minute-level data with local CSV self-caching for daily history.
// Degrades gracefully when the adapter is nil or API calls fail.
type NorthboundService struct {
	adapter *adapters.THSNorthboundAdapter
}

// NewNorthboundService creates a new NorthboundService. adapter may be nil for mock mode.
func NewNorthboundService(adapter *adapters.THSNorthboundAdapter) *NorthboundService {
	return &NorthboundService{adapter: adapter}
}

// GetMinuteFlow returns today's real-time per-minute northbound capital flow.
func (s *NorthboundService) GetMinuteFlow(ctx context.Context) ([]adapters.NorthboundMinute, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.FetchMinuteFlow(ctx)
	if err != nil {
		slog.Warn("northbound: minute flow fetch failed", "error", err)
		return nil, nil
	}
	return data, nil
}

// GetHistory returns cached daily northbound snapshots (most recent N days).
func (s *NorthboundService) GetHistory(n int) ([]adapters.NorthboundSnapshot, error) {
	if s.adapter == nil {
		return nil, nil
	}
	data, err := s.adapter.GetHistory(n)
	if err != nil {
		slog.Warn("northbound: history fetch failed", "error", err)
		return nil, nil
	}
	return data, nil
}
