package research

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/market/adapters"
)

// PeerComparisonService provides peer company comparison analysis.
// When conceptAdapter is set, fetches real industry/sector data; otherwise returns mock.
type PeerComparisonService struct {
	conceptAdapter *adapters.EastMoneyConceptAdapter
	signalsAdapter *adapters.EastMoneySignalsAdapter
}

// NewPeerComparisonService creates a new PeerComparisonService.
// Adapters may be nil for mock mode.
func NewPeerComparisonService(concept *adapters.EastMoneyConceptAdapter, signals *adapters.EastMoneySignalsAdapter) *PeerComparisonService {
	return &PeerComparisonService{conceptAdapter: concept, signalsAdapter: signals}
}

// GetPeers returns peer comparison data for a symbol.
// Uses concept blocks to identify peer stocks in the same industry sectors.
// Deduplicates lead stocks from overlapping concept blocks.
func (s *PeerComparisonService) GetPeers(ctx context.Context, symbol string) ([]PeerComparisonData, error) {
	if s.conceptAdapter == nil {
		return nil, nil
	}

	blocks, err := s.conceptAdapter.FetchConceptBlocks(ctx, symbol)
	if err != nil {
		slog.Warn("peer comparison: concept blocks failed", "symbol", symbol, "error", err)
		return nil, nil
	}

	// Collect unique lead stocks from all blocks, deduplicating by stock code.
	// A single stock often leads multiple concept/industry blocks.
	seen := make(map[string]bool)
	peers := make([]PeerComparisonData, 0)
	for _, b := range blocks {
		if b.LeadStockCode == "" || b.LeadStock == "" {
			continue
		}
		if seen[b.LeadStockCode] {
			continue
		}
		seen[b.LeadStockCode] = true

		peers = append(peers, PeerComparisonData{
			Symbol:        b.LeadStockCode,
			Name:          b.LeadStock,
			MarketCap:     0, // Block-level data, per-stock MarketCap requires additional API calls.
			PE:            0, // Future: batch StockInfo calls for unique lead stocks.
			RevenueGrowth: 0, // Revenue growth not available from concept blocks.
			NetMargin:     0,
			ROE:           0,
		})
	}

	slog.Debug("peer comparison: built peers from concept blocks",
		"symbol", symbol, "blocks", len(blocks), "unique_peers", len(peers))
	return peers, nil
}

// GetIndustryRanks returns industry ranking data.
func (s *PeerComparisonService) GetIndustryRanks(ctx context.Context, topN int) ([]adapters.IndustryRank, error) {
	if s.signalsAdapter == nil {
		return nil, fmt.Errorf("peer comparison: signals adapter not configured")
	}
	return s.signalsAdapter.FetchIndustryRanks(ctx, topN)
}
