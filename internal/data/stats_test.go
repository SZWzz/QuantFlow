package data

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{ohlcvCacheDDL, minuteCacheDDL, dataArchiveDDL} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedOHLCV(t *testing.T, db *sql.DB, symbol string, count int) {
	t.Helper()
	now := time.Now().Unix()
	for i := 0; i < count; i++ {
		ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Unix()
		_, err := db.Exec(
			"INSERT OR IGNORE INTO ohlcv_cache (symbol, interval, ts, open, high, low, close, volume, fetched_at) VALUES (?,?,?,?,?,?,?,?,?)",
			symbol, "1D", ts, 10.0, 11.0, 9.0, 10.5, 100000, now,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetTableStats_returns_counts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedOHLCV(t, db, "000001", 5)
	seedOHLCV(t, db, "600001", 3)

	stats, err := GetTableStats(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 3 {
		t.Fatalf("expected 3 table stats, got %d", len(stats))
	}

	var found bool
	for _, s := range stats {
		if s.Table == "ohlcv_cache" {
			found = true
			if s.Rows != 8 {
				t.Fatalf("expected 8 rows, got %d", s.Rows)
			}
			if s.SizeBytes <= 0 {
				t.Fatal("expected positive size")
			}
			if s.Oldest == "" {
				t.Fatal("expected oldest date")
			}
			if s.Newest == "" {
				t.Fatal("expected newest date")
			}
		}
	}
	if !found {
		t.Fatal("ohlcv_cache not found in stats")
	}
}

func TestGetTableStats_empty_db(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := GetTableStats(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range stats {
		if s.Rows != 0 {
			t.Fatalf("expected 0 rows for %s, got %d", s.Table, s.Rows)
		}
	}
}
