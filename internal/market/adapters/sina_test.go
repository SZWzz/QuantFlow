package adapters

import (
	"context"
	"testing"
	"time"
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

func TestSinaAdapter_Markets_IncludesHK(t *testing.T) {
	a := NewSinaAdapter()
	markets := a.Markets()
	hasHK := false
	for _, m := range markets {
		if m == "HK" {
			hasHK = true
		}
	}
	if !hasHK {
		t.Error("Sina should support HK market")
	}
}

func TestSinaAdapter_FetchHKQuote(t *testing.T) {
	a := NewSinaAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("sina API not available")
	}

	snap, err := a.FetchQuote(ctx, "00700")
	if err != nil {
		t.Fatalf("FetchQuote HK error: %v", err)
	}
	if snap.Last == 0 {
		t.Error("HK last price should be non-zero")
	}
	t.Logf("Tencent(00700): last=%.2f change=%.2f%% vol=%.0f", snap.Last, snap.ChangePct, snap.Volume)
}
