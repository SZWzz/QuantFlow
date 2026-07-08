package data

import (
	"testing"
)

func TestArchiveData_compresses_ohlcv(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "ohlcv_cache" {
		t.Fatalf("expected source ohlcv_cache, got %s", result.Source)
	}
	if result.Symbol != "000001" {
		t.Fatalf("expected symbol 000001, got %s", result.Symbol)
	}
	if result.RowCount != 10 {
		t.Fatalf("expected 10 rows, got %d", result.RowCount)
	}
	if result.CompressedBytes <= 0 {
		t.Fatal("expected positive compressed bytes")
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	if count != 10 {
		t.Fatalf("expected 10 original rows, got %d", count)
	}
}

func TestArchiveData_no_data(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := ArchiveData(db, "ohlcv_cache", "999999", "2025-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 0 {
		t.Fatalf("expected 0 rows, got %d", result.RowCount)
	}
}

func TestArchiveData_invalid_source(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := ArchiveData(db, "nonexistent", "000001", "2025-01-01")
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestUnarchiveData_roundtrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	ar, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if ar.RowCount != 10 {
		t.Fatalf("expected 10 archived rows, got %d", ar.RowCount)
	}

	_, err = db.Exec("DELETE FROM ohlcv_cache WHERE symbol='000001'")
	if err != nil {
		t.Fatal(err)
	}

	restored, err := UnarchiveData(db, ar.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored != 10 {
		t.Fatalf("expected 10 restored rows, got %d", restored)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	if count != 10 {
		t.Fatalf("expected 10 rows after unarchive, got %d", count)
	}
}

func TestUnarchiveData_checksum_mismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	ar, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	if err != nil {
		t.Fatal(err)
	}

	db.Exec("UPDATE data_archive SET data = X'0000' WHERE id = ?", ar.ID)

	_, err = UnarchiveData(db, ar.ID)
	if err == nil {
		t.Fatal("expected checksum error")
	}
}
