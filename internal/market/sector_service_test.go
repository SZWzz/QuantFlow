package market

import (
	"testing"
)

func TestComputePEPercentile(t *testing.T) {
	tests := []struct {
		current  float64
		history  []float64
		expected float64
	}{
		{0, nil, 50},
		{25, []float64{10, 15, 20, 25, 30, 35, 40}, 57.14},  // rank 4/7
		{10, []float64{10, 15, 20, 25, 30, 35, 40}, 14.29},  // rank 1/7
		{40, []float64{10, 15, 20, 25, 30, 35, 40}, 100.0},  // rank 7/7
	}

	for _, tt := range tests {
		got := computePEPercentile(tt.current, tt.history)
		if got != tt.expected {
			t.Errorf("computePEPercentile(%f, %v) = %f, want %f", tt.current, tt.history, got, tt.expected)
		}
	}
}
