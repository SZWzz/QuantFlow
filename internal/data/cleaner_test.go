package data

import (
	"testing"
)

func TestCleanupData_dryRun_returns_count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := CleanupData(db, "ohlcv_cache", "000001", "2024-01-05", true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun {
		t.Fatal("expected dry_run=true")
	}
	if result.AffectedRows != 4 {
		t.Fatalf("expected 4 rows before Jan 5, got %d", result.AffectedRows)
	}
	if len(result.Preview) == 0 {
		t.Fatal("expected preview rows")
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	if count != 10 {
		t.Fatalf("expected 10 rows still present, got %d", count)
	}
}

func TestCleanupData_execute_removes_rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := CleanupData(db, "ohlcv_cache", "000001", "2024-01-05", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.DryRun {
		t.Fatal("expected dry_run=false")
	}
	if result.AffectedRows != 4 {
		t.Fatalf("expected 4 deleted rows, got %d", result.AffectedRows)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	if count != 6 {
		t.Fatalf("expected 6 remaining rows, got %d", count)
	}
}

func TestCleanupData_invalid_table(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := CleanupData(db, "orders", "000001", "2025-01-01", true)
	if err == nil {
		t.Fatal("expected error for invalid table")
	}
}

func TestCleanupData_no_data(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := CleanupData(db, "ohlcv_cache", "999999", "2025-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.AffectedRows != 0 {
		t.Fatalf("expected 0 rows, got %d", result.AffectedRows)
	}
}
