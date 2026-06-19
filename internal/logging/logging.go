// Package logging provides a thin wrapper around slog for application-level logging.
package logging

import (
	"log/slog"
	"os"
)

// Setup configures the default slog logger with the given level.
// Valid levels: "debug", "info", "warn", "error".
func Setup(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
