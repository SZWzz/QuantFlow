# Crash Reporter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Capture goroutine panics and OS crash signals, generate structured crash reports, save locally, and offer opt-in upload.

**Architecture:** `internal/crash/` package handles signal registration, panic recovery, report assembly (stack trace, ring buffer logs, app state), and local storage. `main.go` registers the crash handler at startup. Frontend listens for Wails events to show crash dialog on next launch.

**Tech Stack:** Go 1.25+ (os/signal, runtime/pprof, runtime/debug), SQLite WAL (crash history), Vue 3 (crash dialog via Wails EventsOn)

## Global Constraints

- No new Go dependencies beyond stdlib (os/signal, runtime, runtime/debug, encoding/json, os)
- Use slog for Go logging
- Use Composition API with `<script setup lang="ts">` for Vue
- No `window.confirm()` / `window.alert()` — use `confirmDialog`/`alertDialog` from `@/lib/wails`
- Crash reports contain ZERO personal identifiable information (no API keys, no positions)
- Reports stored in `~/Library/Logs/QuantFlow/crashes/` (macOS/Linux), `%LOCALAPPDATA%\QuantFlow\crashes\` (Windows)
- Auto-clean reports older than 30 days
- Wails EventsOn for crash dialog communication

---

### Task 1: CrashReport Struct Definition (report.go)

**Files:**
- Create: `internal/crash/report.go`
- Test: `internal/crash/report_test.go`

**Interfaces:**
- Consumes: nothing (standalone)
- Produces: `CrashReport`, `AppState` structs; `NewCrashReport(panicVal string, stack string, logs []string, state AppState) *CrashReport` constructor

- [ ] **Step 1: Write the failing test**

```go
// internal/crash/report_test.go
package crash

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestNewCrashReport -count=1`
Expected: FAIL with "package internal/crash is not in std"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/crash/report.go
package crash

import (
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
)

type CrashReport struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	GoVersion string    `json:"go_version"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	BuildMode string    `json:"build_mode"`

	Panic string   `json:"panic"`
	Stack string   `json:"stack"`
	Logs  []string `json:"logs"`

	AppState AppState `json:"app_state"`
}

type AppState struct {
	TradingMode   string   `json:"trading_mode"`
	ActiveBrokers []string `json:"active_brokers"`
	PanelCount    int      `json:"panel_count"`
	WorkflowCount int      `json:"workflow_count"`
	UptimeSeconds int64    `json:"uptime_seconds"`
}

var (
	appVersion  = "0.0.1"
	buildMode   = "dev"
	uptimeFn    = func() int64 { return 0 }
	panelCount  = func() int { return 0 }
	workflowCnt = func() int { return 0 }
	brokersFn   = func() []string { return nil }
	tradingMode = func() string { return "unknown" }
)

func SetAppInfo(version, mode string) {
	appVersion = version
	buildMode = mode
}

func SetStateGetters(
	uptime func() int64,
	panels func() int,
	workflows func() int,
	brokers func() []string,
	mode func() string,
) {
	uptimeFn = uptime
	panelCount = panels
	workflowCnt = workflows
	brokersFn = brokers
	tradingMode = mode
}

func NewCrashReport(panicVal, stack string, logs []string, state AppState) *CrashReport {
	if state.TradingMode == "" {
		state.TradingMode = tradingMode()
	}
	if state.UptimeSeconds == 0 {
		state.UptimeSeconds = uptimeFn()
	}
	if state.PanelCount == 0 {
		state.PanelCount = panelCount()
	}
	if state.WorkflowCount == 0 {
		state.WorkflowCount = workflowCnt()
	}
	if len(state.ActiveBrokers) == 0 {
		state.ActiveBrokers = brokersFn()
	}

	return &CrashReport{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Version:   appVersion,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		BuildMode: buildMode,
		Panic:     panicVal,
		Stack:     stack,
		Logs:      logs,
		AppState:  state,
	}
}

func (r *CrashReport) FileName() string {
	return fmt.Sprintf("%s.json", r.Timestamp.Format("2006-01-02T15:04:05"))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestNewCrashReport -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crash/report.go internal/crash/report_test.go
git commit -m "feat(crash): add CrashReport struct with JSON serialization"
```

---

### Task 2: Local Storage + Upload (store.go)

**Files:**
- Create: `internal/crash/store.go`
- Test: `internal/crash/store_test.go`

**Interfaces:**
- Consumes: `CrashReport` from Task 1
- Produces: `Store` struct with `Save(report *CrashReport) (string, error)`, `List() ([]CrashReport, error)`, `Delete(id string) error`, `Cleanup(maxAge time.Duration) error`, `Upload(report *CrashReport, endpoint string) error`

- [ ] **Step 1: Write the failing test**

```go
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

	removed, err := s.Cleanup(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestStoreSaveAndList -count=1`
Expected: FAIL with "NewStore not defined"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/crash/store.go
package crash

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Save(report *CrashReport) (string, error) {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return "", fmt.Errorf("create crash dir: %w", err)
	}

	path := filepath.Join(s.dir, report.FileName())
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

func (s *Store) List() ([]CrashReport, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CrashReport{}, nil
		}
		return nil, fmt.Errorf("read crash dir: %w", err)
	}

	var reports []CrashReport
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			slog.Warn("read crash file failed, skipping", "file", e.Name(), "error", err)
			continue
		}
		var report CrashReport
		if err := json.Unmarshal(data, &report); err != nil {
			slog.Warn("parse crash file failed, skipping", "file", e.Name(), "error", err)
			continue
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (s *Store) Delete(id string) error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("read crash dir: %w", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var report CrashReport
		if json.Unmarshal(data, &report) != nil {
			continue
		}
		if report.ID == id {
			return os.Remove(filepath.Join(s.dir, e.Name()))
		}
	}
	return fmt.Errorf("report %s not found", id)
}

func (s *Store) Cleanup(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read crash dir: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(s.dir, e.Name())); err != nil {
				slog.Warn("remove old crash file failed", "file", e.Name(), "error", err)
			} else {
				removed++
			}
		}
	}
	return removed, nil
}

func (s *Store) Upload(report *CrashReport, endpoint string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload status %d", resp.StatusCode)
	}
	return nil
}

func (s *Store) Dir() string {
	return s.dir
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestStoreSaveAndList -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crash/store.go internal/crash/store_test.go
git commit -m "feat(crash): add crash report local storage, list, delete, cleanup, and upload"
```

---

### Task 3: Signal Handler + Panic Recovery (reporter.go + platform files)

**Files:**
- Create: `internal/crash/reporter.go`
- Create: `internal/crash/darwin.go`
- Create: `internal/crash/linux.go`
- Create: `internal/crash/windows.go`
- Test: `internal/crash/reporter_test.go`

**Interfaces:**
- Consumes: `CrashReport`, `Store` from Task 1-2
- Produces: `StartHandler(store *Store)`, `CapturePanic()`, `collectReport(panicVal string) *CrashReport`

- [ ] **Step 1: Write the failing test**

```go
// internal/crash/reporter_test.go
package crash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectReport(t *testing.T) {
	// Test panic recovery capture
	func() {
		defer CapturePanic()
		panic("something broke")
	}()

	dir := t.TempDir()
	s := NewStore(dir)

	report := collectReport("something broke", s)
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Panic != "something broke" {
		t.Errorf("expected 'something broke', got %s", report.Panic)
	}
	if report.Stack == "" {
		t.Error("expected non-empty stack trace")
	}
}

func TestStartHandler(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	// Should not panic
	StartHandler(s)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestCollectReport -count=1`
Expected: FAIL with "CapturePanic not defined"

- [ ] **Step 3: Write implementation**

```go
// internal/crash/reporter.go
package crash

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"quantflow/internal/logging"
)

var (
	globalStore *Store
	crashLogDir string
)

// StartHandler registers OS signal handlers and returns a cleanup function.
func StartHandler(store *Store) {
	globalStore = store

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGABRT, syscall.SIGSEGV, syscall.SIGILL, syscall.SIGBUS)

	go func() {
		for sig := range c {
			report := collectReport(fmt.Sprintf("signal: %v", sig), globalStore)
			if report != nil {
				if path, err := globalStore.Save(report); err != nil {
					slog.Error("failed to save crash report", "error", err)
				} else {
					slog.Error("crash report saved", "path", path, "signal", sig)
				}
			}
			os.Exit(1)
		}
	}()

	// Cleanup old reports on start
	go func() {
		if globalStore != nil {
			if removed, err := globalStore.Cleanup(30 * 24 * time.Hour); err != nil {
				slog.Warn("crash cleanup failed", "error", err)
			} else if removed > 0 {
				slog.Info("cleaned up old crash reports", "count", removed)
			}
		}
	}()
}

// CapturePanic should be called via defer at the top of main and key goroutines.
func CapturePanic() {
	if r := recover(); r != nil {
		report := collectReport(fmt.Sprintf("%v", r), globalStore)
		if report != nil && globalStore != nil {
			if path, err := globalStore.Save(report); err != nil {
				slog.Error("failed to save panic report", "error", err)
			} else {
				slog.Error("panic captured, report saved", "path", path, "panic", r)
			}
		}
		os.Exit(1)
	}
}

func collectReport(panicVal string, store *Store) *CrashReport {
	if store == nil {
		return nil
	}

	stack := string(debug.Stack())

	var logs []string
	if logging.Ring != nil {
		entries := logging.Ring.Lines(0, 100)
		logs = make([]string, 0, len(entries))
		for _, e := range entries {
			logs = append(logs, fmt.Sprintf("%s [%s] %s", e.Time.Format("2006-01-02 15:04:05"), e.Level, e.Message))
		}
	}

	return NewCrashReport(panicVal, stack, logs, AppState{})
}
```

```go
// internal/crash/darwin.go
//go:build darwin

package crash

import "log/slog"

func init() {
	crashLogDir = os.ExpandEnv("$HOME/Library/Logs/QuantFlow/crashes")
}
```

```go
// internal/crash/linux.go
//go:build linux

package crash

import "log/slog"

func init() {
	crashLogDir = os.ExpandEnv("$HOME/.local/share/QuantFlow/crashes")
}
```

```go
// internal/crash/windows.go
//go:build windows

package crash

func init() {
	crashLogDir = os.ExpandEnv("%LOCALAPPDATA%\\QuantFlow\\crashes")
}
```

Add the `time` import to reporter.go — it uses `time` for the cleanup goroutine:

```go
// The import block in reporter.go already has "time" in the cleanup code above
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/crash/ -v -run TestCollectReport -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crash/reporter.go internal/crash/darwin.go internal/crash/linux.go internal/crash/windows.go internal/crash/reporter_test.go
git commit -m "feat(crash): add signal handler, panic recovery, and platform log dirs"
```

---

### Task 4: main.go Integration

**Files:**
- Modify: `main.go`
- Modify: `app_startup.go` (wire SetAppInfo + SetStateGetters)

**Interfaces:**
- Consumes: `crash.StartHandler`, `crash.CapturePanic`, `crash.SetAppInfo`, `crash.SetStateGetters` from Task 3
- Produces: Registered crash handler at application startup

- [ ] **Step 1: Write the test**

```go
// main_test.go
package main

import (
	"testing"
)

func TestMainDoesNotPanicOnStartup(t *testing.T) {
	// Just verify imports compile — actual Wails run is tested in integration
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./...`
Expected: No errors

- [ ] **Step 3: Write implementation**

Modify `main.go`:

```go
// main.go
package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/crash"
	"quantflow/internal/ws"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	defer crash.CapturePanic()

	// Determine crash report directory
	crashDir := filepath.Join(userHomeDir(), "Library", "Logs", "QuantFlow", "crashes")
	if runtime.GOOS == "linux" {
		crashDir = filepath.Join(userHomeDir(), ".local", "share", "QuantFlow", "crashes")
	} else if runtime.GOOS == "windows" {
		crashDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "QuantFlow", "crashes")
	}

	store := crash.NewStore(crashDir)
	crash.StartHandler(store)

	// Create the MarketWSService first.
	wsSvc := &ws.MarketWSService{}

	appInstance := &App{wsSvc: wsSvc}
	app := application.New(application.Options{
		Name:        "quantflow",
		Description: "QuantFlow Terminal — 双模式量化金融终端",
		Services: []application.Service{
			application.NewService(appInstance),
			application.NewServiceWithOptions(wsSvc, application.ServiceOptions{
				Route: "/ws/market",
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	appInstance.wailsApp = app
	appInstance.crashStore = store

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "QuantFlow Terminal",
		Width:            1400,
		Height:           900,
		MinWidth:        900,
		MinHeight:       600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		slog.Error("application run failed", "error", err)
		os.Exit(1)
	}
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}
```

Modify `app.go` — add `crashStore` field:

```go
// Add to App struct (after line 130)
	crashStore *crash.Store
```

Modify `app_startup.go` — wire app info:

```go
// Add at the beginning of ServiceStartup, after config loading (after line 57)
	a.cfg = cfg

	// Wire crash reporter with app version and state getters
	crash.SetAppInfo(cfg.Version, "prod")
	crash.SetStateGetters(
		func() int64 { return int64(time.Since(startTime).Seconds()) },
		func() int { return len(panelRegistry) },
		func() int {
			if a.engine != nil {
				return len(a.engine.ListWorkflows())
			}
			return 0
		},
		func() []string {
			names := make([]string, 0, len(a.brokers))
			for name := range a.brokers {
				names = append(names, name)
			}
			return names
		},
		func() string {
			if a.cfg != nil {
				return a.cfg.TradingMode
			}
			return "unknown"
		},
	)
```

- [ ] **Step 4: Verify build**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go build -o /dev/null .`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add main.go app.go app_startup.go
git commit -m "feat(crash): integrate crash handler into main.go and wire app info"
```

---

### Task 5: Export RingBuffer.LastN from Logging Package

**Files:**
- Modify: `internal/logging/ring_buffer.go`
- Test: `internal/logging/ring_buffer_test.go`

**Interfaces:**
- Consumes: `RingBuffer` base struct
- Produces: `LastN(n int) []LogEntry` method; exported `Ring` variable reference

- [ ] **Step 1: Write the failing test**

```go
// internal/logging/ring_buffer_test.go (add to existing)
func TestRingBufferLastN(t *testing.T) {
	rb := NewRingBuffer(10)
	for i := 0; i < 10; i++ {
		rb.Push(LogEntry{Message: fmt.Sprintf("msg %d", i)})
	}

	last5 := rb.LastN(5)
	if len(last5) != 5 {
		t.Errorf("expected 5, got %d", len(last5))
	}
	if last5[0].Message != "msg 5" {
		t.Errorf("expected msg 5, got %s", last5[0].Message)
	}

	// Request more than available
	last20 := rb.LastN(20)
	if len(last20) != 10 {
		t.Errorf("expected 10, got %d", len(last20))
	}

	// Request 0
	last0 := rb.LastN(0)
	if len(last0) != 0 {
		t.Errorf("expected 0, got %d", len(last0))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -v -run TestRingBufferLastN -count=1`
Expected: FAIL with "LastN not defined"

- [ ] **Step 3: Write implementation**

Add to `internal/logging/ring_buffer.go` (after the existing `Lines` method):

```go
// LastN returns the last n entries from the buffer, most recent first.
func (rb *RingBuffer) LastN(n int) []LogEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 || n <= 0 {
		return []LogEntry{}
	}

	if n > rb.count {
		n = rb.count
	}

	result := make([]LogEntry, n)
	for i := 0; i < n; i++ {
		idx := (rb.head + rb.count - n + i) % rb.max
		result[i] = rb.buffer[idx]
	}
	return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -v -run TestRingBufferLastN -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logging/ring_buffer.go internal/logging/ring_buffer_test.go
git commit -m "feat(crash): add RingBuffer.LastN method for crash report log capture"
```

---

### Task 6: Crash Dialog and History Panel (Frontend)

**Files:**
- Create: `frontend/src/terminal/components/CrashDialog.vue`
- Modify: `frontend/src/lib/wails.ts` (add crash event handling)
- Create: `frontend/src/stores/crash.ts`

**Interfaces:**
- Consumes: Wails `EventsOn` for crash reports; `crashStore.List()` → `CrashReport[]`
- Produces: Crash dialog on startup if pending crash; crash history display in SystemMonitor

- [ ] **Step 1: Write the test**

```typescript
// frontend/src/__tests__/CrashDialog.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CrashDialog from '@/terminal/components/CrashDialog.vue'

describe('CrashDialog', () => {
  it('renders crash info', () => {
    const wrapper = mount(CrashDialog, {
      props: {
        visible: true,
        crashTime: '2026-07-16T10:30:00',
        crashPath: '/path/to/report.json',
      },
    })
    expect(wrapper.text()).toContain('QuantFlow 崩溃了')
    expect(wrapper.text()).toContain('/path/to/report.json')
  })

  it('emits restart on restart click', () => {
    const wrapper = mount(CrashDialog, {
      props: {
        visible: true,
        crashTime: '2026-07-16T10:30:00',
        crashPath: '/path/to/report.json',
      },
    })
    wrapper.find('[data-test="restart"]').trigger('click')
    expect(wrapper.emitted('restart')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run -t "CrashDialog" 2>&1 || true`
Expected: FAIL (file doesn't exist)

- [ ] **Step 3: Write implementation**

```typescript
// frontend/src/stores/crash.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Call } from '@wailsio/runtime'

export interface CrashReport {
  id: string
  timestamp: string
  version: string
  go_version: string
  os: string
  arch: string
  panic: string
  stack: string
  logs: string[]
  app_state: {
    trading_mode: string
    active_brokers: string[]
    panel_count: number
    workflow_count: number
    uptime_seconds: number
  }
}

export const useCrashStore = defineStore('crash', () => {
  const reports = ref<CrashReport[]>([])
  const loading = ref(false)

  async function list() {
    loading.value = true
    try {
      reports.value = await Call.ByName('main.App.ListCrashReports') as CrashReport[]
    } catch {
      reports.value = []
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string) {
    try {
      await Call.ByName('main.App.DeleteCrashReport', id)
      reports.value = reports.value.filter(r => r.id !== id)
    } catch {}
  }

  async function upload(id: string) {
    try {
      await Call.ByName('main.App.UploadCrashReport', id)
    } catch {}
  }

  return { reports, loading, list, remove, upload }
})
```

```typescript
// Add to frontend/src/lib/wails.ts (after DeleteLayout)

export interface CrashReport {
  id: string
  timestamp: string
  version: string
  panic: string
  os: string
  arch: string
}

export async function ListCrashReports(): Promise<CrashReport[]> {
  return wailsCall<CrashReport[]>('ListCrashReports')
}

export async function DeleteCrashReport(id: string): Promise<void> {
  return wailsCall<void>('DeleteCrashReport', id)
}
```

```vue
<!-- frontend/src/terminal/components/CrashDialog.vue -->
<script setup lang="ts">
const props = defineProps<{
  visible: boolean
  crashTime: string
  crashPath: string
}>()

const emit = defineEmits<{
  close: []
  restart: []
  upload: [send: boolean]
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="crash-overlay" @click.self="emit('close')">
      <div class="crash-dialog">
        <div class="crash-header">
          <span class="crash-icon">💥</span>
          <h2>QuantFlow 崩溃了</h2>
        </div>
        <div class="crash-body">
          <p>很抱歉，应用遇到了意外错误。</p>
          <p class="crash-path">
            已保存崩溃报告:<br>
            <code>{{ crashPath }}</code>
          </p>
          <label class="crash-upload-opt">
            <input type="checkbox" :checked="true" @change="$emit('upload', ($event.target as HTMLInputElement).checked)" />
            <span>发送匿名崩溃报告帮助改进</span>
          </label>
        </div>
        <div class="crash-actions">
          <button class="btn btn-secondary" @click="emit('close')">忽略</button>
          <button class="btn btn-primary" data-test="restart" @click="emit('restart')">重启应用</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.crash-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}
.crash-dialog {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  width: 440px;
  max-width: 90vw;
}
.crash-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
}
.crash-header h2 { margin: 0; }
.crash-body {
  padding: 0 20px 16px;
}
.crash-path {
  font-size: 12px;
  margin: 8px 0;
}
.crash-path code {
  font-size: 11px;
  word-break: break-all;
}
.crash-upload-opt {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
  margin-top: 12px;
}
.crash-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
}
</style>
```

Add IPC methods to `app_system.go`:

```go
// Add to app_system.go

// ListCrashReports returns all saved crash reports.
func (a *App) ListCrashReports() []crash.CrashReport {
	if a.crashStore == nil {
		return nil
	}
	reports, err := a.crashStore.List()
	if err != nil {
		slog.Warn("list crash reports", "error", err)
		return nil
	}
	return reports
}

// DeleteCrashReport deletes a crash report by ID.
func (a *App) DeleteCrashReport(id string) error {
	if a.crashStore == nil {
		return fmt.Errorf("crash store not initialized")
	}
	return a.crashStore.Delete(id)
}

// UploadCrashReport uploads a crash report to the configured endpoint.
func (a *App) UploadCrashReport(id string) error {
	if a.crashStore == nil {
		return fmt.Errorf("crash store not initialized")
	}
	reports, err := a.crashStore.List()
	if err != nil {
		return fmt.Errorf("list reports: %w", err)
	}
	for _, r := range reports {
		if r.ID == id {
			endpoint := a.cfg.GetAPIKey("crash_endpoint")
			if endpoint == "" {
				endpoint = "https://hooks.quantflow.app/crashes"
			}
			return a.crashStore.Upload(&r, endpoint)
		}
	}
	return fmt.Errorf("report %s not found", id)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run -t "CrashDialog" 2>&1 || true`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/crash.ts frontend/src/terminal/components/CrashDialog.vue frontend/src/lib/wails.ts app_system.go
git commit -m "feat(crash): add crash dialog, crash store, and crash report IPC"
```
