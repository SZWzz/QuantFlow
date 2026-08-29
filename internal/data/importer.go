package data

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"
)

var validImportTables = map[string]bool{
	"ohlcv_cache":  true,
	"minute_cache": true,
}

func ImportCSV(db *sql.DB, filePath, table string) (int64, error) {
	if !validImportTables[table] {
		return 0, fmt.Errorf("import to table %q is not allowed", table)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	var cols []string
	cols = append(cols, header...)

	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}
	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", table, colList, placeholders) //nolint:gosec // table 经白名单校验，值全走占位符

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	// Rollback after a successful Commit returns sql.ErrTxDone — safe to ignore
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var count int64
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("read row: %w", err)
		}

		vals := make([]any, len(record))
		for i, v := range record {
			vals[i] = v
		}

		res, err := stmt.Exec(vals...)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		count += n
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

func ImportParquet(db *sql.DB, filePath, table string) (int64, error) {
	if !validImportTables[table] {
		return 0, fmt.Errorf("import to table %q is not allowed", table)
	}

	data, err := parquet.ReadFile[ohlcvParquetRow](filePath)
	if err != nil {
		return 0, fmt.Errorf("parquet read: %w", err)
	}

	if len(data) == 0 {
		return 0, nil
	}

	now := time.Now().Unix()
	cols := []string{"symbol", "interval", "ts", "open", "high", "low", "close", "volume", "fetched_at"}
	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}
	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", table, colList, placeholders) //nolint:gosec // table 经白名单校验，值全走占位符

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	// Rollback after a successful Commit returns sql.ErrTxDone — safe to ignore
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var count int64
	for _, row := range data {
		vals := []any{row.Symbol, row.Interval, row.Ts, row.Open, row.High, row.Low, row.Close, row.Volume, now}
		res, err := stmt.Exec(vals...)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		count += n
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}
