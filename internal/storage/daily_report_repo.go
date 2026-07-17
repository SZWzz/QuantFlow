package storage

import (
	"database/sql"
	"fmt"
	"time"

	"quantflow/internal/trading"
)

// SaveDailyReport persists a daily report to the database.
func SaveDailyReport(db *sql.DB, report *trading.DailyReport) error {
	data, err := trading.EncodeDailyReport(report)
	if err != nil {
		return err
	}
	query := `INSERT OR REPLACE INTO daily_reports (date, created_at, report_json, summary, pnl, pnl_percent)
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, report.Date, time.Now().Unix(), string(data),
		reportSummary(report), report.DayPNL, report.DayPNLPercent)
	return err
}

// GetDailyReport loads a daily report by date from the database.
func GetDailyReport(db *sql.DB, date string) (*trading.DailyReport, error) {
	var reportJSON string
	query := `SELECT report_json FROM daily_reports WHERE date = ?`
	err := db.QueryRow(query, date).Scan(&reportJSON)
	if err != nil {
		return nil, err
	}
	return trading.DecodeDailyReport([]byte(reportJSON))
}

// ListDailyReports returns the most recent daily reports up to the given limit.
func ListDailyReports(db *sql.DB, limit int) ([]*trading.DailyReport, error) {
	if limit <= 0 {
		limit = 30
	}
	query := `SELECT report_json FROM daily_reports ORDER BY date DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*trading.DailyReport
	for rows.Next() {
		var reportJSON string
		if err := rows.Scan(&reportJSON); err != nil {
			return nil, err
		}
		report, err := trading.DecodeDailyReport([]byte(reportJSON))
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func reportSummary(report *trading.DailyReport) string {
	s := "今日盈亏: "
	if report.DayPNL >= 0 {
		s += "+"
	}
	s += formatMoney(report.DayPNL)
	if report.DayPNLPercent != 0 {
		s += " ("
		if report.DayPNLPercent > 0 {
			s += "+"
		}
		s += formatPct(report.DayPNLPercent) + ")"
	}
	s += fmt.Sprintf(" | 交易: %d 笔", report.Trades)
	return s
}

func formatMoney(v float64) string {
	if v < 0 {
		return fmt.Sprintf("-¥%.2f", -v)
	}
	return fmt.Sprintf("¥%.2f", v)
}

func formatPct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}
