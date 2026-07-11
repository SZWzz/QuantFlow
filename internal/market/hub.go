package market

import (
	"sync"
	"sync/atomic"
	"time"
)

// subscriber wraps a message channel with a closed flag to safely
// prevent send-on-closed-channel panics during concurrent unsubscribe+publish.
type subscriber struct {
	ch     chan MarketMessage
	closed atomic.Bool
}

func newSubscriber() *subscriber {
	return &subscriber{ch: make(chan MarketMessage, 64)}
}

func (s *subscriber) close() {
	s.closed.Store(true)
	// Channel is intentionally NOT closed — closing would cause in-flight
	// publish() sends to panic. The closed flag prevents future sends,
	// and the GC collects the channel when all references are dropped.
}

// topicBroker manages subscribers for a single topic and stores the latest cached message.
type topicBroker struct {
	subscribers map[string]*subscriber
	latest      *CachedMessage
	mu          sync.RWMutex
}

func newTopicBroker() *topicBroker {
	return &topicBroker{
		subscribers: make(map[string]*subscriber),
	}
}

func (b *topicBroker) subscribe(subID string) (<-chan MarketMessage, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := newSubscriber()
	b.subscribers[subID] = sub

	// Send cached message if available
	if b.latest != nil && !b.latest.Expired() {
		select {
		case sub.ch <- b.latest.Msg:
		default:
		}
	}

	unsubscribe := func() {
		b.mu.Lock()
		delete(b.subscribers, subID)
		b.mu.Unlock()
		sub.close()
	}

	return sub.ch, unsubscribe
}

func (b *topicBroker) publish(msg MarketMessage) {
	b.mu.Lock()
	b.latest = &CachedMessage{
		Msg:      msg,
		CachedAt: time.Now(),
		TTL:      30 * time.Second,
	}
	subs := make(map[string]*subscriber)
	for id, s := range b.subscribers {
		subs[id] = s
	}
	b.mu.Unlock()

	for _, s := range subs {
		if s.closed.Load() {
			continue
		}
		select {
		case s.ch <- msg:
		default:
			// Slow consumer — drop message to avoid blocking
		}
	}
}

func (b *topicBroker) subscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// MarketDataProvider defines the public contract for market data pub/sub.
// Implementations include MarketDataHub (production) and mock stubs for tests.
type MarketDataProvider interface {
	Subscribe(topic, subID string) (<-chan MarketMessage, func())
	Publish(topic string, data any)
	GetLatest(topic string) (*MarketMessage, bool)
	SubscriberCount() int
	TopicCount() int
}

var _ MarketDataProvider = (*MarketDataHub)(nil)

// MarketDataHub is a Go channel-based publish/subscribe system for real-time market data.
// Each topic has its own broker with cached latest message and TTL.
type MarketDataHub struct {
	topics map[string]*topicBroker
	mu     sync.RWMutex
}

// NewHub creates a new MarketDataHub.
func NewHub() *MarketDataHub {
	return &MarketDataHub{
		topics: make(map[string]*topicBroker),
	}
}

// Subscribe subscribes to a topic. Returns a receive-only channel and an unsubscribe function.
// Topic format: "market:quote:AAPL", "market:ohlcv:AAPL:1d", etc.
func (h *MarketDataHub) Subscribe(topic, subID string) (<-chan MarketMessage, func()) {
	h.mu.Lock()
	broker, ok := h.topics[topic]
	if !ok {
		broker = newTopicBroker()
		h.topics[topic] = broker
	}
	h.mu.Unlock()

	return broker.subscribe(subID)
}

// Publish publishes a message to a topic. All subscribers receive the message.
func (h *MarketDataHub) Publish(topic string, data any) {
	msg := MarketMessage{
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
	}

	h.mu.Lock()
	broker, ok := h.topics[topic]
	if !ok {
		broker = newTopicBroker()
		h.topics[topic] = broker
	}
	h.mu.Unlock()

	broker.publish(msg)
}

// GetLatest returns the latest cached message for a topic, if not expired.
func (h *MarketDataHub) GetLatest(topic string) (*MarketMessage, bool) {
	h.mu.RLock()
	broker, ok := h.topics[topic]
	h.mu.RUnlock()

	if !ok {
		return nil, false
	}

	broker.mu.RLock()
	defer broker.mu.RUnlock()

	if broker.latest != nil && !broker.latest.Expired() {
		return &broker.latest.Msg, true
	}
	return nil, false
}

// SubscriberCount returns the total number of subscribers across all topics.
func (h *MarketDataHub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, broker := range h.topics {
		count += broker.subscriberCount()
	}
	return count
}

// TopicCount returns the number of active topics.
func (h *MarketDataHub) TopicCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.topics)
}
