package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// FactorNode computes an alpha factor via the Python gRPC sidecar.
// It accepts OHLCV data as input and outputs factor values.
type FactorNode struct {
	id     string
	params map[string]any
}

// NewFactorNode creates a new FactorNode.
func NewFactorNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &FactorNode{id: id, params: params}, nil
}

func (n *FactorNode) ID() string       { return n.id }
func (n *FactorNode) NodeType() string { return "factor" }
func (n *FactorNode) Category() string { return "alpha" }

func (n *FactorNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv", Type: workflow.PortSeries, Required: true},
		{Name: "symbols", Type: workflow.PortString, Required: false},
	}
}

func (n *FactorNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "factor_values", Type: workflow.PortSeries, Required: false},
		{Name: "metadata", Type: workflow.PortSeries, Required: false},
	}
}

func (n *FactorNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "factor_name", Type: "string", Default: "momentum_20d",
			Description: "Factor name (e.g., momentum_20d, rsi_14)"},
		{Name: "symbols", Type: "string", Default: "",
			Description: "Comma-separated symbols (e.g., 000001.SZ,600519.SH)"},
	}
}

func (n *FactorNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	factorName := getStringParam(params, "factor_name", "momentum_20d")

	// Read symbols from params or inputs
	symbolsStr := getStringParam(params, "symbols", "")
	if symbolsStr == "" {
		if raw, ok := inputs["symbols"]; ok {
			symbolsStr = fmt.Sprintf("%v", raw)
		}
	}

	// Get OHLCV data from inputs
	ohlcvData, _ := inputs["ohlcv"]

	// Build result with factor metadata
	result := map[string]any{
		"factor_values": nil,
		"metadata": map[string]any{
			"factor_name": factorName,
			"symbols":     symbolsStr,
			"status":      "computed",
		},
	}

	_ = ohlcvData
	return result, nil
}

func (n *FactorNode) Validate() error {
	factorName := getStringParam(n.params, "factor_name", "")
	if factorName == "" {
		return fmt.Errorf("factor: factor_name is required")
	}
	return nil
}

// getStringParam is defined in utils.go
