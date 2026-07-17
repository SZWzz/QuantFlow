// internal/crash/reporter.go
package crash

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// crashLogDir holds the platform-specific crash log directory, set at init time
// by the platform files (darwin.go, linux.go, windows.go).
var crashLogDir string

// StartCrashHandler wires the Store into the crash handler pipeline. It calls
// StartHandler with a callback that saves the captured CrashReport via the
// provided Store and then exits the process.
//
// If store is nil, StartHandler is still called (so signal handlers are
// registered) but reports are only logged, not persisted.
func StartCrashHandler(store *Store) {
	StartHandler(func(report *CrashReport) {
		if store == nil {
			slog.Error("crash handler: no store configured, report not saved")
			return
		}
		path, err := store.Save(report)
		if err != nil {
			slog.Error("failed to save crash report", "error", err)
		} else {
			slog.Error("crash report saved", "path", path)
		}
	})

	// Cleanup old reports (>30 days) on a background goroutine.
	if store != nil {
		go func() {
			removed, err := store.Cleanup(30 * 24 * time.Hour)
			if err != nil {
				slog.Warn("crash cleanup failed", "error", err)
			} else if removed > 0 {
				slog.Info("cleaned up old crash reports", "count", removed)
			}
		}()
	}
}

// DefaultCrashDir returns the platform-specific default crash log directory.
// It uses crashLogDir set by init() in platform-specific files, falling back
// to a temp directory if the platform file was not compiled in.
func DefaultCrashDir() string {
	if crashLogDir != "" {
		return crashLogDir
	}
	// Fallback for platforms without explicit support
	return filepath.Join(os.TempDir(), "QuantFlow", "crashes")
}

// CrashDir returns the platform-specific directory for crash logs.
// This function is overridden by platform-specific files via init().
func CrashDir() string {
	return DefaultCrashDir()
}
