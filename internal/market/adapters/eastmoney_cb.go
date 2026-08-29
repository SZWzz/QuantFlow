package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"quantflow/internal/market"
	"time"
)

const eastMoneyCBURL = "https://datacenter.eastmoney.com/api/data/v1/get"

// CBQuote represents a convertible bond quote with analysis fields.
type CBQuote struct {
	Code            string  `json:"code"`              // 转债代码 e.g. "123218"
	Name            string  `json:"name"`              // 转债名称
	StockCode       string  `json:"stock_code"`        // 正股代码
	StockName       string  `json:"stock_name"`        // 正股名称
	Price           float64 `json:"price"`             // 转债价格
	StockPrice      float64 `json:"stock_price"`       // 正股价格
	PremiumRate     float64 `json:"premium_rate"`      // 转股溢价率 (%)
	ConversionPrice float64 `json:"conversion_price"`  // 转股价
	ConversionValue float64 `json:"conversion_value"`  // 转股价值
	YTM             float64 `json:"ytm"`               // 到期收益率 (%)
	BondValue       float64 `json:"bond_value"`        // 纯债价值
	Volume          float64 `json:"volume"`            // 成交量
	Amount          float64 `json:"amount"`            // 成交额
	CbChangePct     float64 `json:"cb_change_pct"`     // 转债涨跌幅
	StockChangePct  float64 `json:"stock_change_pct"`  // 正股涨跌幅
	RemainSize      float64 `json:"remain_size"`       // 剩余规模 (亿元)
	ListDate        string  `json:"list_date"`         // 上市日期
	MaturityDate    string  `json:"maturity_date"`     // 到期日期
	PutPrice        float64 `json:"put_price"`         // 回售触发价
	CallPrice       float64 `json:"call_price"`        // 强赎触发价
	Rating          string  `json:"rating"`            // 信用评级
	PutConvertPrice float64 `json:"put_convert_price"` // 回售价
}

// DualLowScore computes the dual-low ranking score (lower = better).
// Score = price + premium_rate (standard formula used by CB traders).
func (c *CBQuote) DualLowScore() float64 {
	return c.Price + c.PremiumRate
}

// EastMoneyCBAdapter fetches convertible bond data from EastMoney.
type EastMoneyCBAdapter struct {
	client *http.Client
}

// NewEastMoneyCBAdapter creates a CB data adapter.
func NewEastMoneyCBAdapter() *EastMoneyCBAdapter {
	return &EastMoneyCBAdapter{
		client: newEastMoneyHTTPClient(15 * time.Second),
	}
}

// FetchCBList fetches the full convertible bond list with real-time quotes.
func (a *EastMoneyCBAdapter) FetchCBList(ctx context.Context) ([]CBQuote, error) {
	// EastMoney convertible bond data API
	url := fmt.Sprintf("%s?reportName=RPT_BOND_CB_LIST&columns=ALL&pageNumber=1&pageSize=500"+
		"&sortTypes=-1&sortColumns=PUBLIC_START_DATE&source=WEB&client=WEB", eastMoneyCBURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cb adapter: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("cb adapter: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cb adapter: HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Data []struct {
				BONDCODE        string      `json:"BONDCODE"`
				BONDNAME        string      `json:"BONDNAME"`
				STOCKCODE       string      `json:"STOCKCODE"`
				STOCKNAME       string      `json:"STOCKNAME"`
				CLOSEPRICE      interface{} `json:"CLOSEPRICE"`
				STOCKPRICE      interface{} `json:"STOCKPRICE"`
				PREMIUMRATE     interface{} `json:"PREMIUMRATE"`
				CONVERSIONPRICE interface{} `json:"CONVERSIONPRICE"`
				CONVERSIONVALUE interface{} `json:"CONVERSIONVALUE"`
				YTM             interface{} `json:"YTM"`
				BONDVALUE       interface{} `json:"BONDVALUE"`
				VOLUME          interface{} `json:"VOLUME"`
				AMOUNT          interface{} `json:"AMOUNT"`
				CBCHANGEPCT     interface{} `json:"CBCHANGEPCT"`
				STOCKCHANGEPCT  interface{} `json:"STOCKCHANGEPCT"`
				REMAINSIZE      interface{} `json:"REMAINSIZE"`
				LISTDATE        string      `json:"LISTDATE"`
				MATURITYDATE    string      `json:"MATURITYDATE"`
				PUTPRICE        interface{} `json:"PUTPRICE"`
				CALLPRICE       interface{} `json:"CALLPRICE"`
				RATING          string      `json:"RATING"`
				PUTCONVERTPRICE interface{} `json:"PUTCONVERTPRICE"`
			} `json:"data"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("cb adapter: parse error: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("cb adapter: API returned success=false")
	}

	quotes := make([]CBQuote, 0, len(result.Result.Data))
	for _, d := range result.Result.Data {
		quotes = append(quotes, CBQuote{
			Code:            d.BONDCODE,
			Name:            d.BONDNAME,
			StockCode:       d.STOCKCODE,
			StockName:       d.STOCKNAME,
			Price:           toFloat64(d.CLOSEPRICE),
			StockPrice:      toFloat64(d.STOCKPRICE),
			PremiumRate:     toFloat64(d.PREMIUMRATE),
			ConversionPrice: toFloat64(d.CONVERSIONPRICE),
			ConversionValue: toFloat64(d.CONVERSIONVALUE),
			YTM:             toFloat64(d.YTM),
			BondValue:       toFloat64(d.BONDVALUE),
			Volume:          toFloat64(d.VOLUME),
			Amount:          toFloat64(d.AMOUNT),
			CbChangePct:     toFloat64(d.CBCHANGEPCT),
			StockChangePct:  toFloat64(d.STOCKCHANGEPCT),
			RemainSize:      toFloat64(d.REMAINSIZE),
			ListDate:        d.LISTDATE,
			MaturityDate:    d.MATURITYDATE,
			PutPrice:        toFloat64(d.PUTPRICE),
			CallPrice:       toFloat64(d.CALLPRICE),
			Rating:          d.RATING,
			PutConvertPrice: toFloat64(d.PUTCONVERTPRICE),
		})
	}

	return quotes, nil
}

// toFloat64 converts an interface{} (json.Number or string) to float64.
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		return parseFloatSafe(val)
	default:
		return 0
	}
}
