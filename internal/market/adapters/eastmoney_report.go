package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ResearchReport represents a single analyst research report.
type ResearchReport struct {
	Title              string  `json:"title"`
	PublishDate        string  `json:"publish_date"`
	OrgName            string  `json:"org_name"`              // 机构简称
	Analyst            string  `json:"analyst"`               // 分析师姓名
	InfoCode           string  `json:"info_code"`             // PDF download code
	PDFURL             string  `json:"pdf_url"`               // full PDF URL
	Rating             string  `json:"rating"`                // 评级(买入/增持/...)
	TargetPriceLow     float64 `json:"target_price_low"`      // 目标价下限
	TargetPriceHigh    float64 `json:"target_price_high"`     // 目标价上限
	PredictThisYearEPS float64 `json:"predict_this_year_eps"` // 今年EPS预测
	PredictNextYearEPS float64 `json:"predict_next_year_eps"` // 明年EPS预测
	PredictNextTwoEPS  float64 `json:"predict_next_two_eps"`  // 后年EPS预测
	Industry           string  `json:"industry"`              // 行业分类
}

// EastMoneyReportAdapter fetches analyst research reports from EastMoney.
// Based on a-stock-data SKILL §2.1 (reportapi.eastmoney.com).
//
// Returns report list with ratings, EPS forecasts, and PDF download links.
// No API key required.
type EastMoneyReportAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// NewEastMoneyReportAdapter creates a new report adapter.
func NewEastMoneyReportAdapter() *EastMoneyReportAdapter {
	return &EastMoneyReportAdapter{
		client:  &http.Client{Timeout: 30 * time.Second},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneyReportAdapter) Name() string { return "eastmoney_report" }

func (a *EastMoneyReportAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchReports(ctx, "600519", 1)
	return err == nil
}

// FetchReports fetches research reports for a stock.
// maxPages: max pages to fetch (1 page = up to 100 reports).
func (a *EastMoneyReportAdapter) FetchReports(ctx context.Context, code string, maxPages int) ([]ResearchReport, error) {
	if maxPages <= 0 {
		maxPages = 1
	}
	if maxPages > 10 {
		maxPages = 10
	}

	var allReports []ResearchReport

	for page := 1; page <= maxPages; page++ {
		a.limiter.Wait()

		url := fmt.Sprintf(
			"https://reportapi.eastmoney.com/report/list"+
				"?industryCode=*&pageSize=100&industry=*&rating=*&ratingChange=*"+
				"&beginTime=2000-01-01&endTime=2030-01-01"+
				"&pageNo=%d&fields=&qType=0&code=%s&rcode=",
			page, code,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("eastmoney_report: %w", err)
		}
		req.Header.Set("User-Agent", emUA)
		req.Header.Set("Referer", "https://data.eastmoney.com/")

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("eastmoney_report: http: %w", err)
		}

		var result struct {
			Data      []reportItem `json:"data"`
			TotalPage int          `json:"TotalPage"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("eastmoney_report: json: %w", err)
		}
		resp.Body.Close()

		if len(result.Data) == 0 {
			break
		}

		for _, item := range result.Data {
			report := ResearchReport{
				Title:              item.Title,
				PublishDate:        item.PublishDate,
				OrgName:            item.OrgSName,
				Analyst:            item.Researcher,
				InfoCode:           item.InfoCode,
				Rating:             item.EmRatingName,
				Industry:           item.IndvInduName,
				TargetPriceLow:     toFloat(item.IndvAimPriceL),
				TargetPriceHigh:    toFloat(item.IndvAimPriceT),
				PredictThisYearEPS: toFloat(item.PredictThisYearEps),
				PredictNextYearEPS: toFloat(item.PredictNextYearEps),
				PredictNextTwoEPS:  toFloat(item.PredictNextTwoYearEps),
			}
			if item.InfoCode != "" {
				report.PDFURL = fmt.Sprintf("https://pdf.dfcfw.com/pdf/H3_%s_1.pdf", item.InfoCode)
			}
			allReports = append(allReports, report)
		}

		if page >= result.TotalPage {
			break
		}
		// Small sleep between pages to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	slog.Debug("eastmoney_report fetched", "code", code, "reports", len(allReports))
	return allReports, nil
}

// FetchConsensusEPS extracts the latest consensus EPS from the most recent reports.
// Returns (thisYearEPS, nextYearEPS, analystCount).
func (a *EastMoneyReportAdapter) FetchConsensusEPS(ctx context.Context, code string) (float64, float64, int, error) {
	reports, err := a.FetchReports(ctx, code, 1)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(reports) == 0 {
		return 0, 0, 0, fmt.Errorf("eastmoney_report: no reports for %s", code)
	}

	// Average the EPS predictions from the most recent reports (up to 10)
	var thisYearSum, nextYearSum float64
	var thisYearCount, nextYearCount int
	limit := len(reports)
	if limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		if reports[i].PredictThisYearEPS > 0 {
			thisYearSum += reports[i].PredictThisYearEPS
			thisYearCount++
		}
		if reports[i].PredictNextYearEPS > 0 {
			nextYearSum += reports[i].PredictNextYearEPS
			nextYearCount++
		}
	}

	var thisYearAvg, nextYearAvg float64
	if thisYearCount > 0 {
		thisYearAvg = thisYearSum / float64(thisYearCount)
	}
	if nextYearCount > 0 {
		nextYearAvg = nextYearSum / float64(nextYearCount)
	}

	return thisYearAvg, nextYearAvg, len(reports), nil
}

// ── Internal response types ───────────────────────────────────────────

type reportItem struct {
	Title                 string      `json:"title"`
	PublishDate           string      `json:"publishDate"`
	OrgSName              string      `json:"orgSName"`
	Researcher            string      `json:"researcher"`
	InfoCode              string      `json:"infoCode"`
	EmRatingName          string      `json:"emRatingName"`
	IndvInduName          string      `json:"indvInduName"`
	IndvAimPriceL         interface{} `json:"indvAimPriceL"`
	IndvAimPriceT         interface{} `json:"indvAimPriceT"`
	PredictThisYearEps    interface{} `json:"predictThisYearEps"`
	PredictNextYearEps    interface{} `json:"predictNextYearEps"`
	PredictNextTwoYearEps interface{} `json:"predictNextTwoYearEps"`
}
