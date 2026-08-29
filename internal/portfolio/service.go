package portfolio

import (
	"quantflow/internal/market"
	"quantflow/internal/trading"
	"sort"
)

// Service computes portfolio summaries, position details, and allocations
// from the live OMS state. It optionally persists daily snapshots via Repo.
type Service struct {
	oms  *trading.OMS
	repo *Repo
}

// NewService creates a portfolio service backed by the given OMS.
func NewService(oms *trading.OMS) *Service {
	return &Service{oms: oms}
}

// SetRepo attaches an optional persistence layer for daily snapshots.
func (s *Service) SetRepo(repo *Repo) {
	s.repo = repo
}

// GetSummary calculates the portfolio summary from current positions and cash ledger.
func (s *Service) GetSummary() *Summary {
	cashBalance := s.oms.GetCashBalance()
	positions := s.oms.GetAllPositions()
	var mv, pnl float64
	for _, p := range positions {
		mv += p.MarketPrice * p.Quantity
		pnl += p.PnL
	}
	cost := mv - pnl
	pnlPct := 0.0
	if cost > 0 {
		pnlPct = (pnl / cost) * 100
	}
	return &Summary{
		TotalValue:  cashBalance + mv,
		CashBalance: cashBalance,
		MarketValue: mv,
		TotalPnL:    pnl,
		TotalPnLPct: pnlPct,
	}
}

// GetPositions returns detailed position analytics, sorted by allocation descending.
func (s *Service) GetPositions() []*PositionDetail {
	positions := s.oms.GetAllPositions()
	tv := 0.0
	for _, p := range positions {
		tv += p.MarketPrice * p.Quantity
	}

	details := make([]*PositionDetail, 0, len(positions))
	for _, p := range positions {
		if p.Quantity == 0 {
			continue
		}
		mv := p.MarketPrice * p.Quantity
		ap := 0.0
		if tv > 0 {
			ap = (mv / tv) * 100
		}
		details = append(details, &PositionDetail{
			Symbol:      p.Symbol,
			Quantity:    p.Quantity,
			AvgPrice:    p.AvgPrice,
			MarketPrice: p.MarketPrice,
			PnL:         p.PnL,
			PnLPct:      p.PnLPct,
			Market:      detectMarket(p.Symbol),
			Currency:    detectCurrency(p.Symbol),
			CostBasis:   p.AvgPrice * p.Quantity,
			AllocPct:    ap,
		})
	}

	sort.Slice(details, func(i, j int) bool {
		return details[i].AllocPct > details[j].AllocPct
	})
	return details
}

// GetAllocation computes allocation breakdowns from current positions.
func (s *Service) GetAllocation() *Allocation {
	positions := s.GetPositions()
	alloc := &Allocation{
		ByMarket:   make(map[string]float64),
		BySector:   make(map[string]float64),
		ByCurrency: make(map[string]float64),
	}
	for _, p := range positions {
		alloc.ByMarket[p.Market] += p.AllocPct
		alloc.ByCurrency[p.Currency] += p.AllocPct
	}
	return alloc
}

// RecordDailySnapshot persists today's portfolio state.
func (s *Service) RecordDailySnapshot() error {
	if s.repo == nil {
		return nil
	}
	return s.repo.RecordDailySnapshot(s.GetSummary())
}

// GetPnLHistory returns daily P&L history from the repository.
func (s *Service) GetPnLHistory(days int) ([]*DailyPnL, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.GetPnLHistory(days)
}

// detectMarket infers the market from a symbol's suffix convention.
// Delegates to market.MarketForSymbol for consistent classification across the app.
func detectMarket(symbol string) string {
	return market.MarketForSymbol(symbol)
}

// detectCurrency infers the currency from the detected market.
func detectCurrency(symbol string) string {
	switch detectMarket(symbol) {
	case "CN":
		return "CNY"
	case "HK":
		return "HKD"
	case "CRYPTO":
		return "USDT"
	default:
		return "USD"
	}
}
