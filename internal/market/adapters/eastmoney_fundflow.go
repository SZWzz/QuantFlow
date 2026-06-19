package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// FundFlowMinute represents per-minute capital flow for a stock.
type FundFlowMinute struct {
	Time     string  `json:"time"`
	MainNet  float64 `json:"main_net"`  // 主力净流入(元)
	SmallNet float64 `json:"small_net"` // 小单净流入(元)
	MidNet   float64 `json:"mid_net"`   // 中单净流入(元)
	LargeNet float64 `json:"large_net"` // 大单净流入(元)
	SuperNet float64 `json:"super_net"` // 超大单净流入(元)
}

// FundFlowDaily represents daily capital flow for a stock.
type FundFlowDaily struct {
	Date     string  `json:"date"`
	MainNet  float64 `json:"main_net"`
	SmallNet float64 `json:"small_net"`
	MidNet   float64 `json:"mid_net"`
	LargeNet float64 `json:"large_net"`
	SuperNet float64 `json:"super_net"`
}

// EastMoneyFundFlowAdapter fetches stock capital flow data (minute + daily).
// Based on a-stock-data SKILL §3.4 (push2 minute) and §4.5 (push2his 120-day).
type EastMoneyFundFlowAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// NewEastMoneyFundFlowAdapter creates a new fund flow adapter.
func NewEastMoneyFundFlowAdapter() *EastMoneyFundFlowAdapter {
	return &EastMoneyFundFlowAdapter{
		client:  &http.Client{Timeout: 15 * time.Second},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneyFundFlowAdapter) Name() string { return "eastmoney_fundflow" }

func (a *EastMoneyFundFlowAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchMinuteFlow(ctx, "600519")
	return err == nil
}

// FetchMinuteFlow fetches today's intraday per-minute capital flow.
func (a *EastMoneyFundFlowAdapter) FetchMinuteFlow(ctx context.Context, code string) ([]FundFlowMinute, error) {
	a.limiter.Wait()

	secID := toEastMoneySecID(code)
	url := fmt.Sprintf(
		"https://push2.eastmoney.com/api/qt/stock/fflow/kline/get"+
			"?secid=%s&klt=1&fields1=f1,f2,f3,f7&fields2=f51,f52,f53,f54,f55,f56,f57",
		secID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	req.Header.Set("Origin", "https://quote.eastmoney.com")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow: http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow: json: %w", err)
	}

	flows := make([]FundFlowMinute, 0, len(result.Data.Klines))
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		flows = append(flows, FundFlowMinute{
			Time:     parts[0],
			MainNet:  parseFloat(parts[1]),
			SmallNet: parseFloat(parts[2]),
			MidNet:   parseFloat(parts[3]),
			LargeNet: parseFloat(parts[4]),
			SuperNet: parseFloat(parts[5]),
		})
	}

	return flows, nil
}

// FetchDailyFlow fetches the last 120 trading days of daily capital flow.
func (a *EastMoneyFundFlowAdapter) FetchDailyFlow(ctx context.Context, code string) ([]FundFlowDaily, error) {
	a.limiter.Wait()

	marketCode := "1"
	if !(code[0] == '6' || code[0] == '9') {
		marketCode = "0"
	}

	url := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get"+
			"?secid=%s.%s&fields1=f1,f2,f3,f7"+
			"&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61,f62,f63,f64,f65"+
			"&lmt=120",
		marketCode, code,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow_daily: %w", err)
	}
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	req.Header.Set("Origin", "https://quote.eastmoney.com")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow_daily: http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney_fundflow_daily: json: %w", err)
	}

	flows := make([]FundFlowDaily, 0, len(result.Data.Klines))
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		flows = append(flows, FundFlowDaily{
			Date:     parts[0],
			MainNet:  parseFloat(parts[1]),
			SmallNet: parseFloat(parts[2]),
			MidNet:   parseFloat(parts[3]),
			LargeNet: parseFloat(parts[4]),
			SuperNet: parseFloat(parts[5]),
		})
	}

	slog.Debug("eastmoney_fundflow daily fetched", "code", code, "days", len(flows))
	return flows, nil
}
