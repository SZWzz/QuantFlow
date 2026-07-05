// Package brokers provides real broker adapter implementations.
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
	oms       *trading.OMS // wired via SetOMS to forward state changes to callbacks
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

// Name returns the broker identifier.
func (b *BinanceBroker) Name() string { return "binance" }

// Connect verifies connectivity to the Binance REST API.
func (b *BinanceBroker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	resp, err := b.client.Get(b.cfg.BaseURL + "/api/v3/time")
	if err != nil {
		return fmt.Errorf("binance connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance connect: status %d", resp.StatusCode)
	}
	b.connected = true
	slog.Info("binance broker connected", "base_url", b.cfg.BaseURL)
	return nil
}

// Disconnect marks the broker as disconnected.
func (b *BinanceBroker) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = false
	return nil
}

// IsConnected returns whether the broker is currently connected.
func (b *BinanceBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// SubmitOrder sends an order to Binance REST API.
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

// CancelOrder cancels an order on Binance.
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

// ModifyOrder is not supported by Binance — cancel and re-submit instead.
func (b *BinanceBroker) ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error {
	return fmt.Errorf("binance does not support direct order modification; cancel and re-submit required")
}

// GetOrders retrieves open orders from Binance.
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

// GetPositions returns balances as positions from the Binance account endpoint.
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

// GetAccount returns the account summary from Binance.
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

// SetOMS wires the broker to the OMS so that order/trade state changes
// in the OMS are forwarded to registered broker callbacks (orderCbs/tradeCbs).
func (b *BinanceBroker) SetOMS(oms *trading.OMS) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.oms = oms
}

// OnOrderUpdate registers a callback for order state changes.
// Delegates to the OMS so callbacks fire on every FillOrder/CancelOrder.
func (b *BinanceBroker) OnOrderUpdate(fn func(*trading.Order)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.orderCbs = append(b.orderCbs, fn)
	if b.oms != nil {
		b.oms.OnOrderUpdate(fn)
	}
}

// OnTradeUpdate registers a callback for trade executions.
// Delegates to the OMS so callbacks fire on every fill.
func (b *BinanceBroker) OnTradeUpdate(fn func(*trading.Trade)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tradeCbs = append(b.tradeCbs, fn)
	if b.oms != nil {
		b.oms.OnTradeUpdate(fn)
	}
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
	// Only treat a known quote suffix as an existing pair when there is
	// a non-empty base asset before it (e.g. ETHUSDT, LINKBTC).
	hasQuote := false
	for _, quote := range []string{"USDT", "BTC", "ETH"} {
		if strings.HasSuffix(symbol, quote) && len(symbol) > len(quote) {
			hasQuote = true
			break
		}
	}
	if !hasQuote {
		symbol = symbol + "USDT"
	}
	return strings.ToUpper(symbol)
}

func binanceOrderType(ot trading.OrderType) string {
	switch ot {
	case trading.TypeMarket:
		return "MARKET"
	case trading.TypeLimit:
		return "LIMIT"
	case trading.TypeStop:
		return "STOP_LOSS_LIMIT"
	default:
		return "MARKET"
	}
}

func binanceTypeToOrderType(t string) trading.OrderType {
	switch strings.ToUpper(t) {
	case "MARKET":
		return trading.TypeMarket
	case "LIMIT":
		return trading.TypeLimit
	case "STOP_LOSS_LIMIT":
		return trading.TypeStop
	default:
		return trading.TypeMarket
	}
}

func binanceStatusToOrderStatus(s string) trading.OrderStatus {
	switch strings.ToUpper(s) {
	case "NEW":
		return trading.StatusPending
	case "PARTIALLY_FILLED":
		return trading.StatusPartial
	case "FILLED":
		return trading.StatusFilled
	case "CANCELED":
		return trading.StatusCancelled
	case "REJECTED", "EXPIRED":
		return trading.StatusRejected
	default:
		return trading.StatusPending
	}
}
