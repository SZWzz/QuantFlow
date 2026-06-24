// Package market provides the MarketDataHub — a Go channel-based pub/sub system
// for real-time market data distribution with L0/L1/L2 caching.
package market

import "time"

// QuoteSnapshot is a real-time quote for a single symbol.
type QuoteSnapshot struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Last      float64 `json:"last"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Volume    float64 `json:"volume"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"change_pct"`
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
