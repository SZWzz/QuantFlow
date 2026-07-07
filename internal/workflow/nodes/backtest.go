package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/backtest"
	"quantflow/internal/market"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

// BacktestNode runs a historical backtest using the backtesting engine.
// It accepts strategy configuration and OHLCV data as inputs.
type BacktestNode struct {
	id     string
	params map[string]any
}

// NewBacktestNode creates a new BacktestNode.
func NewBacktestNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &BacktestNode{id: id, params: params}, nil
}

func (n *BacktestNode) ID() string       { return n.id }
func (n *BacktestNode) NodeType() string { return "backtest" }
func (n *BacktestNode) Category() string { return "backtest" }

func (n *BacktestNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "strategy_config", Type: workflow.PortSeries, Required: false},
		{Name: "ohlcv_data", Type: workflow.PortSeries, Required: true},
		{Name: "signals", Type: workflow.PortSeries, Required: false},
	}
}

func (n *BacktestNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "result", Type: workflow.PortSeries, Required: false},
		{Name: "equity_curve", Type: workflow.PortSeries, Required: false},
		{Name: "metrics", Type: workflow.PortSeries, Required: false},
		{Name: "trades", Type: workflow.PortSeries, Required: false},
	}
}

func (n *BacktestNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "market", Type: "string", Default: "CN",
			Description: "Market: CN, US, HK, CRYPTO"},
		{Name: "start_date", Type: "string", Default: "2024-01-01",
			Description: "Start date (YYYY-MM-DD)"},
		{Name: "end_date", Type: "string", Default: "2024-12-31",
			Description: "End date (YYYY-MM-DD)"},
		{Name: "initial_cash", Type: "float", Default: 1000000,
			Description: "Initial cash for the backtest"},
	}
}

func (n *BacktestNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	marketName := getStringParam(params, "market", "CN")
	startDate := getStringParam(params, "start_date", "2024-01-01")
	endDate := getStringParam(params, "end_date", "2024-12-31")
	initialCash := getFloatParam(params, "initial_cash", 1_000_000)

	// Get OHLCV data from inputs
	rawData, ok := inputs["ohlcv_data"]
	if !ok {
		return nil, fmt.Errorf("backtest: missing required input 'ohlcv_data'")
	}

	// Try to convert input to OHLCV bars (market.OHLCVBar is the canonical type).
	bars, ok := rawData.([]market.OHLCVBar)
	if !ok {
		// Try []any conversion
		if rawSlice, ok := rawData.([]any); ok {
			bars = make([]market.OHLCVBar, 0, len(rawSlice))
			for _, item := range rawSlice {
				if bar, ok := item.(market.OHLCVBar); ok {
					bars = append(bars, bar)
				}
			}
		}
	}
	if len(bars) == 0 {
		return map[string]any{
			"result":       nil,
			"equity_curve": nil,
			"metrics":      nil,
			"trades":       nil,
			"error":        "no OHLCV data available",
		}, nil
	}

	// Convert market.OHLCVBar → trading.OHLCVBar for engine consumption.
	tradingBars := make([]trading.OHLCVBar, len(bars))
	for i, b := range bars {
		tradingBars[i] = trading.OHLCVBar{
			Symbol: b.Symbol,
			Date:   b.Date,
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: b.Volume,
		}
	}

	// Create simple strategy
	// In a full implementation, this would parse strategy_config from inputs
	firstSymbol := ""
	if len(tradingBars) > 0 {
		firstSymbol = tradingBars[0].Symbol
	}
	strategy := backtest.Strategy{
		ID:   n.id,
		Name: "Workflow Strategy",
		SignalFunc: func(openPrice float64, prevBar *trading.OHLCVBar, portfolio *backtest.Portfolio) *trading.Signal {
			// Default: buy and hold on first bar
			if firstSymbol != "" && portfolio.Positions[firstSymbol] <= 0 {
				return &trading.Signal{
					Symbol:    firstSymbol,
					Direction: "buy",
					Quantity:  100,
				}
			}
			return nil
		},
	}

	// Run backtest (StartDate/EndDate are informational — bars determine actual range)
	config := backtest.DefaultConfig()
	config.InitialCash = initialCash
	_ = startDate
	_ = endDate

	var result *backtest.Result
	var err error

	switch marketName {
	case "CN":
		engine := backtest.NewCNEngine(config)
		result, err = engine.Run(ctx, strategy, tradingBars)
	case "US":
		engine := backtest.NewUSEngine(config)
		result, err = engine.Run(ctx, strategy, tradingBars)
	case "HK":
		engine := backtest.NewHKEngine(config)
		result, err = engine.Run(ctx, strategy, tradingBars)
	default:
		runner := backtest.NewRunner(config)
		result, err = runner.Run(ctx, strategy, tradingBars)
	}

	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("backtest failed: %v", err),
		}, nil
	}

	// Extract symbol and strategy name for persistence
	symbol := ""
	if len(tradingBars) > 0 {
		symbol = tradingBars[0].Symbol
	}

	// Extract equity values for series output
	equityValues := make([]float64, len(result.EquityCurve))
	for i, p := range result.EquityCurve {
		equityValues[i] = p.Equity
	}

	outputs := map[string]any{
		"result":          result,
		"equity_curve":    equityValues,
		"metrics":         result.Metrics,
		"trades":          result.Trades,
		"strategy_name":   strategy.Name,
		"symbol":          symbol,
		"engine_type":     marketName,
		"backtest_start":  startDate,
		"backtest_end":    endDate,
	}
	if nctx != nil && nctx.RunID != "" {
		outputs["run_id"] = nctx.RunID
	}
	return outputs, nil
}

func (n *BacktestNode) Validate() error {
	market := getStringParam(n.params, "market", "CN")
	validMarkets := map[string]bool{"CN": true, "US": true, "HK": true, "CRYPTO": true}
	if !validMarkets[market] {
		return fmt.Errorf("backtest: invalid market %q", market)
	}
	return nil
}
