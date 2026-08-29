package market

import (
	"testing"
)

func TestNormalizeCN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
		wantCode string
		wantMkt  string
	}{
		// Plain 6-digit codes (market inferred)
		{name: "plain SH", input: "600519", wantOK: true, wantCode: "600519", wantMkt: "SH"},
		{name: "plain SZ 0", input: "000001", wantOK: true, wantCode: "000001", wantMkt: "SZ"},
		{name: "plain SZ 3", input: "300750", wantOK: true, wantCode: "300750", wantMkt: "SZ"},
		{name: "plain BJ 8", input: "830799", wantOK: true, wantCode: "830799", wantMkt: "BJ"},
		{name: "plain BJ 4", input: "430047", wantOK: true, wantCode: "430047", wantMkt: "BJ"},
		{name: "plain SH 5", input: "500001", wantOK: true, wantCode: "500001", wantMkt: "SH"},
		{name: "plain SH 9", input: "900948", wantOK: true, wantCode: "900948", wantMkt: "SH"},

		// With suffix
		{name: "suffix SH", input: "600519.SH", wantOK: true, wantCode: "600519", wantMkt: "SH"},
		{name: "suffix SS", input: "600519.SS", wantOK: true, wantCode: "600519", wantMkt: "SH"},
		{name: "suffix SZ", input: "000001.SZ", wantOK: true, wantCode: "000001", wantMkt: "SZ"},
		{name: "suffix BJ", input: "830799.BJ", wantOK: true, wantCode: "830799", wantMkt: "BJ"},
		{name: "suffix lowercase", input: "600519.sh", wantOK: true, wantCode: "600519", wantMkt: "SH"},

		// With prefix
		{name: "prefix sh", input: "sh600519", wantOK: true, wantCode: "600519", wantMkt: "SH"},
		{name: "prefix SH upper", input: "SH600519", wantOK: true, wantCode: "600519", wantMkt: "SH"},
		{name: "prefix sz", input: "sz000001", wantOK: true, wantCode: "000001", wantMkt: "SZ"},
		{name: "prefix bj", input: "bj830799", wantOK: true, wantCode: "830799", wantMkt: "BJ"},

		// With whitespace
		{name: "trim spaces", input: "  600519  ", wantOK: true, wantCode: "600519", wantMkt: "SH"},

		// Error cases
		{name: "empty", input: "", wantOK: false},
		{name: "too short", input: "60051", wantOK: false},
		{name: "too long", input: "6005199", wantOK: false},
		{name: "letters", input: "60A519", wantOK: false},
		{name: "random", input: "garbage", wantOK: false},
		{name: "US symbol", input: "AAPL", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NormalizeCN(tt.input)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("NormalizeCN(%q) unexpected error: %v", tt.input, err)
				}
				if id == nil {
					t.Fatalf("NormalizeCN(%q) returned nil", tt.input)
				}
				if id.Code != tt.wantCode {
					t.Errorf("Code = %q, want %q", id.Code, tt.wantCode)
				}
				if id.Market != tt.wantMkt {
					t.Errorf("Market = %q, want %q", id.Market, tt.wantMkt)
				}
			} else if err == nil {
				t.Errorf("NormalizeCN(%q) should have returned error", tt.input)
			}
		})
	}
}

func TestSymbolIdentity_Converters(t *testing.T) {
	// SH stock
	sh, _ := NormalizeCN("600519.SH")
	if sh == nil {
		t.Fatal("NormalizeCN failed")
	}

	if got := sh.ToEastMoney(); got != "1.600519" {
		t.Errorf("ToEastMoney = %q, want %q", got, "1.600519")
	}
	if got := sh.ToTencent(); got != "sh600519" {
		t.Errorf("ToTencent = %q, want %q", got, "sh600519")
	}
	if got := sh.ToSina(); got != "sh600519" {
		t.Errorf("ToSina = %q, want %q", got, "sh600519")
	}
	if got := sh.ToBaidu(); got != "600519" {
		t.Errorf("ToBaidu = %q, want %q", got, "600519")
	}
	if got := sh.ToMootdx(); got != "600519" {
		t.Errorf("ToMootdx = %q, want %q", got, "600519")
	}
	if got := sh.ToYahoo(); got != "600519.SS" {
		t.Errorf("ToYahoo = %q, want %q", got, "600519.SS")
	}
	if got := sh.ToPlain(); got != "600519" {
		t.Errorf("ToPlain = %q, want %q", got, "600519")
	}
	if got := sh.MarketCode(); got != "1" {
		t.Errorf("MarketCode = %q, want %q", got, "1")
	}
	if got := sh.String(); got != "600519.SH" {
		t.Errorf("String = %q, want %q", got, "600519.SH")
	}

	// SZ stock
	sz, _ := NormalizeCN("000001.SZ")
	if sz == nil {
		t.Fatal("NormalizeCN failed")
	}

	if got := sz.ToEastMoney(); got != "0.000001" {
		t.Errorf("ToEastMoney = %q, want %q", got, "0.000001")
	}
	if got := sz.ToTencent(); got != "sz000001" {
		t.Errorf("ToTencent = %q, want %q", got, "sz000001")
	}
	if got := sz.ToYahoo(); got != "000001.SZ" {
		t.Errorf("ToYahoo = %q, want %q", got, "000001.SZ")
	}
	if got := sz.MarketCode(); got != "0" {
		t.Errorf("MarketCode = %q, want %q", got, "0")
	}

	// BJ stock
	bj, _ := NormalizeCN("830799.BJ")
	if bj == nil {
		t.Fatal("NormalizeCN failed")
	}
	if got := bj.MarketCode(); got != "0" {
		t.Errorf("BJ MarketCode = %q, want %q", got, "0")
	}
}

func TestMarketFromCode(t *testing.T) {
	tests := []struct{ code, want string }{
		{"600519", "SH"},
		{"900948", "SH"},
		{"000001", "SZ"},
		{"300750", "SZ"},
		{"830799", "BJ"},
		{"430047", "BJ"},
		{"123456", ""}, // unknown prefix
	}
	for _, tt := range tests {
		got := marketFromCode(tt.code)
		if got != tt.want {
			t.Errorf("marketFromCode(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
