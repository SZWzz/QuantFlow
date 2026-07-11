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

const gateWSURL = "wss://api.gateio.ws/ws/v4/"

var _ wsconn.WSConnector = (*GateIOAdapter)(nil)

// gateTickerEvent is the Gate.io WebSocket spot.tickers event.
type gateTickerEvent struct {
	Time    int64            `json:"time"`
	Channel string           `json:"channel"`
	Event   string           `json:"event"`
	Result  *gateTickerResult `json:"result"`
}

type gateTickerResult struct {
	CurrencyPair     string `json:"currency_pair"`
	Last             string `json:"last"`
	LowestAsk        string `json:"lowest_ask"`
	HighestBid       string `json:"highest_bid"`
	ChangePercentage string `json:"change_percentage"`
	BaseVolume       string `json:"base_volume"`
	QuoteVolume      string `json:"quote_volume"`
	High24h          string `json:"high_24h"`
	Low24h           string `json:"low_24h"`
	Open24h          string `json:"open_24h"`
}

type gateWSState struct {
	conn     *websocket.Conn
	cancel   context.CancelFunc
	hub      *market.MarketDataHub
	mu       sync.Mutex
	reconn   int
}

// ConnectWS connects to Gate.io WebSocket public channel.
func (a *GateIOAdapter) ConnectWS(ctx context.Context, hub *market.MarketDataHub) error {
	state := &gateWSState{hub: hub}
	wsCtx, cancel := context.WithCancel(ctx)
	state.cancel = cancel

	if err := state.connect(wsCtx); err != nil {
		cancel()
		return fmt.Errorf("gate ws: %w", err)
	}

	// Subscribe to spot tickers
	subMsg := map[string]interface{}{
		"time":    time.Now().Unix(),
		"channel": "spot.tickers",
		"event":   "subscribe",
	}
	data, _ := json.Marshal(subMsg)
	if err := state.conn.Write(wsCtx, websocket.MessageText, data); err != nil {
		cancel()
		return fmt.Errorf("gate ws: subscribe: %w", err)
	}

	go state.readLoop(wsCtx)
	go state.heartbeatLoop(wsCtx)
	return nil
}

func (a *GateIOAdapter) DisconnectWS() error  { return nil }
func (a *GateIOAdapter) SupportsWS() bool      { return true }
func (a *GateIOAdapter) ExchangeName() string  { return "gateio" }

func (s *gateWSState) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, gateWSURL, nil)
	if err != nil {
		return err
	}
	s.conn = conn
	s.reconn = 0
	return nil
}

func (s *gateWSState) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, msg, err := s.conn.Read(ctx)
		if err != nil {
			slog.Warn("gate ws: read error", "error", err)
			return
		}
		s.handleMessage(msg)
	}
}

func (s *gateWSState) handleMessage(msg []byte) {
	var event gateTickerEvent
	if err := json.Unmarshal(msg, &event); err != nil || event.Result == nil {
		return
	}

	r := event.Result
	last, _ := strconv.ParseFloat(r.Last, 64)
	open, _ := strconv.ParseFloat(r.Open24h, 64)
	high, _ := strconv.ParseFloat(r.High24h, 64)
	low, _ := strconv.ParseFloat(r.Low24h, 64)
	vol, _ := strconv.ParseFloat(r.BaseVolume, 64)
	bid, _ := strconv.ParseFloat(r.HighestBid, 64)
	ask, _ := strconv.ParseFloat(r.LowestAsk, 64)
	changePct, _ := strconv.ParseFloat(r.ChangePercentage, 64)

	change := last - open

	// Gate.io uses underscore separator (BTC_USDT), convert to standard format
	symbol := r.CurrencyPair

	snapshot := &market.QuoteSnapshot{
		Symbol:    symbol,
		Last:      last,
		Open:      open,
		High:      high,
		Low:       low,
		Volume:    vol,
		Bid:       bid,
		Ask:       ask,
		Change:    change,
		ChangePct: changePct,
		Timestamp: event.Time,
	}

	topic := fmt.Sprintf("market:quote:CRYPTO:%s", symbol)
	s.hub.Publish(topic, snapshot)
}

func (s *gateWSState) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.conn != nil {
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				if err := s.conn.Ping(pingCtx); err != nil {
					slog.Warn("gate ws: ping failed", "error", err)
				}
				cancel()
			}
		}
	}
}
