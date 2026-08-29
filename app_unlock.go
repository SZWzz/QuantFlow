package main

import (
	"context"
	"fmt"
	"quantflow/internal/market/adapters"
)

// GetUnlockCalendar returns upcoming unlock events.
func (a *App) GetUnlockCalendar(ctx context.Context, days int) ([]adapters.UnlockEvent, error) {
	if a.unlockAdpt == nil {
		return nil, fmt.Errorf("unlock adapter not initialized")
	}
	return a.unlockAdpt.FetchUpcoming(ctx, days)
}
