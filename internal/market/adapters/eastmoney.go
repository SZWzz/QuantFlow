package adapters

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quantflow/internal/market"
)

const (
	eastmoneyURL      = "https://push2.eastmoney.com/api/qt/stock/get"
	eastmoneyKlineURL = "https://push2his.eastmoney.com/api/qt/stock/kline/get"
)

// EastMoneyAdapter fetches A-share data from EastMoney (free, no auth).
type EastMoneyAdapter struct {
	client *http.Client
}

// NewEastMoneyAdapter creates a new EastMoney adapter.
func NewEastMoneyAdapter() *EastMoneyAdapter {
	return &EastMoneyAdapter{
		client: newEastMoneyHTTPClient(10 * time.Second),
	}
}

// newEastMoneyHTTPClient creates an HTTP/1.1-only client for EastMoney APIs.
// EastMoney CDN does not handle Go's HTTP/2 ALPN negotiation — connections get
// dropped with EOF if HTTP/2 is allowed. TLSNextProto empty map prevents it.
// All EastMoney adapters MUST use this helper.
func newEastMoneyHTTPClient(timeout time.Duration) *http.Client {
	tr := &http.Transport{
		TLSNextProto:    make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

func (a *EastMoneyAdapter) Name() string      { return "eastmoney" }
func (a *EastMoneyAdapter) Markets() []string  { return []string{"CN"} }
func (a *EastMoneyAdapter) RequiresAuth() bool { return false }

func (a *EastMoneyAdapter) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		eastmoneyURL+"?secid=1.600519&fields=f43,f44", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (a *EastMoneyAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	secid := toEastMoneySecID(symbol)
	url := fmt.Sprintf("%s?secid=%s&fields=f43,f44,f45,f46,f47,f48,f50,f57,f58,f116,f117,f162,f167,f169,f170,f171",
		eastmoneyURL, secid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("eastmoney: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney: HTTP %d", resp.StatusCode)
	}

	var result eastMoneyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney: parse error: %w", err)
	}

	if result.Data == nil {
		return nil, fmt.Errorf("eastmoney: no data for %s", symbol)
	}

	d := result.Data
	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Name:      d.F58,
		Last:      d.F43 / 100.0,  // f43: 最新价 (分)
		Open:      d.F46 / 100.0,  // f46: 开盘价
		High:      d.F44 / 100.0,  // f44: 最高价
		Low:       d.F45 / 100.0,  // f45: 最低价
		Volume:    d.F47,           // f47: 成交量 (手)
		Change:    d.F169 / 100.0,  // f169: 涨跌额
		ChangePct: d.F170 / 100.0,  // f170: 涨跌幅
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *EastMoneyAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, fqfactor string, start, end int64) ([]market.OHLCVBar, error) {
	secid := toEastMoneySecID(symbol)

	// Map interval to EastMoney klt code
	klt := "101" // 日K
	switch strings.ToUpper(interval) {
	case "1W":
		klt = "102" // 周K
	case "1M", "1MONTH":
		klt = "103" // 月K
	}

	// Map fqfactor to EastMoney fqt parameter:
	//   hfq → fqt=2 (后复权, backward-adjusted)
	//   qfq → fqt=1 (前复权, forward-adjusted)
	//   ""  → fqt=0 (不复权, raw prices)
	// Default to hfq for backtesting to avoid look-ahead bias.
	fqt := 2 // default: hfq
	switch strings.ToLower(fqfactor) {
	case "qfq":
		fqt = 1
	case "":
		fqt = 0
	}

	url := fmt.Sprintf("%s?secid=%s&klt=%s&fqt=%d&beg=%s&end=%s&fields=f51,f52,f53,f54,f55,f56,f57",
		eastmoneyKlineURL, secid, klt, fqt,
		time.Unix(start, 0).Format("20060102"),
		time.Unix(end, 0).Format("20060102"))

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("eastmoney kline: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("eastmoney kline: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var klineResp eastMoneyKlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&klineResp); err != nil {
		return nil, fmt.Errorf("eastmoney kline: parse error: %w", err)
	}

	if klineResp.Data == nil || klineResp.Data.Klines == nil {
		return nil, fmt.Errorf("eastmoney kline: no data for %s", symbol)
	}

	endDate := time.Unix(end, 0)
	bars := make([]market.OHLCVBar, 0, len(klineResp.Data.Klines))
	for _, kline := range klineResp.Data.Klines {
		// Each kline is comma-separated: f51,f52,f53,f54,f55,f56,f57
		// f51=date, f52=open, f53=close, f54=high, f55=low, f56=volume, f57=amount
		fields := strings.Split(kline, ",")
		if len(fields) < 6 {
			continue
		}

		// Parse date for post-filtering (prevents look-ahead: EastMoney may return
		// data beyond requested end date).
		barDate, err := time.Parse("2006-01-02", fields[0])
		if err != nil {
			continue
		}
		if barDate.After(endDate) {
			continue // discard bars beyond the requested range
		}

		// Volume from EastMoney is in 手 (lots, 1 lot = 100 shares). Normalize to
		// shares for consistency with other CN adapters (TuShare, Sina, Tencent).
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   fields[0],
			Open:   parseFloatSafe(fields[1]),
			Close:  parseFloatSafe(fields[2]),
			High:   parseFloatSafe(fields[3]),
			Low:    parseFloatSafe(fields[4]),
			Volume: parseFloatSafe(fields[5]) * 100,
		})
	}

	return bars, nil
}

// EastMoneyStockInfo holds basic company information from EastMoney push2 API.
// Based on a-stock-data SKILL §6.3.
type EastMoneyStockInfo struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Industry    string  `json:"industry"`     // 行业分类
	TotalShares float64 `json:"total_shares"` // 总股本(股)
	FloatShares float64 `json:"float_shares"` // 流通股(股)
	MarketCap   float64 `json:"market_cap"`   // 总市值(元)
	FloatCap    float64 `json:"float_cap"`    // 流通市值(元)
	ListDate    string  `json:"list_date"`    // 上市日期 YYYYMMDD
	Price       float64 `json:"price"`        // 最新价(元)
}

// FetchStockInfo returns basic company information for a stock.
func (a *EastMoneyAdapter) FetchStockInfo(ctx context.Context, symbol string) (*EastMoneyStockInfo, error) {
	secid := toEastMoneySecID(symbol)
	url := fmt.Sprintf("%s?secid=%s&fields=f43,f57,f58,f84,f85,f116,f117,f127,f189",
		eastmoneyURL, secid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("eastmoney stock_info: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eastmoney stock_info: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney stock_info: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			F57  string      `json:"f57"`  // 代码
			F58  string      `json:"f58"`  // 名称
			F127 string      `json:"f127"` // 行业
			F84  float64     `json:"f84"`  // 总股本(股)
			F85  float64     `json:"f85"`  // 流通股(股)
			F116 float64     `json:"f116"` // 总市值(元)
			F117 float64     `json:"f117"` // 流通市值(元)
			F189 interface{} `json:"f189"` // 上市日期 (int or string)
			F43  float64     `json:"f43"`  // 最新价(分)
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("eastmoney stock_info: parse: %w", err)
	}

	info := &EastMoneyStockInfo{
		Code:        result.Data.F57,
		Name:        result.Data.F58,
		Industry:    result.Data.F127,
		TotalShares: result.Data.F84,
		FloatShares: result.Data.F85,
		MarketCap:   result.Data.F116,
		FloatCap:    result.Data.F117,
		Price:       result.Data.F43 / 100.0, // 分→元
	}

	// List date can be int (e.g., 20010827) or string ("2001-08-27")
	switch v := result.Data.F189.(type) {
	case float64:
		if v > 19000000 {
			info.ListDate = fmt.Sprintf("%08.0f", v) // YYYYMMDD
		}
	case string:
		info.ListDate = strings.ReplaceAll(v, "-", "")
	}

	return info, nil
}

func (a *EastMoneyAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.FetchQuote(ctx, "600519")
	return err
}

// toEastMoneySecID converts a symbol like "600519.SH" or "000001.SZ" to EastMoney secid "1.600519" or "0.000001".
func toEastMoneySecID(symbol string) string {
	id, err := market.NormalizeCN(symbol)
	if err != nil {
		return "0." + symbol
	}
	return id.ToEastMoney()
}

type eastMoneyResponse struct {
	Data *eastMoneyData `json:"data"`
}

type eastMoneyKlineResponse struct {
	Data *eastMoneyKlineData `json:"data"`
}

type eastMoneyKlineData struct {
	Code   string   `json:"code"`
	Name   string   `json:"name"`
	Klines []string `json:"klines"`
}

type eastMoneyData struct {
	F43  float64 `json:"f43"`  // 最新价
	F44  float64 `json:"f44"`  // 最高价
	F45  float64 `json:"f45"`  // 最低价
	F46  float64 `json:"f46"`  // 开盘价
	F47  float64 `json:"f47"`  // 成交量
	F48  float64 `json:"f48"`  // 成交额
	F50  float64 `json:"f50"`  // 量比
	F57  string  `json:"f57"`  // 名称
	F58  string  `json:"f58"`  // 股票名称
	F116 float64 `json:"f116"` // 总市值
	F117 float64 `json:"f117"` // 流通市值
	F162 float64 `json:"f162"` // 市盈率
	F167 float64 `json:"f167"` // 换手率
	F169 float64 `json:"f169"` // 涨跌额
	F170 float64 `json:"f170"` // 涨跌幅
}
