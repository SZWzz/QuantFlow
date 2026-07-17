package main

import (
	"context"
	"fmt"

	"quantflow/internal/market/adapters"
)

// GetTop10Holders returns top-10 shareholders for a symbol.
func (a *App) GetTop10Holders(ctx context.Context, symbol string) ([]adapters.ShareholderRecord, error) {
	if a.shareholderAdpt == nil {
		return nil, fmt.Errorf("shareholder adapter not initialized")
	}
	return a.shareholderAdpt.FetchTop10Holders(ctx, symbol)
}
