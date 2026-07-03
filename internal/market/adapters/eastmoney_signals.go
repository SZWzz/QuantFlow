package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"quantflow/internal/market"
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

// EastMoneySignalsAdapter provides dragon tiger board, lockup calendar,
// and industry ranking data via the EastMoney datacenter API.
// Based on a-stock-data SKILL §3.5-3.8.
type EastMoneySignalsAdapter struct {
	client  *http.Client
	limiter *EastMoneyRateLimiter

	availMu       sync.RWMutex
	availResult   bool
	availChecked time.Time
}

// NewEastMoneySignalsAdapter creates a new signals adapter.
func NewEastMoneySignalsAdapter() *EastMoneySignalsAdapter {
	return &EastMoneySignalsAdapter{
		client:  newEastMoneyHTTPClient(15 * time.Second),
		limiter: GlobalEMLimiter,
	}
}

func (a *EastMoneySignalsAdapter) Name() string { return "eastmoney_signals" }

func (a *EastMoneySignalsAdapter) Markets() []string  { return []string{"CN"} }
func (a *EastMoneySignalsAdapter) RequiresAuth() bool { return false }

func (a *EastMoneySignalsAdapter) IsAvailable(ctx context.Context) bool {
	// Cache availability result for 120s to avoid self-DOS on rate-limited APIs.
	a.availMu.RLock()
	if time.Since(a.availChecked) < 120*time.Second {
		ok := a.availResult
		a.availMu.RUnlock()
		return ok
	}
	a.availMu.RUnlock()

	a.availMu.Lock()
	defer a.availMu.Unlock()
	// Double-check after acquiring write lock
	if time.Since(a.availChecked) < 120*time.Second {
		return a.availResult
	}
	_, err := a.FetchIndustryRanks(ctx, "CN", 5)
	a.availResult = err == nil
	a.availChecked = time.Now()
	return a.availResult
}

func (a *EastMoneySignalsAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	return nil, fmt.Errorf("eastmoney_signals: quote not supported, use eastmoney adapter instead")
}

func (a *EastMoneySignalsAdapter) FetchOHLCV(ctx context.Context, symbol, interval, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	return nil, fmt.Errorf("eastmoney_signals: OHLCV not supported, use eastmoney adapter instead")
}

func (a *EastMoneySignalsAdapter) HealthCheck(ctx context.Context) error {
	if !a.IsAvailable(ctx) {
		return fmt.Errorf("eastmoney_signals: not available")
	}
	return nil
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
// Falls back to up to 5 earlier calendar dates when the requested date has no data
// (e.g. before market close on the current day).
func (a *EastMoneySignalsAdapter) FetchDailyDragonTiger(ctx context.Context, tradeDate string, minNetBuy float64) ([]DragonTigerStock, error) {
	var lastErr error
	date, err := time.Parse("2006-01-02", tradeDate)
	if err != nil {
		date = time.Now()
	}

	for attempt := 0; attempt < 5; attempt++ {
		tryDate := date.Add(-time.Duration(attempt) * 24 * time.Hour).Format("2006-01-02")
		filter := fmt.Sprintf(`(TRADE_DATE>='%s')(TRADE_DATE<='%s')`, tryDate, tryDate)

		rows, err := a.queryDatacenter("RPT_DAILYBILLBOARD_DETAILSNEW", filter, 500, "BILLBOARD_NET_AMT", "-1")
		if err != nil {
			lastErr = err
			continue
		}
		if len(rows) == 0 {
			lastErr = fmt.Errorf("datacenter RPT_DAILYBILLBOARD_DETAILSNEW: no data for %s", tryDate)
			continue
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
	return nil, lastErr
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

// ── IPO Calendar ──────────────────────────────────────────────────────

// IPORecord represents a new stock issuance/listing record.
type IPORecord struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	IssuePrice     float64 `json:"issue_price"`
	PE             float64 `json:"pe"`
	SubscriptionDate string `json:"subscription_date"`
	ListingDate    string  `json:"listing_date"`
	LotteryRate    float64 `json:"lottery_rate"`
	IssueVolume    float64 `json:"issue_volume"`
	Status         string  `json:"status"`
}

// FetchIPOCalendar fetches upcoming and recent IPO calendar from EastMoney
// datacenter. startDate/endDate format: "YYYY-MM-DD". Max pageSize is 500.
func (a *EastMoneySignalsAdapter) FetchIPOCalendar(ctx context.Context, startDate, endDate string, pageSize int) ([]IPORecord, error) {
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}
	filter := fmt.Sprintf(`(LISTING_DATE>='%s')(LISTING_DATE<='%s')`, startDate, endDate)
	rows, err := a.queryDatacenter("RPT_NEW_SHARE_ISSUE", filter, pageSize, "LISTING_DATE", "-1")
	if err != nil {
		return nil, fmt.Errorf("eastmoney_signals ipo: %w", err)
	}
	records := make([]IPORecord, 0, len(rows))
	for _, r := range rows {
		listingDate := strval(r["LISTING_DATE"])
		if listingDate == "" {
			listingDate = strval(r["IPO_DATE"])
		}
		subDate := strval(r["SUBSCRIPTION_DATE"])
		if subDate == "" {
			subDate = strval(r["APPLY_DATE"])
		}
		records = append(records, IPORecord{
			Code:             strval(r["SECURITY_CODE"]),
			Name:             strval(r["SECURITY_NAME_ABBR"]),
			IssuePrice:       floatval(r["ISSUE_PRICE"]),
			PE:               floatval(r["ISSUE_PE"]),
			SubscriptionDate: subDate,
			ListingDate:      listingDate,
			LotteryRate:      floatval(r["LOTTERY_RATE"]),
			IssueVolume:      floatval(r["ISSUE_VOLUME"]),
			Status:           strval(r["STATUS_CODE"]),
		})
	}
	return records, nil
}

// ── Industry Ranking ──────────────────────────────────────────────────

// FetchIndustryRanks fetches industry ranking by daily change.
// Tries once and returns immediately on failure — better to show empty
// sector data than to block the market overview panel for 5+ seconds.
func (a *EastMoneySignalsAdapter) FetchIndustryRanks(ctx context.Context, mkt string, topN int) ([]market.IndustryRank, error) {
	if mkt != "CN" {
		return []market.IndustryRank{}, nil
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
		return nil, market.NewTransientErrorf("eastmoney_signals industry: %w", err)
	}
	defer resp.Body.Close()

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

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, market.NewTransientErrorf("eastmoney_signals industry: %w", err)
	}

	ranks := make([]market.IndustryRank, 0, len(result.Data.Diff))
	for i, d := range result.Data.Diff {
		ranks = append(ranks, market.IndustryRank{
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

// ── Abnormal Stocks (涨跌停/异动) ─────────────────────────────────────

// push2MarketFilter returns the EastMoney push2 fs parameter for stock lists.
func push2MarketFilter(market string) string {
	switch market {
	case "SH":
		return "m:0+t:6"
	case "SZ":
		return "m:0+t:7"
	default:
		return ""
	}
}

// FetchAbnormalStocks fetches stocks with extreme change_pct via EastMoney push2.
// Makes two requests: top N by change_pct descending (limit-up) and ascending
// (limit-down), then merges and returns combined results.
func (a *EastMoneySignalsAdapter) FetchAbnormalStocks(ctx context.Context, market string, topN int) ([]AbnormalStock, error) {
	fs := push2MarketFilter(market)
	if fs == "" {
		return nil, fmt.Errorf("eastmoney_signals: unsupported market %q", market)
	}
	if topN <= 0 {
		topN = 100
	}

	seen := make(map[string]bool)
	var stocks []AbnormalStock

	for _, ascending := range []bool{false, true} {
		po := 1
		if ascending {
			po = 0
		}
		url := fmt.Sprintf(
			"https://push2.eastmoney.com/api/qt/clist/get?pn=1&pz=%d&po=%d&np=1&fltt=2&invt=2&fs=%s&fields=f2,f3,f5,f6,f12,f14",
			topN, po, fs,
		)

		// Retry loop: EastMoney CDN intermittently drops connections (EOF).
		var result struct {
			Data struct {
				Diff []struct {
					F2  interface{} `json:"f2"`
					F3  interface{} `json:"f3"`
					F5  interface{} `json:"f5"`
					F6  interface{} `json:"f6"`
					F12 string      `json:"f12"`
					F14 string      `json:"f14"`
				} `json:"diff"`
			} `json:"data"`
		}
		retryErr := error(nil)
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				time.Sleep(time.Duration(500*attempt) * time.Millisecond)
			}
			a.limiter.Wait()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			req.Header.Set("Host", "push2.eastmoney.com")
			req.Header.Set("User-Agent", emUA)

			resp, err := a.client.Do(req)
			if err != nil {
				retryErr = fmt.Errorf("eastmoney_signals push2: %w", err)
				continue
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				retryErr = fmt.Errorf("eastmoney_signals push2: decode: %w", err)
				continue
			}
			resp.Body.Close()
			retryErr = nil
			break
		}
		if retryErr != nil {
			return nil, retryErr
		}

		for _, d := range result.Data.Diff {
			if seen[d.F12] {
				continue
			}
			seen[d.F12] = true
			chg := toFloat(d.F3)
			reason := ""
			if chg >= 9.5 {
				reason = "涨停"
			} else if chg <= -9.5 {
				reason = "跌停"
			}
			stocks = append(stocks, AbnormalStock{
				Symbol:    d.F12,
				Name:      d.F14,
				Price:     toFloat(d.F2),
				ChangePct: chg,
				Reason:    reason,
				Volume:    toFloat(d.F5),
				Turnover:  toFloat(d.F6),
			})
		}
	}

	return stocks, nil
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
