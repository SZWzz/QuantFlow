package adapters

import (
	"testing"
)

func TestTuShareAdapter_Name(t *testing.T) {
	a := NewTuShareAdapter()
	if got := a.Name(); got != "tushare" {
		t.Errorf("Name() = %q, want %q", got, "tushare")
	}
}

func TestTuShareAdapter_Markets(t *testing.T) {
	a := NewTuShareAdapter()
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

func TestTuShareAdapter_RequiresAuth(t *testing.T) {
	a := NewTuShareAdapter()
	if got := a.RequiresAuth(); got != true {
		t.Errorf("RequiresAuth() = %v, want %v", got, true)
	}
}

func TestZipFieldsAndItems(t *testing.T) {
	fields := []string{"ts_code", "trade_date", "close"}
	items := [][]any{
		{"000001.SZ", "20240601", 12.5},
		{"600519.SH", "20240601", 1780.0},
	}
	result := zipFieldsAndItems(fields, items)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0]["close"] != 12.5 {
		t.Errorf("result[0][\"close\"] = %v, want 12.5", result[0]["close"])
	}
	if result[1]["ts_code"] != "600519.SH" {
		t.Errorf("result[1][\"ts_code\"] = %v, want \"600519.SH\"", result[1]["ts_code"])
	}
}

func TestZipFieldsAndItems_Empty(t *testing.T) {
	result := zipFieldsAndItems(nil, nil)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestZipFieldsAndItems_MismatchedRow(t *testing.T) {
	fields := []string{"a", "b", "c"}
	items := [][]any{{"x"}} // too short
	result := zipFieldsAndItems(fields, items)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0 (short row skipped)", len(result))
	}
}
