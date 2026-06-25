package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// DragonTigerRecord represents a single stock's appearance on the 龙虎榜.
type DragonTigerRecord struct {
	Date     string  `json:"date"`
	Reason   string  `json:"reason"`
	NetBuy   float64 `json:"net_buy"`  // 净买入(万元)
	Turnover float64 `json:"turnover"` // 换手率%
}

// DragonTigerStock represents a daily market-wide 龙虎榜 entry.
type DragonTigerStock struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Reason    string  `json:"reason"`
	Close     float64 `json:"close"`
	ChangePct float64 `json:"change_pct"`
	NetBuyWan float64 `json:"net_buy_wan"`
	BuyWan    float64 `json:"buy_wan"`
	SellWan   float64 `json:"sell_wan"`
	Turnover  float64 `json:"turnover_pct"`
}

// LockupExpiry represents a lockup expiry event (限售解禁).
type LockupExpiry struct {
	Date   string  `json:"date"`
	Type   string  `json:"type"`
	Shares float64 `json:"shares"`
	Ratio  float64 `json:"ratio"`
}

// IndustryRank represents a single industry's ranking entry.
type IndustryRank struct {
	Rank      int     `json:"rank"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	ChangePct float64 `json:"change_pct"`
	UpCount   int     `json:"up_count"`
	DownCount int     `json:"down_count"`
	Leader    string  `json:"leader"`
	LeaderChg float64 `json:"leader_change"`
}

// EastMoneySignalsAdapter provides dragon tiger board, lockup calendar,
// and industry ranking data via the EastMoney datacenter API.
// Based on a-stock-data SKILL §3.5-3.8.
type EastMoneySignalsAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter
}

// NewEastMoneySignalsAdapter creates a new signals adapter.
func NewEastMoneySignalsAdapter() *EastMoneySignalsAdapter {
	return &EastMoneySignalsAdapter{
		client:  &http.Client{Timeout: 15 * time.Second},
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneySignalsAdapter) Name() string { return "eastmoney_signals" }

func (a *EastMoneySignalsAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchIndustryRanks(ctx, 5)
	return err == nil
}

// ── Dragon Tiger Board ────────────────────────────────────────────────

// FetchDragonTigerStock fetches dragon tiger board records for a single stock.
func (a *EastMoneySignalsAdapter) FetchDragonTigerStock(ctx context.Context, code, endDate string, lookBack int) ([]DragonTigerRecord, error) {
	if lookBack <= 0 {
		lookBack = 30
	}
	startTime := time.Now().Add(-time.Duration(lookBack) * 24 * time.Hour)
	startStr := startTime.Format("2006-01-02")

	filter := fmt.Sprintf(
		`(TRADE_DATE>='%s')(TRADE_DATE<='%s')(SECURITY_CODE="%s")`,
		startStr, endDate, code,
	)

	rows, err := a.queryDatacenter("RPT_DAILYBILLBOARD_DETAILSNEW", filter, 50, "TRADE_DATE", "-1")
	if err != nil {
		return nil, err
	}

	records := make([]DragonTigerRecord, 0, len(rows))
	for _, r := range rows {
		records = append(records, DragonTigerRecord{
			Date:     strval(r["TRADE_DATE"]),
			Reason:   strval(r["EXPLANATION"]),
			NetBuy:   floatval(r["BILLBOARD_NET_AMT"]) / 10000,
			Turnover: floatval(r["TURNOVERRATE"]),
		})
	}
	return records, nil
}

// FetchDailyDragonTiger fetches the full market dragon tiger board for a date.
func (a *EastMoneySignalsAdapter) FetchDailyDragonTiger(ctx context.Context, tradeDate string, minNetBuy float64) ([]DragonTigerStock, error) {
	filter := fmt.Sprintf(`(TRADE_DATE>='%s')(TRADE_DATE<='%s')`, tradeDate, tradeDate)

	rows, err := a.queryDatacenter("RPT_DAILYBILLBOARD_DETAILSNEW", filter, 500, "BILLBOARD_NET_AMT", "-1")
	if err != nil {
		return nil, err
	}

	stocks := make([]DragonTigerStock, 0, len(rows))
	for _, r := range rows {
		netBuy := floatval(r["BILLBOARD_NET_AMT"]) / 10000
		if minNetBuy > 0 && netBuy < minNetBuy {
			continue
		}
		stocks = append(stocks, DragonTigerStock{
			Code:      strval(r["SECURITY_CODE"]),
			Name:      strval(r["SECURITY_NAME_ABBR"]),
			Reason:    strval(r["EXPLANATION"]),
			Close:     floatval(r["CLOSE_PRICE"]),
			ChangePct: floatval(r["CHANGE_RATE"]),
			NetBuyWan: netBuy,
			BuyWan:    floatval(r["BILLBOARD_BUY_AMT"]) / 10000,
			SellWan:   floatval(r["BILLBOARD_SELL_AMT"]) / 10000,
			Turnover:  floatval(r["TURNOVERRATE"]),
		})
	}
	return stocks, nil
}

// ── Lockup Expiry ─────────────────────────────────────────────────────

// FetchLockupExpiry fetches lockup expiry history for a stock.
func (a *EastMoneySignalsAdapter) FetchLockupExpiry(ctx context.Context, code string) ([]LockupExpiry, error) {
	filter := fmt.Sprintf(`(SECURITY_CODE="%s")`, code)

	rows, err := a.queryDatacenter("RPT_LIFT_STAGE", filter, 20, "FREE_DATE", "-1")
	if err != nil {
		return nil, err
	}

	events := make([]LockupExpiry, 0, len(rows))
	for _, r := range rows {
		events = append(events, LockupExpiry{
			Date:   strval(r["FREE_DATE"]),
			Type:   strval(r["LIMITED_STOCK_TYPE"]),
			Shares: floatval(r["FREE_SHARES_NUM"]),
			Ratio:  floatval(r["FREE_RATIO"]),
		})
	}
	return events, nil
}

// ── Industry Ranking ──────────────────────────────────────────────────

// FetchIndustryRanks fetches industry ranking by daily change.
// Retries up to 2 times on transient errors (eastmoney push2 API is occasionally flaky).
func (a *EastMoneySignalsAdapter) FetchIndustryRanks(ctx context.Context, topN int) ([]IndustryRank, error) {
	const maxRetries = 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			time.Sleep(backoff)
			slog.Debug("eastmoney_signals industry: retrying", "attempt", attempt)
		}

		a.limiter.Wait()

		url := "https://push2.eastmoney.com/api/qt/clist/get" +
			"?pn=1&pz=100&po=1&np=1&fltt=2&invt=2" +
			"&fs=m:90+t:2" +
			"&fields=f2,f3,f4,f12,f13,f14,f104,f105,f128,f136,f140,f141,f207"

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req.Header.Set("Host", "push2.eastmoney.com")
		req.Header.Set("User-Agent", emUA)

		resp, err := a.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("eastmoney_signals industry: %w", err)
			continue
		}

		var result struct {
			Data struct {
				Diff []struct {
					F3   interface{} `json:"f3"`
					F12  string      `json:"f12"`
					F14  string      `json:"f14"`
					F104 int         `json:"f104"`
					F105 int         `json:"f105"`
					F140 string      `json:"f140"`
					F136 interface{} `json:"f136"`
				} `json:"diff"`
				Total int `json:"total"`
			} `json:"data"`
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if decodeErr != nil {
			lastErr = fmt.Errorf("eastmoney_signals industry: %w", decodeErr)
			continue
		}

		ranks := make([]IndustryRank, 0, len(result.Data.Diff))
		for i, d := range result.Data.Diff {
			ranks = append(ranks, IndustryRank{
				Rank:      i + 1,
				Name:      d.F14,
				Code:      d.F12,
				ChangePct: toFloat(d.F3),
				UpCount:   d.F104,
				DownCount: d.F105,
				Leader:    d.F140,
				LeaderChg: toFloat(d.F136),
			})
		}

		if topN > 0 && len(ranks) > topN {
			ranks = ranks[:topN]
		}

		slog.Debug("eastmoney_signals industry fetched", "total", result.Data.Total, "returned", len(ranks))
		return ranks, nil
	}

	return nil, lastErr
}

// ── Datacenter query helpers ──────────────────────────────────────────

func (a *EastMoneySignalsAdapter) queryDatacenter(reportName, filter string, pageSize int, sortCol, sortType string) ([]map[string]interface{}, error) {
	a.limiter.Wait()

	url := fmt.Sprintf(
		"https://datacenter-web.eastmoney.com/api/data/v1/get"+
			"?reportName=%s&columns=ALL&filter=%s"+
			"&pageNumber=1&pageSize=%d&sortColumns=%s&sortTypes=%s"+
			"&source=WEB&client=WEB",
		reportName, url.QueryEscape(filter), pageSize, sortCol, sortType,
	)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	req.Header.Set("User-Agent", emUA)
	req.Header.Set("Referer", "https://data.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("datacenter %s: %w", reportName, err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Data []map[string]interface{} `json:"data"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("datacenter %s: %w", reportName, err)
	}
	if !result.Success {
		return nil, fmt.Errorf("datacenter %s: api returned success=false", reportName)
	}
	return result.Result.Data, nil
}

// strval safely extracts a string from a map value.
func strval(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// floatval safely extracts a float64 from a map value.
func floatval(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		return parseFloat(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}
