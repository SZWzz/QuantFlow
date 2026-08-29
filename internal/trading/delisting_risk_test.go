package trading

import "testing"

func TestDetectBoard(t *testing.T) {
	tests := []struct{ symbol, want string }{
		{"600519", "主板"},
		{"000001", "主板"},
		{"300750", "创业板"},
		{"301188", "创业板"},
		{"688001", "科创板"},
		{"689001", "科创板"},
		{"830001", "北交所"},
		{"400001", "北交所"},
		{"00700", "未知"},
		{"AAPL", "未知"},
	}
	for _, tt := range tests {
		got := DetectBoard(tt.symbol)
		if got != tt.want {
			t.Errorf("DetectBoard(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{1.5e9, "15.00亿"},
		{5e7, "5000.00万"},
		{1.23e8, "1.23亿"},
		{-2.5e8, "-2.50亿"},
		{12345, "1.23万"},
		{999, "999.00"},
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

func TestAssessCN_Safe(t *testing.T) {
	m := &FinancialMetrics{Revenue: 12.5e9, NetProfit: 3.2e9, NetAssets: 25e9}
	cats := AssessCN(m, "主板", 15.0, 68e8, 5e6, 10e8)
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Status == "danger" {
				t.Errorf("unexpected danger: %s/%s", cat.Name, it.Indicator)
			}
		}
	}
}

func TestAssessCN_RevenueDanger(t *testing.T) {
	m := &FinancialMetrics{Revenue: 2e8, NetProfit: -5e7, NetAssets: 10e8}
	cats := AssessCN(m, "主板", 15.0, 68e8, 5e6, 10e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Indicator == "营收+净利润组合" && it.Status == "danger" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected danger for revenue<3亿 + negative profit on main board")
	}
}

func TestAssessCN_NetAssetsDanger(t *testing.T) {
	m := &FinancialMetrics{Revenue: 12.5e9, NetProfit: 3.2e9, NetAssets: -1e8}
	cats := AssessCN(m, "主板", 15.0, 68e8, 5e6, 10e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Indicator == "净资产" && it.Status == "danger" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected danger for negative net assets")
	}
}

func TestAssessCN_MarketCapDanger(t *testing.T) {
	m := &FinancialMetrics{Revenue: 12.5e9, NetProfit: 3.2e9, NetAssets: 25e9}
	cats := AssessCN(m, "主板", 15.0, 4e8, 5e6, 10e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Indicator == "总市值" && it.Status == "danger" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected danger for market cap < 5亿 on main board")
	}
}

func TestAssessCN_PriceWarn(t *testing.T) {
	m := &FinancialMetrics{Revenue: 12.5e9, NetProfit: 3.2e9, NetAssets: 25e9}
	cats := AssessCN(m, "主板", 1.15, 68e8, 5e6, 10e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Indicator == "面值(收盘价)" && it.Status == "warn" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected warn for price near 1元")
	}
}

func TestAssessCN_ChiNextThreshold(t *testing.T) {
	m := &FinancialMetrics{Revenue: 5e7, NetProfit: -1e7, NetAssets: 5e8}
	cats := AssessCN(m, "创业板", 10.0, 20e8, 1e6, 5e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Indicator == "营收+净利润组合" && it.Status == "danger" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected danger for revenue<1亿 + negative profit on ChiNext")
	}
}

func TestAssessHK_Danger(t *testing.T) {
	cats := AssessHK(0.8, 4e8, 1e5, 10e8)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Status == "warn" || it.Status == "danger" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected warning for HK penny stock + low market cap")
	}
}

func TestAssessUS_Warn(t *testing.T) {
	cats := AssessUS(0.9, 4e7)
	found := false
	for _, cat := range cats {
		for _, it := range cat.Items {
			if it.Status == "warn" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected warn for US sub-$1 + low market cap")
	}
}

func TestAssessOverall(t *testing.T) {
	g := AssessOverall([]DelistingCategory{{Level: "green", Items: []DelistingItem{{Status: "safe"}}}})
	if g != "low" {
		t.Errorf("green -> %q, want low", g)
	}
	y := AssessOverall([]DelistingCategory{{Level: "yellow", Items: []DelistingItem{{Status: "warn"}}}})
	if y != "medium" {
		t.Errorf("yellow -> %q, want medium", y)
	}
	r := AssessOverall([]DelistingCategory{{Level: "red", Items: []DelistingItem{{Status: "danger"}}}})
	if r != "high" {
		t.Errorf("red -> %q, want high", r)
	}
}
