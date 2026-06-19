package storage

import (
	"testing"
)

func TestRunMigrations_CreatesVersionTable(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := []Migration{
		{Version: 1, SQL: "CREATE TABLE test_t (id INTEGER PRIMARY KEY)"},
	}

	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify the table was created
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_t").Scan(&count); err != nil {
		t.Errorf("test_t should exist: %v", err)
	}

	// Run again — should be idempotent
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() second call error = %v", err)
	}
}

func TestRunMigrations_AppliesInOrder(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := []Migration{
		{Version: 1, SQL: "CREATE TABLE t1 (id INTEGER PRIMARY KEY)"},
		{Version: 2, SQL: "CREATE TABLE t2 (id INTEGER PRIMARY KEY)"},
		{Version: 3, SQL: "INSERT INTO t2 VALUES (1)"},
	}

	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Both tables should exist
	for _, table := range []string{"t1", "t2"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Errorf("%s should exist: %v", table, err)
		}
	}
}
