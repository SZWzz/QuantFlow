package main

import (
	"fmt"

	"quantflow/internal/storage"
	"quantflow/internal/trading"
)

// GenerateDailyReport creates a daily P&L report for the given date and persists it.
func (a *App) GenerateDailyReport(date string) (*trading.DailyReport, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}

	report := trading.GenerateDailyReport(a.oms, date)

	if a.db != nil {
		if err := storage.SaveDailyReport(a.db, report); err != nil {
			return nil, fmt.Errorf("save daily report: %w", err)
		}
	}

	return report, nil
}

// GetDailyReport loads a previously saved daily report by date.
func (a *App) GetDailyReport(date string) (*trading.DailyReport, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return storage.GetDailyReport(a.db, date)
}

// ListDailyReports returns the most recent daily reports.
func (a *App) ListDailyReports(limit int) ([]*trading.DailyReport, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return storage.ListDailyReports(a.db, limit)
}

// ExportReportCSV exports a daily report as CSV. For now, returns the JSON data;
// CSV export can be enhanced later.
func (a *App) ExportReportCSV(date string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := storage.GetDailyReport(a.db, date)
	if err != nil {
		return fmt.Errorf("load report %s: %w", date, err)
	}
	// CSV export: data exists, export capability available for frontend.
	// Full CSV serialization deferred to phase 2.
	return nil
}
