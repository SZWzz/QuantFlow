package storage

import (
	"testing"
)

func TestMigration010_MLModels(t *testing.T) {
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Find migration 010 from builtin migrations
	migs, err := BuiltinMigrations()
	if err != nil {
		t.Fatalf("BuiltinMigrations() error = %v", err)
	}

	var mig010 *Migration
	for _, m := range migs {
		if m.Version == 10 {
			mig010 = &m
			break
		}
	}
	if mig010 == nil {
		t.Fatal("migration 010 not found in builtin migrations")
	}

	// Run migration 010
	if err := Run(db, []Migration{*mig010}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify ml_models table
	_, err = db.Exec(`INSERT INTO ml_models (id, name, model_type, category) VALUES ('test-1', 'test_model', 'xgboost', 'prediction')`)
	if err != nil {
		t.Fatalf("ml_models insert failed: %v", err)
	}

	// Verify ml_predictions table with FK
	_, err = db.Exec(`INSERT INTO ml_predictions (id, model_id, symbol, date, prediction) VALUES ('p-1', 'test-1', 'AAPL', '2026-06-18', 0.05)`)
	if err != nil {
		t.Fatalf("ml_predictions insert failed: %v", err)
	}

	// Verify ml_evaluations table
	_, err = db.Exec(`INSERT INTO ml_evaluations (id, model_id, metric_name, value, period) VALUES ('e-1', 'test-1', 'sharpe', 1.82, 'test')`)
	if err != nil {
		t.Fatalf("ml_evaluations insert failed: %v", err)
	}

	// Verify FK cascade: deleting model should cascade to predictions
	_, err = db.Exec(`DELETE FROM ml_models WHERE id = 'test-1'`)
	if err != nil {
		t.Fatalf("delete model failed: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ml_predictions WHERE model_id = 'test-1'`).Scan(&count); err != nil {
		t.Fatalf("count predictions failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 predictions after cascade delete, got %d", count)
	}
}
