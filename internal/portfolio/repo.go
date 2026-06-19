package portfolio

import (
	"database/sql"
	"time"
)

// Repo persists daily P&L snapshots to SQLite.
type Repo struct {
	db *sql.DB
}

// NewRepo creates a new portfolio repository backed by the given database.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// RecordDailySnapshot inserts or replaces today's portfolio snapshot.
func (r *Repo) RecordDailySnapshot(s *Summary) error {
	today := time.Now().Format("2006-01-02")
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO daily_pnl (date, total_value, cash, market_value, pnl, pnl_pct) VALUES (?, ?, ?, ?, ?, ?)`,
		today, s.TotalValue, s.CashBalance, s.MarketValue, s.TotalPnL, s.TotalPnLPct,
	)
	return err
}

// GetPnLHistory returns the most recent `days` daily P&L records.
func (r *Repo) GetPnLHistory(days int) ([]*DailyPnL, error) {
	rows, err := r.db.Query(
		"SELECT date, total_value, cash, market_value, pnl, pnl_pct FROM daily_pnl ORDER BY date DESC LIMIT ?", days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*DailyPnL
	for rows.Next() {
		d := &DailyPnL{}
		if err := rows.Scan(&d.Date, &d.TotalValue, &d.Cash, &d.MarketValue, &d.PnL, &d.PnLPct); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
