package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// StopLossNode monitors price against a stop-loss level.
// In trailing mode it tracks the highest price and triggers when price
// drops below stop_pct% from the peak. In fixed mode it triggers when
// price falls to or below entry * (1 - stop_pct%).
type StopLossNode struct {
	id     string
	params map[string]any
}

// NewStopLossNode creates a new StopLossNode.
func NewStopLossNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &StopLossNode{id: id, params: params}, nil
}

func (n *StopLossNode) ID() string       { return n.id }
func (n *StopLossNode) NodeType() string { return "stop_loss" }
func (n *StopLossNode) Category() string { return "risk" }

func (n *StopLossNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "price", Type: workflow.PortNumber, Required: true},
		{Name: "entry_price", Type: workflow.PortNumber, Required: true},
	}
}

func (n *StopLossNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "triggered", Type: workflow.PortNumber, Required: false},
		{Name: "stop_price", Type: workflow.PortNumber, Required: false},
	}
}

func (n *StopLossNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "stop_pct", Type: "number", Default: "5", Description: "Stop-loss percentage (e.g. 5 means 5% drop from entry/peak)"},
		{Name: "trailing", Type: "bool", Default: "false", Description: "Use trailing stop that tracks highest price"},
	}
}

func (n *StopLossNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	price := extractFloat(inputs["price"])
	entryPrice := extractFloat(inputs["entry_price"])
	if price == nil || entryPrice == nil {
		return nil, fmt.Errorf("stop_loss: price and entry_price inputs are required")
	}

	stopPct := getFloatParam(params, "stop_pct", 5) / 100.0
	trailing := getStringParam(params, "trailing", "false") == "true"

	var triggered bool
	var stopPrice float64

	if trailing {
		// Track highest price seen (use entry_price as initial peak).
		// For a real implementation this would be stateful across ticks;
		// here we use the higher of entry_price and price as the peak.
		peak := math.Max(*entryPrice, *price)
		stopPrice = peak * (1 - stopPct)
		triggered = *price <= stopPrice
	} else {
		stopPrice = *entryPrice * (1 - stopPct)
		triggered = *price <= stopPrice
	}

	triggeredVal := 0.0
	if triggered {
		triggeredVal = 1.0
	}

	return map[string]any{
		"triggered":  triggeredVal,
		"stop_price": stopPrice,
	}, nil
}

func (n *StopLossNode) Validate() error {
	stopPct := getFloatParam(n.params, "stop_pct", 5)
	if stopPct <= 0 || stopPct > 100 {
		return fmt.Errorf("stop_loss: stop_pct must be between 0 and 100, got %v", stopPct)
	}
	trailing := getStringParam(n.params, "trailing", "false")
	if trailing != "true" && trailing != "false" {
		return fmt.Errorf("stop_loss: trailing must be 'true' or 'false', got %q", trailing)
	}
	return nil
}

// extractFloat extracts a single float64 from an input value.
func extractFloat(val any) *float64 {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case []float64:
		if len(v) > 0 {
			last := v[len(v)-1]
			return &last
		}
	case []any:
		if len(v) > 0 {
			switch f := v[len(v)-1].(type) {
			case float64:
				return &f
			case int:
				fv := float64(f)
				return &fv
			}
		}
	}
	return nil
}
