package market

import "testing"

func TestNormalizeInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Daily
		{"1d", "1D"}, {"1D", "1D"}, {"day", "1D"}, {"daily", "1D"},
		// Weekly
		{"1w", "1W"}, {"1W", "1W"}, {"week", "1W"}, {"weekly", "1W"},
		// Monthly
		{"1M", "1M"}, {"1month", "1M"}, {"monthly", "1M"},
		// Minute-level (lowercase)
		{"1m", "1m"}, {"1min", "1m"},
		{"5m", "5m"}, {"5min", "5m"},
		{"15m", "15m"}, {"15min", "15m"},
		{"30m", "30m"}, {"30min", "30m"},
		// Hourly
		{"1h", "1h"}, {"1hour", "1h"}, {"hourly", "1h"},
		{"4h", "4h"}, {"4hour", "4h"},
		// Passthrough for unknown values
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizeInterval(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeInterval(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
