package research

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
)

// PeerComparisonService provides peer company comparison analysis.
// Uses concept blocks to identify peer stocks, then fills financial metrics
// from EastMoney stock info.
type PeerComparisonService struct {
	conceptAdapter   *adapters.EastMoneyConceptAdapter
	signalsAdapter   *adapters.EastMoneySignalsAdapter
	eastmoneyAdapter *adapters.EastMoneyAdapter
	quoteReg         *market.AdapterRegistry // fallback for peer market caps when Eastmoney is down
}

// NewPeerComparisonService creates a new PeerComparisonService.
// Adapters may be nil (results will have zero-filled metrics).
func NewPeerComparisonService(
	concept *adapters.EastMoneyConceptAdapter,
	signals *adapters.EastMoneySignalsAdapter,
	eastmoney *adapters.EastMoneyAdapter,
	quoteReg *market.AdapterRegistry,
) *PeerComparisonService {
	return &PeerComparisonService{
		conceptAdapter:   concept,
		signalsAdapter:   signals,
		eastmoneyAdapter: eastmoney,
		quoteReg:         quoteReg,
	}
}

// GetPeers returns peer comparison data for a symbol.
// Uses concept blocks to identify peer stocks in the same industry sectors,
// then fetches MarketCap from EastMoney for each peer.
func (s *PeerComparisonService) GetPeers(ctx context.Context, symbol string) ([]PeerComparisonData, error) {
	if s.conceptAdapter == nil {
		return nil, nil
	}

	blocks, err := s.conceptAdapter.FetchConceptBlocks(ctx, symbol)
	if err != nil {
		slog.Warn("peer_comparison: concept blocks failed", "symbol", symbol, "error", err)
		return nil, nil
	}

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

		p := PeerComparisonData{
			Symbol: b.LeadStockCode,
			Name:   b.LeadStock,
		}

		// Fill MarketCap from EastMoney stock info if adapter available.
		// Falls back to quote registry (Tencent/Sina/etc.) when Eastmoney is down.
		if s.eastmoneyAdapter != nil {
			if info, err := s.eastmoneyAdapter.FetchStockInfo(ctx, b.LeadStockCode); err == nil && info != nil {
				p.MarketCap = info.MarketCap
			} else if s.quoteReg != nil {
				if quote, _, qErr := s.quoteReg.FetchQuoteWithFallback(ctx, "CN", b.LeadStockCode); qErr == nil && quote != nil {
					p.MarketCap = quote.MarketCap
				}
			}
		} else if s.quoteReg != nil {
			if quote, _, qErr := s.quoteReg.FetchQuoteWithFallback(ctx, "CN", b.LeadStockCode); qErr == nil && quote != nil {
				p.MarketCap = quote.MarketCap
			}
		}

		peers = append(peers, p)
	}

	slog.Debug("peer_comparison: built peers",
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
