package storage

import (
	"database/sql"
	"quantflow/internal/trading"
)

// SaveReconciliationReport persists a reconciliation report.
func SaveReconciliationReport(db *sql.DB, report *trading.ReconciliationReport) error {
	data, err := trading.EncodeReconciliationReport(report)
	if err != nil {
		return err
	}
	query := `INSERT INTO reconciliation_reports (created_at, report_json, match_count, diff_count, dirt, broker_name)
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, report.CreatedAt.Unix(), string(data),
		report.MatchCount, report.DiffCount, report.Dirt, report.BrokerName)
	return err
}

// ListReconciliationReports returns recent reconciliation reports.
func ListReconciliationReports(db *sql.DB, limit int) ([]*trading.ReconciliationReport, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `SELECT report_json FROM reconciliation_reports ORDER BY created_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*trading.ReconciliationReport
	for rows.Next() {
		var reportJSON string
		if err := rows.Scan(&reportJSON); err != nil {
			return nil, err
		}
		report, err := trading.DecodeReconciliationReport([]byte(reportJSON))
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}
