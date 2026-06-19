package ml

import (
	"math"
	"testing"
)

func TestComputeIC_PerfectCorrelation(t *testing.T) {
	preds := []float64{1, 2, 3, 4, 5}
	actuals := []float64{2, 4, 6, 8, 10}
	ic := ComputeIC(preds, actuals)
	if math.Abs(ic-1.0) > 0.001 {
		t.Errorf("expected IC=1.0, got %f", ic)
	}
}

func TestComputeIC_AntiCorrelation(t *testing.T) {
	preds := []float64{5, 4, 3, 2, 1}
	actuals := []float64{1, 2, 3, 4, 5}
	ic := ComputeIC(preds, actuals)
	if math.Abs(ic+1.0) > 0.001 {
		t.Errorf("expected IC=-1.0, got %f", ic)
	}
}

func TestComputeIR(t *testing.T) {
	icSeries := []float64{0.05, 0.08, 0.06, 0.07, 0.09}
	ir := ComputeIR(icSeries)
	if ir <= 0 {
		t.Errorf("IR should be positive, got %f", ir)
	}
}

func TestComputeSharpe(t *testing.T) {
	returns := []float64{0.01, 0.02, -0.01, 0.03, 0.0, 0.01, 0.02}
	sharpe := ComputeSharpe(returns)
	if sharpe <= 0 {
		t.Errorf("expected positive Sharpe for positive returns, got %f", sharpe)
	}
}
