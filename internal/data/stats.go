package data

import (
	"database/sql"
	"fmt"
)

type TableStat struct {
	Table     string `json:"table"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"size_bytes"`
	Oldest    string `json:"oldest"`
	Newest    string `json:"newest"`
}

var trackedTables = []struct {
	Name     string
	DateCol  string
	SizeExpr string
}{
	{"ohlcv_cache", "ts", "COUNT(*) * 64"},
	{"minute_cache", "date", "COUNT(*) * 48"},
	{"data_archive", "archived_at", "COALESCE(SUM(LENGTH(data)), 0) + COUNT(*) * 128"},
}

func GetTableStats(db *sql.DB) ([]TableStat, error) {
	var stats []TableStat
	for _, t := range trackedTables {
		st, err := tableStat(db, t.Name, t.DateCol, t.SizeExpr)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", t.Name, err)
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func tableStat(db *sql.DB, table, dateCol, sizeExpr string) (TableStat, error) {
	var st TableStat
	st.Table = table

	q := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table) //nolint:gosec // table 来自固定表名枚举，无用户输入
	err := db.QueryRow(q).Scan(&st.Rows)
	if err != nil {
		return st, err
	}

	q = fmt.Sprintf("SELECT %s FROM \"%s\"", sizeExpr, table)
	err = db.QueryRow(q).Scan(&st.SizeBytes)
	if err != nil {
		st.SizeBytes = 0
	}

	if dateCol != "" && st.Rows > 0 {
		q = fmt.Sprintf("SELECT MIN(%s), MAX(%s) FROM \"%s\"", dateCol, dateCol, table)
		var oldest, newest sql.NullString
		err = db.QueryRow(q).Scan(&oldest, &newest)
		if err == nil {
			st.Oldest = oldest.String
			st.Newest = newest.String
		}
	}

	return st, nil
}
