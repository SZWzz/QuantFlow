package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"quantflow/internal/storage"
)

const ohlcvCacheDDL = `
CREATE TABLE IF NOT EXISTS ohlcv_cache (
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    ts INTEGER NOT NULL,
    open REAL, high REAL, low REAL, close REAL, volume REAL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (symbol, interval, ts)
) WITHOUT ROWID;
`

func setupTestAppDB(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Apply all production migrations
	migrations, err := storage.BuiltinMigrations()
	if err != nil {
		t.Fatalf("failed to load migrations: %v", err)
	}
	if err := storage.Run(db, migrations); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return &App{db: db}
}

func seedTestOHLCV(t *testing.T, db *sql.DB, symbol string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		ts := int64(1704067200 + i*86400)
		_, err := db.Exec(
			"INSERT OR IGNORE INTO ohlcv_cache (symbol, interval, ts, open, high, low, close, volume, fetched_at) VALUES (?,?,?,?,?,?,?,?,?)",
			symbol, "1D", ts, 10.0, 11.0, 9.0, 10.5, 100000, ts,
		)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func TestApp_GetStorageStats(t *testing.T) {
	app := setupTestAppDB(t)
	seedTestOHLCV(t, app.db, "000001", 5)

	stats, err := app.GetStorageStats(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) == 0 {
		t.Fatal("expected non-empty stats")
	}
}

func TestApp_ArchiveAndCleanup(t *testing.T) {
	app := setupTestAppDB(t)
	seedTestOHLCV(t, app.db, "000001", 10)

	// Archive
	ar, err := app.ArchiveData(nil, "ohlcv_cache", "000001", "2025-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if ar.RowCount != 10 {
		t.Fatalf("expected 10 archived, got %d", ar.RowCount)
	}

	// Dry-run cleanup
	cr, err := app.CleanupData(nil, "ohlcv_cache", "000001", "2025-01-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if !cr.DryRun {
		t.Fatal("expected dry run")
	}

	// Actual cleanup
	cr, err = app.CleanupData(nil, "ohlcv_cache", "000001", "2025-01-01", false)
	if err != nil {
		t.Fatal(err)
	}
	if cr.AffectedRows != 10 {
		t.Fatalf("expected 10 deleted, got %d", cr.AffectedRows)
	}
}

func TestApp_ExportData(t *testing.T) {
	app := setupTestAppDB(t)
	seedTestOHLCV(t, app.db, "000001", 5)

	path, err := app.ExportData(nil, "ohlcv_cache", "000001", "1D", "csv", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected non-empty export path")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("export file not found: %s", path)
	}
	os.Remove(path)
}

func TestApp_ImportData(t *testing.T) {
	app := setupTestAppDB(t)

	// First export
	seedTestOHLCV(t, app.db, "000001", 3)
	path, err := app.ExportData(nil, "ohlcv_cache", "000001", "1D", "csv", "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Clear and import back
	app.db.Exec("DELETE FROM ohlcv_cache")
	count, err := app.ImportData(nil, path, "ohlcv_cache")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("expected 3 imported, got %d", count)
	}
	os.Remove(path)
}

func TestAppLayout_RoundTrip(t *testing.T) {
	app := setupTestAppDB(t)
	ctx := context.Background()

	layoutJSON := `{"id":"root","type":"tab","tabs":[{"id":"w1","panelId":"watchlist","label":"自选股","icon":"📊"}],"activeTab":"w1"}`

	// Save
	if err := app.SaveLayout(ctx, "trading", layoutJSON); err != nil {
		t.Fatalf("SaveLayout error: %v", err)
	}

	// List
	names, err := app.ListLayouts(ctx)
	if err != nil {
		t.Fatalf("ListLayouts error: %v", err)
	}
	var found bool
	for _, n := range names {
		if n == "trading" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'trading' in layout list")
	}

	// Load
	loaded, err := app.LoadLayout(ctx, "trading")
	if err != nil {
		t.Fatalf("LoadLayout error: %v", err)
	}
	if loaded != layoutJSON {
		t.Fatalf("LoadLayout = %q, want %q", loaded, layoutJSON)
	}

	// Delete
	if err := app.DeleteLayout(ctx, "trading"); err != nil {
		t.Fatalf("DeleteLayout error: %v", err)
	}

	// List after delete
	names, err = app.ListLayouts(ctx)
	if err != nil {
		t.Fatalf("ListLayouts error: %v", err)
	}
	for _, n := range names {
		if n == "trading" {
			t.Fatal("expected 'trading' to be deleted")
		}
	}
}

func TestAppLayout_NotFound(t *testing.T) {
	app := setupTestAppDB(t)
	ctx := context.Background()

	_, err := app.LoadLayout(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent layout")
	}

	err = app.DeleteLayout(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting nonexistent layout")
	}
}

func TestAppLayout_EmptyName(t *testing.T) {
	app := setupTestAppDB(t)
	ctx := context.Background()

	err := app.SaveLayout(ctx, "", "{}")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}
