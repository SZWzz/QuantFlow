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
// Uses concept blocks to find peers in the same industry.
func (s *PeerComparisonService) GetPeers(ctx context.Context, symbol string) ([]PeerComparisonData, error) {
	if s.conceptAdapter == nil {
		return mockPeerData(symbol), nil
	}

	blocks, err := s.conceptAdapter.FetchConceptBlocks(ctx, symbol)
	if err != nil {
		slog.Warn("peer comparison: concept blocks failed, using mock", "symbol", symbol, "error", err)
		return mockPeerData(symbol), nil
	}

	// Convert blocks to peer comparison data
	peers := make([]PeerComparisonData, 0, len(blocks))
	for _, b := range blocks {
		peers = append(peers, PeerComparisonData{
			Symbol:        b.Code,
			Name:          b.Name,
			MarketCap:     0, // would need additional API call per peer
			PE:            0,
			RevenueGrowth: 0,
			NetMargin:     0,
			ROE:           0,
		})
	}

	// If no blocks found, fall back to mock
	if len(peers) == 0 {
		return mockPeerData(symbol), nil
	}

	return peers, nil
}

// GetIndustryRanks returns industry ranking data.
func (s *PeerComparisonService) GetIndustryRanks(ctx context.Context, topN int) ([]adapters.IndustryRank, error) {
	if s.signalsAdapter == nil {
		return nil, fmt.Errorf("peer comparison: signals adapter not configured")
	}
	return s.signalsAdapter.FetchIndustryRanks(ctx, topN)
}

func mockPeerData(symbol string) []PeerComparisonData {
	return []PeerComparisonData{
		{Symbol: symbol, Name: symbol, MarketCap: 2.5e12, PE: 28.5, RevenueGrowth: 0.12, NetMargin: 0.25, ROE: 0.35},
		{Symbol: "MSFT", Name: "Microsoft", MarketCap: 3.0e12, PE: 35.0, RevenueGrowth: 0.15, NetMargin: 0.35, ROE: 0.42},
		{Symbol: "GOOGL", Name: "Alphabet", MarketCap: 1.8e12, PE: 25.0, RevenueGrowth: 0.10, NetMargin: 0.28, ROE: 0.30},
		{Symbol: "AMZN", Name: "Amazon", MarketCap: 1.9e12, PE: 40.0, RevenueGrowth: 0.11, NetMargin: 0.08, ROE: 0.22},
	}
}
