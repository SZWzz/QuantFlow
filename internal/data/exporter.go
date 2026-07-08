package data

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/segmentio/parquet-go"
)

type ohlcvParquetRow struct {
	Symbol   string  `parquet:"symbol"`
	Interval string  `parquet:"interval"`
	Ts       int64   `parquet:"ts"`
	Open     float64 `parquet:"open"`
	High     float64 `parquet:"high"`
	Low      float64 `parquet:"low"`
	Close    float64 `parquet:"close"`
	Volume   float64 `parquet:"volume"`
}

func exportWhere(table, symbol, interval, dateFrom, dateTo string) (string, []any) {
	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if interval != "" {
		where += " AND interval = ?"
		args = append(args, interval)
	}
	if dateFrom != "" {
		if table == "ohlcv_cache" {
			where += " AND ts >= ?"
			args = append(args, dateToTimestamp(dateFrom))
		} else {
			where += " AND date >= ?"
			args = append(args, dateFrom)
		}
	}
	if dateTo != "" {
		if table == "ohlcv_cache" {
			where += " AND ts <= ?"
			args = append(args, dateToTimestamp(dateTo))
		} else {
			where += " AND date <= ?"
			args = append(args, dateTo)
		}
	}
	return where, args
}

func ExportCSV(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error) {
	where, args := exportWhere(table, symbol, interval, dateFrom, dateTo)
	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY symbol, ts", table, where)
	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(cols); err != nil {
		return 0, fmt.Errorf("write header: %w", err)
	}

	var count int64
	vals := make([]any, len(cols))
	valPtrs := make([]any, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			return 0, err
		}
		record := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := w.Write(record); err != nil {
			return 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

func ExportParquet(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error) {
	where, args := exportWhere(table, symbol, interval, dateFrom, dateTo)
	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY symbol, ts", table, where)
	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	var symIdx, intervalIdx, tsIdx, oIdx, hIdx, lIdx, cIdx, vIdx int = -1, -1, -1, -1, -1, -1, -1, -1
	for i, c := range cols {
		switch c {
		case "symbol":
			symIdx = i
		case "interval":
			intervalIdx = i
		case "ts":
			tsIdx = i
		case "open":
			oIdx = i
		case "high":
			hIdx = i
		case "low":
			lIdx = i
		case "close":
			cIdx = i
		case "volume":
			vIdx = i
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	var parquetRows []ohlcvParquetRow
	vals := make([]any, len(cols))
	valPtrs := make([]any, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			return 0, err
		}
		pr := ohlcvParquetRow{}
		if symIdx >= 0 && vals[symIdx] != nil {
			pr.Symbol = fmt.Sprintf("%v", vals[symIdx])
		}
		if intervalIdx >= 0 && vals[intervalIdx] != nil {
			pr.Interval = fmt.Sprintf("%v", vals[intervalIdx])
		}
		if tsIdx >= 0 && vals[tsIdx] != nil {
			pr.Ts = toInt64(vals[tsIdx])
		}
		pr.Open = toFloat64(vals, oIdx)
		pr.High = toFloat64(vals, hIdx)
		pr.Low = toFloat64(vals, lIdx)
		pr.Close = toFloat64(vals, cIdx)
		pr.Volume = toFloat64(vals, vIdx)
		parquetRows = append(parquetRows, pr)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(parquetRows) == 0 {
		return 0, nil
	}

	if err := parquet.WriteFile(outputPath, parquetRows); err != nil {
		return 0, fmt.Errorf("parquet write: %w", err)
	}

	return int64(len(parquetRows)), nil
}

func toFloat64(vals []any, idx int) float64 {
	if idx < 0 || idx >= len(vals) || vals[idx] == nil {
		return 0
	}
	switch v := vals[idx].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case []byte:
		var f float64
		_, _ = fmt.Sscanf(string(v), "%f", &f)
		return f
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch v := v.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case []byte:
		var n int64
		_, _ = fmt.Sscanf(string(v), "%d", &n)
		return n
	default:
		return 0
	}
}
