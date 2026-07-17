package main

import (
	"context"
	"fmt"

	"quantflow/internal/research"
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

	getFD := func(sym string) *research.FinancialData {
		fd, err := a.finSvc.GetFinancials(ctx, sym)
		if err != nil {
			return nil
		}
		return fd
	}

	return research.ComputePeerRadar(symbol, peerSymbols, getFD), nil
}
