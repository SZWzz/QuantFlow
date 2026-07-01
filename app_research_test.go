package main

import "testing"

func TestDetectMarketForSymbol(t *testing.T) {
	tests := []struct{ symbol, want string }{
		{"600519", "CN"}, {"000001", "CN"}, {"300750", "CN"},
		{"00700", "HK"}, {"00700.HK", "HK"},
		{"AAPL", "US"}, {"MSFT", "US"}, {"TSLA", "US"},
	}
	for _, tt := range tests {
		got := detectMarketForSymbol(tt.symbol)
		if got != tt.want {
			t.Errorf("detectMarketForSymbol(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestDetectST(t *testing.T) {
	tests := []struct {
		symbol string
		want   bool
	}{
		{"600519", false}, {"*ST康得", true}, {"ST康得", true}, {"000001", false},
	}
	for _, tt := range tests {
		got := detectST(tt.symbol)
		if got != tt.want {
			t.Errorf("detectST(%q) = %v, want %v", tt.symbol, got, tt.want)
		}
	}
}
