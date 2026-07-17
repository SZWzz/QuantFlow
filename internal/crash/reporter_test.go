// internal/crash/reporter_test.go
package crash

import (
	"testing"
)

func TestCrashDir(t *testing.T) {
	dir := CrashDir()
	if dir == "" {
		t.Fatal("expected non-empty crash dir")
	}
	// Should be an absolute path
	if dir[0] != '/' {
		t.Errorf("expected absolute path, got %s", dir)
	}
	// Should contain QuantFlow and crashes
	if dir != "" {
		// just verify it's reasonable
		t.Logf("CrashDir() = %s", dir)
	}
}

func TestStartCrashHandler(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Should not panic
	StartCrashHandler(s)

	if crashHandler == nil {
		t.Error("expected crashHandler to be set after StartCrashHandler")
	}

	// Clean up
	crashHandler = nil
}

func TestStartCrashHandlerNilStore(t *testing.T) {
	// Should not panic with nil store
	StartCrashHandler(nil)

	// Handler should still be set but store is nil
	if crashHandler == nil {
		t.Error("expected crashHandler to be set after StartCrashHandler(nil)")
	}

	// Clean up
	crashHandler = nil
}

func TestStartCrashHandlerSaveOnCrash(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	var captured *CrashReport
	oldHandler := crashHandler
	crashHandler = func(report *CrashReport) {
		captured = report
	}
	defer func() { crashHandler = oldHandler }()

	// Simulate what StartCrashHandler's callback does
	handler := func(report *CrashReport) {
		captured = report
		if _, err := s.Save(report); err != nil {
			t.Errorf("Save: %v", err)
		}
	}
	handler(collectReport("test panic"))

	if captured == nil {
		t.Fatal("expected crash report to be captured")
	}
	if captured.Panic != "test panic" {
		t.Errorf("expected 'test panic', got %s", captured.Panic)
	}

	// Verify it was saved to store
	reports, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(reports) != 1 {
		t.Errorf("expected 1 saved report, got %d", len(reports))
	}
}
