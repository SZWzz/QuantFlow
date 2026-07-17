package main

import (
	"fmt"

	"quantflow/internal/storage"
	"quantflow/internal/trading"
)

// ReconcileAll runs position reconciliation against all connected brokers.
func (a *App) ReconcileAll() ([]*trading.ReconciliationReport, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}

	reports := trading.ReconcileAll(a.oms, a.brokers)

	// Persist reports
	if a.db != nil {
		for _, report := range reports {
			if err := storage.SaveReconciliationReport(a.db, report); err != nil {
				// Log but don't fail the whole reconciliation
				continue
			}
		}
	}

	return reports, nil
}

// GetReconciliationReports returns recent reconciliation reports.
func (a *App) GetReconciliationReports(limit int) ([]*trading.ReconciliationReport, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return storage.ListReconciliationReports(a.db, limit)
}
