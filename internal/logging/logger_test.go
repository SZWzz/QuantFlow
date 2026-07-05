package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestSetupDoesNotPanic(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown level defaults to info", "unknown"},
		{"empty string defaults to info", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Setup(tt.level)
			slog.Info("alive")
		})
	}
}

func TestSetupOutput(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	Setup("info")
	slog.Info("logger test message")
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	if !strings.Contains(output, "logger test message") {
		t.Errorf("expected log output to contain 'logger test message', got: %s", output)
	}
}

func TestSetupLevelFilter(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	Setup("error")
	slog.Info("this should be suppressed")
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	if strings.Contains(output, "this should be suppressed") {
		t.Error("info-level message should be suppressed when level is error")
	}
}
