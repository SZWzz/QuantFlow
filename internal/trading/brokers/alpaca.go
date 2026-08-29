package brokers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"quantflow/internal/trading"
	"strings"
	"sync"
	"time"
)

// AlpacaConfig holds Alpaca Markets API credentials.
type AlpacaConfig struct {
	APIKey      string `json:"api_key"`
	SecretKey   string `json:"secret_key"`
	BaseURL     string `json:"base_url"`    // default: https://paper-api.alpaca.markets
	Environment string `json:"environment"` // "paper" (default) or "live"
}

// AlpacaBroker implements trading.Broker for Alpaca Markets (US equities).
type AlpacaBroker struct {
	cfg       AlpacaConfig
	client    *http.Client
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

// NewAlpacaBroker creates a new Alpaca broker. Defaults to paper trading.
// Reads ALPACA_API_KEY and ALPACA_SECRET_KEY from env if not provided in config.
// Set Environment to "live" for real-money trading (requires live API key).
func NewAlpacaBroker(cfg AlpacaConfig) *AlpacaBroker {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ALPACA_API_KEY")
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = os.Getenv("ALPACA_SECRET_KEY")
	}
	if cfg.Environment == "" {
		cfg.Environment = "paper"
	}
	if cfg.BaseURL == "" {
		if cfg.Environment == "live" {
			cfg.BaseURL = "https://api.alpaca.markets"
		} else {
			cfg.BaseURL = "https://paper-api.alpaca.markets"
		}
		if v := os.Getenv("ALPACA_BASE_URL"); v != "" {
			cfg.BaseURL = v
		}
	}
	return &AlpacaBroker{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

// IsPaper returns true if this is a paper (simulated) trading account.
func (a *AlpacaBroker) IsPaper() bool {
	return a.cfg.Environment != "live"
}

// Name returns the broker identifier.
func (a *AlpacaBroker) Name() string { return "alpaca" }

// IsConnected returns whether the broker is currently connected.
func (a *AlpacaBroker) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

// Connect verifies connectivity and API key validity via /v2/clock.
func (a *AlpacaBroker) Connect(ctx context.Context) error {
	if a.cfg.APIKey == "" || a.cfg.SecretKey == "" {
		return fmt.Errorf("alpaca: API key not configured (set ALPACA_API_KEY / ALPACA_SECRET_KEY)")
	}
	resp, err := a.doRequest(ctx, http.MethodGet, "/v2/clock", nil)
	if err != nil {
		return fmt.Errorf("alpaca connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("alpaca connect: HTTP %d: %s", resp.StatusCode, string(body))
	}
	a.mu.Lock()
	a.connected = true
	a.mu.Unlock()
	slog.Info("alpaca broker connected", "base_url", a.cfg.BaseURL)
	return nil
}

// Disconnect marks the broker as disconnected.
func (a *AlpacaBroker) Disconnect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connected = false
	return nil
}

// SubmitOrder creates a new order via Alpaca's POST /v2/orders.
func (a *AlpacaBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("alpaca: not connected")
	}
	body := map[string]interface{}{
		"symbol":        order.Symbol,
		"qty":           fmt.Sprintf("%.6f", order.Quantity),
		"side":          string(order.Side),
		"type":          string(order.OrderType),
		"time_in_force": "day",
	}
	if order.OrderType == trading.TypeLimit {
		body["limit_price"] = fmt.Sprintf("%.2f", order.Price)
	}
	if order.OrderType == trading.TypeStop {
		body["stop_price"] = fmt.Sprintf("%.2f", order.StopPrice)
	}
	if order.ClientOrderID != "" {
		body["client_order_id"] = order.ClientOrderID
	}

	resp, err := a.doJSONRequest(ctx, http.MethodPost, "/v2/orders", body)
	if err != nil {
		return nil, fmt.Errorf("alpaca submit order: %w", err)
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("alpaca submit order parse: %w", err)
	}
	return &trading.BrokerOrderResult{
		BrokerOrderID: result.ID,
		Status:        alpacaStatus(result.Status),
		Message:       "",
	}, nil
}

// CancelOrder cancels an order via DELETE /v2/orders/{id}.
func (a *AlpacaBroker) CancelOrder(ctx context.Context, orderID string) error {
	if !a.IsConnected() {
		return fmt.Errorf("alpaca: not connected")
	}
	resp, err := a.doRequest(ctx, http.MethodDelete, "/v2/orders/"+orderID, nil)
	if err != nil {
		return fmt.Errorf("alpaca cancel order: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("alpaca cancel order: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ModifyOrder is not supported by Alpaca REST API. Alpaca requires cancel + replace.
func (a *AlpacaBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	return fmt.Errorf("alpaca: modify not supported — cancel and resubmit instead")
}

// GetOrders returns the list of orders via GET /v2/orders?status=all.
func (a *AlpacaBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("alpaca: not connected")
	}
	resp, err := a.doRequest(ctx, http.MethodGet, "/v2/orders?status=all&limit=100", nil)
	if err != nil {
		return nil, fmt.Errorf("alpaca get orders: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alpaca get orders: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)

	var alpacaOrders []struct {
		ID             string `json:"id"`
		ClientOrderID  string `json:"client_order_id"`
		Symbol         string `json:"symbol"`
		Side           string `json:"side"`
		Type           string `json:"type"`
		Qty            string `json:"qty"`
		LimitPrice     string `json:"limit_price"`
		StopPrice      string `json:"stop_price"`
		FilledQty      string `json:"filled_qty"`
		FilledAvgPrice string `json:"filled_avg_price"`
		Status         string `json:"status"`
		CreatedAt      string `json:"created_at"`
		FilledAt       string `json:"filled_at"`
	}
	if err := json.Unmarshal(body, &alpacaOrders); err != nil {
		return nil, fmt.Errorf("alpaca get orders parse: %w", err)
	}

	orders := make([]*trading.Order, 0, len(alpacaOrders))
	for _, ao := range alpacaOrders {
		orders = append(orders, &trading.Order{
			ID:             ao.ID,
			ClientOrderID:  ao.ClientOrderID,
			Symbol:         ao.Symbol,
			Side:           trading.OrderSide(ao.Side),
			OrderType:      trading.OrderType(ao.Type),
			Quantity:       parseFloat(ao.Qty),
			Price:          parseFloat(ao.LimitPrice),
			StopPrice:      parseFloat(ao.StopPrice),
			FilledQty:      parseFloat(ao.FilledQty),
			FilledAvgPrice: parseFloat(ao.FilledAvgPrice),
			Status:         alpacaStatus(ao.Status),
			PlacedAt:       parseTime(ao.CreatedAt),
			FilledAt:       parseTimePtr(ao.FilledAt),
		})
	}
	return orders, nil
}

// GetPositions returns open positions via GET /v2/positions.
func (a *AlpacaBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("alpaca: not connected")
	}
	resp, err := a.doRequest(ctx, http.MethodGet, "/v2/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("alpaca get positions: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alpaca get positions: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)

	var alpacaPositions []struct {
		Symbol         string `json:"symbol"`
		Qty            string `json:"qty"`
		AvgEntryPrice  string `json:"avg_entry_price"`
		CurrentPrice   string `json:"current_price"`
		UnrealizedPL   string `json:"unrealized_pl"`
		UnrealizedPLPC string `json:"unrealized_plpc"`
	}
	if err := json.Unmarshal(body, &alpacaPositions); err != nil {
		return nil, fmt.Errorf("alpaca get positions parse: %w", err)
	}

	positions := make([]*trading.Position, 0, len(alpacaPositions))
	for _, ap := range alpacaPositions {
		positions = append(positions, &trading.Position{
			Symbol:      ap.Symbol,
			Quantity:    parseFloat(ap.Qty),
			AvgPrice:    parseFloat(ap.AvgEntryPrice),
			MarketPrice: parseFloat(ap.CurrentPrice),
			PnL:         parseFloat(ap.UnrealizedPL),
			PnLPct:      parseFloat(ap.UnrealizedPLPC) * 100, // Alpaca returns decimal, we use %
		})
	}
	return positions, nil
}

// GetAccount returns account info via GET /v2/account.
func (a *AlpacaBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("alpaca: not connected")
	}
	resp, err := a.doRequest(ctx, http.MethodGet, "/v2/account", nil)
	if err != nil {
		return nil, fmt.Errorf("alpaca get account: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alpaca get account: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)

	var acc struct {
		Cash             string `json:"cash"`
		PortfolioValue   string `json:"portfolio_value"`
		BuyingPower      string `json:"buying_power"`
		LongMarketValue  string `json:"long_market_value"`
		Currency         string `json:"currency"`
		PatternDayTrader bool   `json:"pattern_day_trader"`
	}
	if err := json.Unmarshal(body, &acc); err != nil {
		return nil, fmt.Errorf("alpaca get account parse: %w", err)
	}

	return &trading.AccountInfo{
		BrokerName:  "alpaca",
		TotalValue:  parseFloat(acc.PortfolioValue),
		CashBalance: parseFloat(acc.Cash),
		BuyingPower: parseFloat(acc.BuyingPower),
		Currency:    acc.Currency,
	}, nil
}

// OnOrderUpdate registers a callback for order state changes.
func (a *AlpacaBroker) OnOrderUpdate(fn func(*trading.Order)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.orderCbs = append(a.orderCbs, fn)
}

// OnTradeUpdate registers a callback for trade executions.
func (a *AlpacaBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tradeCbs = append(a.tradeCbs, fn)
}

// --- helpers ---

func (a *AlpacaBroker) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.cfg.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("APCA-API-KEY-ID", a.cfg.APIKey)
	req.Header.Set("APCA-API-SECRET-KEY", a.cfg.SecretKey)
	req.Header.Set("Accept", "application/json")
	return a.client.Do(req)
}

func (a *AlpacaBroker) doJSONRequest(ctx context.Context, method, path string, v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, a.cfg.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("APCA-API-KEY-ID", a.cfg.APIKey)
	req.Header.Set("APCA-API-SECRET-KEY", a.cfg.SecretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("alpaca: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// alpacaStatus maps Alpaca order status strings to trading.OrderStatus.
func alpacaStatus(s string) trading.OrderStatus {
	switch strings.ToLower(s) {
	case "new", "accepted", "pending_new", "accepted_for_bidding":
		return trading.StatusPending
	case "partially_filled":
		return trading.StatusPartial
	case "filled":
		return trading.StatusFilled
	case "canceled", "expired", "suspended":
		return trading.StatusCancelled
	case "rejected", "stopped":
		return trading.StatusRejected
	default:
		return trading.StatusPending
	}
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	var v float64
	// Best-effort parse: Alpaca occasionally sends non-numeric placeholders; degrade to 0
	_, _ = fmt.Sscanf(s, "%f", &v)
	return v
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := parseTime(s)
	if t.IsZero() {
		return nil
	}
	return &t
}
