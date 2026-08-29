// Package brokers provides real broker adapter implementations.
package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"quantflow/internal/trading"
	"strings"
	"sync"
	"time"
)

// FutuConfig holds connection parameters for FutuOpenD.
type FutuConfig struct {
	Host     string `json:"host"`     // default: localhost
	Port     int    `json:"port"`     // default: 11111
	Password string `json:"password"` // trade unlock password (optional, depends on FutuOpenD config)
}

// FutuBroker implements trading.Broker for Futu (FutuOpenD).
// Communicates with the locally running FutuOpenD gateway via its HTTP API.
//
// Prerequisites:
//  1. Download and run FutuOpenD (https://www.futunn.com/download/openAPI)
//  2. Login to your Futu account in FutuOpenD
//  3. Enable API access in FutuOpenD settings
type FutuBroker struct {
	cfg       FutuConfig
	client    *http.Client
	baseURL   string
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

// NewFutuBroker creates a new Futu broker with sensible defaults.
func NewFutuBroker(cfg FutuConfig) *FutuBroker {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 11111
	}
	return &FutuBroker{
		cfg:     cfg,
		client:  &http.Client{Timeout: 15 * time.Second},
		baseURL: fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port),
	}
}

// Name returns the broker identifier.
func (f *FutuBroker) Name() string { return "futu" }

// IsConnected returns whether the broker is currently connected.
func (f *FutuBroker) IsConnected() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.connected
}

// Connect verifies connectivity to FutuOpenD by calling the heartbeat endpoint.
func (f *FutuBroker) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", f.baseURL+"/api/v1/heartbeat", nil)
	if err != nil {
		return fmt.Errorf("futu connect: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("futu connect: cannot reach FutuOpenD at %s — ensure FutuOpenD is running: %w", f.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("futu connect: FutuOpenD returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	f.mu.Lock()
	f.connected = true
	f.mu.Unlock()

	slog.Info("futu broker connected", "base_url", f.baseURL)
	return nil
}

// Disconnect marks the broker as disconnected.
func (f *FutuBroker) Disconnect(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connected = false
	return nil
}

// SubmitOrder places an order via FutuOpenD.
// Supports HK and US markets via Futu.
func (f *FutuBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) {
	if !f.IsConnected() {
		return nil, fmt.Errorf("futu: not connected")
	}

	body := map[string]interface{}{
		"code":      order.Symbol,
		"price":     order.Price,
		"qty":       order.Quantity,
		"trdSide":   mapOrderSide(order.Side),
		"orderType": mapOrderType(order.OrderType),
	}
	data, _ := json.Marshal(body)

	resp, err := f.doJSONRequest(ctx, "POST", "/api/v1/trade/order", data)
	if err != nil {
		return nil, fmt.Errorf("futu submit order: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderID string `json:"orderID"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("futu submit order parse: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("futu submit order: %s (code=%d)", result.Message, result.Code)
	}

	return &trading.BrokerOrderResult{
		BrokerOrderID: result.Data.OrderID,
		Status:        trading.StatusPending,
	}, nil
}

// CancelOrder cancels an order via FutuOpenD.
func (f *FutuBroker) CancelOrder(ctx context.Context, orderID string) error {
	if !f.IsConnected() {
		return fmt.Errorf("futu: not connected")
	}

	resp, err := f.doJSONRequest(ctx, "POST", "/api/v1/trade/order/cancel", []byte(`{"orderID":"`+orderID+`"}`))
	if err != nil {
		return fmt.Errorf("futu cancel order: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("futu cancel order: %s", result.Message)
	}
	return nil
}

// ModifyOrder is not natively supported — must cancel and resubmit.
func (f *FutuBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	return fmt.Errorf("futu: modify not supported — cancel and resubmit instead")
}

// GetOrders returns the list of today's orders.
func (f *FutuBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) {
	if !f.IsConnected() {
		return nil, fmt.Errorf("futu: not connected")
	}

	resp, err := f.doJSONRequest(ctx, "GET", "/api/v1/trade/order/list?status=all", nil)
	if err != nil {
		return nil, fmt.Errorf("futu get orders: %w", err)
	}

	var result struct {
		Data []struct {
			OrderID   string  `json:"orderID"`
			Code      string  `json:"code"`
			TrdSide   int     `json:"trdSide"`
			OrderType int     `json:"orderType"`
			Price     float64 `json:"price"`
			Qty       float64 `json:"qty"`
			FilledQty float64 `json:"filledQty"`
			Status    int     `json:"orderStatus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("futu get orders parse: %w", err)
	}

	orders := make([]*trading.Order, 0, len(result.Data))
	for _, o := range result.Data {
		orders = append(orders, &trading.Order{
			ID:        o.OrderID,
			Symbol:    o.Code,
			Side:      mapFutuSide(o.TrdSide),
			OrderType: trading.OrderType(fmt.Sprintf("%d", o.OrderType)),
			Quantity:  o.Qty,
			Price:     o.Price,
			FilledQty: o.FilledQty,
			Status:    mapFutuStatus(o.Status),
		})
	}
	return orders, nil
}

// GetPositions returns current positions.
func (f *FutuBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) {
	if !f.IsConnected() {
		return nil, fmt.Errorf("futu: not connected")
	}

	resp, err := f.doJSONRequest(ctx, "GET", "/api/v1/trade/position/list", nil)
	if err != nil {
		return nil, fmt.Errorf("futu get positions: %w", err)
	}

	var result struct {
		Data []struct {
			Code         string  `json:"code"`
			Name         string  `json:"stockName"`
			Qty          float64 `json:"qty"`
			CostPrice    float64 `json:"costPrice"`
			MarketVal    float64 `json:"marketVal"`
			NominalPrice float64 `json:"nominalPrice"`
			PlVal        float64 `json:"plVal"`
			PlRatio      float64 `json:"plRatio"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("futu get positions parse: %w", err)
	}

	positions := make([]*trading.Position, 0, len(result.Data))
	for _, p := range result.Data {
		positions = append(positions, &trading.Position{
			Symbol:      p.Code,
			Name:        p.Name,
			Quantity:    p.Qty,
			AvgPrice:    p.CostPrice,
			MarketPrice: p.NominalPrice,
			PnL:         p.PlVal,
		})
	}
	return positions, nil
}

// GetAccount returns account summary.
func (f *FutuBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) {
	if !f.IsConnected() {
		return nil, fmt.Errorf("futu: not connected")
	}

	resp, err := f.doJSONRequest(ctx, "GET", "/api/v1/trade/account/info", nil)
	if err != nil {
		return nil, fmt.Errorf("futu get account: %w", err)
	}

	var result struct {
		Data struct {
			TotalAsset  float64 `json:"totalAssets"`
			Cash        float64 `json:"cash"`
			MarketValue float64 `json:"marketValue"`
			BuyingPower float64 `json:"buyingPower"`
			Currency    string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("futu get account parse: %w", err)
	}

	currency := result.Data.Currency
	if currency == "" {
		currency = "HKD"
	}

	return &trading.AccountInfo{
		BrokerName:  "futu",
		TotalValue:  result.Data.TotalAsset,
		CashBalance: result.Data.Cash,
		BuyingPower: result.Data.BuyingPower,
		Currency:    currency,
	}, nil
}

// OnOrderUpdate registers a callback for order state changes.
func (f *FutuBroker) OnOrderUpdate(fn func(*trading.Order)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orderCbs = append(f.orderCbs, fn)
}

// OnTradeUpdate registers a callback for trade executions.
func (f *FutuBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tradeCbs = append(f.tradeCbs, fn)
}

// doJSONRequest sends an HTTP request to FutuOpenD and returns the response body.
func (f *FutuBroker) doJSONRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, f.baseURL+path, strings.NewReader(string(body)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, f.baseURL+path, nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, futuUserFacingError(resp.StatusCode, string(respBody)))
	}

	return respBody, nil
}

// Helper functions

func mapOrderSide(side trading.OrderSide) int {
	switch side {
	case trading.SideBuy:
		return 1
	case trading.SideSell:
		return 2
	default:
		return 0
	}
}

func mapOrderType(ot trading.OrderType) int {
	switch ot {
	case trading.TypeMarket:
		return 1
	case trading.TypeLimit:
		return 2
	case trading.TypeStop:
		return 3
	default:
		return 2
	}
}

func mapFutuSide(side int) trading.OrderSide {
	switch side {
	case 1:
		return trading.SideBuy
	case 2:
		return trading.SideSell
	default:
		return ""
	}
}

func mapFutuStatus(status int) trading.OrderStatus {
	switch status {
	case 0, 4:
		return trading.StatusPending
	case 1, 5:
		return trading.StatusPartial
	case 2, 6:
		return trading.StatusFilled
	case 3, 7:
		return trading.StatusCancelled
	default:
		return trading.StatusPending
	}
}

func futuUserFacingError(statusCode int, body string) string {
	switch {
	case statusCode == 401:
		return "FutuOpenD 认证失败，请检查登录状态"
	case statusCode == 403:
		return "交易权限不足，请在 FutuOpenD 中启用 API 交易"
	case strings.Contains(body, "not login"):
		return "FutuOpenD 未登录，请先登录牛牛账户"
	case strings.Contains(body, "trading password"):
		return "交易密码错误或未设置，请在 FutuOpenD 中解锁交易"
	case strings.Contains(body, "insufficient"):
		return "资金不足"
	case strings.Contains(body, "market closed"):
		return "市场已关闭"
	default:
		return fmt.Sprintf("FutuOpenD 错误 (HTTP %d)", statusCode)
	}
}
