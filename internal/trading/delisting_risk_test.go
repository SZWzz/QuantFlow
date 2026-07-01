package trading

import "testing"

func TestDetectBoard(t *testing.T) {
	tests := []struct{ symbol, want string }{
		{"600519", "主板"}, {"000001", "主板"},
		{"300750", "创业板"}, {"301188", "创业板"},
		{"688001", "科创板"}, {"689001", "科创板"},
		{"830001", "北交所"}, {"400001", "北交所"},
		{"00700", "未知"}, {"AAPL", "未知"},
	}
	for _, tt := range tests {
		got := DetectBoard(tt.symbol)
		if got != tt.want {
			t.Errorf("DetectBoard(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct{ v float64; want string }{
		{1.5e9, "15.00亿"}, {5e7, "5000.00万"},
		{1.23e8, "1.23亿"}, {-2.5e8, "-2.50亿"},
		{12345, "1.23万"}, {999, "999.00"},
	}
	for _, tt := range tests {
		got := formatValue(tt.v)
		if got != tt.want {
			t.Errorf("formatValue(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestExtractFinancialMetrics(t *testing.T) {
	jsonStr := `{
		"income": [{"period": "2026-03-31", "items": [
			{"item_title": "营业总收入", "item_value": "12500000000"},
			{"item_title": "净利润", "item_value": "3200000000"}
		]}],
		"balance": [{"period": "2026-03-31", "items": [
			{"item_title": "归属于母公司所有者的权益合计", "item_value": "25600000000"}
		]}],
		"cashflow": [{"period": "2026-03-31", "items": [
			{"item_title": "经营活动现金流量净额", "item_value": "4500000000"}
		]}]
	}`
	m, err := ExtractFinancialMetrics(jsonStr)
	if err != nil {
		t.Fatalf("ExtractFinancialMetrics() error: %v", err)
	}
	if m.Revenue != 12.5e9 {
		t.Errorf("Revenue = %v, want 12.5e9", m.Revenue)
	}
	if m.NetProfit != 3.2e9 {
		t.Errorf("NetProfit = %v, want 3.2e9", m.NetProfit)
	}
	if m.NetAssets != 25.6e9 {
		t.Errorf("NetAssets = %v, want 25.6e9", m.NetAssets)
	}
	if m.CashFlow != 4.5e9 {
		t.Errorf("CashFlow = %v, want 4.5e9", m.CashFlow)
	}
}

func TestExtractFinancialMetricsEmpty(t *testing.T) {
	_, err := ExtractFinancialMetrics(`{"income":[],"balance":[],"cashflow":[]}`)
	if err == nil {
		t.Error("expected error for empty financial data")
	}
}
