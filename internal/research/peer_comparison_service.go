package research

import "context"

// PeerComparisonService provides peer company comparison analysis.
type PeerComparisonService struct{}

// NewPeerComparisonService creates a new PeerComparisonService.
func NewPeerComparisonService() *PeerComparisonService {
	return &PeerComparisonService{}
}

// GetPeers returns mock peer comparison data for a symbol.
func (s *PeerComparisonService) GetPeers(ctx context.Context, symbol string) ([]PeerComparisonData, error) {
	return []PeerComparisonData{
		{Symbol: symbol, Name: symbol, MarketCap: 2.5e12, PE: 28.5, RevenueGrowth: 0.12, NetMargin: 0.25, ROE: 0.35},
		{Symbol: "MSFT", Name: "Microsoft", MarketCap: 3.0e12, PE: 35.0, RevenueGrowth: 0.15, NetMargin: 0.35, ROE: 0.42},
		{Symbol: "GOOGL", Name: "Alphabet", MarketCap: 1.8e12, PE: 25.0, RevenueGrowth: 0.10, NetMargin: 0.28, ROE: 0.30},
		{Symbol: "AMZN", Name: "Amazon", MarketCap: 1.9e12, PE: 40.0, RevenueGrowth: 0.11, NetMargin: 0.08, ROE: 0.22},
	}, nil
}
