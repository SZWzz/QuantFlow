package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UnlockEvent represents a restricted share unlock event.
type UnlockEvent struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	UnlockDate   string  `json:"unlock_date"`
	UnlockShares float64 `json:"unlock_shares"`
	UnlockPct    float64 `json:"unlock_pct"`
	FloatRatio   float64 `json:"float_ratio"`
	MarketValue  float64 `json:"market_value"`
}

// EastMoneyUnlockAdapter fetches unlock calendar from EastMoney.
type EastMoneyUnlockAdapter struct {
	client *http.Client
}

// NewEastMoneyUnlockAdapter creates a new adapter.
func NewEastMoneyUnlockAdapter() *EastMoneyUnlockAdapter {
	return &EastMoneyUnlockAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchUpcoming returns unlock events within the next N days.
func (a *EastMoneyUnlockAdapter) FetchUpcoming(ctx context.Context, days int) ([]UnlockEvent, error) {
	url := fmt.Sprintf("https://datacenter.eastmoney.com/api/data/v1/get?"+
		"reportName=RPT_F10_SHAREHOLDER_UNLOCK&columns=ALL&"+
		"filter=(PLAN_UNLOCK_DATE>=^%s^)&pageSize=100&"+
		"sortTypes=1&sortColumns=PLAN_UNLOCK_DATE",
		time.Now().Format("2006-01-02"))

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unlock fetch: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Data []struct {
				SECURITYCODE   string  `json:"SECURITY_CODE"`
				SECURITYNAME   string  `json:"SECURITY_NAME_ABBR"`
				PLANUNLOCKDATE string  `json:"PLAN_UNLOCK_DATE"`
				UNLOCKNUM      float64 `json:"UNLOCK_NUM"`
				UNLOCKRATIO    float64 `json:"UNLOCK_RATIO"`
				FLOATRATIO     float64 `json:"FLOAT_RATIO"`
				MARKETVALUE    float64 `json:"MARKET_VALUE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unlock parse: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	var events []UnlockEvent
	for _, d := range result.Result.Data {
		if d.PLANUNLOCKDATE > cutoff {
			continue
		}
		events = append(events, UnlockEvent{
			Symbol:       d.SECURITYCODE,
			Name:         d.SECURITYNAME,
			UnlockDate:   d.PLANUNLOCKDATE,
			UnlockShares: d.UNLOCKNUM,
			UnlockPct:    d.UNLOCKRATIO,
			FloatRatio:   d.FLOATRATIO,
			MarketValue:  d.MARKETVALUE,
		})
	}
	return events, nil
}
