package market

import (
	"sync"
	"time"
)

// topicBroker manages subscribers for a single topic and stores the latest cached message.
type topicBroker struct {
	subscribers map[string]chan MarketMessage
	latest      *CachedMessage
	mu          sync.RWMutex
}

func newTopicBroker() *topicBroker {
	return &topicBroker{
		subscribers: make(map[string]chan MarketMessage),
	}
}

func (b *topicBroker) subscribe(subID string) (<-chan MarketMessage, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan MarketMessage, 64)
	b.subscribers[subID] = ch

	// Send cached message if available
	if b.latest != nil && !b.latest.Expired() {
		select {
		case ch <- b.latest.Msg:
		default:
		}
	}

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers, subID)
		// We do NOT close the channel here — there is a known race with
		// publish(), which snapshots the subscriber map under the lock but
		// sends outside the lock. Closing while a send is in flight panics.
		// Instead we rely on the GC to collect the channel after the
		// subscriber stops reading and publish drops its reference.
		// The delete above ensures no future publish will route to this
		// subscriber, so the idle channel simply leaks once per subscription
		// lifecycle (acceptable for a desktop app with bounded subscriptions).
	}

	return ch, unsubscribe
}

func (b *topicBroker) publish(msg MarketMessage) {
	b.mu.Lock()
	b.latest = &CachedMessage{
		Msg:      msg,
		CachedAt: time.Now(),
		TTL:      30 * time.Second,
	}
	subs := make(map[string]chan MarketMessage)
	for id, ch := range b.subscribers {
		subs[id] = ch
	}
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
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

	h.mu.RLock()
	broker, ok := h.topics[topic]
	h.mu.RUnlock()

	if !ok {
		// No subscribers yet; create broker to cache the message
		h.mu.Lock()
		broker = newTopicBroker()
		h.topics[topic] = broker
		h.mu.Unlock()
	}

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
