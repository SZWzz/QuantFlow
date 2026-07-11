package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"

	"quantflow/internal/market"
	"quantflow/internal/market/wsconn"
)

const binanceWSBaseURL = "wss://stream.binance.com:9443/ws"

// Compile-time check: BinanceAdapter implements WSConnector.
var _ wsconn.WSConnector = (*BinanceAdapter)(nil)

// binanceWSEvent is the Binance WebSocket ticker event format.
type binanceWSEvent struct {
	EventType       string `json:"e"` // "24hrTicker"
	EventTime       int64  `json:"E"`
	Symbol          string `json:"s"`
	PriceChange     string `json:"p"`
	PriceChangePct  string `json:"P"`
	WeightedAvgPrice string `json:"w"`
	LastPrice       string `json:"c"`
	LastQty         string `json:"Q"`
	BidPrice        string `json:"b"`
	BidQty          string `json:"B"`
	AskPrice        string `json:"a"`
	AskQty          string `json:"A"`
	Open            string `json:"o"`
	High            string `json:"h"`
	Low             string `json:"l"`
	Volume          string `json:"v"`
	QuoteVolume     string `json:"q"`
	OpenTime        int64  `json:"O"`
	CloseTime       int64  `json:"C"`
	TradeCount      int64  `json:"n"`
}

// binanceWSState holds the mutable WebSocket connection state.
type binanceWSState struct {
	conn     *websocket.Conn
	cancel   context.CancelFunc
	hub      *market.MarketDataHub
	subsMu   sync.Mutex
	subs     map[string]bool // subscribed symbols (uppercase)
	reconnMu sync.Mutex
	reconn   int // reconnect attempt counter
}

// ConnectWS connects to Binance WebSocket and begins pushing ticker data.
func (a *BinanceAdapter) ConnectWS(ctx context.Context, hub *market.MarketDataHub) error {
	state := &binanceWSState{
		hub:  hub,
		subs: make(map[string]bool),
	}

	wsCtx, cancel := context.WithCancel(ctx)
	state.cancel = cancel

	if err := state.connect(wsCtx); err != nil {
		cancel()
		return fmt.Errorf("binance ws: initial connect: %w", err)
	}

	go state.readLoop(wsCtx)
	go state.heartbeatLoop(wsCtx)

	return nil
}

// DisconnectWS closes the Binance WebSocket connection.
func (a *BinanceAdapter) DisconnectWS() error {
	// State is managed externally by wsconn.Manager; no-op here.
	// The Manager cancels the context which stops the goroutines.
	return nil
}

// SupportsWS returns true.
func (a *BinanceAdapter) SupportsWS() bool { return true }

// ExchangeName returns "binance".
func (a *BinanceAdapter) ExchangeName() string { return "binance" }

func (s *binanceWSState) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, binanceWSBaseURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	s.conn = conn
	s.reconn = 0
	return nil
}

func (s *binanceWSState) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, msg, err := s.conn.Read(ctx)
		if err != nil {
			slog.Warn("binance ws: read error, reconnecting", "error", err)
			if !s.reconnect(ctx) {
				return
			}
			continue
		}

		s.handleMessage(msg)
	}
}

func (s *binanceWSState) handleMessage(msg []byte) {
	var event binanceWSEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		// Not a ticker event; skip
		return
	}

	if event.EventType != "24hrTicker" {
		return
	}

	last, _ := strconv.ParseFloat(event.LastPrice, 64)
	open, _ := strconv.ParseFloat(event.Open, 64)
	high, _ := strconv.ParseFloat(event.High, 64)
	low, _ := strconv.ParseFloat(event.Low, 64)
	volume, _ := strconv.ParseFloat(event.Volume, 64)
	change, _ := strconv.ParseFloat(event.PriceChange, 64)
	changePct, _ := strconv.ParseFloat(event.PriceChangePct, 64)
	bid, _ := strconv.ParseFloat(event.BidPrice, 64)
	ask, _ := strconv.ParseFloat(event.AskPrice, 64)

	snapshot := &market.QuoteSnapshot{
		Symbol:    event.Symbol,
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Volume:    volume,
		Change:    change,
		ChangePct: changePct,
		Bid:       bid,
		Ask:       ask,
		Timestamp: event.EventTime,
	}

	topic := fmt.Sprintf("market:quote:CRYPTO:%s", event.Symbol)
	s.hub.Publish(topic, snapshot)
}

func (s *binanceWSState) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.conn == nil {
				continue
			}
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := s.conn.Ping(pingCtx); err != nil {
				slog.Warn("binance ws: ping failed, reconnecting", "error", err)
				cancel()
				if !s.reconnect(ctx) {
					return
				}
			}
			cancel()
		}
	}
}

func (s *binanceWSState) reconnect(ctx context.Context) bool {
	s.reconnMu.Lock()
	attempt := s.reconn
	s.reconn++
	s.reconnMu.Unlock()

	delay := time.Duration(1<<min(attempt, 4)) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	slog.Info("binance ws: reconnecting", "attempt", attempt+1, "delay", delay)

	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
	}

	conn, _, err := websocket.Dial(ctx, binanceWSBaseURL, nil)
	if err != nil {
		slog.Warn("binance ws: reconnect failed", "error", err)
		return false
	}

	// Close old connection
	if s.conn != nil {
		s.conn.Close(websocket.StatusNormalClosure, "reconnect")
	}
	s.conn = conn

	// Re-subscribe to previously subscribed symbols
	s.subsMu.Lock()
	var symbols []string
	for sym := range s.subs {
		symbols = append(symbols, sym)
	}
	s.subsMu.Unlock()

	for _, sym := range symbols {
		subMsg := map[string]interface{}{
			"method": "SUBSCRIBE",
			"params": []string{fmt.Sprintf("%s@ticker", sym)},
			"id":     time.Now().UnixNano(),
		}
		data, _ := json.Marshal(subMsg)
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			slog.Warn("binance ws: resubscribe failed", "symbol", sym, "error", err)
		}
	}

	return true
}

// SubscribeSymbol adds a symbol to the Binance WS subscription.
func (s *binanceWSState) SubscribeSymbol(symbol string) error {
	upper := symbol // Binance symbols are already uppercase
	s.subsMu.Lock()
	if s.subs[upper] {
		s.subsMu.Unlock()
		return nil
	}
	s.subs[upper] = true
	s.subsMu.Unlock()

	subMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": []string{fmt.Sprintf("%s@ticker", upper)},
		"id":     time.Now().UnixNano(),
	}
	data, err := json.Marshal(subMsg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.conn.Write(ctx, websocket.MessageText, data)
}
