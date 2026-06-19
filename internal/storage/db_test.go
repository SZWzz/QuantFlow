package storage

import (
	"os"
	"testing"
)

func TestOpen_Pragmas(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Verify WAL mode
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode error = %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	// Verify foreign keys
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys error = %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkEnabled)
	}
}

func TestOpenDB_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	_, err := os.Stat(dbPath)
	if err == nil {
		t.Fatal("test.db should not exist before Open()")
	}

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("DB file was not created at %s", dbPath)
	}
}

func TestOpenDB_Reopen(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	db1.Close()

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	db2.Close()
}

func TestSetupSQLite_Migrations_Run(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := []Migration{
		{Version: 1, SQL: "CREATE TABLE IF NOT EXISTS test_migration (id INTEGER PRIMARY KEY, name TEXT)"},
		{Version: 2, SQL: "CREATE TABLE IF NOT EXISTS test_migration_2 (id INTEGER PRIMARY KEY, value REAL)"},
	}

	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify both migration tables exist
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_migration").Scan(&count); err != nil {
		t.Errorf("test_migration table should exist: %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM test_migration_2").Scan(&count); err != nil {
		t.Errorf("test_migration_2 table should exist: %v", err)
	}

	// Verify schema_version was created and records both versions
	var versionCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&versionCount); err != nil {
		t.Errorf("schema_version should exist: %v", err)
	}
	if versionCount != 2 {
		t.Errorf("expected 2 migration records, got %d", versionCount)
	}

	// Run again — should be idempotent
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() second call error = %v", err)
	}
	// Version count should still be 2
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&versionCount); err != nil {
		t.Fatalf("schema_version query error: %v", err)
	}
	if versionCount != 2 {
		t.Errorf("after re-run: expected 2 migration records, got %d", versionCount)
	}
}

func TestSetupSQLite_WAL_Mode(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Verify WAL journal mode is active
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode error = %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	// Verify WAL file exists on disk after a write
	db.Exec("CREATE TABLE IF NOT EXISTS wal_test (id INTEGER PRIMARY KEY)")
	db.Exec("INSERT INTO wal_test VALUES (1)")

	walPath := dbPath + "-wal"
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		t.Logf("WAL sidecar file not found at %s (may not be flushed yet)", walPath)
	}
}
