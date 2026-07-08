# IBKR Broker Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `trading.Broker` adapter for Interactive Brokers via Client Portal REST API.

**Architecture:** New `IBKRBroker` in `internal/trading/brokers/ibkr.go` (main) + `ibkr_session.go` (session management) + `ibkr_test.go` (HTTP mock tests). Follows the exact same pattern as AlpacaBroker and BinanceBroker. Registration in `app_startup.go`.

**Tech Stack:** Go 1.22+, `net/http` (no external dependencies), `net/http/httptest` (tests)

## Global Constraints

- Must implement all 11 methods of `trading.Broker` interface (Connect, Disconnect, IsConnected, Name, SubmitOrder, CancelOrder, ModifyOrder, GetOrders, GetPositions, GetAccount, OnOrderUpdate, OnTradeUpdate)
- HTTP client must skip TLS verification (`InsecureSkipVerify: true` for IB Gateway self-signed certs)
- Session refresh goroutine every 4 minutes via `GET /sso/validate`
- Follow Alpaca/Binance patterns exactly (error wrapping style, mutex usage, logging)

---
### Task 1: Config + Types + Session Management

**Files:**
- Create: `internal/trading/brokers/ibkr_config.go`
- Create: `internal/trading/brokers/ibkr_session.go`

**Interfaces:**
- Consumes: `trading.Broker` interface (from parent package)
- Produces: `IBKRConfig` struct, `ibkrAPIOrder` / `ibkrAPIOrderReply` / `ibkrPosition` / `ibkrAccountSummary` types, session validation + refresh functions

---

- [ ] **Step 1: Create `ibkr_config.go`**

```go
package brokers

// IBKRConfig holds configuration for the Interactive Brokers Client Portal REST API.
type IBKRConfig struct {
	Host      string `json:"host"`       // IB Gateway host (default: localhost)
	Port      int    `json:"port"`       // IB Gateway port (default: 5000)
	AccountID string `json:"account_id"` // IBKR numeric account ID
}
```

- [ ] **Step 2: Create `ibkr_session.go`**

```go
package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ibkrSession manages the IBKR Client Portal session token and expiry.
type ibkrSession struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	stopCh    chan struct{}
}

// isValid checks if the current session is still valid locally.
func (s *ibkrSession) isValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token != "" && time.Now().Before(s.expiresAt)
}

// setToken stores the session token with a 30-minute expiry.
func (s *ibkrSession) setToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	s.expiresAt = time.Now().Add(30 * time.Minute)
}

// getToken returns the current session token.
func (s *ibkrSession) getToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// clear resets the session.
func (s *ibkrSession) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
	s.expiresAt = time.Time{}
}

// startRefresh launches a background goroutine that validates the session every 4 minutes.
func (s *ibkrSession) startRefresh(ctx context.Context, client *http.Client, baseURL string) {
	s.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.validate(ctx, client, baseURL); err != nil {
					slog.Warn("ibkr session refresh failed", "error", err)
				}
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// stopRefresh stops the background session refresh goroutine.
func (s *ibkrSession) stopRefresh() {
	if s.stopCh != nil {
		close(s.stopCh)
	}
}

// validate performs GET /sso/validate to check if the session is still valid.
func (s *ibkrSession) validate(ctx context.Context, client *http.Client, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/sso/validate", nil)
	if err != nil {
		return fmt.Errorf("ibkr session validate: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ibkr session validate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ibkr session validate: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Authenticated bool   `json:"authenticated"`
		Token         string `json:"token"`
		Expires       int    `json:"expires"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("ibkr session validate parse: %w", err)
	}
	if !result.Authenticated {
		return fmt.Errorf("ibkr session not authenticated — user must log into IB Gateway")
	}

	s.setToken(result.Token)
	slog.Debug("ibkr session refreshed", "expires_in_s", result.Expires)
	return nil
}
```

- [ ] **Step 3: Create `ibkr_session_test.go`**

```go
package brokers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIBKRSession_Validate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sso/validate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"token":         "test-session-token",
			"expires":       1800,
		})
	}))
	defer server.Close()

	sess := &ibkrSession{}
	err := sess.validate(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("validate() error: %v", err)
	}
	token := sess.getToken()
	if token != "test-session-token" {
		t.Errorf("getToken() = %q, want %q", token, "test-session-token")
	}
	if !sess.isValid() {
		t.Error("expected session to be valid after validate()")
	}
}

func TestIBKRSession_Validate_NotAuthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
	}))
	defer server.Close()

	sess := &ibkrSession{}
	err := sess.validate(context.Background(), server.Client(), server.URL)
	if err == nil {
		t.Fatal("expected error for not authenticated")
	}
}

func TestIBKRSession_Clear(t *testing.T) {
	sess := &ibkrSession{}
	sess.setToken("tok")
	if !sess.isValid() {
		t.Error("expected valid after set")
	}
	sess.clear()
	if sess.isValid() {
		t.Error("expected invalid after clear")
	}
	if sess.getToken() != "" {
		t.Error("expected empty token after clear")
	}
}

func TestIBKRSession_Invalid_AfterExpiry(t *testing.T) {
	sess := &ibkrSession{}
	sess.mu.Lock()
	sess.token = "tok"
	sess.expiresAt = time.Now().Add(-1 * time.Second) // already expired
	sess.mu.Unlock()
	if sess.isValid() {
		t.Error("expected invalid for expired session")
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/trading/brokers/ -run TestIBKRSession -v`
Expected: 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/trading/brokers/ibkr_config.go internal/trading/brokers/ibkr_session.go internal/trading/brokers/ibkr_session_test.go docs/superpowers/plans/2026-07-08-ibkr-broker-adapter.md
git commit -m "feat(brokers): add IBKR config types and session management

Core session validation + refresh goroutine for IBKR Client Portal API.
Part of IBKR broker adapter (spec: docs/specs/2026-07-08-ibkr-broker-adapter.md)."
```

---

### Task 2: Broker Core Implementation

**Files:**
- Create: `internal/trading/brokers/ibkr.go`

**Interfaces:**
- Consumes: `IBKRConfig` from Task 1, `ibkrSession` from Task 1, `trading.Broker` / `trading.Order` / `trading.Position` / `trading.AccountInfo` / `trading.BrokerOrderResult` / `trading.OrderSide` / `trading.OrderType` / `trading.OrderStatus` / `trading.Trade` from `internal/trading`
- Produces: `NewIBKRBroker(cfg IBKRConfig) *IBKRBroker` (factory), `IBKRBroker` implementing all `Broker` interface methods

---

- [ ] **Step 1: Create `ibkr.go`** — broker struct, factory, connection methods, callbacks

```go
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
	cfg      IBKRConfig
	client   *http.Client
	session  ibkrSession
	baseURL  string
	connected bool
	mu        sync.RWMutex
	orderCbs []func(*trading.Order)
	tradeCbs []func(*trading.Trade)
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
```

- [ ] **Step 2: Run test to verify compilation**

Run: `cd app && go build ./internal/trading/brokers/`
Expected: Build succeeds

- [ ] **Step 3: Add SubmitOrder + CancelOrder + ModifyOrder**

Append to `ibkr.go`:

```go
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
		"orderId":    orderID,
		"totalQuantity": fmt.Sprintf("%.6f", newQty),
		"limitPrice":    fmt.Sprintf("%.2f", newPrice),
	}
	_, err := b.doJSONRequest(ctx, http.MethodPost, fmt.Sprintf("/iserver/account/%s/order/%s", b.cfg.AccountID, orderID), body)
	if err != nil {
		return fmt.Errorf("ibkr modify order: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Add GetOrders + GetPositions + GetAccount**

Append to `ibkr.go`:

```go
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
		OrderID     int     `json:"orderId"`
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		OrderType   string  `json:"orderType"`
		Quantity    float64 `json:"quantity"`
		LimitPrice  float64 `json:"limitPrice"`
		StopPrice   float64 `json:"auxPrice"`
		FilledQty   float64 `json:"filledQuantity"`
		AvgPrice    float64 `json:"avgPrice"`
		Status      string  `json:"status"`
		PlacedAt    string  `json:"placedTime"`
		ClientOrderID string `json:"clientOrderId"`
	}
	if err := json.Unmarshal(body, &rawOrders); err != nil {
		return nil, fmt.Errorf("ibkr get orders parse: %w", err)
	}

	orders := make([]*trading.Order, 0, len(rawOrders))
	for _, ro := range rawOrders {
		o := &trading.Order{
			ID:            fmt.Sprintf("%d", ro.OrderID),
			ClientOrderID: ro.ClientOrderID,
			Symbol:        ro.Symbol,
			Side:          trading.OrderSide(ro.Side),
			OrderType:     ibkrTypeToOrderType(ro.OrderType),
			Quantity:      ro.Quantity,
			Price:         ro.LimitPrice,
			StopPrice:     ro.StopPrice,
			FilledQty:     ro.FilledQty,
			FilledAvgPrice: ro.AvgPrice,
			Status:        ibkrOrderStatus(ro.Status),
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
```

- [ ] **Step 5: Add helper types + functions to `ibkr.go`**

Append to `ibkr.go`:

```go
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
	Value        float64 `json:"value"`
	ValueString  string  `json:"valueString"`
	Currency     string  `json:"currency"`
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
```

- [ ] **Step 6: Run test to verify compilation**

Run: `cd app && go build ./internal/trading/brokers/`
Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add internal/trading/brokers/ibkr.go
git commit -m "feat(brokers): implement IBKRBroker with all Broker interface methods

Connect/Disconnect/SubmitOrder/CancelOrder/ModifyOrder/GetOrders/
GetPositions/GetAccount/OnOrderUpdate/OnTradeUpdate for IBKR
Client Portal REST API.
Part of IBKR broker adapter (spec: docs/specs/2026-07-08-ibkr-broker-adapter.md)."
```

---

### Task 3: Broker Tests

**Files:**
- Create: `internal/trading/brokers/ibkr_test.go`

**Interfaces:**
- Consumes: `NewIBKRBroker`, `IBKRBroker`, `IBKRConfig` from Task 1+2
- Produces: Test verification of all Broker interface methods

---

- [ ] **Step 1: Create `ibkr_test.go`** — mock server + tests

```go
package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quantflow/internal/trading"
)

func setupIBKRTestServer() (*httptest.Server, *IBKRBroker) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		method := r.Method

		switch {
		case path == "/sso/validate":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"authenticated": true,
				"token":         "sess-tok",
				"expires":       1800,
			})

		case path == "/logout":
			w.WriteHeader(http.StatusOK)

		case path == fmt.Sprintf("/iserver/%s/orders", mockAccountID):
			switch method {
			case http.MethodGet:
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"orderId": 1001, "symbol": "AAPL", "side": "BUY", "orderType": "MKT",
						"quantity": 100.0, "filledQuantity": 100.0, "avgPrice": 198.50,
						"status": "Filled", "placedTime": "2026-07-08T10:00:00Z"},
					{"orderId": 1002, "symbol": "TSLA", "side": "SELL", "orderType": "LMT",
						"quantity": 50.0, "limitPrice": 250.00, "filledQuantity": 0,
						"status": "Submitted", "placedTime": "2026-07-08T11:00:00Z"},
				})
			case http.MethodPost:
				var reply struct {
					ID     string `json:"id"`
					Status string `json:"order_status"`
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id": "ord-ibkr-001", "order_status": "Submitted",
				})
			}

		case strings.HasPrefix(path, fmt.Sprintf("/iserver/%s/order/", mockAccountID)):
			if method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
			} else if method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "Modified"})
			}

		case path == fmt.Sprintf("/portfolio/%s/positions/0", mockAccountID):
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"symbol": "AAPL", "position": 100.0, "avgCost": 190.25,
					"marketPrice": 198.50, "unrealizedPnl": 825.00, "unrealizedPnlPerc": 4.34},
				{"symbol": "MSFT", "position": 50.0, "avgCost": 420.00,
					"marketPrice": 435.75, "unrealizedPnl": 787.50, "unrealizedPnlPerc": 3.75},
			})

		case path == fmt.Sprintf("/portfolio/%s/summary", mockAccountID):
			json.NewEncoder(w).Encode(map[string]ibkrAccountSummary{
				"TotalCashValue": {Value: 25000.50, Currency: "USD"},
				"CashBalance":    {Value: 25000.50, Currency: "USD"},
				"BuyingPower":    {Value: 100000.00, Currency: "USD"},
				"Currency":       {ValueString: "USD", Currency: "USD"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	}))

	broker := NewIBKRBroker(IBKRConfig{
		Host:      "localhost",
		Port:      0, // will be overridden below
		AccountID: mockAccountID,
	})
	broker.baseURL = server.URL
	broker.client = server.Client()
	return server, broker
}

const mockAccountID = "U1234567"

func TestIBKRBroker_Name(t *testing.T) {
	b := NewIBKRBroker(IBKRConfig{})
	if b.Name() != "ibkr" {
		t.Errorf("Name() = %q, want %q", b.Name(), "ibkr")
	}
}

func TestIBKRBroker_Connect(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	if err := broker.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	if !broker.IsConnected() {
		t.Error("expected connected after Connect()")
	}
}

func TestIBKRBroker_Connect_NoAccountID(t *testing.T) {
	b := NewIBKRBroker(IBKRConfig{})
	if err := b.Connect(context.Background()); err == nil {
		t.Error("expected error when AccountID not configured")
	}
}

func TestIBKRBroker_Disconnect(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	broker.Connect(context.Background())
	broker.Disconnect(context.Background())
	if broker.IsConnected() {
		t.Error("expected disconnected after Disconnect()")
	}
}

func TestIBKRBroker_SubmitOrder_Market(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeMarket, Quantity: 100,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
	if result.Status != trading.StatusPending {
		t.Errorf("Status = %q, want Pending", result.Status)
	}
}

func TestIBKRBroker_SubmitOrder_Limit(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeLimit,
		Quantity: 100, Price: 195.50,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
}

func TestIBKRBroker_SubmitOrder_Stop(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	result, err := broker.SubmitOrder(ctx, &trading.Order{
		Symbol: "AAPL", Side: trading.SideBuy, OrderType: trading.TypeStop,
		Quantity: 100, StopPrice: 190.00,
	})
	if err != nil {
		t.Fatalf("SubmitOrder() error: %v", err)
	}
	if result.BrokerOrderID != "ord-ibkr-001" {
		t.Errorf("BrokerOrderID = %q, want ord-ibkr-001", result.BrokerOrderID)
	}
}

func TestIBKRBroker_CancelOrder(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	if err := broker.CancelOrder(ctx, "1001"); err != nil {
		t.Errorf("CancelOrder() error: %v", err)
	}
}

func TestIBKRBroker_ModifyOrder(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	if err := broker.ModifyOrder(ctx, "1001", 200.00, 150); err != nil {
		t.Errorf("ModifyOrder() error: %v", err)
	}
}

func TestIBKRBroker_GetOrders(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	orders, err := broker.GetOrders(ctx)
	if err != nil {
		t.Fatalf("GetOrders() error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}
	if orders[0].Symbol != "AAPL" {
		t.Errorf("orders[0].Symbol = %q, want AAPL", orders[0].Symbol)
	}
	if orders[0].Status != trading.StatusFilled {
		t.Errorf("orders[0].Status = %q, want Filled", orders[0].Status)
	}
}

func TestIBKRBroker_GetPositions(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	positions, err := broker.GetPositions(ctx)
	if err != nil {
		t.Fatalf("GetPositions() error: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}
	if positions[0].Symbol != "AAPL" {
		t.Errorf("positions[0].Symbol = %q, want AAPL", positions[0].Symbol)
	}
}

func TestIBKRBroker_GetAccount(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()
	ctx := context.Background()
	broker.Connect(ctx)

	acc, err := broker.GetAccount(ctx)
	if err != nil {
		t.Fatalf("GetAccount() error: %v", err)
	}
	if acc.CashBalance != 25000.50 {
		t.Errorf("CashBalance = %v, want 25000.50", acc.CashBalance)
	}
	if acc.BuyingPower != 100000.00 {
		t.Errorf("BuyingPower = %v, want 100000.00", acc.BuyingPower)
	}
}

func TestIBKRBroker_NotConnected_ReturnsError(t *testing.T) {
	broker := NewIBKRBroker(IBKRConfig{AccountID: "test"})

	if _, err := broker.SubmitOrder(context.Background(), &trading.Order{}); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetOrders(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetPositions(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
	if _, err := broker.GetAccount(context.Background()); err == nil {
		t.Error("expected error when not connected")
	}
}

func TestIBKRBroker_Callbacks(t *testing.T) {
	server, broker := setupIBKRTestServer()
	defer server.Close()

	var called int
	broker.OnOrderUpdate(func(o *trading.Order) { called++ })
	broker.OnTradeUpdate(func(tr *trading.Trade) { called++ })
	if called != 0 {
		t.Error("callbacks should not fire on registration")
	}
}

func TestIBKROrderStatus_Mapping(t *testing.T) {
	tests := []struct {
		ibkr string
		want trading.OrderStatus
	}{
		{"Submitted", trading.StatusPending},
		{"PreSubmitted", trading.StatusPending},
		{"Filled", trading.StatusFilled},
		{"Cancelled", trading.StatusCancelled},
		{"ApiCancelled", trading.StatusCancelled},
		{"Inactive", trading.StatusPending},
		{"Unknown", trading.StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.ibkr, func(t *testing.T) {
			if got := ibkrOrderStatus(tt.ibkr); got != tt.want {
				t.Errorf("ibkrOrderStatus(%q) = %q, want %q", tt.ibkr, got, tt.want)
			}
		})
	}
}

func TestIBKROrderType_Mapping(t *testing.T) {
	tests := []struct {
		input trading.OrderType
		want  string
	}{
		{trading.TypeMarket, "MKT"},
		{trading.TypeLimit, "LMT"},
		{trading.TypeStop, "STP"},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if got := ibkrOrderType(tt.input); got != tt.want {
				t.Errorf("ibkrOrderType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `cd app && go test ./internal/trading/brokers/ -run TestIBKR -v`
Expected: All 17 IBKR tests PASS

- [ ] **Step 3: Run full test suite**

Run: `cd app && go test ./internal/trading/brokers/ -v -count=1`
Expected: All tests PASS (Alpaca + Binance + Futu + IBKR)

- [ ] **Step 4: Commit**

```bash
git add internal/trading/brokers/ibkr_test.go
git commit -m "test(brokers): add comprehensive IBKR broker tests

HTTP mock test coverage for all Broker interface methods,
order status/type mapping, and error paths.
Part of IBKR broker adapter (spec: docs/specs/2026-07-08-ibkr-broker-adapter.md)."
```

---

### Task 4: App Registration + Changelog

**Files:**
- Modify: `app_startup.go` (add IBKR broker registration)
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: `NewIBKRBroker(cfg IBKRConfig) *IBKRBroker` from Task 2

---

- [ ] **Step 1: Read app_startup.go around Alpaca registration**

Read lines 220-240 to see the exact pattern to follow.

- [ ] **Step 2: Add IBKR broker registration after Alpaca in `app_startup.go`**

Replace the comment block after the Alpaca broker section (around line 237) to include:

```go
	// Phase 5: Initialize broker adapters. IBKR is optional — when AccountID
	// is not set, the broker stays disconnected and panels show "Not Configured".
	if ibkrCfg := brokers.IBKRConfig{
		Host:      os.Getenv("IBKR_HOST"),
		Port:      5000,
		AccountID: os.Getenv("IBKR_ACCOUNT_ID"),
	}; ibkrCfg.AccountID != "" {
		ibkrBroker := brokers.NewIBKRBroker(ibkrCfg)
		if err := ibkrBroker.Connect(context.Background()); err != nil {
			slog.Warn("ibkr broker not available — IBKR trading disabled", "error", err)
		} else {
			a.oms.SetBroker(ibkrBroker)
			slog.Info("ibkr broker connected — IBKR trading enabled")
		}
	}
```

**Important:** Find the exact location in the file and make the edit. The pattern to follow is the existing Alpaca registration at `app_startup.go:230-237`. Insert the IBKR block right after the Alpaca block.

The `os` import should already be in app_startup.go's imports (since it's used elsewhere for ALPACA_API_KEY).

- [ ] **Step 3: Verify compilation**

Run: `cd app && go build ./...`
Expected: Build succeeds

- [ ] **Step 4: Update CHANGELOG.md**

Read the current CHANGELOG.md header to find the latest version and date. Add entry:

```markdown
## [2026.7.8] - 2026-07-08

### Added
- [Broker] IBKR broker adapter via Client Portal REST API — Connect/Disconnect/SubmitOrder/
  CancelOrder/ModifyOrder/GetOrders/GetPositions/GetAccount, session management with
  4-minute auto-refresh, three order types (market/limit/stop), HTTP mock test suite
```

- [ ] **Step 5: Update version date if needed**

Check `frontend/package.json` version field and `README.md` version badge. Update to `2026.7.8` if stale.

- [ ] **Step 6: Run full check**

```bash
cd app && go vet ./... && go test ./internal/trading/brokers/ -v -count=1
```

Expected: `go vet` clean, all tests PASS

- [ ] **Step 7: Commit**

```bash
git add app_startup.go CHANGELOG.md frontend/package.json README.md
git commit -m "feat: register IBKR broker at startup and update changelog

IBKR broker is configured via IBKR_HOST and IBKR_ACCOUNT_ID env vars.
Optional — IBKR trading gracefully disabled when not configured.
Part of IBKR broker adapter (spec: docs/specs/2026-07-08-ibkr-broker-adapter.md)."
```
