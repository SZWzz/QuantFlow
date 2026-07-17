// internal/crash/capture_test.go
package crash

import (
	"strings"
	"testing"
)

func TestCollectReport(t *testing.T) {
	report := collectReport("something broke")
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Panic != "something broke" {
		t.Errorf("expected 'something broke', got %s", report.Panic)
	}
	if report.Stack == "" {
		t.Error("expected non-empty stack trace")
	}
	if !strings.Contains(report.Stack, "collectReport") {
		t.Error("expected stack trace to contain collectReport")
	}
}

func TestCapturePanic(t *testing.T) {
	oldExit := osExit
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
	}
	defer func() { osExit = oldExit }()

	var captured *CrashReport
	crashHandler = func(report *CrashReport) {
		captured = report
	}
	defer func() { crashHandler = nil }()

	func() {
		defer CapturePanic()
		panic("test crash capture")
	}()

	if !exitCalled {
		t.Error("expected osExit to be called after panic recovery")
	}
	if captured == nil {
		t.Fatal("expected crash report to be captured by handler")
	}
	if captured.Panic != "test crash capture" {
		t.Errorf("expected 'test crash capture', got %s", captured.Panic)
	}
	if captured.Stack == "" {
		t.Error("expected non-empty stack trace in captured report")
	}
}

func TestCapturePanicNilHandler(t *testing.T) {
	oldExit := osExit
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true
	}
	defer func() { osExit = oldExit }()

	crashHandler = nil

	func() {
		defer CapturePanic()
		panic("nil handler crash")
	}()

	if !exitCalled {
		t.Error("expected osExit to be called even with nil handler")
	}
}

func TestStartHandler(t *testing.T) {
	StartHandler(func(report *CrashReport) {
		_ = report // handler counts as a use
	})

	if crashHandler == nil {
		t.Error("expected crashHandler to be set after StartHandler")
	}

	// Clean up for subsequent tests
	crashHandler = nil
}
