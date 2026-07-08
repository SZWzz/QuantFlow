package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportCSV_writes_valid_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "000001", "1D", "2024-01-01", "2024-01-05", path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected 5 rows, got %d", count)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !contains(content, "symbol") {
		t.Fatal("CSV missing header")
	}
	if !contains(content, "000001") {
		t.Fatal("CSV missing data")
	}
}

func TestExportCSV_empty_result(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "999999", "1D", "2024-01-01", "2024-01-05", path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows, got %d", count)
	}
}

func TestExportCSV_no_date_filter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "all.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "000001", "1D", "", "", path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected 5 rows, got %d", count)
	}
}

func TestExportParquet_writes_valid_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	count, err := ExportParquet(db, "ohlcv_cache", "000001", "1D", "2024-01-01", "2024-01-05", path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected 5 rows, got %d", count)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 0 {
		t.Fatal("expected non-empty parquet file")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
