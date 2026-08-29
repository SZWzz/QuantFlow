// internal/crash/store_test.go
package crash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
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

func TestStoreUpload(t *testing.T) {
	// Server that validates the POST body is our crash report JSON and returns 200.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		var received CrashReport
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if received.Panic == "" {
			t.Error("expected non-empty panic field")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := NewStore(t.TempDir())
	report := NewCrashReport("upload test panic", "stack", []string{"log"}, AppState{})
	if err := s.Upload(report, server.URL); err != nil {
		t.Fatalf("Upload: %v", err)
	}
}

func TestStoreUploadNon200(t *testing.T) {
	// Server that returns 500 to test error handling.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	s := NewStore(t.TempDir())
	report := NewCrashReport("upload error", "stack", nil, AppState{})
	if err := s.Upload(report, server.URL); err == nil {
		t.Error("expected error for non-200 status")
	}
}
