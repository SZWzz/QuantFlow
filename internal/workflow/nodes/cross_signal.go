package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"
)

// SignalEvent represents a cross signal event (golden cross or death cross).
type SignalEvent struct {
	Index      int     `json:"index"`
	Direction  string  `json:"direction"`
	FastValue  float64 `json:"fast_value"`
	SlowValue  float64 `json:"slow_value"`
	Confidence float64 `json:"confidence"`
}

// CrossSignalNode detects cross signals (golden cross / death cross)
// between two input series (fast and slow).
type CrossSignalNode struct {
	id     string
	params map[string]any
}

// NewCrossSignalNode creates a new CrossSignalNode.
func NewCrossSignalNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &CrossSignalNode{id: id, params: params}, nil
}

func (n *CrossSignalNode) ID() string       { return n.id }
func (n *CrossSignalNode) NodeType() string { return "cross_signal" }
func (n *CrossSignalNode) Category() string { return "signal" }

func (n *CrossSignalNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "fast", Type: workflow.PortSeries, Required: true},
		{Name: "slow", Type: workflow.PortSeries, Required: true},
	}
}

func (n *CrossSignalNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "signal", Type: workflow.PortSignal, Required: false},
	}
}

func (n *CrossSignalNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *CrossSignalNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	fast, ok := toFloat64Slice(inputs["fast"])
	if !ok {
		return nil, fmt.Errorf("cross_signal: fast input must be []float64")
	}
	slow, ok := toFloat64Slice(inputs["slow"])
	if !ok {
		return nil, fmt.Errorf("cross_signal: slow input must be []float64")
	}
	if len(fast) != len(slow) {
		return nil, fmt.Errorf("cross_signal: fast(%d) and slow(%d) must have same length", len(fast), len(slow))
	}

	var signals []SignalEvent
	for i := 1; i < len(fast); i++ {
		prevAbove := fast[i-1] > slow[i-1]
		currAbove := fast[i] > slow[i]
		if !prevAbove && currAbove {
			signals = append(signals, SignalEvent{
				Index:      i,
				Direction:  "buy",
				FastValue:  fast[i],
				SlowValue:  slow[i],
				Confidence: (fast[i] - slow[i]) / slow[i],
			})
		} else if prevAbove && !currAbove {
			signals = append(signals, SignalEvent{
				Index:      i,
				Direction:  "sell",
				FastValue:  fast[i],
				SlowValue:  slow[i],
				Confidence: (slow[i] - fast[i]) / slow[i],
			})
		}
	}
	return map[string]any{"signal": signals}, nil
}

func (n *CrossSignalNode) Validate() error { return nil }
