package market

import (
	"context"
	"testing"
	"time"

	"quantflow/internal/ws"
)

type mockPollAdapter struct {
	name      string
	available bool
	quotes    map[string]*QuoteSnapshot
}

func (m *mockPollAdapter) Name() string                          { return m.name }
func (m *mockPollAdapter) Markets() []string                     { return nil }
func (m *mockPollAdapter) RequiresAuth() bool                    { return false }
func (m *mockPollAdapter) IsAvailable(ctx context.Context) bool   { return m.available }
func (m *mockPollAdapter) HealthCheck(ctx context.Context) error  { return nil }
func (m *mockPollAdapter) FetchQuote(_ context.Context, symbol string) (*QuoteSnapshot, error) {
	q, ok := m.quotes[symbol]
	if !ok {
		return nil, nil
	}
	return q, nil
}
func (m *mockPollAdapter) FetchOHLCV(_ context.Context, _, _, _ string, _, _ int64) ([]OHLCVBar, error) {
	return nil, nil
}

func TestQuotePoller_SubscribeUnsubscribe(t *testing.T) {
	reg := NewAdapterRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, NewHub(), wsHub)
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers initially, got %d", poller.SubscriberCount())
	}

	poller.Subscribe("CN", "600519")
	if poller.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", poller.SubscriberCount())
	}

	poller.Unsubscribe("CN", "600519")
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_DeduplicateSubscriptions(t *testing.T) {
	poller := NewQuotePoller(nil, NewHub(), ws.NewHub())
	poller.Subscribe("CN", "600519")
	poller.Subscribe("CN", "600519")
	if poller.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after dedup, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_SubscribeDifferentSymbols(t *testing.T) {
	poller := NewQuotePoller(nil, NewHub(), ws.NewHub())
	poller.Subscribe("CN", "600519")
	poller.Subscribe("US", "AAPL")
	if poller.SubscriberCount() != 2 {
		t.Fatalf("expected 2 subscribers, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_StopStart(t *testing.T) {
	reg := NewAdapterRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, NewHub(), wsHub)
	poller.interval = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go poller.Run(ctx)

	poller.Subscribe("CN", "600519")
	time.Sleep(30 * time.Millisecond)
	poller.Unsubscribe("CN", "600519")
	time.Sleep(20 * time.Millisecond)

	cancel()
	poller.Stop()
	if poller.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after stop, got %d", poller.SubscriberCount())
	}
}

func TestQuotePoller_FetchesAndPublishesData(t *testing.T) {
	reg := NewAdapterRegistry()
	adapter := &mockPollAdapter{
		name: "test", available: true,
		quotes: map[string]*QuoteSnapshot{
			"600519": {Symbol: "600519", Last: 1800.0, Change: 10.0, ChangePct: 0.56, Volume: 10000, Timestamp: time.Now().UnixMilli()},
		},
	}
	reg.Register(adapter)

	origChain := FallbackChains["CRYPTO"]
	FallbackChains["CRYPTO"] = []string{"test"}
	defer func() { FallbackChains["CRYPTO"] = origChain }()

	marketHub := NewHub()
	wsHub := ws.NewHub()
	go wsHub.Run()

	poller := NewQuotePoller(reg, marketHub, wsHub)
	poller.interval = 10 * time.Millisecond

	// Simulate a frontend WS client subscribing to the topic
	client := ws.NewClient(wsHub, nil)
	wsHub.Subscribe(client, "market:quote:CRYPTO:600519")

	ctx, cancel := context.WithCancel(context.Background())
	go poller.Run(ctx)

	// Poll until data arrives (max 500ms)
	var msg *MarketMessage
	var ok bool
	for i := 0; i < 50; i++ {
		msg, ok = marketHub.GetLatest("market:quote:CRYPTO:600519")
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	poller.Stop()

	if !ok {
		t.Fatal("expected market data hub to have cached message")
	}
	quote, ok := msg.Data.(*QuoteSnapshot)
	if !ok {
		t.Fatalf("expected *QuoteSnapshot, got %T", msg.Data)
	}
	if quote.Last != 1800.0 {
		t.Fatalf("expected Last=1800, got %f", quote.Last)
	}
}
