package adapters

import (
	"testing"
)

func TestYahooAdapter_Name(t *testing.T) {
	a := NewYahooAdapter()
	if a.Name() != "yahoo" {
		t.Errorf("Name() = %s, want yahoo", a.Name())
	}
}

func TestYahooAdapter_Markets(t *testing.T) {
	a := NewYahooAdapter()
	markets := a.Markets()
	if len(markets) == 0 {
		t.Error("Markets() should not be empty")
	}
	hasUS := false
	for _, m := range markets {
		if m == "US" {
			hasUS = true
		}
	}
	if !hasUS {
		t.Error("Yahoo should support US market")
	}
}

func TestYahooAdapter_RequiresAuth(t *testing.T) {
	a := NewYahooAdapter()
	if a.RequiresAuth() {
		t.Error("Yahoo should not require auth")
	}
}

func TestSafeFloat(t *testing.T) {
	tests := []struct {
		name string
		arr  []float64
		i    int
		want float64
	}{
		{"in bounds", []float64{1.0, 2.0, 3.0}, 1, 2.0},
		{"out of bounds", []float64{1.0, 2.0}, 5, 0},
		{"empty slice", []float64{}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeFloat(tt.arr, tt.i); got != tt.want {
				t.Errorf("safeFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
