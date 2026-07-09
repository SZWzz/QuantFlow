package backtest

import "testing"

func TestPriceLimitFor_BoardRules(t *testing.T) {
	tests := []struct {
		symbol string
		ratio  float64
	}{
		{"600519", 0.10}, // main board
		{"000001", 0.10}, // main board
		{"300750", 0.20}, // ChiNext
		{"301001", 0.20}, // ChiNext
		{"688001", 0.20}, // STAR
		{"689001", 0.20}, // STAR
		{"830001", 0.30}, // BSE
		{"400001", 0.30}, // BSE (old BSE code)
		{"999999", 0.10}, // unknown → safe default
	}
	for _, tt := range tests {
		got := PriceLimitFor(tt.symbol)
		if got.Ratio != tt.ratio {
			t.Errorf("PriceLimitFor(%s) = %v, want ratio %v", tt.symbol, got, tt.ratio)
		}
	}
}
