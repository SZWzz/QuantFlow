package adapters

import (
	"testing"
)

func TestOKXAdapter_Name(t *testing.T) {
	a := NewOKXAdapter()
	if got := a.Name(); got != "okx" {
		t.Errorf("Name() = %q, want %q", got, "okx")
	}
}

func TestOKXAdapter_Markets(t *testing.T) {
	a := NewOKXAdapter()
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

func TestOKXAdapter_RequiresAuth(t *testing.T) {
	a := NewOKXAdapter()
	if got := a.RequiresAuth(); got != false {
		t.Errorf("RequiresAuth() = %v, want %v", got, false)
	}
}
