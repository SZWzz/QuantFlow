package main

import (
	"context"
	"fmt"
	"strings"

	"quantflow/internal/portfolio"
	"quantflow/internal/trading"
)

// PlaceOrder submits an order to the trading engine (paper or live broker).
func (a *App) PlaceOrder(symbol, side, orderType, brokerName string, qty, price float64) (*trading.Order, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}

	// Configure daily price limit from cached last close.
	if a.lastClose != nil {
		if prevClose, ok := a.lastClose[symbol]; ok && prevClose > 0 {
			ratio := 0.10
			if strings.HasPrefix(symbol, "300") || strings.HasPrefix(symbol, "301") || strings.HasPrefix(symbol, "688") {
				ratio = 0.20
			}
			a.oms.SetPriceLimit(symbol, prevClose, ratio)
		}
	}

	return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), brokerName, qty, price)
}

// GetPositions returns all current positions.
func (a *App) GetPositions() []*trading.Position {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetAllPositions()
}

// GetOrders returns all orders.
func (a *App) GetOrders() []*trading.Order {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetOrders()
}

// GetTrades returns all filled trades.
func (a *App) GetTrades() []*trading.Trade {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetTrades()
}

// GetPortfolioSummary returns current portfolio summary.
func (a *App) GetPortfolioSummary() map[string]interface{} {
	if a.portfolioSvc == nil {
		return map[string]interface{}{"total_value": 0}
	}
	var s *portfolio.Summary
	if a.portfolioSvc != nil {
		s = a.portfolioSvc.GetSummary()
	} else {
		s = &portfolio.Summary{}
	}
	return map[string]interface{}{
		"total_value": s.TotalValue, "cash_balance": s.CashBalance,
		"market_value": s.MarketValue, "total_pnl": s.TotalPnL, "total_pnl_pct": s.TotalPnLPct,
	}
}

// GetPortfolioAllocation returns allocation breakdowns.
func (a *App) GetPortfolioAllocation() *portfolio.Allocation {
	if a.portfolioSvc == nil {
		return &portfolio.Allocation{}
	}
	return a.portfolioSvc.GetAllocation()
}

// GetRebalanceSuggestions returns rebalance advice.
func (a *App) GetRebalanceSuggestions(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// BrokerStatus reports connection state for a broker.
type BrokerStatus struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Market    string `json:"market"`
	Connected bool   `json:"connected"`
	Detail    string `json:"detail"`
}

// brokerByName looks up a registered live broker by name. Returns nil if not found.
func (a *App) brokerByName(name string) trading.Broker {
	if a.brokers == nil {
		return nil
	}
	return a.brokers[name]
}

// GetBrokerStatuses returns connection status of all registered brokers.
func (a *App) GetBrokerStatuses() []BrokerStatus {
	statuses := []BrokerStatus{
		{Name: "paper", Label: "Paper Trading", Market: "模拟", Connected: true, Detail: "本地模拟撮合"},
	}

	// Probe registered live brokers.
	brokerNames := []string{"futu", "binance", "alpaca", "ibkr"}
	brokerLabels := map[string]string{
		"futu":    "富途牛牛",
		"binance": "Binance",
		"alpaca":  "Alpaca",
		"ibkr":    "Interactive Brokers",
	}
	brokerMarkets := map[string]string{
		"futu":    "港股/A股/美股",
		"binance": "加密",
		"alpaca":  "美股",
		"ibkr":    "全球",
	}

	for _, name := range brokerNames {
		br := a.brokerByName(name)
		connected := false
		detail := "未配置"
		if br != nil {
			connected = br.IsConnected()
			if connected {
				detail = "已连接"
			} else {
				detail = "已配置，未连接"
			}
		}
		label := brokerLabels[name]
		if label == "" {
			label = name
		}
		statuses = append(statuses, BrokerStatus{
			Name:      name,
			Label:     label,
			Market:    brokerMarkets[name],
			Connected: connected,
			Detail:    detail,
		})
	}

	return statuses
}

// RunBacktest executes a backtest from a workflow JSON definition.
func (a *App) RunBacktest(jsonDef string) (map[string]interface{}, error) {
	_ = jsonDef
	return nil, fmt.Errorf("backtest engine available but RunBacktest not yet wired — see internal/backtest/runner.go")
}

// CheckWashSale detects wash sale events for a symbol using trade history.
func (a *App) CheckWashSale(symbol string) ([]trading.WashSaleEvent, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}
	checker := trading.NewWashSaleChecker()
	return checker.CheckSymbol(context.Background(), symbol, a.oms)
}
