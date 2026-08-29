package data

import (
	"database/sql"
	"fmt"
	"time"
)

var allowedCleanupTables = map[string]bool{
	"ohlcv_cache":  true,
	"minute_cache": true,
}

type CleanupResult struct {
	AffectedRows int64            `json:"affected_rows"`
	Preview      []map[string]any `json:"preview"`
	Table        string           `json:"table"`
	DryRun       bool             `json:"dry_run"`
}

func cleanupDateCol(table string) string {
	switch table {
	case "ohlcv_cache":
		return "ts"
	case "minute_cache":
		return "date"
	default:
		return ""
	}
}

func dateToTimestamp(dateStr string) int64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func CleanupData(db *sql.DB, table, symbol, before string, dryRun bool) (*CleanupResult, error) {
	if !allowedCleanupTables[table] {
		return nil, fmt.Errorf("table %s is not allowed for cleanup", table)
	}

	dateCol := cleanupDateCol(table)
	if dateCol == "" {
		return nil, fmt.Errorf("no date column for table %s", table)
	}

	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if before != "" {
		if table == "ohlcv_cache" {
			where += " AND " + dateCol + " < ?"
			args = append(args, dateToTimestamp(before))
		} else {
			where += " AND " + dateCol + " < ?"
			args = append(args, before)
		}
	}

	countQ := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\" WHERE %s", table, where) //nolint:gosec // table 经 allowedCleanupTables 白名单校验，值全走 args 占位符
	var total int64
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count: %w", err)
	}

	previewQ := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s LIMIT 10", table, where) //nolint:gosec // table 经 allowedCleanupTables 白名单校验
	rows, err := db.Query(previewQ, args...)
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var preview []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		valPtrs := make([]any, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any)
		for i, col := range cols {
			row[col] = vals[i]
		}
		preview = append(preview, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &CleanupResult{
		AffectedRows: total,
		Preview:      preview,
		Table:        table,
		DryRun:       dryRun,
	}

	if !dryRun && total > 0 {
		deleteQ := fmt.Sprintf("DELETE FROM \"%s\" WHERE %s", table, where) //nolint:gosec // table 经 allowedCleanupTables 白名单校验
		res, err := db.Exec(deleteQ, args...)
		if err != nil {
			return nil, fmt.Errorf("delete: %w", err)
		}
		n, _ := res.RowsAffected()
		result.AffectedRows = n
	}

	return result, nil
}
