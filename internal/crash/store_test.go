// internal/crash/store_test.go
package crash

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveAndList(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	report := NewCrashReport("test panic", "stack", []string{"log"}, AppState{TradingMode: "paper"})
	path, err := s.Save(report)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}

	reports, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}
	if reports[0].Panic != "test panic" {
		t.Errorf("expected test panic, got %s", reports[0].Panic)
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	report := NewCrashReport("delete me", "stack", nil, AppState{})
	_, err := s.Save(report)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Delete(report.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	reports, _ := s.List()
	if len(reports) != 0 {
		t.Errorf("expected 0 reports after delete, got %d", len(reports))
	}
}

func TestStoreCleanup(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	report := NewCrashReport("old", "stack", nil, AppState{})
	report.Timestamp = time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	path := filepath.Join(dir, report.FileName())
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set file modification time to match the old timestamp so Cleanup detects it
	oldTime := report.Timestamp
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	removed, err := s.Cleanup(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}
