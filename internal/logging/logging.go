package logging

import (
	"log/slog"
	"os"
)

var Ring = NewRingBuffer(5000)

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
	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	handler := newDualHandler(textHandler, Ring)
	slog.SetDefault(slog.New(handler))
}
