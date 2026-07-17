// internal/crash/capture.go
package crash

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

// osExit is overridable in tests to prevent os.Exit from killing the test binary.
var osExit = os.Exit

// crashHandler is the registered callback invoked when a crash is captured.
var crashHandler func(report *CrashReport)

// StartHandler registers OS signal handlers for common crash signals (SIGABRT,
// SIGSEGV, SIGILL, SIGBUS) and enables panic-on-fault so that memory access
// violations become recoverable panics rather than hard crashes. When a signal
// is received, it constructs a CrashReport and calls the provided handler
// before exiting the process.
//
// The handler should be called once at application startup. Passing nil for
// handler is safe — reports will be logged but not forwarded.
func StartHandler(handler func(report *CrashReport)) {
	crashHandler = handler
	debug.SetPanicOnFault(true)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGABRT, syscall.SIGSEGV, syscall.SIGILL, syscall.SIGBUS)

	go func() {
		for sig := range c {
			slog.Error("crash signal received", "signal", sig)
			report := collectReport(fmt.Sprintf("signal: %v", sig))
			if crashHandler != nil {
				crashHandler(report)
			}
			osExit(1)
		}
	}()
}

// CapturePanic should be called via defer at the top of main and key goroutines.
// It recovers from panics (including those originating from memory faults when
// debug.SetPanicOnFault is enabled), assembles a CrashReport, and invokes the
// registered crash handler before exiting the process.
//
// Usage:
//
//	defer crash.CapturePanic()
func CapturePanic() {
	if r := recover(); r != nil {
		report := collectReport(fmt.Sprintf("%v", r))
		if crashHandler != nil {
			crashHandler(report)
		}
		osExit(1)
	}
}

// collectReport assembles a CrashReport from a panic value, capturing the
// current goroutine stack trace via runtime/debug.Stack.
func collectReport(panicVal string) *CrashReport {
	stack := string(debug.Stack())
	return NewCrashReport(panicVal, stack, nil, AppState{})
}
