package normalize

import (
	"testing"
)

func TestOHLCVBar_Fields(t *testing.T) {
	bar := OHLCVBar{
		Symbol: "000001", Date: "2026-01-02",
		Open: 10.0, High: 11.0, Low: 9.0, Close: 10.5, Volume: 100000,
	}
	if bar.Symbol != "000001" {
		t.Errorf("Symbol = %q, want %q", bar.Symbol, "000001")
	}
	if bar.Close != 10.5 {
		t.Errorf("Close = %v, want 10.5", bar.Close)
	}
}
