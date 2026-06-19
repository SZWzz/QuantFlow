package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// THSHotStock represents a single strong-performing stock with its reason tags.
type THSHotStock struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Reason   string  `json:"reason"`    // 题材归因 tags, e.g. "算力租赁+Token工厂+AI政务"
	Close    float64 `json:"close"`
	Change   float64 `json:"change_amt"` // 涨跌额
	ChangePct float64 `json:"change_pct"` // 涨幅%
	Turnover  float64 `json:"turnover_pct"` // 换手率%
	Amount   float64 `json:"amount"`       // 成交额(元)
	Volume   float64 `json:"volume"`       // 成交量(股)
	DDENet   float64 `json:"dde_net"`      // 大单净量
	Market   string  `json:"market"`       // 市场(沪/深/北)
}

// THSHotAdapter fetches daily strong-performing stocks with analyst-curated
// reason tags from 同花顺 (10jqka). Based on a-stock-data SKILL §3.1.
//
// Core value: not just "which stocks are strong", but "WHY they're strong"
// — the reason field is manually curated by THS editors.
type THSHotAdapter struct {
	client *http.Client
}

// NewTHSHotAdapter creates a new THS hot stocks adapter.
func NewTHSHotAdapter() *THSHotAdapter {
	return &THSHotAdapter{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *THSHotAdapter) Name() string { return "ths_hot" }

func (a *THSHotAdapter) IsAvailable(ctx context.Context) bool {
	_, err := a.FetchHotStocks(ctx, "")
	return err == nil
}

// FetchHotStocks fetches today's strong-performing stocks with reason tags.
// date: "YYYY-MM-DD" format, empty string = today.
// Returns typically ~125 stocks at ~73ms latency.
func (a *THSHotAdapter) FetchHotStocks(ctx context.Context, date string) ([]THSHotStock, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	url := fmt.Sprintf(
		"http://zx.10jqka.com.cn/event/api/getharden/"+
			"date/%s/orderby/date/orderway/desc/charset/GBK/", date,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ths_hot: %w", err)
	}
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/117.0.0.0 Safari/537.36")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ths_hot: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ths_hot: http %d", resp.StatusCode)
	}

	var result struct {
		ErroCode int `json:"errocode"`
		Data     []struct {
			Code           string      `json:"code"`
			Name           string      `json:"name"`
			Reason         string      `json:"reason"`
			Close          interface{} `json:"close"`
			Zhangdie       interface{} `json:"zhangdie"`
			Zhangfu        interface{} `json:"zhangfu"`
			Huanshou       interface{} `json:"huanshou"`
			Chengjiaoe     interface{} `json:"chengjiaoe"`
			Chengjiaoliang interface{} `json:"chengjiaoliang"`
			Ddejingliang   interface{} `json:"ddejingliang"`
			Market         interface{} `json:"market"` // API returns number, not string
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ths_hot: json: %w", err)
	}

	if result.ErroCode != 0 {
		return nil, fmt.Errorf("ths_hot: api error code %d", result.ErroCode)
	}

	stocks := make([]THSHotStock, 0, len(result.Data))
	for _, d := range result.Data {
		stocks = append(stocks, THSHotStock{
			Code:      d.Code,
			Name:      d.Name,
			Reason:    d.Reason,
			Close:     toFloat(d.Close),
			Change:    toFloat(d.Zhangdie),
			ChangePct: toFloat(d.Zhangfu),
			Turnover:  toFloat(d.Huanshou),
			Amount:    toFloat(d.Chengjiaoe),
			Volume:    toFloat(d.Chengjiaoliang),
			DDENet:    toFloat(d.Ddejingliang),
			Market:    strval(d.Market),
		})
	}

	slog.Debug("ths_hot fetched", "date", date, "count", len(stocks))
	return stocks, nil
}
