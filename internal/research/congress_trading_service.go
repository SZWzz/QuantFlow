package research

import "context"

// CongressTradingService monitors US Congress trading activity.
type CongressTradingService struct{}

// NewCongressTradingService creates a new CongressTradingService.
func NewCongressTradingService() *CongressTradingService {
	return &CongressTradingService{}
}

// GetCongressTrades returns mock congress trading records.
func (s *CongressTradingService) GetCongressTrades(ctx context.Context) ([]CongressTrade, error) {
	return []CongressTrade{
		{Name: "Nancy Pelosi", Chamber: "House", Party: "Democrat", Symbol: "AAPL", Type: "buy", Amount: "$1M-$5M", Date: "2026-05-20"},
		{Name: "Dan Crenshaw", Chamber: "House", Party: "Republican", Symbol: "XOM", Type: "buy", Amount: "$100K-$250K", Date: "2026-05-15"},
		{Name: "Tommy Tuberville", Chamber: "Senate", Party: "Republican", Symbol: "MSFT", Type: "sell", Amount: "$50K-$100K", Date: "2026-05-10"},
	}, nil
}
