package adapters

import (
	"testing"
)

func TestBaiduAdapter_Name(t *testing.T) {
	a := NewBaiduAdapter()
	if got := a.Name(); got != "baidu" {
		t.Errorf("Name() = %q, want %q", got, "baidu")
	}
}

func TestBaiduAdapter_Markets(t *testing.T) {
	a := NewBaiduAdapter()
	want := []string{"CN"}
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

func TestBaiduAdapter_RequiresAuth(t *testing.T) {
	a := NewBaiduAdapter()
	if got := a.RequiresAuth(); got != false {
		t.Errorf("RequiresAuth() = %v, want %v", got, false)
	}
}
