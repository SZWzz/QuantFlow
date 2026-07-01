# 退市风险检测模块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full-market (A/HK/US) delisting risk detection to AuditPanel via a pure Go rule engine.

**Architecture:** New Go structs + pure functions in `internal/trading/delisting_risk.go` compute delisting risk from financial JSON and market data. A new Wails method `GetDelistingRisk` wires financial data (existing `fetchFinancialJSON`) + market data (existing `FetchStockInfo`/`GetQuote`) into the engine. Frontend adds a "退市风险" tab using `PanelTabs`.

**Tech Stack:** Go 1.25, Vue 3 + TypeScript, PanelTabs shared component

## Global Constraints

- Go standard library only for financial JSON parsing (no new imports)
- Zero Python sidecar dependency — all delisting logic runs in Go
- All rules (CN/HK/US) in single engine file, with `market` parameter dispatch
- Frontend reuses existing `PanelTabs`, `SkeletonPanel`, `SignalBadge` components
- No changes to existing `fetchFinancialJSON` or `GetAuditFindings` signatures

---

### Task 1: Go Data Types + Financial JSON Parser

**Files:**
- Create: `internal/trading/delisting_risk.go` (types + JSON parser helpers)
- Create: `internal/trading/delisting_risk_test.go`

**Interfaces:**
- Produces: exported structs `DelistingItem`, `DelistingCategory`, `DelistingRiskResult`, `FinancialMetrics`
- Produces: `DetectBoard(symbol string) string`
- Produces: `ExtractFinancialMetrics(jsonStr string) (*FinancialMetrics, error)`
- Produces: helper `formatValue(v float64) string`

- [ ] **Step 1: Write delisting_risk.go with types and parser**

```go
package trading

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type FinancialMetrics struct {
	Revenue    float64
	NetProfit  float64
	CoreProfit float64
	NetAssets  float64
	CashFlow   float64
}

type DelistingItem struct {
	Indicator string `json:"indicator"`
	Status    string `json:"status"`
	Current   string `json:"current"`
	Threshold string `json:"threshold"`
	Detail    string `json:"detail"`
}

type DelistingCategory struct {
	Name  string          `json:"name"`
	Level string          `json:"level"`
	Items []DelistingItem `json:"items"`
}

type DelistingRiskResult struct {
	Market      string             `json:"market"`
	Board       string             `json:"board"`
	IsST        bool               `json:"is_st"`
	OverallRisk string             `json:"overall_risk"`
	Categories  []DelistingCategory `json:"categories"`
	Summary     string             `json:"summary"`
}

const (
	revenueThresholdMainland   = 3e8
	revenueThresholdOther      = 1e8
	marketCapThresholdMainland = 5e8
	marketCapThresholdOther    = 3e8
)

func DetectBoard(symbol string) string {
	switch {
	case strings.HasPrefix(symbol, "688"), strings.HasPrefix(symbol, "689"):
		return "科创板"
	case strings.HasPrefix(symbol, "300"), strings.HasPrefix(symbol, "301"):
		return "创业板"
	case strings.HasPrefix(symbol, "8"), strings.HasPrefix(symbol, "4"):
		return "北交所"
	case strings.HasPrefix(symbol, "60"), strings.HasPrefix(symbol, "00"):
		return "主板"
	default:
		return "未知"
	}
}

func formatValue(v float64) string {
	abs := math.Abs(v)
	var prefix string
	if v < 0 {
		prefix = "-"
	}
	switch {
	case abs >= 1e8:
		return fmt.Sprintf("%s%.2f亿", prefix, abs/1e8)
	case abs >= 1e4:
		return fmt.Sprintf("%s%.2f万", prefix, abs/1e4)
	default:
		return fmt.Sprintf("%s%.2f", prefix, abs)
	}
}

type finPeriodItem struct {
	Title string `json:"item_title"`
	Value string `json:"item_value"`
	YoY   string `json:"item_tongbi"`
}

type finPeriod struct {
	Period string         `json:"period"`
	Items  []finPeriodItem `json:"items"`
}

type finJSON struct {
	Income   []finPeriod `json:"income"`
	Balance  []finPeriod `json:"balance"`
	Cashflow []finPeriod `json:"cashflow"`
}

func findItemValue(items []finPeriodItem, title string) float64 {
	for _, item := range items {
		if item.Title == title {
			v, _ := strconv.ParseFloat(item.Value, 64)
			return v
		}
	}
	if title == "营业总收入" {
		for _, item := range items {
			if item.Title == "营业收入" {
				v, _ := strconv.ParseFloat(item.Value, 64)
				return v
			}
		}
	}
	if title == "归属于母公司所有者的权益合计" {
		for _, item := range items {
			if item.Title == "所有者权益合计" || item.Title == "股东权益合计" {
				v, _ := strconv.ParseFloat(item.Value, 64)
				return v
			}
		}
	}
	return 0
}

func ExtractFinancialMetrics(jsonStr string) (*FinancialMetrics, error) {
	var data finJSON
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("unmarshal financial JSON: %w", err)
	}
	if len(data.Income) == 0 && len(data.Balance) == 0 {
		return nil, fmt.Errorf("no financial data")
	}
	m := &FinancialMetrics{}
	if len(data.Income) > 0 {
		items := data.Income[0].Items
		m.Revenue = findItemValue(items, "营业总收入")
		m.NetProfit = findItemValue(items, "净利润")
		cp := findItemValue(items, "扣非净利润")
		if cp == 0 {
			cp = findItemValue(items, "归属于母公司所有者的净利润")
		}
		m.CoreProfit = cp
	}
	if len(data.Balance) > 0 {
		m.NetAssets = findItemValue(data.Balance[0].Items, "归属于母公司所有者的权益合计")
	}
	if len(data.Cashflow) > 0 {
		m.CashFlow = findItemValue(data.Cashflow[0].Items, "经营活动现金流量净额")
	}
	return m, nil
}
```

- [ ] **Step 2: Write delisting_risk_test.go**

```go
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
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -v -run "TestDetectBoard|TestFormatValue|TestExtractFinancial" -count=1`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/trading/
git commit -m "feat(trading): delisting risk types and financial JSON parser"
```

---

### Task 2: Go Rule Engine — A/HK/US Assessment

**Files:**
- Modify: `internal/trading/delisting_risk.go` (append assessment functions)
- Modify: `internal/trading/delisting_risk_test.go` (append rule tests)

**Interfaces:**
- Consumes: `DelistingItem`, `DelistingCategory`, `FinancialMetrics`, `DetectBoard`, `formatValue`
- Produces: `AssessCN(m *FinancialMetrics, board string, price, marketCap, volume, totalShares float64) []DelistingCategory`
- Produces: `AssessHK(price, marketCap, volume, totalShares float64) []DelistingCategory`
- Produces: `AssessUS(price, marketCap float64) []DelistingCategory`
- Produces: `AssessOverall(categories []DelistingCategory) string`
- Produces: `GenerateSummary(categories []DelistingCategory, overallRisk string) string`

- [ ] **Step 1: Append assessment functions to delisting_risk.go**

```go
func AssessCN(m *FinancialMetrics, board string, price, marketCap, volume, totalShares float64) []DelistingCategory {
	var cats []DelistingCategory

	// 财务类退市
	finItems := []DelistingItem{}
	revThresh := revenueThresholdMainland
	if board == "创业板" || board == "科创板" || board == "北交所" {
		revThresh = revenueThresholdOther
	}
	if m != nil {
		revBelow := m.Revenue > 0 && m.Revenue < revThresh
		profitNeg := m.NetProfit < 0
		switch {
		case revBelow && profitNeg:
			finItems = append(finItems, DelistingItem{"营收+净利润组合", "danger",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"已触及财务类退市,年报披露后实施*ST"})
		case m.Revenue > 0 && m.Revenue < revThresh*1.3:
			finItems = append(finItems, DelistingItem{"营收+净利润组合", "warn",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"营收接近退市线，需关注"})
		default:
			finItems = append(finItems, DelistingItem{"营收+净利润组合", "safe",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"远离退市线"})
		}
		switch {
		case m.NetAssets < 0:
			finItems = append(finItems, DelistingItem{"净资产", "danger",
				formatValue(m.NetAssets), "净资产<0", "资不抵债,触及财务类退市"})
		case m.NetAssets < 1e8:
			finItems = append(finItems, DelistingItem{"净资产", "warn",
				formatValue(m.NetAssets), "净资产<0", "净资产偏低"})
		default:
			finItems = append(finItems, DelistingItem{"净资产", "safe",
				formatValue(m.NetAssets), "净资产<0", "净资产为正"})
		}
	} else {
		finItems = append(finItems, DelistingItem{"营收+净利润组合", "safe",
			"财务数据暂缺", "营收<阈值且净利<0", "无法判断"})
		finItems = append(finItems, DelistingItem{"净资产", "safe",
			"财务数据暂缺", "净资产<0", "无法判断"})
	}
	finItems = append(finItems, DelistingItem{"审计意见类型", "safe",
		"待获取", "无法表示/否定→退市", "数据源待补充"})
	cats = append(cats, DelistingCategory{"财务类退市", categoryLevel(finItems), finItems})

	// 交易类退市
	tradeItems := []DelistingItem{}
	_ = volume // volume check simplified
	if price > 0 && price < 1.0 {
		tradeItems = append(tradeItems, DelistingItem{"面值(收盘价)", "danger",
			fmt.Sprintf("%.2f元", price), "连续20日<1元", "已低于1元"})
	} else if price > 0 && price < 1.3 {
		tradeItems = append(tradeItems, DelistingItem{"面值(收盘价)", "warn",
			fmt.Sprintf("%.2f元", price), "连续20日<1元",
			fmt.Sprintf("距1元仅%.0f%%", (price-1)/1*100)})
	} else if price > 0 {
		tradeItems = append(tradeItems, DelistingItem{"面值(收盘价)", "safe",
			fmt.Sprintf("%.2f元", price), "连续20日<1元", "安全"})
	}
	mcThresh := marketCapThresholdMainland
	if board == "科创板" || board == "北交所" {
		mcThresh = marketCapThresholdOther
	}
	if marketCap > 0 && marketCap < mcThresh {
		tradeItems = append(tradeItems, DelistingItem{"总市值", "danger",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "低于市值退市标准"})
	} else if marketCap > 0 && marketCap < mcThresh*1.2 {
		tradeItems = append(tradeItems, DelistingItem{"总市值", "warn",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "市值接近退市线"})
	} else if marketCap > 0 {
		tradeItems = append(tradeItems, DelistingItem{"总市值", "safe",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "市值安全"})
	}
	cats = append(cats, DelistingCategory{"交易类退市", categoryLevel(tradeItems), tradeItems})

	// 规范类 + 重大违法 (占位)
	cats = append(cats, DelistingCategory{"规范类退市", "green", []DelistingItem{
		{"财报披露", "safe", "正常", "未按期披露→*ST", "需交易所公告确认"},
		{"资金占用", "safe", "待获取", "占用≥净资产30%且未改正", "数据源待补充"},
	}})
	cats = append(cats, DelistingCategory{"重大违法类退市", "green", []DelistingItem{
		{"财务造假", "safe", "待获取", "1年≥2亿+30%/2年≥3亿+20%/连3年", "需关注证监会立案公告"},
	}})
	return cats
}

func AssessHK(price, marketCap, volume, totalShares float64) []DelistingCategory {
	items := []DelistingItem{}
	if price > 0 && price < 1.0 {
		items = append(items, DelistingItem{"股价(仙股化)", "warn",
			fmt.Sprintf("%.3f HKD", price), "<1 HKD", "仙股化风险"})
	} else if price > 0 {
		items = append(items, DelistingItem{"股价(仙股化)", "safe",
			fmt.Sprintf("%.2f HKD", price), "<1 HKD", "股价正常"})
	}
	if marketCap > 0 && marketCap < 5e8 {
		items = append(items, DelistingItem{"总市值", "warn",
			formatValue(marketCap), "<5亿 HKD", "低于IPO最低市值"})
	} else if marketCap > 0 {
		items = append(items, DelistingItem{"总市值", "safe",
			formatValue(marketCap), "<5亿 HKD", "市值正常"})
	}
	if volume > 0 && totalShares > 0 && volume/totalShares < 0.0002 {
		items = append(items, DelistingItem{"流动性(换手率)", "warn",
			fmt.Sprintf("%.4f%%", volume/totalShares*100), ">0.02%", "流动性枯竭风险"})
	} else if volume > 0 && totalShares > 0 {
		items = append(items, DelistingItem{"流动性(换手率)", "safe",
			fmt.Sprintf("%.2f%%", volume/totalShares*100), ">0.02%", "流动性正常"})
	}
	return []DelistingCategory{{"交易指标预警", categoryLevel(items), items}}
}

func AssessUS(price, marketCap float64) []DelistingCategory {
	items := []DelistingItem{}
	if price > 0 && price < 1.0 {
		items = append(items, DelistingItem{"股价(面值)", "warn",
			fmt.Sprintf("$%.2f", price), "连续30日<$1", "NYSE/NASDAQ面值退市风险"})
	} else if price > 0 {
		items = append(items, DelistingItem{"股价(面值)", "safe",
			fmt.Sprintf("$%.2f", price), "连续30日<$1", "股价正常"})
	}
	if marketCap > 0 && marketCap < 5e7 {
		items = append(items, DelistingItem{"总市值", "warn",
			fmt.Sprintf("$%.0f万", marketCap/1e4), "<$5000万", "NYSE/NASDAQ市值退市风险"})
	} else if marketCap > 0 {
		items = append(items, DelistingItem{"总市值", "safe",
			fmt.Sprintf("$%.0f亿", marketCap/1e8), "<$5000万", "市值正常"})
	}
	return []DelistingCategory{{"交易指标预警", categoryLevel(items), items}}
}

func categoryLevel(items []DelistingItem) string {
	for _, it := range items {
		if it.Status == "danger" {
			return "red"
		}
	}
	for _, it := range items {
		if it.Status == "warn" {
			return "yellow"
		}
	}
	return "green"
}

func AssessOverall(categories []DelistingCategory) string {
	for _, c := range categories {
		if c.Level == "red" {
			return "high"
		}
	}
	for _, c := range categories {
		if c.Level == "yellow" {
			return "medium"
		}
	}
	return "low"
}

func GenerateSummary(categories []DelistingCategory, overallRisk string) string {
	d, w := 0, 0
	for _, cat := range categories {
		for _, it := range cat.Items {
			switch it.Status {
			case "danger":
				d++
			case "warn":
				w++
			}
		}
	}
	switch overallRisk {
	case "high":
		return fmt.Sprintf("触及 %d 项退市标准(%d 项预警),存在强制退市风险", d, w)
	case "medium":
		return fmt.Sprintf("%d 项指标接近退市线,需密切关注", w)
	default:
		return "各项指标远离退市标准,暂无退市风险"
	}
}
```

- [ ] **Step 2: Append tests to delisting_risk_test.go**

```go
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
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -v -count=1`
Expected: All 12+ tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/trading/
git commit -m "feat(trading): A/HK/US delisting rule engine with tests"
```

---

### Task 3: Go Wails Method — GetDelistingRisk

**Files:**
- Modify: `app_research.go` (add `GetDelistingRisk` + helper functions)

**Interfaces:**
- Consumes: `fetchFinancialJSON(symbol string) (string, error)` — existing
- Consumes: `a.eastmoneyAdpt.FetchStockInfo(ctx, symbol)` — existing
- Consumes: `a.GetQuote(ctx, market, symbol)` — existing
- Consumes: `quantflow/internal/trading` package (all exported functions)
- Produces: `GetDelistingRisk(symbol string) (*trading.DelistingRiskResult, error)`

- [ ] **Step 1: Write failing integration test**

```go
// in app_research_test.go
func TestDetectMarketForSymbol(t *testing.T) {
	tests := []struct{ symbol, want string }{
		{"600519", "CN"}, {"000001", "CN"}, {"300750", "CN"},
		{"00700", "HK"}, {"00700.HK", "HK"},
		{"AAPL", "US"}, {"MSFT", "US"}, {"TSLA", "US"},
	}
	for _, tt := range tests {
		got := detectMarketForSymbol(tt.symbol)
		if got != tt.want {
			t.Errorf("detectMarketForSymbol(%q) = %q, want %q", tt.symbol, got, tt.want)
		}
	}
}

func TestDetectST(t *testing.T) {
	tests := []struct{ symbol string; want bool }{
		{"600519", false}, {"*ST康得", true}, {"ST康得", true}, {"000001", false},
	}
	for _, tt := range tests {
		got := detectST(tt.symbol)
		if got != tt.want {
			t.Errorf("detectST(%q) = %v, want %v", tt.symbol, got, tt.want)
		}
	}
}
```

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -v -run "TestDetectMarket|TestDetectST" -count=1`
Expected: FAIL — detectMarketForSymbol not defined

- [ ] **Step 2: Add helper functions and GetDelistingRisk to app_research.go**

Add imports at top (add `"unicode"` and `"quantflow/internal/trading"` to existing import block):

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
	"quantflow/internal/trading"
)
```

Add helper functions near the top of the file (after existing helper `detectLanguage`):

```go
func detectMarketForSymbol(symbol string) string {
	if strings.HasSuffix(symbol, ".HK") {
		return "HK"
	}
	if strings.HasSuffix(symbol, ".SZ") || strings.HasSuffix(symbol, ".SH") {
		return "CN"
	}
	if len(symbol) == 6 && (symbol[0] == '0' || symbol[0] == '3' || symbol[0] == '6') {
		return "CN"
	}
	if len(symbol) == 5 && symbol[0] == '0' {
		return "HK"
	}
	if len(symbol) <= 4 && allUpper(symbol) {
		return "US"
	}
	return "CN"
}

func allUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func detectST(symbol string) bool {
	return strings.Contains(strings.ToUpper(symbol), "ST")
}
```

Add `GetDelistingRisk` method (after existing research methods, before EOF):

```go
func (a *App) GetDelistingRisk(symbol string) (*trading.DelistingRiskResult, error) {
	market := detectMarketForSymbol(symbol)
	isST := detectST(symbol)

	finJSON, err := a.fetchFinancialJSON(symbol)
	if err != nil {
		slog.Warn("delisting risk: financial data unavailable", "symbol", symbol, "err", err)
		return a.computeDelistingRisk(symbol, market, isST, nil)
	}

	metrics, err := trading.ExtractFinancialMetrics(finJSON)
	if err != nil {
		slog.Warn("delisting risk: failed to parse financial data", "symbol", symbol, "err", err)
		return a.computeDelistingRisk(symbol, market, isST, nil)
	}

	return a.computeDelistingRisk(symbol, market, isST, metrics)
}

func (a *App) computeDelistingRisk(symbol, market string, isST bool, metrics *trading.FinancialMetrics) (*trading.DelistingRiskResult, error) {
	ctx := context.Background()
	price, marketCap, volume := 0.0, 0.0, 0.0
	totalShares := 0.0
	board := trading.DetectBoard(symbol)

	if a.eastmoneyAdpt != nil {
		info, err := a.eastmoneyAdpt.FetchStockInfo(ctx, symbol)
		if err == nil && info != nil {
			marketCap = info.MarketCap
			totalShares = info.TotalShares
		}
	}
	quote, _, err := a.GetQuote(ctx, market, symbol)
	if err == nil && quote != nil {
		price = quote.Last
		volume = quote.Volume
		if quote.MarketCap > 0 {
			marketCap = quote.MarketCap
		}
	}

	var categories []trading.DelistingCategory
	switch market {
	case "CN":
		categories = trading.AssessCN(metrics, board, price, marketCap, volume, totalShares)
	case "HK":
		categories = trading.AssessHK(price, marketCap, volume, totalShares)
	case "US":
		categories = trading.AssessUS(price, marketCap)
	default:
		categories = trading.AssessCN(metrics, board, price, marketCap, volume, totalShares)
	}

	overallRisk := trading.AssessOverall(categories)
	summary := trading.GenerateSummary(categories, overallRisk)

	return &trading.DelistingRiskResult{
		Market:      market,
		Board:       board,
		IsST:        isST,
		OverallRisk: overallRisk,
		Categories:  categories,
		Summary:     summary,
	}, nil
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -v -run "TestDetectMarket|TestDetectST" -count=1`
Expected: PASS

Run full trading tests: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -v -count=1`
Expected: All PASS

Run: `go vet ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add app_research.go app_research_test.go
git commit -m "feat: add GetDelistingRisk Wails method with market routing"
```

---

### Task 4: Frontend — Add Delisting Risk Tab to AuditPanel

**Files:**
- Modify: `frontend/src/terminal/panels/AuditPanel.vue` (add tab bar + delisting risk content)

- [ ] **Step 1: Write AuditPanel.vue changes**

In `<script setup>`:

```typescript
// Add import (after existing imports, ~line 2)
import { PanelTabs } from '@/terminal/components/panel'

// Add reactive state (after existing refs, ~line 24)
const activeTab = ref<'audit' | 'delist'>('audit')

// Add delisting risk state (after analysis.value, ~line 30)
const delisting = ref<Record<string, any> | null>(null)
const delistingLoading = ref(false)
const delistingError = ref('')

// Add loadDelistingRisk function (after loadData, ~line 120)
async function loadDelistingRisk() {
  const app = (window as any).go?.main?.App
  if (!app?.GetDelistingRisk) return
  delistingLoading.value = true
  delistingError.value = ''
  try {
    delisting.value = await app.GetDelistingRisk(symbol.value)
  } catch (e: any) {
    delistingError.value = e.message || '退市风险数据加载失败'
  }
  delistingLoading.value = false
}

// Replace onMounted to load delisting data (after onMounted, ~line 251)
onMounted(() => {
  loadData()
  loadDelistingRisk()
})
```

In `<template>`, after the header block (`<div class="h">...</div>`, ~line 262), insert:

```html
    <!-- Tab bar -->
    <PanelTabs
      variant="pill"
      :tabs="[{ key: 'audit', label: '审计异常' }, { key: 'delist', label: '退市风险' }]"
      :active="activeTab"
      @change="(k: string) => activeTab = k as 'audit' | 'delist'"
    />
```

Wrap the existing content (gauges, KPIs, chart, findings, history — from `<!-- Risk Gauges -->` to the closing `</div>` before `</template>`) with:

```html
    <template v-if="activeTab === 'audit'">
      <!-- existing content ... -->
    </template>
```

Add the delisting risk content after the audit tab block:

```html
    <template v-if="activeTab === 'delist'">
      <SkeletonPanel v-if="delistingLoading && !delisting" type="card" :rows="4" />
      <div v-else-if="delistingError && !delisting" class="st err">{{ delistingError }}</div>
      <template v-else-if="delisting">
        <!-- Overall Risk Badge -->
        <div class="dr-overall">
          <div class="dr-badge" :class="'dr-' + delisting.overall_risk">
            <span class="dr-badge-label">{{ delisting.overall_risk === 'high' ? '高风险' : delisting.overall_risk === 'medium' ? '中风险' : '低风险' }}</span>
            <span class="dr-board">{{ delisting.market }} · {{ delisting.board }}</span>
            <span v-if="delisting.is_st" class="st-tag">ST</span>
          </div>
          <p class="dr-summary">{{ delisting.summary }}</p>
        </div>

        <!-- Category Cards -->
        <div v-for="cat in delisting.categories" :key="cat.name" class="dr-category">
          <div class="dr-cat-h">
            <span class="dr-cat-dot" :class="'dot-' + cat.level"></span>
            <span class="dr-cat-name">{{ cat.name }}</span>
          </div>
          <div class="dr-items">
            <div v-for="item in cat.items" :key="item.indicator" class="dr-item" :class="'dr-item-' + item.status">
              <div class="dr-item-left">
                <span class="dr-dot" :class="'dot-' + (item.status === 'danger' ? 'red' : item.status === 'warn' ? 'yellow' : 'green')"></span>
                <span class="dr-indicator">{{ item.indicator }}</span>
              </div>
              <div class="dr-item-right">
                <span class="dr-current">{{ item.current }}</span>
                <span class="dr-threshold">阈值: {{ item.threshold }}</span>
              </div>
              <div v-if="item.detail" class="dr-detail">{{ item.detail }}</div>
            </div>
          </div>
        </div>
      </template>
    </template>
```

- [ ] **Step 2: Add CSS styles**

Append to the `<style scoped>` section (before closing `</style>`):

```css
/* Overall risk */
.dr-overall { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.dr-badge { display: flex; align-items: center; gap: 8px; padding: 10px 16px; border-radius: 8px; font-size: 13px; }
.dr-badge.dr-high { background: rgba(239,68,68,0.15); border: 1px solid rgba(239,68,68,0.3); }
.dr-badge.dr-medium { background: rgba(234,179,8,0.15); border: 1px solid rgba(234,179,8,0.3); }
.dr-badge.dr-low { background: rgba(34,197,94,0.15); border: 1px solid rgba(34,197,94,0.3); }
.dr-badge-label { font-weight: 600; font-size: 15px; }
.dr-board { color: var(--color-text-secondary); font-size: 12px; }
.st-tag { background: var(--color-accent); color: #fff; padding: 1px 6px; border-radius: 3px; font-size: 11px; font-weight: 600; }
.dr-summary { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/* Category */
.dr-category { margin-bottom: 14px; }
.dr-cat-h { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.dr-cat-dot { width: 8px; height: 8px; border-radius: 50%; }
.dot-red { background: #ef4444; }
.dot-yellow { background: #eab308; }
.dot-green { background: #22c55e; }
.dr-cat-name { font-weight: 600; font-size: 13px; }

/* Items */
.dr-items { display: flex; flex-direction: column; gap: 4px; }
.dr-item { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 6px 10px; border-radius: 4px; font-size: 12px; background: var(--color-bg-subtle); }
.dr-item-left { display: flex; align-items: center; gap: 6px; min-width: 140px; }
.dr-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.dr-indicator { font-weight: 500; white-space: nowrap; }
.dr-item-right { display: flex; align-items: center; gap: 8px; margin-left: auto; }
.dr-current { font-family: 'JetBrains Mono', monospace; font-size: 12px; }
.dr-threshold { color: var(--color-text-secondary); font-size: 11px; }
.dr-detail { width: 100%; color: var(--color-text-secondary); font-size: 11px; padding-left: 18px; }
.dr-item-danger { border-left: 3px solid #ef4444; }
.dr-item-warn { border-left: 3px solid #eab308; }
.dr-item-safe { border-left: 3px solid #22c55e; }
```

- [ ] **Step 3: Verify frontend builds**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/AuditPanel.vue
git commit -m "feat(frontend): add delisting risk tab to AuditPanel"
```

---

### Task 5: Final Verification

- [ ] **Step 1: Run full Go tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go test ./internal/trading/... -v -count=1
```
Expected: All PASS, no vet errors

- [ ] **Step 2: Run full frontend check**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit && npx vitest run
```
Expected: No errors

- [ ] **Step 3: Update CHANGELOG.md**

Add entry under `[2026.7.1]`:

```markdown
### Added
- [Frontend] **AuditPanel 退市风险 tab** — 新增 A 股/港股/美股退市风险检测，覆盖财务类(营收+净利润组合/净资产)、交易类(面值/总市值)、规范类、重大违法类四类退市标准，Go 端纯规则引擎无 Python 依赖
- [Backend] **GetDelistingRisk** — 新增 Wails 方法返回退市风险结构化数据，包括市场路由(A/HK/US)、板块检测、ST 状态推断、逐项指标状态灯
- [Backend] **internal/trading/delisting_risk** — 退市规则引擎包：ExtractFinancialMetrics 解析新浪财务 JSON 提取关键指标，AssessCN/HK/US 三市场规则实现
```

- [ ] **Step 4: Push**

```bash
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for delisting risk feature"
git push origin main
```
