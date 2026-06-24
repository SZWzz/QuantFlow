package portfolio

import (
	"math"
	"testing"
)

func TestCorrelationMatrix_Diagonal(t *testing.T) {
	returns := map[string][]float64{
		"A": {0.01, -0.02, 0.03, -0.01, 0.02},
	}
	m := CorrelationMatrix(returns)
	if v := m["A"]["A"]; math.Abs(v-1.0) > 1e-9 {
		t.Errorf("diagonal = %f, want 1.0", v)
	}
}

func TestCorrelationMatrix_PerfectPositive(t *testing.T) {
	// Identical series → correlation = 1
	r := []float64{0.01, 0.02, 0.03, 0.04, 0.05}
	returns := map[string][]float64{"A": r, "B": r}
	m := CorrelationMatrix(returns)
	if v := m["A"]["B"]; math.Abs(v-1.0) > 1e-6 {
		t.Errorf("corr identical = %f, want 1.0", v)
	}
}

func TestCorrelationMatrix_Empty(t *testing.T) {
	returns := map[string][]float64{"A": {0.01}}
	m := CorrelationMatrix(returns)
	if v := m["A"]["A"]; math.Abs(v-1.0) > 1e-9 {
		t.Errorf("diagonal single = %f, want 1.0", v)
	}
}

func TestReturnDistribution(t *testing.T) {
	returns := []float64{-0.02, -0.01, 0, 0.01, 0.02, 0.03}
	bins, counts := ReturnDistribution(returns, 3)
	if len(bins) != 3 || len(counts) != 3 {
		t.Fatalf("expected 3 bins, got %d bins, %d counts", len(bins), len(counts))
	}
	total := 0.0
	for _, c := range counts {
		total += c
	}
	if total != 6 {
		t.Errorf("total count = %f, want 6", total)
	}
}

func TestReturnDistribution_Empty(t *testing.T) {
	bins, counts := ReturnDistribution([]float64{}, 5)
	if bins != nil || counts != nil {
		t.Error("empty input should return nil,nil")
	}
}

func TestReturnDistribution_AllSame(t *testing.T) {
	returns := []float64{0.01, 0.01, 0.01, 0.01}
	bins, counts := ReturnDistribution(returns, 3)
	if len(bins) != 1 || counts[0] != 4 {
		t.Errorf("all-same: bins=%v counts=%v", bins, counts)
	}
}

func TestVolatilitySurface(t *testing.T) {
	// 300 random-ish returns
	returns := make([]float64, 300)
	for i := range returns {
		returns[i] = 0.001 * float64(i%21-10) // small oscillation
	}
	windows := []int{5, 20, 60}
	surface := VolatilitySurface(returns, windows)
	if len(surface) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(surface))
	}
	for _, row := range surface {
		if row[0] <= 0 || row[1] <= 0 {
			t.Errorf("row %v: window or vol should be positive", row)
		}
	}
}

func TestVolatilitySurface_InsufficientData(t *testing.T) {
	returns := []float64{0.01, 0.02}
	surface := VolatilitySurface(returns, []int{5, 20})
	if len(surface) != 0 {
		t.Errorf("insufficient data should return empty, got %d rows", len(surface))
	}
}
