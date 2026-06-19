package adapters

import (
	"testing"
)

func TestBinanceAdapter_Name(t *testing.T) {
	a := NewBinanceAdapter()
	if got := a.Name(); got != "binance" {
		t.Errorf("Name() = %q, want %q", got, "binance")
	}
}

func TestBinanceAdapter_Markets(t *testing.T) {
	a := NewBinanceAdapter()
	markets := a.Markets()
	found := false
	for _, m := range markets {
		if m == "CRYPTO" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Markets() = %v, want it to include %q", markets, "CRYPTO")
	}
}

func TestBinanceAdapter_RequiresAuth(t *testing.T) {
	a := NewBinanceAdapter()
	if got := a.RequiresAuth(); got != false {
		t.Errorf("RequiresAuth() = %v, want %v", got, false)
	}
}
