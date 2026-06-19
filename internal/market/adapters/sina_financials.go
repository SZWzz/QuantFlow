package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

// FinancialStatementItem holds a single line item from a financial statement.
type FinancialStatementItem struct {
	Title   string `json:"item_title"`   // 科目名称
	Value   string `json:"item_value"`   // 值(新浪原始格式)
	YoYPct  string `json:"item_tongbi"`  // 同比变化
}

// FinancialStatementPeriod holds one reporting period's financial data.
type FinancialStatementPeriod struct {
	Period string                  `json:"period"` // e.g. "2026-03-31"
	Items  []FinancialStatementItem `json:"items"`
}

// SinaFinancialsAdapter fetches balance sheet, income statement, and cash flow
// statement from Sina Finance (quotes.sina.cn).
// Based on a-stock-data SKILL §6.4.
type SinaFinancialsAdapter struct {
	client *http.Client
}

// NewSinaFinancialsAdapter creates a new Sina financials adapter.
func NewSinaFinancialsAdapter() *SinaFinancialsAdapter {
	return &SinaFinancialsAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *SinaFinancialsAdapter) Name() string { return "sina_financials" }

func (a *SinaFinancialsAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchIncomeStatement(ctx, "600519", 1)
	return err == nil
}

// report types
const (
	sinaBalanceSheet    = "fzb" // 资产负债表
	sinaIncomeStatement = "lrb" // 利润表
	sinaCashFlow        = "llb" // 现金流量表
)

// FetchIncomeStatement fetches the income statement (利润表).
func (a *SinaFinancialsAdapter) FetchIncomeStatement(ctx context.Context, code string, num int) ([]FinancialStatementPeriod, error) {
	return a.fetchReport(ctx, code, sinaIncomeStatement, num)
}

// FetchBalanceSheet fetches the balance sheet (资产负债表).
func (a *SinaFinancialsAdapter) FetchBalanceSheet(ctx context.Context, code string, num int) ([]FinancialStatementPeriod, error) {
	return a.fetchReport(ctx, code, sinaBalanceSheet, num)
}

// FetchCashFlow fetches the cash flow statement (现金流量表).
func (a *SinaFinancialsAdapter) FetchCashFlow(ctx context.Context, code string, num int) ([]FinancialStatementPeriod, error) {
	return a.fetchReport(ctx, code, sinaCashFlow, num)
}

func (a *SinaFinancialsAdapter) fetchReport(ctx context.Context, code, reportType string, num int) ([]FinancialStatementPeriod, error) {
	if num <= 0 {
		num = 4
	}

	prefix := "sz"
	if code[0] == '6' || code[0] == '9' {
		prefix = "sh"
	}

	url := fmt.Sprintf(
		"https://quotes.sina.cn/cn/api/openapi.php/CompanyFinanceService.getFinanceReport2022"+
			"?paperCode=%s%s&source=%s&type=0&page=1&num=%d",
		prefix, code, reportType, num,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("sina_financials: %w", err)
	}
	req.Header.Set("User-Agent", emUA)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sina_financials: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sina_financials: http %d", resp.StatusCode)
	}

	// Sina's response structure: result.data.report_list is a map keyed by period (e.g. "20260331"),
	// each value has a "data" array of [{item_title, item_value, item_tongbi}].
	var result struct {
		Result struct {
			Data struct {
				ReportList map[string]struct {
					Data []struct {
						ItemTitle  string      `json:"item_title"`
						ItemValue  interface{} `json:"item_value"`
						ItemTongbi interface{} `json:"item_tongbi"`
					} `json:"data"`
				} `json:"report_list"`
			} `json:"data"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("sina_financials: json: %w", err)
	}

	reportList := result.Result.Data.ReportList

	// Sort periods descending
	periods := make([]string, 0, len(reportList))
	for p := range reportList {
		periods = append(periods, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(periods)))

	if len(periods) > num {
		periods = periods[:num]
	}

	statements := make([]FinancialStatementPeriod, 0, len(periods))
	for _, period := range periods {
		obj := reportList[period]
		formattedPeriod := fmt.Sprintf("%s-%s-%s", period[:4], period[4:6], period[6:8])

		items := make([]FinancialStatementItem, 0, len(obj.Data))
		for _, it := range obj.Data {
			item := FinancialStatementItem{
				Title: it.ItemTitle,
				Value: strval(it.ItemValue),
			}
			if it.ItemTongbi != nil {
				item.YoYPct = strval(it.ItemTongbi)
			}
			items = append(items, item)
		}

		statements = append(statements, FinancialStatementPeriod{
			Period: formattedPeriod,
			Items:  items,
		})
	}

	slog.Debug("sina_financials fetched", "code", code, "type", reportType, "periods", len(statements))
	return statements, nil
}

// 新浪用 20260331 这样的格式。排序需要按字符串倒序。先确保导入 sort 包了。
var _ = sort.Reverse // force import
var _ = time.Now     // force import
