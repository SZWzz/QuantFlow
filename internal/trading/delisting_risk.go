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
	Market      string              `json:"market"`
	Board       string              `json:"board"`
	IsST        bool                `json:"is_st"`
	OverallRisk string              `json:"overall_risk"`
	Categories  []DelistingCategory `json:"categories"`
	Summary     string              `json:"summary"`
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
	Period string          `json:"period"`
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

// latestAnnualPeriod returns the period with the latest date ending in "12-31",
// or the first period if no annual period is found.
func latestAnnualPeriod(periods []finPeriod) *finPeriod {
	var ann *finPeriod
	for i, p := range periods {
		if strings.HasSuffix(p.Period, "12-31") {
			if ann == nil || p.Period > ann.Period {
				ann = &periods[i]
			}
		}
	}
	if ann == nil && len(periods) > 0 {
		ann = &periods[0]
	}
	return ann
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
	if inc := latestAnnualPeriod(data.Income); inc != nil {
		items := inc.Items
		m.Revenue = findItemValue(items, "营业总收入")
		m.NetProfit = findItemValue(items, "净利润")
		cp := findItemValue(items, "扣非净利润")
		if cp == 0 {
			cp = findItemValue(items, "归属于母公司所有者的净利润")
		}
		m.CoreProfit = cp
	}
	if bal := latestAnnualPeriod(data.Balance); bal != nil {
		m.NetAssets = findItemValue(bal.Items, "归属于母公司所有者的权益合计")
	}
	if cf := latestAnnualPeriod(data.Cashflow); cf != nil {
		m.CashFlow = findItemValue(cf.Items, "经营活动现金流量净额")
	}
	return m, nil
}

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
			finItems = append(finItems, DelistingItem{
				"营收+净利润组合", "danger",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"已触及财务类退市,年报披露后实施*ST",
			})
		case m.Revenue > 0 && m.Revenue < revThresh*1.3:
			finItems = append(finItems, DelistingItem{
				"营收+净利润组合", "warn",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"营收接近退市线，需关注",
			})
		default:
			finItems = append(finItems, DelistingItem{
				"营收+净利润组合", "safe",
				formatValue(m.Revenue) + "/" + formatValue(m.NetProfit),
				fmt.Sprintf("营收<%.0f亿且净利<0", revThresh/1e8),
				"远离退市线",
			})
		}
		switch {
		case m.NetAssets < 0:
			finItems = append(finItems, DelistingItem{
				"净资产", "danger",
				formatValue(m.NetAssets), "净资产<0", "资不抵债,触及财务类退市",
			})
		case m.NetAssets < 1e8:
			finItems = append(finItems, DelistingItem{
				"净资产", "warn",
				formatValue(m.NetAssets), "净资产<0", "净资产偏低",
			})
		default:
			finItems = append(finItems, DelistingItem{
				"净资产", "safe",
				formatValue(m.NetAssets), "净资产<0", "净资产为正",
			})
		}
	} else {
		finItems = append(finItems, DelistingItem{
			"营收+净利润组合", "safe",
			"财务数据暂缺", "营收<阈值且净利<0", "无法判断",
		})
		finItems = append(finItems, DelistingItem{
			"净资产", "safe",
			"财务数据暂缺", "净资产<0", "无法判断",
		})
	}
	finItems = append(finItems, DelistingItem{
		"审计意见类型", "safe",
		"待获取", "无法表示/否定→退市", "数据源待补充",
	})
	cats = append(cats, DelistingCategory{"财务类退市", categoryLevel(finItems), finItems})

	// 交易类退市
	tradeItems := []DelistingItem{}
	_ = volume // volume check simplified
	switch {
	case price > 0 && price < 1.0:
		tradeItems = append(tradeItems, DelistingItem{
			"面值(收盘价)", "danger",
			fmt.Sprintf("%.2f元", price), "连续20日<1元", "已低于1元",
		})
	case price > 0 && price < 1.3:
		tradeItems = append(tradeItems, DelistingItem{
			"面值(收盘价)", "warn",
			fmt.Sprintf("%.2f元", price), "连续20日<1元",
			fmt.Sprintf("距1元仅%.0f%%", (price-1)/1*100),
		})
	case price > 0:
		tradeItems = append(tradeItems, DelistingItem{
			"面值(收盘价)", "safe",
			fmt.Sprintf("%.2f元", price), "连续20日<1元", "安全",
		})
	}
	mcThresh := marketCapThresholdMainland
	if board == "科创板" || board == "北交所" {
		mcThresh = marketCapThresholdOther
	}
	switch {
	case marketCap > 0 && marketCap < mcThresh:
		tradeItems = append(tradeItems, DelistingItem{
			"总市值", "danger",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "低于市值退市标准",
		})
	case marketCap > 0 && marketCap < mcThresh*1.2:
		tradeItems = append(tradeItems, DelistingItem{
			"总市值", "warn",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "市值接近退市线",
		})
	case marketCap > 0:
		tradeItems = append(tradeItems, DelistingItem{
			"总市值", "safe",
			formatValue(marketCap), fmt.Sprintf("<%.0f亿", mcThresh/1e8), "市值安全",
		})
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
		items = append(items, DelistingItem{
			"股价(仙股化)", "warn",
			fmt.Sprintf("%.3f HKD", price), "<1 HKD", "仙股化风险",
		})
	} else if price > 0 {
		items = append(items, DelistingItem{
			"股价(仙股化)", "safe",
			fmt.Sprintf("%.2f HKD", price), "<1 HKD", "股价正常",
		})
	}
	if marketCap > 0 && marketCap < 5e8 {
		items = append(items, DelistingItem{
			"总市值", "warn",
			formatValue(marketCap), "<5亿 HKD", "低于IPO最低市值",
		})
	} else if marketCap > 0 {
		items = append(items, DelistingItem{
			"总市值", "safe",
			formatValue(marketCap), "<5亿 HKD", "市值正常",
		})
	}
	if volume > 0 && totalShares > 0 && volume/totalShares < 0.0002 {
		items = append(items, DelistingItem{
			"流动性(换手率)", "warn",
			fmt.Sprintf("%.4f%%", volume/totalShares*100), ">0.02%", "流动性枯竭风险",
		})
	} else if volume > 0 && totalShares > 0 {
		items = append(items, DelistingItem{
			"流动性(换手率)", "safe",
			fmt.Sprintf("%.2f%%", volume/totalShares*100), ">0.02%", "流动性正常",
		})
	}
	return []DelistingCategory{{"交易指标预警", categoryLevel(items), items}}
}

func AssessUS(price, marketCap float64) []DelistingCategory {
	items := []DelistingItem{}
	if price > 0 && price < 1.0 {
		items = append(items, DelistingItem{
			"股价(面值)", "warn",
			fmt.Sprintf("$%.2f", price), "连续30日<$1", "NYSE/NASDAQ面值退市风险",
		})
	} else if price > 0 {
		items = append(items, DelistingItem{
			"股价(面值)", "safe",
			fmt.Sprintf("$%.2f", price), "连续30日<$1", "股价正常",
		})
	}
	if marketCap > 0 && marketCap < 5e7 {
		items = append(items, DelistingItem{
			"总市值", "warn",
			fmt.Sprintf("$%.0f万", marketCap/1e4), "<$5000万", "NYSE/NASDAQ市值退市风险",
		})
	} else if marketCap > 0 {
		items = append(items, DelistingItem{
			"总市值", "safe",
			fmt.Sprintf("$%.0f亿", marketCap/1e8), "<$5000万", "市值正常",
		})
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
