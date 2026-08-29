package main

import (
	"fmt"
	"quantflow/internal/market"
)

// GetMacroSnapshot returns the latest macro snapshot for a country.
func (a *App) GetMacroSnapshot(country string) (*market.MacroSnapshot, error) {
	if a.macroSvc == nil {
		return nil, fmt.Errorf("macro service not initialized")
	}
	return a.macroSvc.GetMacroSnapshot(country)
}
