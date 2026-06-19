package adapters

import (
	"testing"
)

func TestCoinGeckoAdapter_Name(t *testing.T) {
	a := NewCoinGeckoAdapter()
	if got := a.Name(); got != "coingecko" {
		t.Errorf("Name() = %q, want %q", got, "coingecko")
	}
}

func TestCoinGeckoAdapter_Markets(t *testing.T) {
	a := NewCoinGeckoAdapter()
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

func TestCoinGeckoAdapter_RequiresAuth(t *testing.T) {
	a := NewCoinGeckoAdapter()
	if got := a.RequiresAuth(); got != false {
		t.Errorf("RequiresAuth() = %v, want %v", got, false)
	}
}
