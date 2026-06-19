package research

import "context"

// InsiderTradingService monitors insider transactions.
type InsiderTradingService struct{}

// NewInsiderTradingService creates a new InsiderTradingService.
func NewInsiderTradingService() *InsiderTradingService {
	return &InsiderTradingService{}
}

// GetInsiderTrades returns mock insider transactions for a symbol.
func (s *InsiderTradingService) GetInsiderTrades(ctx context.Context, symbol string) ([]InsiderTransaction, error) {
	return []InsiderTransaction{
		{Name: "Tim Cook", Role: "CEO", Type: "sell", Shares: 50000, Price: 195.0, Value: 9_750_000, Date: "2026-06-10"},
		{Name: "CFO", Role: "CFO", Type: "sell", Shares: 10000, Price: 192.0, Value: 1_920_000, Date: "2026-06-08"},
		{Name: "VP Engineering", Role: "VP", Type: "buy", Shares: 5000, Price: 188.0, Value: 940_000, Date: "2026-06-05"},
	}, nil
}
