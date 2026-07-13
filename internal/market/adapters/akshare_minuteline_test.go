package adapters

import (
	"context"
	"testing"
)

func TestAKShareMinuteAdapter_Name(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	if a.Name() != "akshare_hk_minute" {
		t.Errorf("Name() = %s, want akshare_hk_minute", a.Name())
	}
}

func TestAKShareMinuteAdapter_Markets(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	mkts := a.Markets()
	if len(mkts) != 1 || mkts[0] != "HK" {
		t.Errorf("Markets() = %v, want [HK]", mkts)
	}
}

func TestAKShareMinuteAdapter_IsAvailable_NilClient(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	if a.IsAvailable(context.Background()) {
		t.Error("expected false when client is nil")
	}
}

func TestAKShareMinuteAdapter_FetchMinuteLine_NoClient(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	_, err := a.FetchMinuteLine("00700")
	if err == nil {
		t.Error("FetchMinuteLine should error without Python bridge")
	}
}

func TestAKShareMinuteAdapter_FetchQuote_NotImplemented(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	_, err := a.FetchQuote(context.Background(), "00700")
	if err == nil {
		t.Error("expected error for FetchQuote")
	}
}

func TestAKShareMinuteAdapter_RequiresAuth(t *testing.T) {
	a := NewAKShareMinuteAdapter(nil)
	if a.RequiresAuth() {
		t.Error("AKShare minute adapter should not require auth")
	}
}
