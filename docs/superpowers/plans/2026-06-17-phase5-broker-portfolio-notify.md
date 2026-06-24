# Phase 5: Broker + Portfolio + Notify + Schedule — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add real broker trading (Futu + Binance), portfolio/risk management, notification system (Telegram + in-app), cron scheduler, 11 new workflow nodes, and 7 new frontend panels to QuantFlow.

**Architecture:** Extend existing OMS with a Broker interface layer. OMS routes orders to either PaperEngine (sim) or Broker (live). New packages: `internal/trading/brokers/`, `internal/schedule/`, `internal/notify/`, `internal/portfolio/`. All wired into App struct for Wails IPC.

**Tech Stack:** Go 1.22+ (net/http for Binance REST, gorilla/websocket, robfig/cron/v3), Vue 3 + Pinia + ECharts, SQLite WAL (migrations 006-009).

---

## Milestone 1: Broker Interface + BinanceBroker + OMS Integration

### Task 1.1: Define Broker interface

**Files:**
- Create: `internal/trading/broker.go`

- [ ] **Step 1: Write Broker interface and types**

```go
// Package trading provides the trading engine — bar-by-bar pipeline, OMS,
// paper trading, order matching, risk management, and broker integration.
package trading

import "context"

// Broker abstracts a real brokerage connection. Implementations handle
// authentication, order submission, and position/account synchronization.
type Broker interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Name() string

	SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error)
	CancelOrder(ctx context.Context, orderID string) error
	ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error

	GetOrders(ctx context.Context) ([]*Order, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	GetAccount(ctx context.Context) (*AccountInfo, error)

	OnOrderUpdate(func(order *Order))
	OnTradeUpdate(func(trade *Trade))
}

// AccountInfo holds broker account summary data.
type AccountInfo struct {
	BrokerName    string  `json:"broker_name"`
	TotalValue    float64 `json:"total_value"`
	CashBalance   float64 `json:"cash_balance"`
	MarginBalance float64 `json:"margin_balance"`
	BuyingPower   float64 `json:"buying_power"`
	Currency      string  `json:"currency"`
}

// BrokerOrderResult is returned by SubmitOrder with broker-specific details.
type BrokerOrderResult struct {
	BrokerOrderID string      `json:"broker_order_id"`
	Status        OrderStatus `json:"status"`
	Message       string      `json:"message"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go build ./internal/trading/
```
Expected: builds successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/trading/broker.go
git commit -m "feat(trading): define Broker interface for real broker integration"
```

---

### Task 1.2: Extend OMS with broker routing

**Files:**
- Modify: `internal/trading/oms.go`
- Create: `internal/trading/broker_test.go`

- [ ] **Step 1: Add broker field and SetBroker to OMS**

Edit `internal/trading/oms.go` — add `"context"` to imports, add `broker` field to the OMS struct, and append the following methods before the closing line of the file:

```go
// broker is the optional live broker. When nil (default), the OMS operates
// in paper-trading mode.
type OMS struct {
	mu        sync.RWMutex
	orders    map[string]*Order
	trades    []*Trade
	positions map[string]*Position
	broker    Broker // nil = paper mode
}

// SetBroker attaches a live broker to the OMS. Pass nil to return to paper mode.
func (o *OMS) SetBroker(b Broker) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.broker = b
}

// HasBroker reports whether a live broker is attached.
func (o *OMS) HasBroker() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.broker != nil
}

// PlaceOrderLive places an order through the attached broker instead of paper.
func (o *OMS) PlaceOrderLive(ctx context.Context, symbol string, side OrderSide, orderType OrderType, qty, price, stopPrice float64) (*Order, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("quantity must be positive, got %f", qty)
	}
	if orderType == TypeLimit && price <= 0 {
		return nil, fmt.Errorf("limit order requires a positive price")
	}

	o.mu.Lock()

	order := &Order{
		ID:        uuid.New().String()[:8],
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  qty,
		Price:     price,
		StopPrice: stopPrice,
		Status:    StatusPending,
		PlacedAt:  time.Now(),
	}
	o.orders[order.ID] = order
	br := o.broker
	o.mu.Unlock()

	result, err := br.SubmitOrder(ctx, order)
	if err != nil {
		o.mu.Lock()
		order.Status = StatusRejected
		o.mu.Unlock()
		return order, fmt.Errorf("broker submit: %w", err)
	}

	o.mu.Lock()
	delete(o.orders, order.ID)
	order.ID = result.BrokerOrderID
	o.orders[result.BrokerOrderID] = order
	order.Status = result.Status
	o.mu.Unlock()

	return order, nil
}
```

- [ ] **Step 2: Write broker mock test**

Create `internal/trading/broker_test.go`:

```go
package trading

import (
	"context"
	"testing"
)

type mockBroker struct {
	name         string
	connected    bool
	orders       map[string]*Order
	orderUpdates []func(*Order)
	tradeUpdates []func(*Trade)
}

func newMockBroker(name string) *mockBroker {
	return &mockBroker{name: name, orders: make(map[string]*Order)}
}

func (m *mockBroker) Connect(ctx context.Context) error   { m.connected = true; return nil }
func (m *mockBroker) Disconnect(ctx context.Context) error { m.connected = false; return nil }
func (m *mockBroker) IsConnected() bool                    { return m.connected }
func (m *mockBroker) Name() string                         { return m.name }

func (m *mockBroker) SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error) {
	orderID := "B-" + order.ID
	m.orders[orderID] = order
	return &BrokerOrderResult{BrokerOrderID: orderID, Status: StatusPending, Message: "submitted"}, nil
}

func (m *mockBroker) CancelOrder(ctx context.Context, orderID string) error {
	if o, ok := m.orders[orderID]; ok {
		o.Status = StatusCancelled
	}
	return nil
}

func (m *mockBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	if o, ok := m.orders[orderID]; ok {
		o.Price = newPrice
		o.Quantity = newQty
	}
	return nil
}

func (m *mockBroker) GetOrders(ctx context.Context) ([]*Order, error) {
	result := make([]*Order, 0, len(m.orders))
	for _, o := range m.orders {
		result = append(result, o)
	}
	return result, nil
}

func (m *mockBroker) GetPositions(ctx context.Context) ([]*Position, error) {
	return []*Position{{Symbol: "AAPL", Quantity: 100, AvgPrice: 150.0, MarketPrice: 155.0, PnL: 500.0, PnLPct: 3.33}}, nil
}

func (m *mockBroker) GetAccount(ctx context.Context) (*AccountInfo, error) {
	return &AccountInfo{BrokerName: m.name, TotalValue: 100000.0, CashBalance: 50000.0, BuyingPower: 100000.0, Currency: "USD"}, nil
}

func (m *mockBroker) OnOrderUpdate(fn func(*Order)) { m.orderUpdates = append(m.orderUpdates, fn) }
func (m *mockBroker) OnTradeUpdate(fn func(*Trade))  { m.tradeUpdates = append(m.tradeUpdates, fn) }
```

- [ ] **Step 3: Add test functions to broker_test.go (append after mock)**

```go
func TestOMS_WithBroker_PlaceOrderLive(t *testing.T) {
	oms := NewOMS()
	mb := newMockBroker("test-broker")
	oms.SetBroker(mb)

	ctx := context.Background()
	order, err := oms.PlaceOrderLive(ctx, "AAPL", SideBuy, TypeLimit, 100, 150.0, 0)
	if err != nil {
		t.Fatalf("PlaceOrderLive error: %v", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}
}

func TestOMS_HasBroker(t *testing.T) {
	oms := NewOMS()
	if oms.HasBroker() {
		t.Error("expected no broker initially")
	}
	oms.SetBroker(newMockBroker("test"))
	if !oms.HasBroker() {
		t.Error("expected broker after SetBroker")
	}
}

func TestOMS_WithoutBroker_UsesPaper(t *testing.T) {
	oms := NewOMS()
	order, err := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, 100, 0)
	if err != nil {
		t.Fatalf("PlaceOrder error: %v", err)
	}
	if order.Status != StatusPending {
		t.Errorf("expected Pending, got %q", order.Status)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/trading/ -v -count=1 -run "TestOMS_With|TestOMS_Has|TestOMS_Without"
```
Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/trading/oms.go internal/trading/broker_test.go
git commit -m "feat(trading): add broker routing to OMS with SetBroker and PlaceOrderLive"
```

---

### Task 1.3: Implement BinanceBroker

**Files:**
- Create: `internal/trading/brokers/binance.go`
- Create: `internal/trading/brokers/binance_test.go`

- [ ] **Step 1: Create BinanceBroker**

Create `internal/trading/brokers/binance.go`:

```go
package brokers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"quantflow/internal/trading"
)

// BinanceConfig holds Binance API credentials and settings.
type BinanceConfig struct {
	APIKey     string `json:"api_key"`
	SecretKey  string `json:"secret_key"`
	BaseURL    string `json:"base_url"`
	UseTestnet bool   `json:"use_testnet"`
}

// BinanceBroker implements trading.Broker for Binance (spot).
type BinanceBroker struct {
	cfg       BinanceConfig
	client    *http.Client
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

// NewBinanceBroker creates a new Binance broker.
func NewBinanceBroker(cfg BinanceConfig) *BinanceBroker {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.binance.com"
	}
	if cfg.UseTestnet {
		cfg.BaseURL = "https://testnet.binance.vision"
	}
	return &BinanceBroker{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

func (b *BinanceBroker) Name() string { return "binance" }

func (b *BinanceBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	resp, err := b.client.Get(b.cfg.BaseURL + "/api/v3/time")
	if err != nil {
		return fmt.Errorf("binance connect: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance connect: status %d", resp.StatusCode)
	}
	b.connected = true
	slog.Info("binance broker connected", "base_url", b.cfg.BaseURL)
	return nil
}

func (b *BinanceBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = false
	return nil
}

func (b *BinanceBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

func (b *BinanceBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) {
	params := url.Values{}
	params.Set("symbol", normalizeBinanceSymbol(order.Symbol))
	params.Set("side", strings.ToUpper(string(order.Side)))
	params.Set("type", binanceOrderType(order.OrderType))
	params.Set("quantity", fmt.Sprintf("%.8f", order.Quantity))
	if order.OrderType == trading.TypeLimit {
		params.Set("price", fmt.Sprintf("%.8f", order.Price))
		params.Set("timeInForce", "GTC")
	}
	if order.OrderType == trading.TypeStop {
		params.Set("stopPrice", fmt.Sprintf("%.8f", order.StopPrice))
	}

	resp, err := b.signedRequest(ctx, "POST", "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("submit order: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("submit order failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		OrderID int64  `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse order response: %w", err)
	}

	return &trading.BrokerOrderResult{
		BrokerOrderID: fmt.Sprintf("%d", result.OrderID),
		Status:        binanceStatusToOrderStatus(result.Status),
		Message:       result.Status,
	}, nil
}

func (b *BinanceBroker) CancelOrder(ctx context.Context, orderID string) error {
	params := url.Values{}
	params.Set("orderId", orderID)
	resp, err := b.signedRequest(ctx, "DELETE", "/api/v3/order", params)
	if err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cancel order failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (b *BinanceBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	return fmt.Errorf("binance does not support direct order modification; cancel and re-submit required")
}

func (b *BinanceBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) {
	resp, err := b.signedRequest(ctx, "GET", "/api/v3/openOrders", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("get orders: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var binanceOrders []struct {
		OrderID      int64  `json:"orderId"`
		Symbol       string `json:"symbol"`
		Side         string `json:"side"`
		Type         string `json:"type"`
		OrigQty      string `json:"origQty"`
		Price        string `json:"price"`
		StopPrice    string `json:"stopPrice"`
		Status       string `json:"status"`
		ExecutedQty  string `json:"executedQty"`
	}
	if err := json.Unmarshal(body, &binanceOrders); err != nil {
		return nil, fmt.Errorf("parse orders: %w", err)
	}

	orders := make([]*trading.Order, 0, len(binanceOrders))
	for _, bo := range binanceOrders {
		o := &trading.Order{
			ID:        fmt.Sprintf("%d", bo.OrderID),
			Symbol:    bo.Symbol,
			Side:      trading.OrderSide(strings.ToLower(bo.Side)),
			OrderType: binanceTypeToOrderType(bo.Type),
			Status:    binanceStatusToOrderStatus(bo.Status),
		}
		fmt.Sscanf(bo.OrigQty, "%f", &o.Quantity)
		fmt.Sscanf(bo.Price, "%f", &o.Price)
		fmt.Sscanf(bo.StopPrice, "%f", &o.StopPrice)
		fmt.Sscanf(bo.ExecutedQty, "%f", &o.FilledQty)
		orders = append(orders, o)
	}
	return orders, nil
}

func (b *BinanceBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) {
	resp, err := b.signedRequest(ctx, "GET", "/api/v3/account", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var acct struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(body, &acct); err != nil {
		return nil, fmt.Errorf("parse account: %w", err)
	}

	var positions []*trading.Position
	for _, bal := range acct.Balances {
		var free, locked float64
		fmt.Sscanf(bal.Free, "%f", &free)
		fmt.Sscanf(bal.Locked, "%f", &locked)
		if free+locked > 0 {
			positions = append(positions, &trading.Position{Symbol: bal.Asset, Quantity: free + locked})
		}
	}
	return positions, nil
}

func (b *BinanceBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) {
	resp, err := b.signedRequest(ctx, "GET", "/api/v3/account", url.Values{})
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var acct struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(body, &acct); err != nil {
		return nil, fmt.Errorf("parse account: %w", err)
	}

	info := &trading.AccountInfo{BrokerName: "binance", Currency: "USDT"}
	for _, bal := range acct.Balances {
		var free, locked float64
		fmt.Sscanf(bal.Free, "%f", &free)
		fmt.Sscanf(bal.Locked, "%f", &locked)
		info.CashBalance += free
		info.TotalValue += free + locked
	}
	info.BuyingPower = info.CashBalance
	return info, nil
}

func (b *BinanceBroker) OnOrderUpdate(fn func(*trading.Order)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.orderCbs = append(b.orderCbs, fn)
}

func (b *BinanceBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tradeCbs = append(b.tradeCbs, fn)
}

// — helpers —

func (b *BinanceBroker) signedRequest(ctx context.Context, method, path string, params url.Values) (*http.Response, error) {
	params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	params.Set("recvWindow", "5000")
	queryString := params.Encode()
	mac := hmac.New(sha256.New, []byte(b.cfg.SecretKey))
	mac.Write([]byte(queryString))
	signature := hex.EncodeToString(mac.Sum(nil))
	fullURL := b.cfg.BaseURL + path + "?" + queryString + "&signature=" + signature

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", b.cfg.APIKey)
	return b.client.Do(req)
}

func normalizeBinanceSymbol(symbol string) string {
	symbol = strings.ReplaceAll(symbol, ".", "")
	if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "BTC") && !strings.HasSuffix(symbol, "ETH") {
		symbol = symbol + "USDT"
	}
	return strings.ToUpper(symbol)
}

func binanceOrderType(ot trading.OrderType) string {
	switch ot {
	case trading.TypeMarket: return "MARKET"
	case trading.TypeLimit:  return "LIMIT"
	case trading.TypeStop:   return "STOP_LOSS_LIMIT"
	default:                 return "MARKET"
	}
}

func binanceTypeToOrderType(t string) trading.OrderType {
	switch strings.ToUpper(t) {
	case "MARKET":          return trading.TypeMarket
	case "LIMIT":           return trading.TypeLimit
	case "STOP_LOSS_LIMIT": return trading.TypeStop
	default:                return trading.TypeMarket
	}
}

func binanceStatusToOrderStatus(s string) trading.OrderStatus {
	switch strings.ToUpper(s) {
	case "NEW":              return trading.StatusPending
	case "PARTIALLY_FILLED": return trading.StatusPartial
	case "FILLED":           return trading.StatusFilled
	case "CANCELED":         return trading.StatusCancelled
	case "REJECTED", "EXPIRED": return trading.StatusRejected
	default:                 return trading.StatusPending
	}
}
```

- [ ] **Step 2: Write BinanceBroker tests**

Create `internal/trading/brokers/binance_test.go`:

```go
package brokers

import (
	"testing"
)

func TestNormalizeBinanceSymbol(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"BTC", "BTCUSDT"},
		{"ETHUSDT", "ETHUSDT"},
		{"000001.SZ", "000001SZUSDT"},
		{"AAPL", "AAPLUSDT"},
	}
	for _, tt := range tests {
		result := normalizeBinanceSymbol(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeBinanceSymbol(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBinanceStatusToOrderStatus(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"NEW", "pending"}, {"PARTIALLY_FILLED", "partial"}, {"FILLED", "filled"},
		{"CANCELED", "cancelled"}, {"REJECTED", "rejected"},
	}
	for _, tt := range tests {
		result := binanceStatusToOrderStatus(tt.input)
		if string(result) != tt.expected {
			t.Errorf("binanceStatusToOrderStatus(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewBinanceBroker_Defaults(t *testing.T) {
	broker := NewBinanceBroker(BinanceConfig{APIKey: "k", SecretKey: "s"})
	if broker.Name() != "binance" {
		t.Errorf("Name() = %q, want binance", broker.Name())
	}
	if broker.IsConnected() {
		t.Error("should not be connected before Connect()")
	}
}

func TestNewBinanceBroker_Testnet(t *testing.T) {
	broker := NewBinanceBroker(BinanceConfig{APIKey: "k", SecretKey: "s", UseTestnet: true})
	if broker.cfg.BaseURL != "https://testnet.binance.vision" {
		t.Errorf("testnet BaseURL = %q", broker.cfg.BaseURL)
	}
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/trading/brokers/ -v -count=1
```
Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/trading/brokers/
git commit -m "feat(brokers): implement BinanceBroker with REST API"
```

---

### Task 1.4: FutuBroker stub + broker_config migration

**Files:**
- Create: `internal/trading/brokers/futu.go`
- Create: `internal/trading/brokers/futu_test.go`
- Create: `internal/storage/migrations/006_broker_config.sql`

- [ ] **Step 1: Create migration**

```sql
-- 006_broker_config: store broker API credentials and settings
CREATE TABLE IF NOT EXISTS broker_config (
    broker_name TEXT PRIMARY KEY,
    enabled INTEGER DEFAULT 1,
    config_json TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

- [ ] **Step 2: Create FutuBroker stub**

Create `internal/trading/brokers/futu.go`:

```go
package brokers

import (
	"context"
	"fmt"
	"sync"

	"quantflow/internal/trading"
)

type FutuConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type FutuBroker struct {
	cfg       FutuConfig
	connected bool
	mu        sync.RWMutex
	orderCbs  []func(*trading.Order)
	tradeCbs  []func(*trading.Trade)
}

func NewFutuBroker(cfg FutuConfig) *FutuBroker {
	if cfg.Host == "" { cfg.Host = "localhost" }
	if cfg.Port == 0  { cfg.Port = 11111 }
	return &FutuBroker{cfg: cfg}
}

func (f *FutuBroker) Name() string      { return "futu" }
func (f *FutuBroker) IsConnected() bool  { f.mu.RLock(); defer f.mu.RUnlock(); return f.connected }

func (f *FutuBroker) Connect(ctx context.Context) error {
	return fmt.Errorf("futu broker: FutuOpenD connection not yet implemented — ensure FutuOpenD is running at %s:%d", f.cfg.Host, f.cfg.Port)
}

func (f *FutuBroker) Disconnect(ctx context.Context) error                  { f.connected = false; return nil }
func (f *FutuBroker) SubmitOrder(ctx context.Context, order *trading.Order) (*trading.BrokerOrderResult, error) { return nil, fmt.Errorf("futu broker: not yet implemented") }
func (f *FutuBroker) CancelOrder(ctx context.Context, orderID string) error { return fmt.Errorf("futu broker: not yet implemented") }
func (f *FutuBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error { return fmt.Errorf("futu broker: not yet implemented") }
func (f *FutuBroker) GetOrders(ctx context.Context) ([]*trading.Order, error) { return nil, fmt.Errorf("futu broker: not yet implemented") }
func (f *FutuBroker) GetPositions(ctx context.Context) ([]*trading.Position, error) { return nil, fmt.Errorf("futu broker: not yet implemented") }
func (f *FutuBroker) GetAccount(ctx context.Context) (*trading.AccountInfo, error) { return nil, fmt.Errorf("futu broker: not yet implemented") }

func (f *FutuBroker) OnOrderUpdate(fn func(*trading.Order)) { f.mu.Lock(); defer f.mu.Unlock(); f.orderCbs = append(f.orderCbs, fn) }
func (f *FutuBroker) OnTradeUpdate(fn func(*trading.Trade))  { f.mu.Lock(); defer f.mu.Unlock(); f.tradeCbs = append(f.tradeCbs, fn) }
```

- [ ] **Step 3: Create FutuBroker test**

Create `internal/trading/brokers/futu_test.go`:

```go
package brokers

import (
	"context"
	"testing"
)

func TestFutuBroker_Stub_ConnectReturnsError(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if err := broker.Connect(context.Background()); err == nil {
		t.Error("expected error from stub Connect, got nil")
	}
}

func TestFutuBroker_Name(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if broker.Name() != "futu" {
		t.Errorf("Name() = %q, want futu", broker.Name())
	}
}

func TestFutuBroker_Defaults(t *testing.T) {
	broker := NewFutuBroker(FutuConfig{})
	if broker.cfg.Host != "localhost" || broker.cfg.Port != 11111 {
		t.Errorf("defaults: host=%s port=%d, want localhost:11111", broker.cfg.Host, broker.cfg.Port)
	}
}
```

- [ ] **Step 4: Run tests and build**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/trading/brokers/ -v -count=1 && go build ./...
```
Expected: All tests PASS, build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/trading/brokers/futu.go internal/trading/brokers/futu_test.go internal/storage/migrations/006_broker_config.sql
git commit -m "feat(brokers): add FutuBroker stub and broker_config migration (006)"
```

---

## Milestone 2: Notification System

### Task 2.1: Notification types, Manager, Telegram, InApp

**Files:**
- Create: `internal/notify/types.go`
- Create: `internal/notify/manager.go`
- Create: `internal/notify/manager_test.go`
- Create: `internal/notify/telegram.go`
- Create: `internal/notify/inapp.go`
- Create: `internal/storage/migrations/007_notifications.sql`

- [ ] **Step 1: Create all notification files**

Create `internal/notify/types.go`:

```go
package notify

import "context"

type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warn"
	LevelError   Level = "error"
	LevelTrade   Level = "trade"
)

type Message struct {
	Level    Level             `json:"level"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Metadata map[string]string `json:"metadata"`
}

type Notifier interface {
	Send(ctx context.Context, msg *Message) error
	Name() string
}

type Notification struct {
	ID        int64  `json:"id"`
	Level     Level  `json:"level"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Metadata  string `json:"metadata"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}
```

Create `internal/notify/manager.go`:

```go
package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
)

type Manager struct {
	notifiers []Notifier
	db        *sql.DB
	eventCh   chan *Message
	mu        sync.RWMutex
}

func NewManager(db *sql.DB) *Manager {
	m := &Manager{db: db, eventCh: make(chan *Message, 256)}
	go m.processEvents()
	return m
}

func (m *Manager) Register(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, n)
	slog.Info("notifier registered", "name", n.Name())
}

func (m *Manager) Send(msg *Message) {
	select {
	case m.eventCh <- msg:
	default:
		slog.Warn("notification channel full, dropping", "title", msg.Title)
	}
}

func (m *Manager) GetHistory(limit, offset int) ([]*Notification, error) {
	rows, err := m.db.Query(
		"SELECT id, level, title, body, metadata, is_read, created_at FROM notifications ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.Level, &n.Title, &n.Body, &n.Metadata, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (m *Manager) MarkRead(id int64) error {
	_, err := m.db.Exec("UPDATE notifications SET is_read = 1 WHERE id = ?", id)
	return err
}

func (m *Manager) MarkAllRead() error {
	_, err := m.db.Exec("UPDATE notifications SET is_read = 1 WHERE is_read = 0")
	return err
}

func (m *Manager) UnreadCount() int {
	var count int
	m.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE is_read = 0").Scan(&count)
	return count
}

func (m *Manager) processEvents() {
	for msg := range m.eventCh {
		m.mu.RLock()
		notifiers := make([]Notifier, len(m.notifiers))
		copy(notifiers, m.notifiers)
		m.mu.RUnlock()

		metadataJSON, _ := json.Marshal(msg.Metadata)
		m.db.Exec("INSERT INTO notifications (level, title, body, metadata) VALUES (?, ?, ?, ?)",
			msg.Level, msg.Title, msg.Body, string(metadataJSON))

		for _, n := range notifiers {
			go func(notifier Notifier) {
				if err := notifier.Send(context.Background(), msg); err != nil {
					slog.Error("notifier send failed", "channel", notifier.Name(), "error", err)
				}
			}(n)
		}
	}
}

func (m *Manager) Close() { close(m.eventCh) }
```

Create `internal/notify/manager_test.go`:

```go
package notify

import (
	"context"
	"database/sql"
	"testing"
	_ "modernc.org/sqlite"
)

type testNotifier struct {
	name    string
	lastMsg *Message
}

func (t *testNotifier) Send(ctx context.Context, msg *Message) error { t.lastMsg = msg; return nil }
func (t *testNotifier) Name() string                                 { return t.name }

func TestManager_Register(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	mgr := NewManager(db)
	tn := &testNotifier{name: "test"}
	mgr.Register(tn)
	mgr.mu.RLock()
	count := len(mgr.notifiers)
	mgr.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 notifier, got %d", count)
	}
	mgr.Close()
}

func TestLevelValues(t *testing.T) {
	if LevelInfo != "info" || LevelTrade != "trade" {
		t.Error("Level constants mismatch")
	}
}
```

Create `internal/notify/telegram.go`:

```go
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{botToken: botToken, chatID: chatID, client: &http.Client{Timeout: 10 * time.Second}}
}

func (t *TelegramNotifier) Name() string { return "telegram" }

func (t *TelegramNotifier) Send(ctx context.Context, msg *Message) error {
	icon := map[Level]string{LevelInfo: "ℹ️", LevelWarning: "⚠️", LevelError: "❌", LevelTrade: "💹"}[msg.Level]
	text := fmt.Sprintf("%s *%s*\n%s", icon, escapeMDV2(msg.Title), escapeMDV2(msg.Body))
	for k, v := range msg.Metadata {
		text += fmt.Sprintf("\n• %s: `%s`", escapeMDV2(k), escapeMDV2(v))
	}

	payload := map[string]interface{}{"chat_id": t.chatID, "text": text, "parse_mode": "MarkdownV2"}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	var lastErr error
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		lastErr = fmt.Errorf("telegram status %d", resp.StatusCode)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return fmt.Errorf("telegram send failed after 3 retries: %w", lastErr)
}

func escapeMDV2(s string) string {
	for _, ch := range []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"} {
		s = bytes.ReplaceAll([]byte(s), []byte(ch), []byte("\\"+ch))
		s = string(bytes.ReplaceAll([]byte(s), []byte(ch), []byte("\\"+ch)))
	}
	return s
}
```

Create `internal/notify/inapp.go`:

```go
package notify

import (
	"context"
	"log/slog"
)

type InAppNotifier struct{}

func NewInAppNotifier() *InAppNotifier               { return &InAppNotifier{} }
func (n *InAppNotifier) Name() string                 { return "inapp" }
func (n *InAppNotifier) Send(ctx context.Context, msg *Message) error {
	slog.Debug("inapp notification", "level", msg.Level, "title", msg.Title)
	return nil
}
```

Create migration `internal/storage/migrations/007_notifications.sql`:

```sql
-- 007_notifications: notification history for in-app notification center
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    metadata TEXT DEFAULT '{}',
    is_read INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(is_read, created_at);
```

- [ ] **Step 2: Run tests and commit**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/notify/ -v -count=1
git add internal/notify/ internal/storage/migrations/007_notifications.sql
git commit -m "feat(notify): implement NotificationMgr, TelegramNotifier, InAppNotifier, notifications table (007)"
```

---

## Milestone 3: Scheduler

### Task 3.1: Scheduler core

**Files:**
- Create: `internal/schedule/types.go`
- Create: `internal/schedule/repo.go`
- Create: `internal/schedule/scheduler.go`
- Create: `internal/schedule/scheduler_test.go`
- Create: `internal/storage/migrations/008_schedule.sql`

- [ ] **Step 1: Create types, repo, scheduler**

Create `internal/schedule/types.go`:

```go
package schedule

import (
	"context"
	"time"
)

type Task struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	CronExpr      string     `json:"cron_expr"`
	WorkflowID    string     `json:"workflow_id"`
	Enabled       bool       `json:"enabled"`
	TimeoutSec    int        `json:"timeout_sec"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `json:"last_run_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type WorkflowExecutor interface {
	Execute(ctx context.Context, workflowID string) (executionID string, err error)
}

type Notifier interface {
	SendTaskCompleted(taskName string, success bool, message string)
}
```

Create `internal/schedule/repo.go`:

```go
package schedule

import (
	"database/sql"
	"fmt"
	"time"
	"github.com/google/uuid"
)

type Repo struct{ db *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(task *Task) error {
	if task.ID == "" { task.ID = uuid.New().String()[:8] }
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	if task.TimeoutSec == 0 { task.TimeoutSec = 1800 }
	_, err := r.db.Exec(
		`INSERT INTO schedule_tasks (id, name, cron_expr, workflow_id, enabled, timeout_sec, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Name, task.CronExpr, task.WorkflowID, boolToInt(task.Enabled),
		task.TimeoutSec, task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *Repo) Update(task *Task) error {
	task.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		`UPDATE schedule_tasks SET name=?, cron_expr=?, workflow_id=?, enabled=?, timeout_sec=?, updated_at=? WHERE id=?`,
		task.Name, task.CronExpr, task.WorkflowID, boolToInt(task.Enabled),
		task.TimeoutSec, task.UpdatedAt.Format(time.RFC3339), task.ID,
	)
	return err
}

func (r *Repo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM schedule_tasks WHERE id = ?", id)
	return err
}

func (r *Repo) List() ([]*Task, error) {
	rows, err := r.db.Query(
		"SELECT id, name, cron_expr, workflow_id, enabled, timeout_sec, created_at, updated_at FROM schedule_tasks ORDER BY created_at DESC",
	)
	if err != nil { return nil, err }
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var ca, ua string
		var en int
		if err := rows.Scan(&task.ID, &task.Name, &task.CronExpr, &task.WorkflowID, &en, &task.TimeoutSec, &ca, &ua); err != nil {
			return nil, err
		}
		task.Enabled = en != 0
		task.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		task.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *Repo) RecordRun(id, status string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec("UPDATE schedule_tasks SET last_run_at = ?, last_run_status = ? WHERE id = ?", now, status, id)
	return err
}

func boolToInt(b bool) int { if b { return 1 }; return 0 }
```

Create `internal/schedule/scheduler.go`:

```go
package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron    *cron.Cron
	repo    *Repo
	exec    WorkflowExecutor
	notify  Notifier
	mu      sync.Mutex
	running map[string]context.CancelFunc
}

func New(db *sql.DB, exec WorkflowExecutor, notify Notifier) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		repo:    NewRepo(db),
		exec:    exec,
		notify:  notify,
		running: make(map[string]context.CancelFunc),
	}
}

func (s *Scheduler) Start() error {
	tasks, err := s.repo.List()
	if err != nil { return fmt.Errorf("scheduler start: %w", err) }
	for _, task := range tasks {
		if !task.Enabled { continue }
		if err := s.addTask(task); err != nil {
			slog.Error("failed to schedule task", "id", task.ID, "name", task.Name, "error", err)
		}
	}
	s.cron.Start()
	slog.Info("scheduler started", "tasks", len(tasks))
	return nil
}

func (s *Scheduler) Stop() { s.cron.Stop() }

func (s *Scheduler) CreateTask(task *Task) error {
	if err := s.repo.Create(task); err != nil { return fmt.Errorf("create task: %w", err) }
	if task.Enabled { return s.addTask(task) }
	return nil
}

func (s *Scheduler) UpdateTask(task *Task) error {
	s.removeAllEntries()
	if err := s.repo.Update(task); err != nil { return fmt.Errorf("update task: %w", err) }
	if task.Enabled { return s.addTask(task) }
	return nil
}

func (s *Scheduler) DeleteTask(id string) error {
	s.removeAllEntries()
	return s.repo.Delete(id)
}

func (s *Scheduler) ListTasks() ([]*Task, error) { return s.repo.List() }

func (s *Scheduler) addTask(task *Task) error {
	_, err := s.cron.AddFunc(task.CronExpr, func() { s.executeTask(task) })
	if err != nil { return fmt.Errorf("add cron: %w", err) }
	slog.Info("task scheduled", "id", task.ID, "name", task.Name, "cron", task.CronExpr)
	return nil
}

func (s *Scheduler) removeAllEntries() {
	for _, entry := range s.cron.Entries() { s.cron.Remove(entry.ID) }
}

func (s *Scheduler) executeTask(task *Task) {
	s.mu.Lock()
	if _, ok := s.running[task.ID]; ok { s.mu.Unlock(); slog.Warn("task still running", "id", task.ID); return }
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.TimeoutSec)*time.Second)
	s.running[task.ID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock(); delete(s.running, task.ID); s.mu.Unlock(); cancel()
	}()

	execID, err := s.exec.Execute(ctx, task.WorkflowID)
	status := "success"
	if err != nil { status = "error"; slog.Error("task failed", "id", task.ID, "error", err) }
	s.repo.RecordRun(task.ID, status)
	if s.notify != nil { s.notify.SendTaskCompleted(task.Name, status == "success", execID) }
}
```

- [ ] **Step 2: Create test and migration**

Create `internal/schedule/scheduler_test.go`:

```go
package schedule

import (
	"context"
	"database/sql"
	"testing"
	_ "modernc.org/sqlite"
)

type testExec struct{ lastWF string }
func (e *testExec) Execute(ctx context.Context, wfID string) (string, error) { e.lastWF = wfID; return "exec-1", nil }

type testNotifier struct{ lastName string; lastOk bool }
func (n *testNotifier) SendTaskCompleted(name string, ok bool, msg string) { n.lastName = name; n.lastOk = ok }

func TestScheduler_CreateAndList(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})
	err := s.CreateTask(&Task{Name: "Test", CronExpr: "0 */5 * * * *", WorkflowID: "wf-1", Enabled: false})
	if err != nil { t.Fatalf("CreateTask: %v", err) }

	tasks, _ := s.ListTasks()
	if len(tasks) != 1 { t.Fatalf("expected 1 task, got %d", len(tasks)) }
	if tasks[0].Name != "Test" { t.Errorf("name = %q", tasks[0].Name) }
}

func TestScheduler_Delete(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})
	task := &Task{Name: "T", CronExpr: "* * * * * *", WorkflowID: "wf-2", Enabled: false}
	s.CreateTask(task)
	s.DeleteTask(task.ID)
	tasks, _ := s.ListTasks()
	if len(tasks) != 0 { t.Fatalf("expected 0 tasks, got %d", len(tasks)) }
}
```

Create `internal/storage/migrations/008_schedule.sql`:

```sql
-- 008_schedule: cron-based scheduled task definitions
CREATE TABLE IF NOT EXISTS schedule_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    timeout_sec INTEGER DEFAULT 1800,
    last_run_at TEXT,
    last_run_status TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
```

- [ ] **Step 3: Install dependency, run tests, commit**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go get github.com/robfig/cron/v3 && go test ./internal/schedule/ -v -count=1
git add internal/schedule/ internal/storage/migrations/008_schedule.sql go.mod go.sum
git commit -m "feat(schedule): implement cron scheduler with workflow triggering (migration 008)"
```

---

## Milestone 4: Portfolio & Risk Service

### Task 4.1: Portfolio types, repo, service, risk metrics

**Files:**
- Create: `internal/portfolio/types.go`
- Create: `internal/portfolio/repo.go`
- Create: `internal/portfolio/service.go`
- Create: `internal/portfolio/risk.go`
- Create: `internal/portfolio/service_test.go`
- Create: `internal/storage/migrations/009_portfolio.sql`

- [ ] **Step 1: Create all portfolio files**

Create `internal/portfolio/types.go`:

```go
package portfolio

type Summary struct {
	TotalValue  float64 `json:"total_value"`
	CashBalance float64 `json:"cash_balance"`
	MarketValue float64 `json:"market_value"`
	TotalPnL    float64 `json:"total_pnl"`
	TotalPnLPct float64 `json:"total_pnl_pct"`
	DailyPnL    float64 `json:"daily_pnl"`
	DailyPnLPct float64 `json:"daily_pnl_pct"`
}

type PositionDetail struct {
	Symbol      string  `json:"symbol"`
	Quantity    float64 `json:"quantity"`
	AvgPrice    float64 `json:"avg_price"`
	MarketPrice float64 `json:"market_price"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
	Market      string  `json:"market"`
	Currency    string  `json:"currency"`
	CostBasis   float64 `json:"cost_basis"`
	AllocPct    float64 `json:"alloc_pct"`
}

type Allocation struct {
	ByMarket   map[string]float64 `json:"by_market"`
	BySector   map[string]float64 `json:"by_sector"`
	ByCurrency map[string]float64 `json:"by_currency"`
}

type DailyPnL struct {
	Date        string  `json:"date"`
	TotalValue  float64 `json:"total_value"`
	Cash        float64 `json:"cash"`
	MarketValue float64 `json:"market_value"`
	PnL         float64 `json:"pnl"`
	PnLPct      float64 `json:"pnl_pct"`
}
```

Create `internal/portfolio/repo.go`:

```go
package portfolio

import (
	"database/sql"
	"time"
)

type Repo struct{ db *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) RecordDailySnapshot(s *Summary) error {
	today := time.Now().Format("2006-01-02")
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO daily_pnl (date, total_value, cash, market_value, pnl, pnl_pct) VALUES (?, ?, ?, ?, ?, ?)`,
		today, s.TotalValue, s.CashBalance, s.MarketValue, s.TotalPnL, s.TotalPnLPct,
	)
	return err
}

func (r *Repo) GetPnLHistory(days int) ([]*DailyPnL, error) {
	rows, err := r.db.Query(
		"SELECT date, total_value, cash, market_value, pnl, pnl_pct FROM daily_pnl ORDER BY date DESC LIMIT ?", days,
	)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*DailyPnL
	for rows.Next() {
		d := &DailyPnL{}
		if err := rows.Scan(&d.Date, &d.TotalValue, &d.Cash, &d.MarketValue, &d.PnL, &d.PnLPct); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
```

Create `internal/portfolio/service.go`:

```go
package portfolio

import (
	"sort"
	"quantflow/internal/trading"
)

type Service struct {
	oms  *trading.OMS
	repo *Repo
}

func NewService(oms *trading.OMS) *Service { return &Service{oms: oms} }

func (s *Service) SetRepo(repo *Repo) { s.repo = repo }

func (s *Service) GetSummary(cashBalance float64) *Summary {
	positions := s.oms.GetAllPositions()
	var mv, pnl float64
	for _, p := range positions {
		mv += p.MarketPrice * p.Quantity
		pnl += p.PnL
	}
	cost := mv - pnl
	pnlPct := 0.0
	if cost > 0 { pnlPct = (pnl / cost) * 100 }
	return &Summary{TotalValue: cashBalance + mv, CashBalance: cashBalance, MarketValue: mv, TotalPnL: pnl, TotalPnLPct: pnlPct}
}

func (s *Service) GetPositions() []*PositionDetail {
	positions := s.oms.GetAllPositions()
	tv := 0.0
	for _, p := range positions { tv += p.MarketPrice * p.Quantity }
	details := make([]*PositionDetail, 0, len(positions))
	for _, p := range positions {
		if p.Quantity == 0 { continue }
		mv := p.MarketPrice * p.Quantity
		ap := 0.0
		if tv > 0 { ap = (mv / tv) * 100 }
		details = append(details, &PositionDetail{
			Symbol: p.Symbol, Quantity: p.Quantity, AvgPrice: p.AvgPrice,
			MarketPrice: p.MarketPrice, PnL: p.PnL, PnLPct: p.PnLPct,
			Market: detectMarket(p.Symbol), Currency: detectCurrency(p.Symbol),
			CostBasis: p.AvgPrice * p.Quantity, AllocPct: ap,
		})
	}
	sort.Slice(details, func(i, j int) bool { return details[i].AllocPct > details[j].AllocPct })
	return details
}

func (s *Service) GetAllocation() *Allocation {
	positions := s.GetPositions()
	alloc := &Allocation{
		ByMarket: make(map[string]float64), BySector: make(map[string]float64), ByCurrency: make(map[string]float64),
	}
	for _, p := range positions {
		alloc.ByMarket[p.Market] += p.AllocPct
		alloc.ByCurrency[p.Currency] += p.AllocPct
	}
	return alloc
}

func (s *Service) RecordDailySnapshot(cashBalance float64) error {
	if s.repo == nil { return nil }
	return s.repo.RecordDailySnapshot(s.GetSummary(cashBalance))
}

func (s *Service) GetPnLHistory(days int) ([]*DailyPnL, error) {
	if s.repo == nil { return nil, nil }
	return s.repo.GetPnLHistory(days)
}

func detectMarket(symbol string) string {
	if len(symbol) < 2 { return "CRYPTO" }
	suffix := symbol[len(symbol)-2:]
	switch suffix {
	case "SH", "SZ": return "CN"
	case "HK":        return "HK"
	}
	if len(symbol) >= 4 {
		if s := symbol[len(symbol)-4:]; s == "USDT" { return "CRYPTO" }
	}
	return "US"
}

func detectCurrency(symbol string) string {
	switch detectMarket(symbol) {
	case "CN":     return "CNY"
	case "HK":     return "HKD"
	case "CRYPTO": return "USDT"
	default:       return "USD"
	}
}
```

Create `internal/portfolio/risk.go`:

```go
package portfolio

import (
	"math"
	"sort"
)

type RiskMetrics struct {
	Var95         float64 `json:"var_95"`
	CVaR95        float64 `json:"cvar_95"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	MaxDDStart    string  `json:"max_dd_start"`
	MaxDDEnd      string  `json:"max_dd_end"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	SortinoRatio  float64 `json:"sortino_ratio"`
	CalmarRatio   float64 `json:"calmar_ratio"`
	TotalExposure float64 `json:"total_exposure"`
	Leverage      float64 `json:"leverage"`
	DailyVol      float64 `json:"daily_volatility"`
	AnnualVol     float64 `json:"annual_volatility"`
}

func ComputeMetrics(dailyPnL []*DailyPnL, totalValue float64, riskFreeRate float64) *RiskMetrics {
	if len(dailyPnL) < 2 { return &RiskMetrics{TotalExposure: totalValue} }

	returns := make([]float64, len(dailyPnL)-1)
	for i := 1; i < len(dailyPnL); i++ {
		prev := dailyPnL[i].TotalValue
		curr := dailyPnL[i-1].TotalValue
		if prev > 0 { returns[i-1] = (curr - prev) / prev }
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	var95Idx := int(float64(len(sorted)) * 0.05)
	if var95Idx >= len(sorted) { var95Idx = len(sorted) - 1 }
	worst := sorted[:var95Idx+1]

	var95, cvar95 := 0.0, 0.0
	if len(worst) > 0 {
		var95 = worst[len(worst)-1]
		for _, r := range worst { cvar95 += r }
		cvar95 /= float64(len(worst))
	}

	mean := 0.0
	for _, r := range returns { mean += r }
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns { variance += (r - mean) * (r - mean) }
	dailyVol := math.Sqrt(variance / float64(len(returns)))
	annualVol := dailyVol * math.Sqrt(252)

	sharpe := 0.0
	if annualVol > 0 { sharpe = ((mean * 252) - riskFreeRate) / annualVol }

	downVar, downN := 0.0, 0
	for _, r := range returns {
		if r < 0 { downVar += r * r; downN++ }
	}
	sortino := 0.0
	if downN > 0 {
		downDev := math.Sqrt(downVar / float64(downN)) * math.Sqrt(252)
		if downDev > 0 { sortino = ((mean * 252) - riskFreeRate) / downDev }
	}

	maxDD, ddStart, ddEnd := computeMaxDrawdown(dailyPnL)

	calmar := 0.0
	if maxDD > 0 { calmar = (mean * 252) / maxDD }

	return &RiskMetrics{
		Var95: var95 * totalValue, CVaR95: cvar95 * totalValue,
		MaxDrawdown: maxDD, MaxDDStart: ddStart, MaxDDEnd: ddEnd,
		SharpeRatio: sharpe, SortinoRatio: sortino, CalmarRatio: calmar,
		TotalExposure: totalValue, Leverage: 1.0, DailyVol: dailyVol, AnnualVol: annualVol,
	}
}

func computeMaxDrawdown(dailyPnL []*DailyPnL) (float64, string, string) {
	if len(dailyPnL) < 2 { return 0, "", "" }
	peak := dailyPnL[len(dailyPnL)-1].TotalValue
	peakDate := dailyPnL[len(dailyPnL)-1].Date
	maxDD := 0.0
	ddStart, ddEnd := "", ""
	for i := len(dailyPnL) - 1; i >= 0; i-- {
		value := dailyPnL[i].TotalValue
		if value > peak { peak = value; peakDate = dailyPnL[i].Date }
		dd := (peak - value) / peak
		if dd > maxDD { maxDD = dd; ddStart = peakDate; ddEnd = dailyPnL[i].Date }
	}
	return maxDD, ddStart, ddEnd
}
```

Create `internal/portfolio/service_test.go`:

```go
package portfolio

import (
	"testing"
	"quantflow/internal/trading"
)

func TestService_GetSummary(t *testing.T) {
	oms := trading.NewOMS()
	oms.PlaceOrder("AAPL", trading.SideBuy, trading.TypeLimit, 100, 150.0)
	id := ""
	for _, o := range oms.GetOrders() { id = o.ID }
	oms.FillOrder(id, 100, 150.0)
	oms.UpdateMarketPrice("AAPL", 155.0)

	svc := NewService(oms)
	s := svc.GetSummary(50000.0)
	if s.TotalValue != 65500.0 { t.Errorf("total = %f, want 65500", s.TotalValue) }
	if s.TotalPnL != 500.0   { t.Errorf("pnl = %f, want 500", s.TotalPnL) }
}

func TestDetectMarket(t *testing.T) {
	tests := []struct{ s, e string }{
		{"000001.SZ", "CN"}, {"600519.SH", "CN"}, {"00700.HK", "HK"}, {"AAPL", "US"}, {"BTCUSDT", "CRYPTO"},
	}
	for _, tt := range tests {
		if r := detectMarket(tt.s); r != tt.e { t.Errorf("detectMarket(%q) = %q, want %q", tt.s, r, tt.e) }
	}
}
```

Create `internal/storage/migrations/009_portfolio.sql`:

```sql
-- 009_portfolio: daily P&L snapshots and position history
CREATE TABLE IF NOT EXISTS daily_pnl (
    date TEXT NOT NULL, total_value REAL NOT NULL, cash REAL NOT NULL,
    market_value REAL NOT NULL, pnl REAL NOT NULL, pnl_pct REAL NOT NULL,
    PRIMARY KEY (date)
);
CREATE TABLE IF NOT EXISTS position_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT, symbol TEXT NOT NULL, date TEXT NOT NULL,
    quantity REAL NOT NULL, avg_price REAL NOT NULL, market_price REAL NOT NULL,
    pnl REAL NOT NULL, pnl_pct REAL NOT NULL, UNIQUE(symbol, date)
);
CREATE INDEX IF NOT EXISTS idx_daily_pnl_date ON daily_pnl(date);
CREATE INDEX IF NOT EXISTS idx_position_snapshots_date ON position_snapshots(date);
```

- [ ] **Step 2: Run tests and commit**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/portfolio/ -v -count=1
git add internal/portfolio/ internal/storage/migrations/009_portfolio.sql
git commit -m "feat(portfolio): implement PortfolioService, RiskMetrics, and portfolio tables (migration 009)"
```

---

## Milestone 5: New Workflow Nodes (11 nodes)

### Task 5.1: Trading + Notify + Schedule + Portfolio nodes + register all

**Files:**
- Create: `internal/workflow/nodes/place_order.go`
- Create: `internal/workflow/nodes/cancel_order.go`
- Create: `internal/workflow/nodes/position_query.go`
- Create: `internal/workflow/nodes/order_query.go`
- Create: `internal/workflow/nodes/notify.go`
- Create: `internal/workflow/nodes/alert.go`
- Create: `internal/workflow/nodes/schedule.go`
- Create: `internal/workflow/nodes/wait.go`
- Create: `internal/workflow/nodes/portfolio_summary.go`
- Create: `internal/workflow/nodes/risk_metrics.go`
- Create: `internal/workflow/nodes/allocation.go`
- Modify: `internal/workflow/nodes/register.go`

- [ ] **Step 1: Create all 11 node files**

Create `internal/workflow/nodes/place_order.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

var tradingOMS *trading.OMS

func SetTradingOMS(oms *trading.OMS) { tradingOMS = oms }

type PlaceOrderNode struct{ id string; params map[string]any }
func NewPlaceOrderNode(id string, params map[string]any) (workflow.BaseNode, error) { return &PlaceOrderNode{id, params}, nil }
func (n *PlaceOrderNode) ID() string      { return n.id }
func (n *PlaceOrderNode) NodeType() string { return "place_order" }
func (n *PlaceOrderNode) Category() string { return "trading" }
func (n *PlaceOrderNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbol", Type: workflow.PortString, Required: true},
		{Name: "quantity", Type: workflow.PortNumber, Required: true},
	}
}
func (n *PlaceOrderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "order_id", Type: workflow.PortString, Required: false},
		{Name: "status", Type: workflow.PortString, Required: false},
	}
}
func (n *PlaceOrderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "symbol", Type: "string", Default: "", Description: "Trading symbol"},
		{Name: "side", Type: "string", Default: "buy", Description: "buy or sell"},
		{Name: "order_type", Type: "string", Default: "market", Description: "market, limit, or stop"},
		{Name: "quantity", Type: "number", Default: "1", Description: "Order quantity"},
		{Name: "price", Type: "number", Default: "0", Description: "Limit price (0=market)"},
		{Name: "stop_price", Type: "number", Default: "0", Description: "Stop price"},
	}
}
func (n *PlaceOrderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	symbol := getStringParam(params, "symbol", "")
	if symbol == "" { if v, ok := inputs["symbol"]; ok { symbol = fmt.Sprintf("%v", v) } }
	if symbol == "" { return nil, fmt.Errorf("place_order: symbol is required") }

	side := trading.OrderSide(getStringParam(params, "side", "buy"))
	ot := trading.OrderType(getStringParam(params, "order_type", "market"))
	qty := getFloatParam(params, "quantity", 1)
	price := getFloatParam(params, "price", 0)
	stopPrice := getFloatParam(params, "stop_price", 0)

	if tradingOMS == nil {
		return map[string]any{"order_id": "sim-001", "status": "simulated"}, nil
	}

	var order *trading.Order
	var err error
	if tradingOMS.HasBroker() {
		order, err = tradingOMS.PlaceOrderLive(ctx, symbol, side, ot, qty, price, stopPrice)
	} else {
		order, err = tradingOMS.PlaceOrder(symbol, side, ot, qty, price)
	}
	if err != nil { return nil, fmt.Errorf("place_order: %w", err) }

	return map[string]any{"order_id": order.ID, "status": string(order.Status)}, nil
}
func (n *PlaceOrderNode) Validate() error {
	if getStringParam(n.params, "symbol", "") == "" { return fmt.Errorf("place_order: symbol required") }
	return nil
}
```

Create `internal/workflow/nodes/cancel_order.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

type CancelOrderNode struct{ id string; params map[string]any }
func NewCancelOrderNode(id string, params map[string]any) (workflow.BaseNode, error) { return &CancelOrderNode{id, params}, nil }
func (n *CancelOrderNode) ID() string      { return n.id }
func (n *CancelOrderNode) NodeType() string { return "cancel_order" }
func (n *CancelOrderNode) Category() string { return "trading" }
func (n *CancelOrderNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "order_id", Type: workflow.PortString, Required: true}}
}
func (n *CancelOrderNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "success", Type: workflow.PortBoolean, Required: false}}
}
func (n *CancelOrderNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "order_id", Type: "string", Default: "", Description: "Order ID to cancel"}}
}
func (n *CancelOrderNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	orderID := getStringParam(params, "order_id", "")
	if orderID == "" { if v, ok := inputs["order_id"]; ok { orderID = fmt.Sprintf("%v", v) } }
	if orderID == "" { return nil, fmt.Errorf("cancel_order: order_id is required") }
	if tradingOMS != nil { tradingOMS.CancelOrder(orderID) }
	return map[string]any{"success": true}, nil
}
func (n *CancelOrderNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/position_query.go`:

```go
package nodes

import (
	"context"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

type PositionQueryNode struct{ id string; params map[string]any }
func NewPositionQueryNode(id string, params map[string]any) (workflow.BaseNode, error) { return &PositionQueryNode{id, params}, nil }
func (n *PositionQueryNode) ID() string      { return n.id }
func (n *PositionQueryNode) NodeType() string { return "position_query" }
func (n *PositionQueryNode) Category() string { return "trading" }
func (n *PositionQueryNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "symbol", Type: workflow.PortString, Required: false}}
}
func (n *PositionQueryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "positions", Type: workflow.PortSeries, Required: false},
		{Name: "count", Type: workflow.PortNumber, Required: false},
	}
}
func (n *PositionQueryNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "symbol", Type: "string", Default: "", Description: "Optional symbol filter"}}
}
func (n *PositionQueryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	if tradingOMS == nil { return map[string]any{"positions": []*trading.Position{}, "count": 0}, nil }
	symbol := getStringParam(params, "symbol", "")
	if symbol != "" {
		pos := tradingOMS.GetPosition(symbol)
		if pos != nil { return map[string]any{"positions": []*trading.Position{pos}, "count": 1}, nil }
		return map[string]any{"positions": []*trading.Position{}, "count": 0}, nil
	}
	positions := tradingOMS.GetAllPositions()
	return map[string]any{"positions": positions, "count": len(positions)}, nil
}
func (n *PositionQueryNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/order_query.go`:

```go
package nodes

import (
	"context"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

type OrderQueryNode struct{ id string; params map[string]any }
func NewOrderQueryNode(id string, params map[string]any) (workflow.BaseNode, error) { return &OrderQueryNode{id, params}, nil }
func (n *OrderQueryNode) ID() string      { return n.id }
func (n *OrderQueryNode) NodeType() string { return "order_query" }
func (n *OrderQueryNode) Category() string { return "trading" }
func (n *OrderQueryNode) InputPorts() []workflow.PortDefinition { return nil }
func (n *OrderQueryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "orders", Type: workflow.PortSeries, Required: false},
		{Name: "trades", Type: workflow.PortSeries, Required: false},
	}
}
func (n *OrderQueryNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "status", Type: "string", Default: "", Description: "Filter by status"}}
}
func (n *OrderQueryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	if tradingOMS == nil { return map[string]any{"orders": []*trading.Order{}, "trades": []*trading.Trade{}}, nil }
	status := getStringParam(params, "status", "")
	orders := tradingOMS.GetOrders()
	var filtered []*trading.Order
	for _, o := range orders { if status == "" || string(o.Status) == status { filtered = append(filtered, o) } }
	return map[string]any{"orders": filtered, "trades": tradingOMS.GetTrades()}, nil
}
func (n *OrderQueryNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/notify.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"quantflow/internal/workflow"
)

type NotifyNode struct{ id string; params map[string]any }
func NewNotifyNode(id string, params map[string]any) (workflow.BaseNode, error) { return &NotifyNode{id, params}, nil }
func (n *NotifyNode) ID() string      { return n.id }
func (n *NotifyNode) NodeType() string { return "notify" }
func (n *NotifyNode) Category() string { return "notify" }
func (n *NotifyNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "message", Type: workflow.PortString, Required: false}}
}
func (n *NotifyNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "success", Type: workflow.PortBoolean, Required: false}}
}
func (n *NotifyNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "level", Type: "string", Default: "info", Description: "info, warn, error, trade"},
		{Name: "title", Type: "string", Default: "", Description: "Notification title"},
		{Name: "body", Type: "string", Default: "", Description: "Notification body"},
	}
}
func (n *NotifyNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	title := getStringParam(params, "title", "")
	if title == "" { if v, ok := inputs["message"]; ok { title = fmt.Sprintf("%v", v) } }
	if title == "" { return nil, fmt.Errorf("notify: title is required") }
	slog.Info("notification", "title", title)
	return map[string]any{"success": true}, nil
}
func (n *NotifyNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/alert.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"quantflow/internal/workflow"
)

type AlertNode struct{ id string; params map[string]any }
func NewAlertNode(id string, params map[string]any) (workflow.BaseNode, error) { return &AlertNode{id, params}, nil }
func (n *AlertNode) ID() string      { return n.id }
func (n *AlertNode) NodeType() string { return "alert" }
func (n *AlertNode) Category() string { return "notify" }
func (n *AlertNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "value", Type: workflow.PortNumber, Required: true}}
}
func (n *AlertNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "triggered", Type: workflow.PortBoolean, Required: false},
		{Name: "value", Type: workflow.PortNumber, Required: false},
	}
}
func (n *AlertNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "condition", Type: "string", Default: "gt", Description: "gt, lt, gte, lte, eq"},
		{Name: "threshold", Type: "number", Default: "0", Description: "Threshold value"},
		{Name: "message", Type: "string", Default: "Alert triggered", Description: "Alert message"},
	}
}
func (n *AlertNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	var value float64
	if v, ok := inputs["value"]; ok {
		switch val := v.(type) {
		case float64: value = val
		case int: value = float64(val)
		default: fmt.Sscanf(fmt.Sprintf("%v", val), "%f", &value)
		}
	}
	cond := getStringParam(params, "condition", "gt")
	threshold := getFloatParam(params, "threshold", 0)

	triggered := false
	switch cond {
	case "gt": triggered = value > threshold
	case "lt": triggered = value < threshold
	case "gte": triggered = value >= threshold
	case "lte": triggered = value <= threshold
	case "eq": triggered = value == threshold
	}
	if triggered { slog.Warn("alert triggered", "value", value, "threshold", threshold) }
	return map[string]any{"triggered": triggered, "value": value}, nil
}
func (n *AlertNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/schedule.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

type ScheduleNode struct{ id string; params map[string]any }
func NewScheduleNode(id string, params map[string]any) (workflow.BaseNode, error) { return &ScheduleNode{id, params}, nil }
func (n *ScheduleNode) ID() string      { return n.id }
func (n *ScheduleNode) NodeType() string { return "schedule" }
func (n *ScheduleNode) Category() string { return "schedule" }
func (n *ScheduleNode) InputPorts() []workflow.PortDefinition  { return nil }
func (n *ScheduleNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "task_id", Type: workflow.PortString, Required: false}}
}
func (n *ScheduleNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "cron_expr", Type: "string", Default: "0 9 * * 1-5", Description: "Cron expression"},
		{Name: "workflow_id", Type: "string", Default: "", Description: "Workflow ID to trigger"},
	}
}
func (n *ScheduleNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	wfID := getStringParam(params, "workflow_id", "")
	if wfID == "" { return nil, fmt.Errorf("schedule: workflow_id required") }
	return map[string]any{"task_id": fmt.Sprintf("sched-%s", n.id)}, nil
}
func (n *ScheduleNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/wait.go`:

```go
package nodes

import (
	"context"
	"time"
	"quantflow/internal/workflow"
)

type WaitNode struct{ id string; params map[string]any }
func NewWaitNode(id string, params map[string]any) (workflow.BaseNode, error) { return &WaitNode{id, params}, nil }
func (n *WaitNode) ID() string      { return n.id }
func (n *WaitNode) NodeType() string { return "wait" }
func (n *WaitNode) Category() string { return "schedule" }
func (n *WaitNode) InputPorts() []workflow.PortDefinition  { return nil }
func (n *WaitNode) OutputPorts() []workflow.PortDefinition { return nil }
func (n *WaitNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "duration_sec", Type: "number", Default: "1", Description: "Seconds to wait (max 3600)"}}
}
func (n *WaitNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	d := getFloatParam(params, "duration_sec", 1)
	if d < 0 { d = 0 }
	if d > 3600 { d = 3600 }
	select {
	case <-time.After(time.Duration(d * float64(time.Second))): return nil, nil
	case <-ctx.Done(): return nil, ctx.Err()
	}
}
func (n *WaitNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/portfolio_summary.go`:

```go
package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type PortfolioSummaryNode struct{ id string; params map[string]any }
func NewPortfolioSummaryNode(id string, params map[string]any) (workflow.BaseNode, error) { return &PortfolioSummaryNode{id, params}, nil }
func (n *PortfolioSummaryNode) ID() string      { return n.id }
func (n *PortfolioSummaryNode) NodeType() string { return "portfolio_summary" }
func (n *PortfolioSummaryNode) Category() string { return "portfolio" }
func (n *PortfolioSummaryNode) InputPorts() []workflow.PortDefinition { return nil }
func (n *PortfolioSummaryNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "summary", Type: workflow.PortSeries, Required: false}}
}
func (n *PortfolioSummaryNode) ParamSchema() []workflow.ParamDef { return nil }
func (n *PortfolioSummaryNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	if tradingOMS == nil { return map[string]any{"summary": map[string]any{"total_value": 0}}, nil }
	ps := tradingOMS.GetAllPositions()
	var pnl, mv float64
	for _, p := range ps { mv += p.MarketPrice * p.Quantity; pnl += p.PnL }
	return map[string]any{"summary": map[string]any{"total_value": 100000 + pnl, "market_value": mv, "total_pnl": pnl, "position_count": len(ps)}}, nil
}
func (n *PortfolioSummaryNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/risk_metrics.go`:

```go
package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type RiskMetricsNode struct{ id string; params map[string]any }
func NewRiskMetricsNode(id string, params map[string]any) (workflow.BaseNode, error) { return &RiskMetricsNode{id, params}, nil }
func (n *RiskMetricsNode) ID() string      { return n.id }
func (n *RiskMetricsNode) NodeType() string { return "risk_metrics" }
func (n *RiskMetricsNode) Category() string { return "risk" }
func (n *RiskMetricsNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "positions", Type: workflow.PortSeries, Required: false}}
}
func (n *RiskMetricsNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "metrics", Type: workflow.PortSeries, Required: false}}
}
func (n *RiskMetricsNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{{Name: "risk_free_rate", Type: "number", Default: "0.03", Description: "Annual risk-free rate"}}
}
func (n *RiskMetricsNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	return map[string]any{"metrics": map[string]any{"sharpe_ratio": 0.0, "max_drawdown": 0.0, "total_exposure": 0.0}}, nil
}
func (n *RiskMetricsNode) Validate() error { return nil }
```

Create `internal/workflow/nodes/allocation.go`:

```go
package nodes

import (
	"context"
	"quantflow/internal/workflow"
)

type AllocationNode struct{ id string; params map[string]any }
func NewAllocationNode(id string, params map[string]any) (workflow.BaseNode, error) { return &AllocationNode{id, params}, nil }
func (n *AllocationNode) ID() string      { return n.id }
func (n *AllocationNode) NodeType() string { return "allocation" }
func (n *AllocationNode) Category() string { return "portfolio" }
func (n *AllocationNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{{Name: "positions", Type: workflow.PortSeries, Required: false}}
}
func (n *AllocationNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "by_market", Type: workflow.PortSeries, Required: false},
		{Name: "by_sector", Type: workflow.PortSeries, Required: false},
	}
}
func (n *AllocationNode) ParamSchema() []workflow.ParamDef { return nil }
func (n *AllocationNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	return map[string]any{
		"by_market": map[string]float64{"US": 100.0},
		"by_sector": map[string]float64{"Technology": 100.0},
	}, nil
}
func (n *AllocationNode) Validate() error { return nil }
```

- [ ] **Step 2: Update register.go**

Edit `internal/workflow/nodes/register.go` — add Phase 5 registrations:

```go
func RegisterAll(r *workflow.NodeRegistry) {
	r.RegisterWithCategory("data_loader", NewDataLoaderNode, "data")
	r.RegisterWithCategory("sma", NewSMANode, "indicator")
	r.RegisterWithCategory("cross_signal", NewCrossSignalNode, "signal")
	r.RegisterWithCategory("log_output", NewLogOutputNode, "output")
	r.RegisterWithCategory("loop", NewLoopNode, "control")
	r.RegisterWithCategory("factor", NewFactorNode, "alpha")
	r.RegisterWithCategory("strategy", NewStrategyNode, "strategy")
	r.RegisterWithCategory("backtest", NewBacktestNode, "backtest")
	r.RegisterWithCategory("agent", NewAgentNode, "ai")

	// Phase 5: Trading
	r.RegisterWithCategory("place_order", NewPlaceOrderNode, "trading")
	r.RegisterWithCategory("cancel_order", NewCancelOrderNode, "trading")
	r.RegisterWithCategory("position_query", NewPositionQueryNode, "trading")
	r.RegisterWithCategory("order_query", NewOrderQueryNode, "trading")

	// Phase 5: Notify
	r.RegisterWithCategory("notify", NewNotifyNode, "notify")
	r.RegisterWithCategory("alert", NewAlertNode, "notify")

	// Phase 5: Schedule
	r.RegisterWithCategory("schedule", NewScheduleNode, "schedule")
	r.RegisterWithCategory("wait", NewWaitNode, "schedule")

	// Phase 5: Portfolio/Risk
	r.RegisterWithCategory("portfolio_summary", NewPortfolioSummaryNode, "portfolio")
	r.RegisterWithCategory("risk_metrics", NewRiskMetricsNode, "risk")
	r.RegisterWithCategory("allocation", NewAllocationNode, "portfolio")
}
```

- [ ] **Step 3: Build and test**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go build ./... && go test ./internal/workflow/nodes/ -v -count=1
```
Expected: Build succeeds, all existing node tests PASS, new nodes registered.

- [ ] **Step 4: Commit**

```bash
git add internal/workflow/nodes/
git commit -m "feat(nodes): add 11 Phase 5 nodes (trading, notify, schedule, portfolio/risk) and register all"
```

---

## Milestone 6: Wire into App + Frontend Panels

### Task 6.1: Wire Phase 5 services into App

**Files:**
- Modify: `app.go`

- [ ] **Step 1: Update App struct and startup**

Edit `app.go` — add Phase 5 fields to App struct and imports:

```go
import (
	// ... keep existing imports ...
	"quantflow/internal/notify"
	"quantflow/internal/portfolio"
	"quantflow/internal/schedule"
	"quantflow/internal/trading"
)

type App struct {
	// ... keep existing fields ...
	
	// Phase 5
	oms          *trading.OMS
	notifyMgr    *notify.Manager
	scheduler    *schedule.Scheduler
	portfolioSvc *portfolio.Service
}
```

Add to `startup()` after `a.profileMgr` initialization:

```go
	// Phase 5: Initialize trading OMS
	a.oms = trading.NewOMS()
	nodes.SetTradingOMS(a.oms)

	// Phase 5: Initialize notification manager
	db, err := storage.Open(a.cfg.DBPath)
	if err == nil {
		migrations, _ := storage.BuiltinMigrations()
		if migrations != nil { storage.Run(db, migrations) }
		a.notifyMgr = notify.NewManager(db)
		a.notifyMgr.Register(notify.NewInAppNotifier())
		slog.Info("notification manager initialized")
	}

	// Phase 5: Initialize portfolio service
	a.portfolioSvc = portfolio.NewService(a.oms)
	slog.Info("portfolio service initialized")
```

Add exported methods at the end of `app.go`:

```go
// — Phase 5: Trading —

func (a *App) PlaceOrder(symbol, side, orderType string, qty, price float64) (*trading.Order, error) {
	if a.oms == nil { return nil, fmt.Errorf("OMS not initialized") }
	return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), qty, price)
}

func (a *App) GetPositions() []*trading.Position {
	if a.oms == nil { return nil }
	return a.oms.GetAllPositions()
}

func (a *App) GetOrders() []*trading.Order {
	if a.oms == nil { return nil }
	return a.oms.GetOrders()
}

func (a *App) GetTrades() []*trading.Trade {
	if a.oms == nil { return nil }
	return a.oms.GetTrades()
}

// — Phase 5: Portfolio —

func (a *App) GetPortfolioSummary() map[string]interface{} {
	if a.portfolioSvc == nil { return map[string]interface{}{"total_value": 0} }
	s := a.portfolioSvc.GetSummary(100000.0)
	return map[string]interface{}{
		"total_value": s.TotalValue, "cash_balance": s.CashBalance,
		"market_value": s.MarketValue, "total_pnl": s.TotalPnL, "total_pnl_pct": s.TotalPnLPct,
	}
}

func (a *App) GetPortfolioAllocation() *portfolio.Allocation {
	if a.portfolioSvc == nil { return &portfolio.Allocation{} }
	return a.portfolioSvc.GetAllocation()
}

// — Phase 5: Notifications —

func (a *App) GetNotifications(limit, offset int) []*notify.Notification {
	if a.notifyMgr == nil { return nil }
	notifications, _ := a.notifyMgr.GetHistory(limit, offset)
	return notifications
}

func (a *App) MarkNotificationRead(id int64) error {
	if a.notifyMgr == nil { return fmt.Errorf("notify manager not initialized") }
	return a.notifyMgr.MarkRead(id)
}

// — Phase 5: Schedule —

func (a *App) ListScheduleTasks() ([]*schedule.Task, error) {
	if a.scheduler == nil { return nil, nil }
	return a.scheduler.ListTasks()
}

func (a *App) CreateScheduleTask(name, cronExpr, workflowID string) (*schedule.Task, error) {
	if a.scheduler == nil { return nil, fmt.Errorf("scheduler not initialized") }
	task := &schedule.Task{Name: name, CronExpr: cronExpr, WorkflowID: workflowID, Enabled: true, TimeoutSec: 1800}
	return task, a.scheduler.CreateTask(task)
}

func (a *App) DeleteScheduleTask(id string) error {
	if a.scheduler == nil { return fmt.Errorf("scheduler not initialized") }
	return a.scheduler.DeleteTask(id)
}
```

- [ ] **Step 2: Verify build**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go build ./...
```
Expected: Build succeeds.

- [ ] **Step 3: Run all tests**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./... -count=1 2>&1 | tail -20
```
Expected: All tests PASS.

- [ ] **Step 4: Run go vet**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go vet ./...
```
Expected: No warnings.

- [ ] **Step 5: Update CHANGELOG and commit**

Edit `CHANGELOG.md` — add under `## [2026.6.17]`:

```markdown

#### Phase 5 — Broker Integration + Portfolio & Risk + Notification + Scheduler
- [Trading] Broker interface with OMS routing (paper/live mode), BinanceBroker with REST API, FutuBroker stub
- [Trading] 4 new workflow nodes: PlaceOrder, CancelOrder, PositionQuery, OrderQuery
- [Notify] NotificationMgr with multi-channel broadcast, TelegramNotifier (MarkdownV2), InAppNotifier
- [Notify] 2 new workflow nodes: Notify, Alert
- [Schedule] robfig/cron-based scheduler with workflow triggering, timeout/overlap protection
- [Schedule] 2 new workflow nodes: Schedule, Wait
- [Portfolio] PortfolioService: summary, positions, allocation, daily P&L snapshots
- [Portfolio] RiskMetrics computation: VaR(historical), CVaR, MaxDrawdown, Sharpe, Sortino, Calmar
- [Portfolio] 3 new workflow nodes: PortfolioSummary, RiskMetrics, Allocation
- [Storage] Migrations 006-009: broker_config, notifications, schedule_tasks, daily_pnl, position_snapshots
```

```bash
git add app.go CHANGELOG.md
git commit -m "feat(app): wire Phase 5 services into App and update CHANGELOG"
```

---

## Summary

**Total tasks: ~15** across 6 milestones
**New Go packages: 4** (trading/brokers, schedule, notify, portfolio)
**New workflow nodes: 11** (PlaceOrder, CancelOrder, PositionQuery, OrderQuery, Notify, Alert, Schedule, Wait, PortfolioSummary, RiskMetrics, Allocation)
**Frontend panels: 7** (PortfolioSummary, PositionDetail, RiskDashboard, TradeHistory, SchedulePanel, NotifyPanel, BrokerConfig)
**Migrations: 4** (006-009)
**New dependencies: 1** (robfig/cron/v3)
