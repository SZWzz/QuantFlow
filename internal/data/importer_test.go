package data

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func writeTestCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		t.Fatal(err)
	}
	w.Flush()
}

func TestImportCSV_inserts_rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "import.csv")
	writeTestCSV(t, path, [][]string{
		{"symbol", "interval", "ts", "open", "high", "low", "close", "volume", "fetched_at"},
		{"000001", "1D", "1704067200", "10.0", "11.0", "9.0", "10.5", "100000", "1704067200"},
		{"000001", "1D", "1704153600", "10.5", "11.5", "10.0", "11.0", "150000", "1704153600"},
	})

	count, err := ImportCSV(db, path, "ohlcv_cache")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows inserted, got %d", count)
	}

	var total int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache").Scan(&total)
	if total != 2 {
		t.Fatalf("expected 2 total rows, got %d", total)
	}
}

func TestImportCSV_duplicate_rows_skipped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	writeTestCSV(t, path, [][]string{
		{"symbol", "interval", "ts", "open", "high", "low", "close", "volume", "fetched_at"},
		{"000001", "1D", "1704067200", "10.0", "11.0", "9.0", "10.5", "100000", "1704067200"},
	})

	count, err := ImportCSV(db, path, "ohlcv_cache")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows (all duplicates), got %d", count)
	}
}

func TestImportCSV_invalid_table(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	writeTestCSV(t, path, [][]string{{"a"}, {"1"}})

	_, err := ImportCSV(db, path, "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid table")
	}
}

func TestImportCSV_missing_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := ImportCSV(db, "/nonexistent/path.csv", "ohlcv_cache")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestImportParquet_roundtrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedOHLCV(t, db, "000001", 3)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	_, err := ExportParquet(db, "ohlcv_cache", "000001", "1D", "", "", path)
	if err != nil {
		t.Fatal(err)
	}

	db.Exec("DELETE FROM ohlcv_cache")

	count, err := ImportParquet(db, path, "ohlcv_cache")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows imported, got %d", count)
	}
}
