package adapters

import (
	"testing"
)

func TestEastMoneyAdapter_Name(t *testing.T) {
	a := NewEastMoneyAdapter()
	if a.Name() != "eastmoney" {
		t.Errorf("Name() = %s, want eastmoney", a.Name())
	}
}

func TestEastMoneyAdapter_Markets(t *testing.T) {
	a := NewEastMoneyAdapter()
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
		t.Error("EastMoney should support CN market")
	}
}

func TestEastMoneyAdapter_RequiresAuth(t *testing.T) {
	a := NewEastMoneyAdapter()
	if a.RequiresAuth() {
		t.Error("EastMoney should not require auth")
	}
}

func TestToEastMoneySecID(t *testing.T) {
	tests := []struct {
		name   string
		symbol string
		want   string
	}{
		{"SH stock", "600519.SH", "1.600519"},
		{"SZ stock", "000001.SZ", "0.000001"},
		{"SH without suffix", "600519", "1.600519"},
		{"SZ without suffix", "000001", "0.000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toEastMoneySecID(tt.symbol); got != tt.want {
				t.Errorf("toEastMoneySecID(%q) = %q, want %q", tt.symbol, got, tt.want)
			}
		})
	}
}
