package adapters

import (
	"context"
	"testing"
)

func TestMootdxAdapter_Name(t *testing.T) {
	a := NewMootdxAdapter(nil)
	if a.Name() != "mootdx" {
		t.Errorf("Name() = %s, want mootdx", a.Name())
	}
}

func TestMootdxAdapter_Markets(t *testing.T) {
	a := NewMootdxAdapter(nil)
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
		t.Error("Mootdx should support CN market")
	}
}

func TestMootdxAdapter_RequiresAuth(t *testing.T) {
	a := NewMootdxAdapter(nil)
	if a.RequiresAuth() {
		t.Error("Mootdx should not require auth")
	}
}

func TestMootdxAdapter_IsAvailable_NoBridge(t *testing.T) {
	// IsAvailable is a cheap nil-check on the DataClient (no TDX round-trip).
	// A nil dataClient ⇒ unavailable. The non-nil ⇒ true case is structural
	// (return a.dataClient != nil) and is exercised at the registry level in
	// app_test.go, where a real *python.DataClient can be constructed.
	a := NewMootdxAdapter(nil)
	if a.IsAvailable(context.Background()) {
		t.Error("IsAvailable should be false when dataClient is nil")
	}
}

func TestMootdxAdapter_FetchQuote_NoBridge(t *testing.T) {
	a := NewMootdxAdapter(nil)
	_, err := a.FetchQuote(context.Background(), "600519")
	if err == nil {
		t.Error("FetchQuote should return error without Python bridge")
	}
}

func TestMootdxAdapter_FetchOHLCV_NoBridge(t *testing.T) {
	a := NewMootdxAdapter(nil)
	_, err := a.FetchOHLCV(context.Background(), "600519", "1D", 0, 0)
	if err == nil {
		t.Error("FetchOHLCV should return error without Python bridge")
	}
}
