package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CongressTradesAdapter fetches US congressional stock trading data
// from the free telep.io Capitol Trades API (no API key required).
type CongressTradesAdapter struct {
	client  *http.Client
	baseURL string
}

// NewCongressTradesAdapter creates a new CongressTradesAdapter.
func NewCongressTradesAdapter() *CongressTradesAdapter {
	return &CongressTradesAdapter{
		client:  newEastMoneyHTTPClient(15 * time.Second),
		baseURL: "https://trades.telep.io/api",
	}
}

// CongressTradeItem is a single trade from the telep.io API.
type CongressTradeItem struct {
	PoliticianName string `json:"politician_name"`
	Chamber        string `json:"chamber"`
	State          string `json:"state"`
	Party          string `json:"party"`
	Ticker         string `json:"ticker"`
	AssetName      string `json:"asset_name"`
	AssetType      string `json:"asset_type"`
	TransactionType string `json:"transaction_type"`
	TransactionDate string `json:"transaction_date"`
	DisclosureDate  string `json:"disclosure_date"`
	AmountText      string `json:"amount_text"`
	FilingURL       string `json:"filing_url"`
}

// congressTradesResponse is the API response envelope.
type congressTradesResponse struct {
	Trades  []CongressTradeItem `json:"trades"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
	Pages   int                 `json:"pages"`
}

// FetchRecentTrades returns the most recent N congressional trades.
func (a *CongressTradesAdapter) FetchRecentTrades(ctx context.Context, limit int) ([]CongressTradeItem, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	url := fmt.Sprintf("%s/trades?per_page=%d&page=1", a.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("congress_trades: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("congress_trades: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("congress_trades: http %d", resp.StatusCode)
	}

	var result congressTradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("congress_trades: json: %w", err)
	}

	return result.Trades, nil
}

// FetchTradesByTicker returns trades for a specific stock symbol.
func (a *CongressTradesAdapter) FetchTradesByTicker(ctx context.Context, ticker string, limit int) ([]CongressTradeItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	url := fmt.Sprintf("%s/trades?ticker=%s&per_page=%d", a.baseURL, ticker, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("congress_trades: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("congress_trades: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("congress_trades: http %d", resp.StatusCode)
	}

	var result congressTradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("congress_trades: json: %w", err)
	}

	return result.Trades, nil
}
