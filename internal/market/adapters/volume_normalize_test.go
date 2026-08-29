package adapters

import (
	"quantflow/internal/normalize"
	"testing"
)

// TestAllCNAdaptersNormalizeVolume verifies all A-share adapters are mapped
// in NormalizeVolume with multiplier 100.
func TestAllCNAdaptersNormalizeVolume(t *testing.T) {
	tests := []struct {
		adapterName string
		rawVolume   float64
		expected    float64
	}{
		{"eastmoney", 1000, 100000},
		{"sina", 1000, 100000},
		{"tencent", 1000, 100000},
		{"tushare", 1000, 100000},
		{"mootdx", 1000, 100000},
		{"baidu", 1000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.adapterName, func(t *testing.T) {
			got := normalize.NormalizeVolume(tt.adapterName, tt.rawVolume)
			if got != tt.expected {
				t.Errorf("NormalizeVolume(%q, %v) = %v, want %v",
					tt.adapterName, tt.rawVolume, got, tt.expected)
			}
		})
	}
}

// TestNonCNAdaptersNotNormalized verifies non-A-share adapters are not normalized.
func TestNonCNAdaptersNotNormalized(t *testing.T) {
	tests := []string{
		"binance", "yahoo", "finnhub", "polygon", "alpaca",
		"gateio", "okx", "coingecko", "unknown",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalize.NormalizeVolume(name, 1000)
			if got != 1000 {
				t.Errorf("NormalizeVolume(%q, 1000) = %v, want 1000 (no multiplier)", name, got)
			}
		})
	}
}

// TestVolumeMultiplierConsistency verifies VolumeMultiplier and NormalizeVolume agree.
func TestVolumeMultiplierConsistency(t *testing.T) {
	for _, source := range normalize.VolumeSources() {
		mult := normalize.VolumeMultiplier(source)
		got := normalize.NormalizeVolume(source, 1)
		if got != mult {
			t.Errorf("mismatch for %q: NormalizeVolume(1)=%v, VolumeMultiplier=%v",
				source, got, mult)
		}
	}
}
