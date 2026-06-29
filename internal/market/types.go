// Package market provides the MarketDataHub — a Go channel-based pub/sub system
// for real-time market data distribution with L0/L1/L2 caching.
package market

import "time"

// DepthLevel represents a single price level in the order book.
type DepthLevel struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// DepthSnapshot represents the order book depth for a symbol.
// Bids are sorted highest-first; Asks sorted lowest-first.
type DepthSnapshot struct {
	Symbol    string       `json:"symbol"`
	Bids      []DepthLevel `json:"bids"`
	Asks      []DepthLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
}

// QuoteSnapshot is a real-time quote for a single symbol.
type QuoteSnapshot struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Last      float64 `json:"last"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	PrevClose float64 `json:"prevClose"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"change_pct"`
	MarketCap float64 `json:"marketCap"`
	Pe        float64 `json:"pe"`
	Exchange  string  `json:"exchange"`
	Timestamp int64   `json:"timestamp"`
}

// OHLCVBar is a price bar.
type OHLCVBar struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// MarketMessage wraps data published to a topic.
type MarketMessage struct {
	Topic     string    `json:"topic"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// CachedMessage holds a message with TTL metadata.
type CachedMessage struct {
	Msg       MarketMessage
	CachedAt  time.Time
	TTL       time.Duration
}

// Expired returns true if the cached message has exceeded its TTL.
func (c *CachedMessage) Expired() bool {
	return time.Since(c.CachedAt) > c.TTL
}
