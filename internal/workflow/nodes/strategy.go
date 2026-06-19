package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// StrategyNode defines a trading strategy configuration.
// It outputs strategy parameters that can be consumed by a BacktestNode.
type StrategyNode struct {
	id     string
	params map[string]any
}

// NewStrategyNode creates a new StrategyNode.
func NewStrategyNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &StrategyNode{id: id, params: params}, nil
}

func (n *StrategyNode) ID() string       { return n.id }
func (n *StrategyNode) NodeType() string { return "strategy" }
func (n *StrategyNode) Category() string { return "strategy" }

func (n *StrategyNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "factor_signals", Type: workflow.PortSeries, Required: false},
		{Name: "constraints", Type: workflow.PortSeries, Required: false},
	}
}

func (n *StrategyNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "strategy_config", Type: workflow.PortSeries, Required: false},
		{Name: "signals", Type: workflow.PortSeries, Required: false},
	}
}

func (n *StrategyNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "signal_type", Type: "string", Default: "sma_cross",
			Description: "Signal type: sma_cross, rsi_threshold, momentum, custom"},
		{Name: "fast_period", Type: "int", Default: 5,
			Description: "Fast moving average period (for sma_cross)"},
		{Name: "slow_period", Type: "int", Default: 20,
			Description: "Slow moving average period (for sma_cross)"},
		{Name: "rsi_oversold", Type: "float", Default: 30,
			Description: "RSI oversold threshold (buy signal)"},
		{Name: "rsi_overbought", Type: "float", Default: 70,
			Description: "RSI overbought threshold (sell signal)"},
		{Name: "position_size", Type: "float", Default: 1000,
			Description: "Fixed position size in shares or currency units"},
	}
}

func (n *StrategyNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	signalType := getStringParam(params, "signal_type", "sma_cross")

	strategyConfig := map[string]any{
		"signal_type":    signalType,
		"fast_period":    getIntParam(params, "fast_period", 5),
		"slow_period":    getIntParam(params, "slow_period", 20),
		"rsi_oversold":   getFloatParam(params, "rsi_oversold", 30),
		"rsi_overbought": getFloatParam(params, "rsi_overbought", 70),
		"position_size":  getFloatParam(params, "position_size", 1000),
	}

	return map[string]any{
		"strategy_config": strategyConfig,
		"signals":         nil,
	}, nil
}

func (n *StrategyNode) Validate() error {
	signalType := getStringParam(n.params, "signal_type", "sma_cross")
	validTypes := map[string]bool{
		"sma_cross": true, "rsi_threshold": true, "momentum": true, "custom": true,
	}
	if !validTypes[signalType] {
		return fmt.Errorf("strategy: invalid signal_type %q", signalType)
	}
	return nil
}

func getIntParam(params map[string]any, key string, defaultVal int) int {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return defaultVal
}

func getFloatParam(params map[string]any, key string, defaultVal float64) float64 {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return defaultVal
}
