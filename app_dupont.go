package main

import (
	"context"
	"fmt"
	"quantflow/internal/research"
	"strings"
)

// GetDupontAnalysis returns Dupont decomposition for a symbol.
func (a *App) GetDupontAnalysis(ctx context.Context, symbol string) (*research.DupontBreakdown, error) {
	if a.finSvc == nil {
		return nil, fmt.Errorf("financials service not initialized")
	}

	fd, err := a.finSvc.GetFinancials(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("get financials: %w", err)
	}

	return research.ComputeDupont(fd), nil
}

// GetPeerRadar returns peer comparison radar data.
func (a *App) GetPeerRadar(ctx context.Context, symbol string) ([]research.PeerRadar, error) {
	peers, err := a.peerSvc.GetPeers(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	peerSymbols := make([]string, 0, len(peers))
	for _, p := range peers {
		peerSymbols = append(peerSymbols, p.Symbol)
	}

	// Filter out invalid peer symbols (BSE, partial codes, etc.)
	validPeers := make([]string, 0, len(peerSymbols))
	for _, s := range peerSymbols {
		if len(s) >= 5 && (strings.Contains(s, ".SH") || strings.Contains(s, ".SZ") || strings.Contains(s, ".HK")) {
			validPeers = append(validPeers, s)
		}
	}
	// Limit to 5 peers to avoid excessive API calls
	if len(validPeers) > 5 {
		validPeers = validPeers[:5]
	}

	getFD := func(sym string) *research.FinancialData {
		fd, err := a.finSvc.GetFinancials(ctx, sym)
		if err != nil {
			return nil
		}
		return fd
	}

	return research.ComputePeerRadar(symbol, validPeers, getFD), nil
}
