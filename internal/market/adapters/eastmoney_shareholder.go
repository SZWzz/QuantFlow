package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ShareholderRecord represents a single shareholder position.
type ShareholderRecord struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Shares      float64 `json:"shares"`
	Pct         float64 `json:"pct"`
	Change      float64 `json:"change"`
	MarketValue float64 `json:"market_value"`
	ReportDate  string  `json:"report_date"`
}

// EastMoneyShareholderAdapter fetches shareholder data from EastMoney.
type EastMoneyShareholderAdapter struct {
	client *http.Client
}

// NewEastMoneyShareholderAdapter creates a new adapter.
func NewEastMoneyShareholderAdapter() *EastMoneyShareholderAdapter {
	return &EastMoneyShareholderAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchTop10Holders fetches top-10 circulating shareholders for a symbol.
func (a *EastMoneyShareholderAdapter) FetchTop10Holders(ctx context.Context, symbol string) ([]ShareholderRecord, error) {
	secid := toEastMoneySecID(symbol)
	url := fmt.Sprintf("https://datacenter.eastmoney.com/api/data/v1/get?"+
		"reportName=RPT_F10_SHAREHOLDER_TOP10&columns=ALL&"+
		"filter=(SECURITY_CODE=%q)&pageSize=10&sortTypes=-1&sortColumns=HOLD_NUM",
		secid)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shareholder fetch: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Data []struct {
				HOLDERNAME  string  `json:"HOLDER_NAME"`
				HOLDERTYPE  string  `json:"HOLDER_TYPE"`
				HOLDNUM     float64 `json:"HOLD_NUM"`
				HOLDRATIO   float64 `json:"HOLD_RATIO"`
				CHANGENUM   float64 `json:"CHANGE_NUM"`
				MARKETVALUE float64 `json:"MARKET_VALUE"`
				ENDDATE     string  `json:"END_DATE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("shareholder parse: %w", err)
	}

	var records []ShareholderRecord
	for _, d := range result.Result.Data {
		records = append(records, ShareholderRecord{
			Name:        d.HOLDERNAME,
			Type:        d.HOLDERTYPE,
			Shares:      d.HOLDNUM,
			Pct:         d.HOLDRATIO,
			Change:      d.CHANGENUM,
			MarketValue: d.MARKETVALUE,
			ReportDate:  d.ENDDATE,
		})
	}
	return records, nil
}
