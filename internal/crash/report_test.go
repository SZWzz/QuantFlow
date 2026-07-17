// internal/crash/report_test.go
package crash

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewCrashReport(t *testing.T) {
	state := AppState{
		TradingMode:   "paper",
		ActiveBrokers: []string{"alpaca"},
		PanelCount:    5,
		UptimeSeconds: 3600,
	}
	report := NewCrashReport("runtime error: index out of range", "goroutine 1 [running]:\nmain.foo()", []string{"2026/07/16 10:00:00 INFO startup complete"}, state)

	if report.Panic != "runtime error: index out of range" {
		t.Errorf("expected panic value, got %s", report.Panic)
	}
	if !strings.Contains(report.Stack, "main.foo") {
		t.Errorf("expected stack trace, got %s", report.Stack)
	}
	if len(report.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(report.Logs))
	}
	if report.AppState.TradingMode != "paper" {
		t.Errorf("expected paper mode, got %s", report.AppState.TradingMode)
	}

	// Fields that should be set automatically
	if report.ID == "" {
		t.Error("expected non-empty ID")
	}
	if report.Version == "" {
		t.Error("expected non-empty Version")
	}
	if report.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	if report.OS == "" || report.Arch == "" {
		t.Error("expected OS/Arch to be set")
	}
	if report.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestFileNameWindowsSafe(t *testing.T) {
	report := NewCrashReport("p", "s", nil, AppState{})
	name := report.FileName()

	// Colons are illegal in Windows file names.
	if strings.ContainsAny(name, `:<>"/\|?*`) {
		t.Errorf("filename contains Windows-illegal characters: %s", name)
	}
	if !strings.HasPrefix(name, "crash-") || !strings.HasSuffix(name, ".json") {
		t.Errorf("expected crash-<ts>-<id>.json format, got %s", name)
	}
	// ID prefix prevents same-second collisions.
	if !strings.Contains(name, report.ID[:8]) {
		t.Errorf("expected filename to contain ID prefix %s, got %s", report.ID[:8], name)
	}

	// Short IDs must not panic and are used whole.
	report.ID = "abc"
	if got := report.FileName(); !strings.Contains(got, "-abc.json") {
		t.Errorf("expected short ID used whole, got %s", got)
	}
}

func TestCrashReportJSONSerialization(t *testing.T) {
	report := NewCrashReport("test panic", "stack trace", []string{"log1"}, AppState{TradingMode: "live"})
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded CrashReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Panic != "test panic" {
		t.Errorf("expected test panic, got %s", decoded.Panic)
	}
	if decoded.AppState.TradingMode != "live" {
		t.Errorf("expected live mode, got %s", decoded.AppState.TradingMode)
	}
}
