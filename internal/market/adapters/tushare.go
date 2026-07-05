package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"quantflow/internal/market"
)

const tushareBaseURL = "https://api.tushare.pro"

// TuShareAdapter fetches A-share data from TuShare Pro (requires token).
// TuShare provides the most comprehensive financial data for Chinese markets.
type TuShareAdapter struct {
	client *http.Client
	token  string
}

// NewTuShareAdapter creates a new TuShare adapter.
// Token is read from TUSHARE_TOKEN env var.
func NewTuShareAdapter() *TuShareAdapter {
	return &TuShareAdapter{
		client: &http.Client{Timeout: 15 * time.Second},
		token:  os.Getenv("TUSHARE_TOKEN"),
	}
}

func (a *TuShareAdapter) Name() string      { return "tushare" }
func (a *TuShareAdapter) Markets() []string  { return []string{"CN"} }
func (a *TuShareAdapter) RequiresAuth() bool { return true }

func (a *TuShareAdapter) IsAvailable(ctx context.Context) bool {
	if a.token == "" {
		return false
	}
	// Lightweight check: try a simple API call
	_, err := a.callAPI(ctx, "stock_basic", map[string]any{"limit": "1"}, []string{"ts_code"})
	return err == nil
}

func (a *TuShareAdapter) FetchQuote(ctx context.Context, symbol string) (*market.QuoteSnapshot, error) {
	if a.token == "" {
		return nil, fmt.Errorf("tushare: TUSHARE_TOKEN not set")
	}

	// TuShare daily quote
	result, err := a.callAPI(ctx, "daily", map[string]any{
		"ts_code":    symbol,
		"limit":      "1",
	}, []string{"ts_code", "trade_date", "open", "high", "low", "close", "vol", "pct_chg", "change"})
	if err != nil {
		return nil, err
	}

	items := result.Items
	if len(items) == 0 {
		return nil, fmt.Errorf("tushare: no data for %s", symbol)
	}

	row := items[0]
	return &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      rowFloat(row, "close"),
		Open:      rowFloat(row, "open"),
		High:      rowFloat(row, "high"),
		Low:       rowFloat(row, "low"),
		Volume:    rowFloat(row, "vol") * 100, // 手→股
		Change:    rowFloat(row, "change"),
		ChangePct: rowFloat(row, "pct_chg"),
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func (a *TuShareAdapter) FetchOHLCV(ctx context.Context, symbol string, interval string, _ string, start, end int64) ([]market.OHLCVBar, error) {
	if a.token == "" {
		return nil, fmt.Errorf("tushare: TUSHARE_TOKEN not set")
	}

	startDate := time.Unix(start, 0).Format("20060102")
	endDate := time.Unix(end, 0).Format("20060102")

	result, err := a.callAPI(ctx, "daily", map[string]any{
		"ts_code":    symbol,
		"start_date": startDate,
		"end_date":   endDate,
	}, []string{"ts_code", "trade_date", "open", "high", "low", "close", "vol"})
	if err != nil {
		return nil, err
	}

	bars := make([]market.OHLCVBar, 0, len(result.Items))
	for _, row := range result.Items {
		bars = append(bars, market.OHLCVBar{
			Symbol: symbol,
			Date:   rowString(row, "trade_date"),
			Open:   rowFloat(row, "open"),
			High:   rowFloat(row, "high"),
			Low:    rowFloat(row, "low"),
			Close:  rowFloat(row, "close"),
			Volume: rowFloat(row, "vol") * 100,
		})
	}
	return bars, nil
}

// StockBasicInfo holds stock listing metadata for backtest survivorship handling.
type StockBasicInfo struct {
	TsCode     string `json:"ts_code"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Area       string `json:"area"`
	Industry   string `json:"industry"`
	Market     string `json:"market"`
	ListDate   string `json:"list_date"`
	DelistDate string `json:"delist_date"`
	ListStatus string `json:"list_status"` // L=listed, D=delisted, P=paused
}

// FetchStockList returns all stocks (listed + delisted) for backtest universe construction.
// Uses list_status='L+D' to eliminate survivorship bias.
func (a *TuShareAdapter) FetchStockList(ctx context.Context) ([]StockBasicInfo, error) {
	if a.token == "" {
		return nil, fmt.Errorf("tushare: TUSHARE_TOKEN not set")
	}

	result, err := a.callAPI(ctx, "stock_basic", map[string]any{
		"list_status": "L,D",
	}, []string{"ts_code", "symbol", "name", "area", "industry", "market", "list_date", "delist_date", "list_status"})
	if err != nil {
		return nil, err
	}

	stocks := make([]StockBasicInfo, 0, len(result.Items))
	for _, row := range result.Items {
		stocks = append(stocks, StockBasicInfo{
			TsCode:     rowString(row, "ts_code"),
			Symbol:     rowString(row, "symbol"),
			Name:       rowString(row, "name"),
			Area:       rowString(row, "area"),
			Industry:   rowString(row, "industry"),
			Market:     rowString(row, "market"),
			ListDate:   rowString(row, "list_date"),
			DelistDate: rowString(row, "delist_date"),
			ListStatus: rowString(row, "list_status"),
		})
	}
	return stocks, nil
}

func (a *TuShareAdapter) HealthCheck(ctx context.Context) error {
	if a.token == "" {
		return fmt.Errorf("tushare: TUSHARE_TOKEN not set")
	}
	_, err := a.callAPI(ctx, "stock_basic", map[string]any{"limit": "1"}, []string{"ts_code"})
	return err
}

// callAPI makes a generic TuShare API call.
func (a *TuShareAdapter) callAPI(ctx context.Context, apiName string, params map[string]any, fields []string) (*tushareResponse, error) {
	reqBody := map[string]any{
		"api_name": apiName,
		"token":    a.token,
		"params":   params,
		"fields":   fieldsToString(fields),
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tushare: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", tushareBaseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tushare: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, market.NewTransientErrorf("tushare: http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tushare: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result tushareResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("tushare: parse error: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("tushare: API error code=%d msg=%s", result.Code, result.Msg)
	}

	// TuShare returns data as fields+items (parallel arrays), not maps.
	// Convert to the map form expected by FetchQuote/FetchOHLCV.
	if len(result.Items) == 0 && len(result.Data.Fields) > 0 {
		result.Items = zipFieldsAndItems(result.Data.Fields, result.Data.Items)
	}

	return &result, nil
}

type tushareResponse struct {
	Code  int              `json:"code"`
	Msg   string           `json:"msg"`
	Data  tushareData      `json:"data"`
	Items []map[string]any `json:"items"` // may be in data.fields+data.items format
}

type tushareData struct {
	Fields []string   `json:"fields"`
	Items  [][]any    `json:"items"`
}

func fieldsToString(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ","
		}
		result += f
	}
	return result
}

func rowFloat(row map[string]any, key string) float64 {
	v, ok := row[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		return parseFloatSafe(val)
	}
	return 0
}

func rowString(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(v)
}

// zipFieldsAndItems converts TuShare's fields+items parallel-array format
// into []map[string]any, matching the API shape expected by callers.
// TuShare returns column names in data.fields and row values in data.items
// as [][]any. This function zips them into named maps.
func zipFieldsAndItems(fields []string, items [][]any) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, row := range items {
		if len(row) < len(fields) {
			continue
		}
		m := make(map[string]any, len(fields))
		for i, f := range fields {
			m[f] = row[i]
		}
		result = append(result, m)
	}
	return result
}
