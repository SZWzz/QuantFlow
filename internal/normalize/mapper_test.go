package normalize

import (
	"testing"
)

func TestOrderStatusMapper_IBKR(t *testing.T) {
	m := NewOrderStatusMapper("ibkr")
	tests := []struct {
		input string
		want  string
	}{
		{"Submitted", "pending"},
		{"PreSubmitted", "pending"},
		{"Filled", "filled"},
		{"Cancelled", "cancelled"},
		{"ApiCancelled", "cancelled"},
		{"Inactive", "pending"},
		{"Unknown", "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderStatusMapper_Binance(t *testing.T) {
	m := NewOrderStatusMapper("binance")
	tests := []struct {
		input string
		want  string
	}{
		{"NEW", "pending"},
		{"PARTIALLY_FILLED", "pending"},
		{"FILLED", "filled"},
		{"CANCELED", "cancelled"},
		{"EXPIRED", "cancelled"},
		{"REJECTED", "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderStatusMapper_Alpaca(t *testing.T) {
	m := NewOrderStatusMapper("alpaca")
	tests := []struct {
		input string
		want  string
	}{
		{"accepted", "pending"},
		{"new", "pending"},
		{"partially_filled", "pending"},
		{"filled", "filled"},
		{"canceled", "cancelled"},
		{"expired", "cancelled"},
		{"rejected", "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderTypeMapper_IBKR(t *testing.T) {
	m := NewOrderTypeMapper("ibkr")
	tests := []struct {
		input string
		want  string
	}{
		{"MKT", "market"},
		{"LMT", "limit"},
		{"STP", "stop"},
		{"UNKNOWN", "market"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOHLCVMapper_Parse(t *testing.T) {
	m := NewOHLCVMapper("eastmoney", map[string]string{
		"symbol": "code",
		"date":   "trade_date",
		"open":   "opn",
		"high":   "hi",
		"low":    "lo",
		"close":  "cls",
		"volume": "vol",
	})

	raw := map[string]any{
		"code":       "000001",
		"trade_date": "2026-01-02",
		"opn":        "10.0",
		"hi":         "11.0",
		"lo":         "9.0",
		"cls":        "10.5",
		"vol":        100.0,
	}

	bar, err := m.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if bar.Symbol != "000001" {
		t.Errorf("Symbol = %q, want %q", bar.Symbol, "000001")
	}
	if bar.Open != 10.0 {
		t.Errorf("Open = %v, want 10.0", bar.Open)
	}
	if bar.Volume != 10000 {
		t.Errorf("Volume = %v, want 10000 (100手×100)", bar.Volume)
	}
}

func TestOHLCVMapper_MissingSymbol(t *testing.T) {
	m := NewOHLCVMapper("test", map[string]string{
		"symbol": "code",
		"open":   "opn",
	})
	_, err := m.Parse(map[string]any{"opn": 10.0})
	if err == nil {
		t.Fatal("expected error for missing symbol")
	}
}
