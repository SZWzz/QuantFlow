package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// SignalCombineNode combines multiple signal arrays using AND, OR, or majority vote.
type SignalCombineNode struct{ id string; params map[string]any }

// NewSignalCombineNode creates a new SignalCombineNode.
func NewSignalCombineNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SignalCombineNode{id: id, params: params}, nil
}

func (n *SignalCombineNode) ID() string       { return n.id }
func (n *SignalCombineNode) NodeType() string { return "signal_combine" }
func (n *SignalCombineNode) Category() string { return "signal" }

func (n *SignalCombineNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "signals", Type: workflow.PortSeries, Required: true},
	}
}

func (n *SignalCombineNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "combined", Type: workflow.PortSeries, Required: false},
		{Name: "confidence", Type: workflow.PortNumber, Required: false},
	}
}

func (n *SignalCombineNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "majority", Description: "Combine method: and, or, majority"},
	}
}

func (n *SignalCombineNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	method := getStringParam(params, "method", "majority")
	raw := inputs["signals"]
	if raw == nil {
		return nil, fmt.Errorf("signal_combine: signals input is required")
	}

	// Expect signals as [][]float64
	signalSets, ok := raw.([][]float64)
	if !ok {
		// Try []any wrapping
		if arr, ok2 := raw.([]any); ok2 {
			signalSets = make([][]float64, len(arr))
			for i, a := range arr {
				signalSets[i] = extractFloatSlice(a)
			}
		}
	}
	if len(signalSets) == 0 {
		return nil, fmt.Errorf("signal_combine: signals input is empty")
	}

	nSignals := len(signalSets)
	length := len(signalSets[0])
	combined := make([]float64, length)
	for i := 0; i < length; i++ {
		count := 0.0
		for s := 0; s < nSignals; s++ {
			if i < len(signalSets[s]) {
				count += signalSets[s][i]
			}
		}
		switch method {
		case "and":
			if count == float64(nSignals) {
				combined[i] = 1
			} else if count == -float64(nSignals) {
				combined[i] = -1
			}
		case "or":
			if count > 0 {
				combined[i] = 1
			} else if count < 0 {
				combined[i] = -1
			}
		default: // majority
			avg := count / float64(nSignals)
			if avg > 0.33 {
				combined[i] = 1
			} else if avg < -0.33 {
				combined[i] = -1
			}
		}
	}
	confidence := 1.0 / float64(max(1, nSignals))
	return map[string]any{"combined": combined, "confidence": confidence}, nil
}

func (n *SignalCombineNode) Validate() error {
	method := getStringParam(n.params, "method", "majority")
	switch method {
	case "and", "or", "majority":
		return nil
	default:
		return fmt.Errorf("signal_combine: invalid method %q, expected and/or/majority", method)
	}
}
