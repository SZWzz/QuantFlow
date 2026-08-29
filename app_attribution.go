package main

import (
	"fmt"
	"quantflow/internal/portfolio"
)

// GetFactorAttribution returns portfolio factor attribution.
func (a *App) GetFactorAttribution(totalReturn float64) (*portfolio.FactorAttribution, error) {
	if a.attrSvc == nil {
		return nil, fmt.Errorf("attribution service not initialized")
	}
	return a.attrSvc.ComputeAttribution(totalReturn), nil
}
