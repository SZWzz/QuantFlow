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

const okxWSPublicURL = "wss://ws.okx.com:8443/ws/v5/public"

var _ wsconn.WSConnector = (*OKXAdapter)(nil)

// okxTickerEvent is the OKX WebSocket tickers channel event.
type okxTickerEvent struct {
	Arg  okxChannelArg `json:"arg"`
	Data []okxTickerData `json:"data"`
}

type okxChannelArg struct {
	Channel  string `json:"channel"`
	InstID   string `json:"instId"`
}

type okxTickerData struct {
	InstID    string `json:"instId"`
	Last      string `json:"last"`
	Open24h   string `json:"open24h"`
	High24h   string `json:"high24h"`
	Low24h    string `json:"low24h"`
	Vol24h    string `json:"vol24h"`
	BidPx     string `json:"bidPx"`
	AskPx     string `json:"askPx"`
	SodUtc0   string `json:"sodUtc0"`
	Ts        string `json:"ts"`
}

type okxWSState struct {
	conn     *websocket.Conn
	cancel   context.CancelFunc
	hub      *market.MarketDataHub
	mu       sync.Mutex
	reconn   int
}

// ConnectWS connects to OKX WebSocket public channel.
func (a *OKXAdapter) ConnectWS(ctx context.Context, hub *market.MarketDataHub) error {
	state := &okxWSState{hub: hub}
	wsCtx, cancel := context.WithCancel(ctx)
	state.cancel = cancel

	if err := state.connect(wsCtx); err != nil {
		cancel()
		return fmt.Errorf("okx ws: %w", err)
	}

	// Subscribe to all tickers
	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": []map[string]string{{"channel": "tickers", "instType": "SPOT"}},
	}
	data, _ := json.Marshal(subMsg)
	if err := state.conn.Write(wsCtx, websocket.MessageText, data); err != nil {
		cancel()
		return fmt.Errorf("okx ws: subscribe: %w", err)
	}

	go state.readLoop(wsCtx)
	go state.heartbeatLoop(wsCtx)
	return nil
}

func (a *OKXAdapter) DisconnectWS() error  { return nil }
func (a *OKXAdapter) SupportsWS() bool      { return true }
func (a *OKXAdapter) ExchangeName() string  { return "okx" }

func (s *okxWSState) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, okxWSPublicURL, nil)
	if err != nil {
		return err
	}
	s.conn = conn
	s.reconn = 0
	return nil
}

func (s *okxWSState) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, msg, err := s.conn.Read(ctx)
		if err != nil {
			slog.Warn("okx ws: read error", "error", err)
			return
		}
		s.handleMessage(msg)
	}
}

func (s *okxWSState) handleMessage(msg []byte) {
	var event okxTickerEvent
	if err := json.Unmarshal(msg, &event); err != nil || len(event.Data) == 0 {
		return
	}

	for _, d := range event.Data {
		last, _ := strconv.ParseFloat(d.Last, 64)
		open, _ := strconv.ParseFloat(d.Open24h, 64)
		high, _ := strconv.ParseFloat(d.High24h, 64)
		low, _ := strconv.ParseFloat(d.Low24h, 64)
		vol, _ := strconv.ParseFloat(d.Vol24h, 64)
		bid, _ := strconv.ParseFloat(d.BidPx, 64)
		ask, _ := strconv.ParseFloat(d.AskPx, 64)

		change := last - open
		changePct := 0.0
		if open > 0 {
			changePct = (change / open) * 100
		}

		ts, _ := strconv.ParseInt(d.Ts, 10, 64)

		snapshot := &market.QuoteSnapshot{
			Symbol:    d.InstID,
			Last:      last,
			Open:      open,
			High:      high,
			Low:       low,
			Volume:    vol,
			Bid:       bid,
			Ask:       ask,
			Change:    change,
			ChangePct: changePct,
			Timestamp: ts,
		}

		topic := fmt.Sprintf("market:quote:CRYPTO:%s", d.InstID)
		s.hub.Publish(topic, snapshot)
	}
}

func (s *okxWSState) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.conn != nil {
				// OKX uses "ping" text frame
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				s.conn.Write(pingCtx, websocket.MessageText, []byte("ping"))
				cancel()
			}
		}
	}
}
