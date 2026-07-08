package data

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type ArchiveResult struct {
	ID              int64  `json:"id"`
	Source          string `json:"source"`
	Symbol          string `json:"symbol"`
	Interval        string `json:"interval"`
	DateFrom        string `json:"date_from"`
	DateTo          string `json:"date_to"`
	RowCount        int    `json:"row_count"`
	CompressedBytes int    `json:"compressed_bytes"`
}

var validArchiveSources = map[string]bool{
	"ohlcv_cache":      true,
	"minute_cache":     true,
	"backtest_results": true,
}

func sourceDateCol(source string) string {
	switch source {
	case "ohlcv_cache":
		return "ts"
	case "minute_cache":
		return "date"
	case "backtest_results":
		return "finished_at"
	default:
		return ""
	}
}

func ArchiveData(db *sql.DB, source, symbol, before string) (*ArchiveResult, error) {
	if !validArchiveSources[source] {
		return nil, fmt.Errorf("invalid archive source: %s", source)
	}

	dateCol := sourceDateCol(source)
	if dateCol == "" {
		return nil, fmt.Errorf("no date column mapping for source: %s", source)
	}

	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if before != "" {
		where += " AND " + dateCol + " < ?"
		args = append(args, before)
	}

	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY %s", source, where, dateCol)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", source, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var allRows []map[string]any
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
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(allRows) == 0 {
		return &ArchiveResult{Source: source, Symbol: symbol}, nil
	}

	raw, err := json.Marshal(allRows)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	hash := sha256.Sum256(raw)
	checksum := hex.EncodeToString(hash[:])

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip init: %w", err)
	}
	if _, err := gz.Write(raw); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	compressed := buf.Bytes()

	dateFrom := before
	dateTo := time.Now().Format("2006-01-02")
	if len(allRows) > 0 {
		if v, ok := allRows[0][dateCol]; ok {
			dateFrom = fmt.Sprintf("%v", v)
		}
		if v, ok := allRows[len(allRows)-1][dateCol]; ok {
			dateTo = fmt.Sprintf("%v", v)
		}
	}

	interval := ""
	if v, ok := allRows[0]["interval"]; ok {
		interval = fmt.Sprintf("%v", v)
	}

	result := &ArchiveResult{
		Source:          source,
		Symbol:          symbol,
		Interval:        interval,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		RowCount:        len(allRows),
		CompressedBytes: len(compressed),
	}

	err = db.QueryRow(
		`INSERT INTO data_archive (source, symbol, interval, date_from, date_to, row_count, data, checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		source, symbol, interval, dateFrom, dateTo, len(allRows), compressed, checksum,
	).Scan(&result.ID)
	if err != nil {
		return nil, fmt.Errorf("insert archive: %w", err)
	}

	return result, nil
}

func UnarchiveData(db *sql.DB, archiveID int64) (int64, error) {
	var source, checksum string
	var compressed []byte
	err := db.QueryRow(
		"SELECT source, data, checksum FROM data_archive WHERE id = ?", archiveID,
	).Scan(&source, &compressed, &checksum)
	if err != nil {
		return 0, fmt.Errorf("find archive %d: %w", archiveID, err)
	}

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return 0, fmt.Errorf("gzip reader: %w", err)
	}
	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(gz); err != nil {
		return 0, fmt.Errorf("gzip read: %w", err)
	}
	gz.Close()

	hash := sha256.Sum256(decompressed.Bytes())
	if hex.EncodeToString(hash[:]) != checksum {
		return 0, fmt.Errorf("checksum mismatch: archive may be corrupted")
	}

	var rows []map[string]any
	if err := json.Unmarshal(decompressed.Bytes(), &rows); err != nil {
		return 0, fmt.Errorf("unmarshal: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	cols := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		if k == "id" && source != "backtest_results" {
			continue
		}
		cols = append(cols, k)
	}

	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}

	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", source, colList, placeholders)
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for _, row := range rows {
		vals := make([]any, 0, len(cols))
		for _, c := range cols {
			vals = append(vals, row[c])
		}
		res, err := stmt.Exec(vals...)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		inserted += n
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
