package brokers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"quantflow/internal/trading"
)

// IBKRBroker implements trading.Broker for Interactive Brokers via Client Portal REST API.
type IBKRBroker struct {
	cfg       IBKRConfig
	client    *http.Client
	session   ibkrSession
	baseURL   string
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

// NewIBKRBroker creates a new IBKR broker adapter.
// NOTE: IB Gateway must be running on the configured host:port before Connect().
func NewIBKRBroker(cfg IBKRConfig) *IBKRBroker {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 5000
	}
	baseURL := fmt.Sprintf("https://%s:%d/v1/api", cfg.Host, cfg.Port)
	return &IBKRBroker{
		cfg:     cfg,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// Name returns the broker identifier.
func (b *IBKRBroker) Name() string { return "ibkr" }

// IsConnected returns whether the broker is currently connected.
func (b *IBKRBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// Connect establishes a session with IB Gateway.
func (b *IBKRBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cfg.AccountID == "" {
		return fmt.Errorf("ibkr: AccountID not configured")
	}

	if err := b.session.validate(ctx, b.client, b.baseURL); err != nil {
		return fmt.Errorf("ibkr connect: %w\n\nPlease ensure IB Gateway / Client Portal is running on %s:%d and you are logged in.",
			err, b.cfg.Host, b.cfg.Port)
	}

	b.session.startRefresh(ctx, b.client, b.baseURL)
	b.connected = true
	slog.Info("ibkr broker connected", "host", b.cfg.Host, "port", b.cfg.Port)
	return nil
}

// Disconnect logs out and marks the broker as disconnected.
func (b *IBKRBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.session.stopRefresh()
	b.session.clear()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/logout", nil)
	if err == nil {
		if resp, err := b.client.Do(req); err == nil {
			resp.Body.Close()
		}
	}

	b.connected = false
	slog.Info("ibkr broker disconnected")
	return nil
}

// SubmitOrder sends an order to IBKR via POST /iserver/account/{id}/orders.
func (b *IBKRBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) {
	if !b.IsConnected() {
		return nil, fmt.Errorf("ibkr: not connected")
	}

	ibkrOrder := ibkrAPIOrder{
		AcctID:      b.cfg.AccountID,
		Symbol:      order.Symbol,
		OrderType:   ibkrOrderType(order.OrderType),
		Side:        string(order.Side),
		Quantity:    order.Quantity,
		TimeInForce: "DAY",
	}
	if order.OrderType == trading.TypeLimit {
		ibkrOrder.Price = order.Price
	}
	if order.OrderType == trading.TypeStop {
		ibkrOrder.StopPrice = order.StopPrice
	}

	body, err := b.doJSONRequest(ctx, http.MethodPost, fmt.Sprintf("/iserver/account/%s/orders", b.cfg.AccountID), ibkrOrder)
	if err != nil {
		return nil, fmt.Errorf("ibkr submit order: %w", err)
	}

	var reply ibkrAPIOrderReply
	if err := json.Unmarshal(body, &reply); err != nil {
		return nil, fmt.Errorf("ibkr submit order parse: %w", err)
	}

	return &trading.BrokerOrderResult{
		BrokerOrderID: reply.ID,
		Status:        ibkrOrderStatus(reply.Status),
		Message:       reply.Message,
	}, nil
}

// CancelOrder cancels an order via DELETE /iserver/account/{id}/order/{id}.
func (b *IBKRBroker) CancelOrder(ctx context.Context, orderID string) error {
	if !b.IsConnected() {
		return fmt.Errorf("ibkr: not connected")
	}
	resp, err := b.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/iserver/account/%s/order/%s", b.cfg.AccountID, orderID), nil)
	if err != nil {
		return fmt.Errorf("ibkr cancel order: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ibkr cancel order: HTTP %d", resp.StatusCode)
	}
	return nil
}

// ModifyOrder modifies an existing order via POST /iserver/account/{id}/order/{id}.
func (b *IBKRBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	if !b.IsConnected() {
		return fmt.Errorf("ibkr: not connected")
	}
	body := map[string]interface{}{
		"orderId":       orderID,
		"totalQuantity": fmt.Sprintf("%.6f", newQty),
		"limitPrice":    fmt.Sprintf("%.2f", newPrice),
	}
	_, err := b.doJSONRequest(ctx, http.MethodPost, fmt.Sprintf("/iserver/account/%s/order/%s", b.cfg.AccountID, orderID), body)
	if err != nil {
		return fmt.Errorf("ibkr modify order: %w", err)
	}
	return nil
}

// GetOrders retrieves orders from IBKR via GET /iserver/account/{id}/orders.
func (b *IBKRBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) {
	if !b.IsConnected() {
		return nil, fmt.Errorf("ibkr: not connected")
	}
	resp, err := b.doRequest(ctx, http.MethodGet, fmt.Sprintf("/iserver/account/%s/orders", b.cfg.AccountID), nil)
	if err != nil {
		return nil, fmt.Errorf("ibkr get orders: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var rawOrders []struct {
		OrderID       int     `json:"orderId"`
		Symbol        string  `json:"symbol"`
		Side          string  `json:"side"`
		OrderType     string  `json:"orderType"`
		Quantity      float64 `json:"quantity"`
		LimitPrice    float64 `json:"limitPrice"`
		StopPrice     float64 `json:"auxPrice"`
		FilledQty     float64 `json:"filledQuantity"`
		AvgPrice      float64 `json:"avgPrice"`
		Status        string  `json:"status"`
		PlacedAt      string  `json:"placedTime"`
		ClientOrderID string  `json:"clientOrderId"`
	}
	if err := json.Unmarshal(body, &rawOrders); err != nil {
		return nil, fmt.Errorf("ibkr get orders parse: %w", err)
	}

	orders := make([]*trading.Order, 0, len(rawOrders))
	for _, ro := range rawOrders {
		o := &trading.Order{
			ID:             fmt.Sprintf("%d", ro.OrderID),
			ClientOrderID:  ro.ClientOrderID,
			Symbol:         ro.Symbol,
			Side:           trading.OrderSide(ro.Side),
			OrderType:      ibkrTypeToOrderType(ro.OrderType),
			Quantity:       ro.Quantity,
			Price:          ro.LimitPrice,
			StopPrice:      ro.StopPrice,
			FilledQty:      ro.FilledQty,
			FilledAvgPrice: ro.AvgPrice,
			Status:         ibkrOrderStatus(ro.Status),
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// GetPositions returns open portfolio positions via GET /portfolio/{id}/positions/0.
func (b *IBKRBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) {
	if !b.IsConnected() {
		return nil, fmt.Errorf("ibkr: not connected")
	}
	resp, err := b.doRequest(ctx, http.MethodGet, fmt.Sprintf("/portfolio/%s/positions/0", b.cfg.AccountID), nil)
	if err != nil {
		return nil, fmt.Errorf("ibkr get positions: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var rawPositions []ibkrPosition
	if err := json.Unmarshal(body, &rawPositions); err != nil {
		return nil, fmt.Errorf("ibkr get positions parse: %w", err)
	}

	positions := make([]*trading.Position, 0, len(rawPositions))
	for _, rp := range rawPositions {
		positions = append(positions, &trading.Position{
			Symbol:      rp.Symbol,
			Quantity:    rp.Position,
			AvgPrice:    rp.AvgCost,
			MarketPrice: rp.MarketPrice,
			PnL:         rp.UnrealizedPNL,
			PnLPct:      rp.UnrealizedPNLPercent,
		})
	}
	return positions, nil
}

// GetAccount returns account summary via GET /portfolio/{id}/summary.
func (b *IBKRBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) {
	if !b.IsConnected() {
		return nil, fmt.Errorf("ibkr: not connected")
	}
	resp, err := b.doRequest(ctx, http.MethodGet, fmt.Sprintf("/portfolio/%s/summary", b.cfg.AccountID), nil)
	if err != nil {
		return nil, fmt.Errorf("ibkr get account: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var raw map[string]ibkrAccountSummary
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ibkr get account parse: %w", err)
	}

	return &trading.AccountInfo{
		BrokerName:  "ibkr",
		TotalValue:  raw["TotalCashValue"].Value,
		CashBalance: raw["CashBalance"].Value,
		BuyingPower: raw["BuyingPower"].Value,
		Currency:    raw["Currency"].ValueString,
	}, nil
}

// OnOrderUpdate registers a callback for order state changes.
func (b *IBKRBroker) OnOrderUpdate(fn func(*trading.Order)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.orderCbs = append(b.orderCbs, fn)
}

// OnTradeUpdate registers a callback for trade executions.
func (b *IBKRBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tradeCbs = append(b.tradeCbs, fn)
}

// — IBKR API types —

type ibkrAPIOrder struct {
	AcctID      string  `json:"acctId"`
	Symbol      string  `json:"symbol"`
	OrderType   string  `json:"orderType"`
	Side        string  `json:"side"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price,omitempty"`
	StopPrice   float64 `json:"auxPrice,omitempty"`
	TimeInForce string  `json:"tif,omitempty"`
}

type ibkrAPIOrderReply struct {
	ID      string `json:"id"`
	Status  string `json:"order_status"`
	Message string `json:"message,omitempty"`
}

type ibkrPosition struct {
	Symbol               string  `json:"symbol"`
	Position             float64 `json:"position"`
	AvgCost              float64 `json:"avgCost"`
	MarketPrice          float64 `json:"marketPrice"`
	UnrealizedPNL        float64 `json:"unrealizedPnl"`
	UnrealizedPNLPercent float64 `json:"unrealizedPnlPerc"`
}

type ibkrAccountSummary struct {
	Value       float64 `json:"value"`
	ValueString string  `json:"valueString"`
	Currency    string  `json:"currency"`
}

// — HTTP helpers —

func (b *IBKRBroker) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return b.client.Do(req)
}

func (b *IBKRBroker) doJSONRequest(ctx context.Context, method, path string, v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, b.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ibkr: HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// — Order type / status mappers —

func ibkrOrderType(ot trading.OrderType) string {
	switch ot {
	case trading.TypeMarket:
		return "MKT"
	case trading.TypeLimit:
		return "LMT"
	case trading.TypeStop:
		return "STP"
	default:
		return "MKT"
	}
}

func ibkrTypeToOrderType(t string) trading.OrderType {
	switch t {
	case "MKT":
		return trading.TypeMarket
	case "LMT":
		return trading.TypeLimit
	case "STP":
		return trading.TypeStop
	default:
		return trading.TypeMarket
	}
}

func ibkrOrderStatus(s string) trading.OrderStatus {
	switch s {
	case "Submitted", "PreSubmitted":
		return trading.StatusPending
	case "Filled":
		return trading.StatusFilled
	case "Cancelled", "ApiCancelled":
		return trading.StatusCancelled
	case "Inactive":
		return trading.StatusPending
	default:
		return trading.StatusPending
	}
}
