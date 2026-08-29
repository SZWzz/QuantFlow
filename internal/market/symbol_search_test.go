package market

import (
	"context"
	"testing"
)

func TestFetchCNStockList(t *testing.T) {
	entries, err := fetchCNStockList(context.Background())
	if err != nil {
		// Accept network failure in CI; only fail if we get data that's clearly wrong
		t.Logf("stock list fetch failed (network?): %v", err)
		return
	}
	if len(entries) < 4000 {
		t.Errorf("expected >=4000 stocks, got %d", len(entries))
	}
	if len(entries) > 10000 {
		t.Errorf("expected <=10000 stocks, got %d", len(entries))
	}
	// Verify we have 贵州茅台
	found := false
	for _, e := range entries {
		if e.Code == "600519" && e.Name == "贵州茅台" {
			found = true
			break
		}
	}
	if !found {
		t.Error("600519 贵州茅台 not found in stock list")
	}
	t.Logf("fetched %d stocks, sample: %s %s", len(entries), entries[0].Code, entries[0].Name)
}

func TestPinyinAbbr(t *testing.T) {
	tests := []struct{ name, want string }{
		{"贵州茅台", "gzmt"},
		{"平安银行", "payx"},
		{"中国平安", "zgpa"},
		{"比亚迪", "byd"},
		{"宁德时代", "ndsd"},
		{"", ""},
		{"ABC", ""}, // no Chinese chars
	}
	for _, tt := range tests {
		got := pinyinAbbr(tt.name)
		if got != tt.want {
			t.Errorf("pinyinAbbr(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSymbolSearchService_Search(t *testing.T) {
	// Build with a small test set (don't depend on network in unit tests)
	svc := &SymbolSearchService{
		entries: []StockEntry{
			{Code: "600519", Name: "贵州茅台", Market: "SH", Pinyin: "gzmt"},
			{Code: "000001", Name: "平安银行", Market: "SZ", Pinyin: "payx"},
			{Code: "000002", Name: "万科A", Market: "SZ", Pinyin: "wka"},
			{Code: "600036", Name: "招商银行", Market: "SH", Pinyin: "zsyx"},
			{Code: "300750", Name: "宁德时代", Market: "SZ", Pinyin: "ndsd"},
			{Code: "600519", Name: "贵州茅台", Market: "SH", Pinyin: "gzmt"},
		},
	}

	tests := []struct {
		query string
		want  string // expected first result code
	}{
		{"600519", "600519"}, // exact code
		{"6005", "600519"},   // code prefix
		{"茅台", "600519"},     // name contains
		{"gzmt", "600519"},   // pinyin exact
		{"gzm", "600519"},    // pinyin prefix
		{"平安", "000001"},     // name contains
		{"ndsd", "300750"},   // pinyin
		{"招商", "600036"},     // name contains
	}

	for _, tt := range tests {
		results := svc.Search(tt.query, 5)
		if len(results) == 0 {
			t.Errorf("Search(%q) returned no results", tt.query)
			continue
		}
		if results[0].Code != tt.want {
			t.Errorf("Search(%q) first = %s, want %s", tt.query, results[0].Code, tt.want)
		}
	}

	// Test limit
	results := svc.Search("0", 2)
	if len(results) > 2 {
		t.Errorf("Search limit: got %d results, want <=2", len(results))
	}
}

func TestSymbolSearchService_Empty(t *testing.T) {
	svc := &SymbolSearchService{}
	results := svc.Search("600519", 10)
	if len(results) != 0 {
		t.Error("empty index should return empty results")
	}
}
