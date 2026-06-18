package research

import "context"

// AnalystEstimatesService provides analyst rating data.
type AnalystEstimatesService struct{}

// NewAnalystEstimatesService creates a new AnalystEstimatesService.
func NewAnalystEstimatesService() *AnalystEstimatesService {
	return &AnalystEstimatesService{}
}

// GetEstimates returns mock analyst estimates for a symbol.
func (s *AnalystEstimatesService) GetEstimates(ctx context.Context, symbol string) ([]AnalystEstimate, error) {
	return []AnalystEstimate{
		{Analyst: "John Smith", Firm: "Goldman Sachs", Rating: "buy", TargetLow: 180.0, TargetHigh: 220.0, Date: "2026-06-15"},
		{Analyst: "Jane Doe", Firm: "Morgan Stanley", Rating: "hold", TargetLow: 175.0, TargetHigh: 210.0, Date: "2026-06-14"},
		{Analyst: "Bob Lee", Firm: "JP Morgan", Rating: "buy", TargetLow: 190.0, TargetHigh: 230.0, Date: "2026-06-13"},
		{Analyst: "Alice Wang", Firm: "Citigroup", Rating: "sell", TargetLow: 150.0, TargetHigh: 170.0, Date: "2026-06-12"},
		{Analyst: "Tom Chen", Firm: "UBS", Rating: "strong_buy", TargetLow: 200.0, TargetHigh: 250.0, Date: "2026-06-11"},
	}, nil
}
