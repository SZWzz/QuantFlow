package trading

import "sync"

// T1Tracker tracks shares locked by T+1 settlement rule.
// Shares bought today cannot be sold until the next trading day.
// T+1 applies to A-share markets only.
type T1Tracker struct {
	mu     sync.Mutex
	locked map[string]float64 // symbol → locked quantity from today's buys
}

func NewT1Tracker() *T1Tracker {
	return &T1Tracker{locked: make(map[string]float64)}
}

func (t *T1Tracker) Lock(symbol string, qty float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locked[symbol] += qty
}

func (t *T1Tracker) Available(symbol string, totalQty float64) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	locked := t.locked[symbol]
	avail := totalQty - locked
	if avail <= 0 {
		return 0
	}
	return avail
}

func (t *T1Tracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locked = make(map[string]float64)
}
