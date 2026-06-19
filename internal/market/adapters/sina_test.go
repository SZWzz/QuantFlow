package adapters

import (
	"testing"
)

func TestSinaAdapter_Name(t *testing.T) {
	a := NewSinaAdapter()
	if a.Name() != "sina" {
		t.Errorf("Name() = %s, want sina", a.Name())
	}
}

func TestSinaAdapter_Markets(t *testing.T) {
	a := NewSinaAdapter()
	markets := a.Markets()
	if len(markets) == 0 {
		t.Error("Markets() should not be empty")
	}
	hasCN := false
	for _, m := range markets {
		if m == "CN" {
			hasCN = true
		}
	}
	if !hasCN {
		t.Error("Sina should support CN market")
	}
}

func TestSinaAdapter_RequiresAuth(t *testing.T) {
	a := NewSinaAdapter()
	if a.RequiresAuth() {
		t.Error("Sina should not require auth")
	}
}
