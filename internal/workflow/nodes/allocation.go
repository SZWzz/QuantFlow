package nodes

import (
	"context"
	"fmt"
	"math"
	"quantflow/internal/workflow"
)

type AllocationNode struct {
	id     string
	params map[string]any
}

func NewAllocationNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AllocationNode{id: id, params: params}, nil
}

func (n *AllocationNode) ID() string       { return n.id }
func (n *AllocationNode) NodeType() string { return "allocation" }
func (n *AllocationNode) Category() string { return "portfolio" }

func (n *AllocationNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "symbols", Type: workflow.PortAny, Required: true},
		{Name: "total_capital", Type: workflow.PortNumber, Required: true},
		{Name: "method", Type: workflow.PortString, Required: false},
	}
}

func (n *AllocationNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "allocations", Type: workflow.PortAny, Required: false},
		{Name: "weights", Type: workflow.PortAny, Required: false},
	}
}

func (n *AllocationNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "equal", Description: "Allocation method: equal, risk_parity"},
	}
}

func (n *AllocationNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	symbolsRaw, ok := inputs["symbols"]
	if !ok || symbolsRaw == nil {
		return nil, fmt.Errorf("allocation: symbols input is required")
	}

	symbols := toStringSlice(symbolsRaw)
	if len(symbols) == 0 {
		return nil, fmt.Errorf("allocation: symbols list is empty")
	}

	totalCapital, ok := toFloat64(inputs["total_capital"])
	if !ok || totalCapital <= 0 {
		return nil, fmt.Errorf("allocation: total_capital must be a positive number")
	}

	method := getStringParam(params, "method", "equal")
	if m, ok := inputs["method"].(string); ok && m != "" {
		method = m
	}

	var allocations map[string]float64
	switch method {
	case "equal":
		allocations = equalAllocation(symbols, totalCapital)
	case "risk_parity":
		allocations = equalAllocation(symbols, totalCapital)
	default:
		return nil, fmt.Errorf("allocation: unknown method %q (supported: equal, risk_parity)", method)
	}

	weights := make(map[string]float64, len(allocations))
	for sym, amt := range allocations {
		weights[sym] = math.Round(amt/totalCapital*10000) / 10000
	}

	return map[string]any{
		"allocations": allocations,
		"weights":     weights,
	}, nil
}

func (n *AllocationNode) Validate() error { return nil }

func equalAllocation(symbols []string, total float64) map[string]float64 {
	n := len(symbols)
	base := math.Floor(total/float64(n)*100) / 100
	alloc := make(map[string]float64, n)
	var used float64
	for i, sym := range symbols {
		if i == n-1 {
			alloc[sym] = math.Round((total-used)*100) / 100
		} else {
			alloc[sym] = base
			used += base
		}
	}
	return alloc
}

func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		s := make([]string, 0, len(val))
		for _, x := range val {
			if str, ok := x.(string); ok {
				s = append(s, str)
			}
		}
		return s
	default:
		return nil
	}
}
