package adapters

import (
	"math/rand"
	"sync"
	"time"
)

// EastMoneyRateLimiter enforces rate limits for all EastMoney API endpoints.
// Based on community-measured thresholds (2026-05): >5 req/s → ban, ≥10 concurrent → ban.
// All eastmoney.com requests must go through this limiter to avoid IP blocking.
//
// Usage: GlobalEMLimiter.Wait() before every EastMoney HTTP request.
type EastMoneyRateLimiter struct {
	mu       sync.Mutex
	lastCall time.Time
	minGap   time.Duration
}

// GlobalEMLimiter is the shared rate limiter for all EastMoney adapters.
var GlobalEMLimiter = &EastMoneyRateLimiter{
	minGap: 500 * time.Millisecond,
}

// Wait blocks until it's safe to make the next EastMoney request.
// Adds random jitter (0-200ms) on top of the minimum gap.
func (r *EastMoneyRateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	elapsed := time.Since(r.lastCall)
	jitter := time.Duration(rand.Intn(200)) * time.Millisecond //nolint:gosec // 限流抖动非安全用途，math/rand 足够
	wait := r.minGap - elapsed + jitter
	if wait > 0 {
		time.Sleep(wait)
	}
	r.lastCall = time.Now()
}
