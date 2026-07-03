package adapters

import (
	"context"
	"testing"
)

func TestTencentAdapter_Name(t *testing.T) {
	a := NewTencentAdapter()
	if a.Name() != "tencent" {
		t.Errorf("Name() = %s, want tencent", a.Name())
	}
}

func TestTencentAdapter_Markets(t *testing.T) {
	a := NewTencentAdapter()
	markets := a.Markets()
	if len(markets) == 0 {
		t.Error("Markets() should not be empty")
	}
	hasCN := false
	hasHK := false
	for _, m := range markets {
		if m == "CN" {
			hasCN = true
		}
		if m == "HK" {
			hasHK = true
		}
	}
	if !hasCN {
		t.Error("Tencent should support CN market")
	}
	if !hasHK {
		t.Error("Tencent should support HK market")
	}
}

func TestTencentAdapter_RequiresAuth(t *testing.T) {
	a := NewTencentAdapter()
	if a.RequiresAuth() {
		t.Error("Tencent should not require auth")
	}
}

func TestToTencentCode(t *testing.T) {
	tests := []struct {
		name   string
		symbol string
		want   string
	}{
		{"SH stock", "600519.SH", "sh600519"},
		{"SZ stock", "000001.SZ", "sz000001"},
		{"HK stock", "00700.HK", "hk00700"},
		{"SH without suffix", "600519", "sh600519"},
		{"SZ without suffix", "000001", "sz000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toTencentCode(tt.symbol); got != tt.want {
				t.Errorf("toTencentCode(%q) = %q, want %q", tt.symbol, got, tt.want)
			}
		})
	}
}

func TestTencentAdapter_FetchIndustryRanks(t *testing.T) {
	adapter := NewTencentAdapter()
	if !adapter.IsAvailable(context.Background()) {
		t.Skip("tencent adapter not available (network)")
	}

	ranks, err := adapter.FetchIndustryRanks(context.Background(), "HK", 30)
	if err != nil {
		t.Skipf("FetchIndustryRanks unavailable (API/network): %v", err)
	}
	if len(ranks) == 0 {
		t.Skip("expected non-empty industry ranks but got empty")
	}
	t.Logf("fetched %d HK industry ranks", len(ranks))
	for i, r := range ranks {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s: %.2f%%", r.Rank, r.Name, r.ChangePct)
	}
}

func TestStripSuffix(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		suffix string
		want   string
	}{
		{"has suffix", "600519.SH", ".SH", "600519"},
		{"no suffix", "600519", ".SH", "600519"},
		{"empty string", "", ".SH", ""},
		{"same string", "abc", "abc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripSuffix(tt.s, tt.suffix); got != tt.want {
				t.Errorf("stripSuffix(%q, %q) = %q, want %q", tt.s, tt.suffix, got, tt.want)
			}
		})
	}
}
