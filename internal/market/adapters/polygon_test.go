package adapters

import (
	"testing"
)

func TestPolygonAdapter_Name(t *testing.T) {
	a := NewPolygonAdapter(PolygonConfig{})
	if got := a.Name(); got != "polygon" {
		t.Errorf("Name() = %q, want %q", got, "polygon")
	}
}

func TestPolygonAdapter_Markets(t *testing.T) {
	a := NewPolygonAdapter(PolygonConfig{})
	want := []string{"US"}
	got := a.Markets()
	if len(got) != len(want) {
		t.Fatalf("Markets() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Markets()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPolygonAdapter_RequiresAuth(t *testing.T) {
	a := NewPolygonAdapter(PolygonConfig{})
	if got := a.RequiresAuth(); got != true {
		t.Errorf("RequiresAuth() = %v, want %v", got, true)
	}
}
