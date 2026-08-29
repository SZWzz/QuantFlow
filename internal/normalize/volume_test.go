package normalize

import (
	"testing"
)

func TestNormalizeVolume_KnownSources(t *testing.T) {
	tests := []struct {
		source string
		input  float64
		want   float64
	}{
		{"eastmoney", 100, 10000},
		{"sina", 100, 10000},
		{"tencent", 100, 10000},
		{"tushare", 100, 10000},
		{"mootdx", 100, 10000},
		{"baidu", 100, 10000},
		{"yahoo", 100, 100},   // US source, no multiplier
		{"binance", 1.5, 1.5}, // crypto, no multiplier
		{"unknown", 100, 100}, // unknown source
		{"", 100, 100},        // empty source
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := NormalizeVolume(tt.source, tt.input)
			if got != tt.want {
				t.Errorf("NormalizeVolume(%q, %v) = %v, want %v", tt.source, tt.input, got, tt.want)
			}
		})
	}
}

func TestVolumeSources_NotEmpty(t *testing.T) {
	sources := VolumeSources()
	if len(sources) == 0 {
		t.Fatal("VolumeSources() returned empty")
	}
}
