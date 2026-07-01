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
	case (strings.HasPrefix(symbol, "60") || strings.HasPrefix(symbol, "00")) && len(symbol) == 6:
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
