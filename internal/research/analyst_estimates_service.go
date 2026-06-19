package research

import (
	"context"
	"log/slog"
	"strings"

	"quantflow/internal/market/adapters"
)

// AnalystEstimatesService provides analyst rating and consensus data.
// When reportAdapter or consensusAdapter is set, fetches real data from EastMoney/THS;
// otherwise returns mock data.
type AnalystEstimatesService struct {
	reportAdapter    *adapters.EastMoneyReportAdapter
	consensusAdapter *adapters.THSConsensusAdapter
}

// NewAnalystEstimatesService creates a new AnalystEstimatesService.
// Both adapters may be nil for mock mode.
func NewAnalystEstimatesService(reportAdapter *adapters.EastMoneyReportAdapter, consensusAdapter *adapters.THSConsensusAdapter) *AnalystEstimatesService {
	return &AnalystEstimatesService{
		reportAdapter:    reportAdapter,
		consensusAdapter: consensusAdapter,
	}
}

// GetEstimates returns analyst estimates for a symbol.
// When report adapter is available, fetches real research reports and converts
// them to AnalystEstimate records. Falls back to mock on error.
func (s *AnalystEstimatesService) GetEstimates(ctx context.Context, symbol string) ([]AnalystEstimate, error) {
	if s.reportAdapter != nil {
		reports, err := s.reportAdapter.FetchReports(ctx, symbol, 1)
		if err != nil {
			slog.Warn("analyst_estimates: report fetch failed, using mock", "symbol", symbol, "error", err)
			return s.mockEstimates(symbol), nil
		}
		if len(reports) > 0 {
			estimates := make([]AnalystEstimate, 0, len(reports))
			for _, r := range reports {
				estimates = append(estimates, AnalystEstimate{
					Analyst:    r.OrgName,
					Firm:       r.OrgName,
					Rating:     normalizeRating(r.Rating),
					TargetLow:  r.PredictNextYearEPS,
					TargetHigh: r.PredictThisYearEPS,
					Date:       r.PublishDate,
				})
			}
			slog.Debug("analyst_estimates: fetched from reports", "symbol", symbol, "count", len(estimates))
			return estimates, nil
		}
	}
	slog.Debug("analyst_estimates: no adapter, using mock", "symbol", symbol)
	return s.mockEstimates(symbol), nil
}

// GetConsensusEPS returns analyst consensus EPS forecast.
// Tries THSConsensusAdapter first, then falls back to EastMoneyReportAdapter average.
func (s *AnalystEstimatesService) GetConsensusEPS(ctx context.Context, symbol string) (*ConsensusData, error) {
	// Try THS consensus first (dedicated consensus data)
	if s.consensusAdapter != nil {
		entries, err := s.consensusAdapter.FetchConsensus(ctx, symbol)
		if err != nil {
			slog.Warn("analyst_estimates: consensus fetch failed", "symbol", symbol, "error", err)
		} else if len(entries) > 0 {
			// Return the most recent year's consensus
			latest := entries[len(entries)-1]
			return &ConsensusData{
				Symbol:       symbol,
				ThisYearEPS:  latest.AvgEPS,
				AnalystCount: latest.AnalystCount,
				Source:       "ths_consensus",
			}, nil
		}
	}

	// Fallback: average EPS from EastMoney reports
	if s.reportAdapter != nil {
		thisYear, nextYear, count, err := s.reportAdapter.FetchConsensusEPS(ctx, symbol)
		if err != nil {
			slog.Warn("analyst_estimates: report consensus failed", "symbol", symbol, "error", err)
		} else if count > 0 {
			return &ConsensusData{
				Symbol:       symbol,
				ThisYearEPS:  thisYear,
				NextYearEPS:  nextYear,
				AnalystCount: count,
				Source:       "eastmoney_report",
			}, nil
		}
	}

	return nil, nil // no data available
}

// ConsensusData holds analyst consensus EPS forecast data.
type ConsensusData struct {
	Symbol       string  `json:"symbol"`
	ThisYearEPS  float64 `json:"this_year_eps"`
	NextYearEPS  float64 `json:"next_year_eps"`
	AnalystCount int     `json:"analyst_count"`
	Source       string  `json:"source"`
}

// mockEstimates returns mock analyst estimates for development/testing.
func (s *AnalystEstimatesService) mockEstimates(symbol string) []AnalystEstimate {
	return []AnalystEstimate{
		{Analyst: "John Smith", Firm: "Goldman Sachs", Rating: "buy", TargetLow: 180.0, TargetHigh: 220.0, Date: "2026-06-15"},
		{Analyst: "Jane Doe", Firm: "Morgan Stanley", Rating: "hold", TargetLow: 175.0, TargetHigh: 210.0, Date: "2026-06-14"},
		{Analyst: "Bob Lee", Firm: "JP Morgan", Rating: "buy", TargetLow: 190.0, TargetHigh: 230.0, Date: "2026-06-13"},
		{Analyst: "Alice Wang", Firm: "Citigroup", Rating: "sell", TargetLow: 150.0, TargetHigh: 170.0, Date: "2026-06-12"},
		{Analyst: "Tom Chen", Firm: "UBS", Rating: "strong_buy", TargetLow: 200.0, TargetHigh: 250.0, Date: "2026-06-11"},
	}
}

// normalizeRating converts Chinese or abbreviated ratings to standard labels.
func normalizeRating(rating string) string {
	switch rating {
	case "买入":
		return "buy"
	case "增持", "推荐", "强烈推荐", "强推":
		return "buy"
	case "中性", "谨慎推荐":
		return "hold"
	case "减持", "卖出":
		return "sell"
	default:
		return strings.ToLower(rating)
	}
}
