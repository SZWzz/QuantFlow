package trading

import (
	"fmt"
	"sync"
	"time"
)

// CashLedger tracks cash balance with an immutable audit trail.
type CashLedger struct {
	mu      sync.RWMutex
	balance float64
	entries []CashEntry
}

// CashEntry is a single row in the cash ledger.
type CashEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`    // "deposit", "withdraw", "buy", "sell", "dividend", "fee"
	Amount    float64   `json:"amount"`  // positive = increase, negative = decrease
	Balance   float64   `json:"balance"` // running balance after this entry
	OrderID   string    `json:"order_id,omitempty"`
}

// NewCashLedger creates a new CashLedger with zero balance.
func NewCashLedger() *CashLedger {
	return &CashLedger{}
}

// Deposit adds funds to the cash ledger.
func (cl *CashLedger) Deposit(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive, got %.2f", amount)
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.balance += amount
	cl.entries = append(cl.entries, CashEntry{
		Timestamp: time.Now(),
		Type:      "deposit",
		Amount:    amount,
		Balance:   cl.balance,
	})
	return nil
}

// Withdraw removes funds from the cash ledger.
func (cl *CashLedger) Withdraw(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("withdraw amount must be positive, got %.2f", amount)
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if amount > cl.balance {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", cl.balance, amount)
	}
	cl.balance -= amount
	cl.entries = append(cl.entries, CashEntry{
		Timestamp: time.Now(),
		Type:      "withdraw",
		Amount:    -amount,
		Balance:   cl.balance,
	})
	return nil
}

// RecordTrade records the cash impact of a filled trade.
// For buys: cash decreases by tradeAmount + commission.
// For sells: cash increases by tradeAmount - commission - stampTax.
func (cl *CashLedger) RecordTrade(side OrderSide, orderID string, tradeAmount, commission, stampTax float64) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	switch side {
	case SideBuy:
		amount := -(tradeAmount + commission)
		cl.balance += amount
		cl.entries = append(cl.entries, CashEntry{
			Timestamp: time.Now(),
			Type:      "buy",
			Amount:    amount,
			Balance:   cl.balance,
			OrderID:   orderID,
		})
		if commission > 0 {
			cl.entries = append(cl.entries, CashEntry{
				Timestamp: time.Now(),
				Type:      "fee",
				Amount:    -commission,
				Balance:   cl.balance,
				OrderID:   orderID,
			})
		}
	case SideSell:
		amount := tradeAmount - commission - stampTax
		cl.balance += amount
		cl.entries = append(cl.entries, CashEntry{
			Timestamp: time.Now(),
			Type:      "sell",
			Amount:    amount,
			Balance:   cl.balance,
			OrderID:   orderID,
		})
	}
}

// GetBalance returns the current cash balance.
func (cl *CashLedger) GetBalance() float64 {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.balance
}

// GetEntries returns a copy of all cash ledger entries.
func (cl *CashLedger) GetEntries() []CashEntry {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	result := make([]CashEntry, len(cl.entries))
	copy(result, cl.entries)
	return result
}
