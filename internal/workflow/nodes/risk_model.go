package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// RiskModelNode computes GARCH volatility or covariance matrix via Python sidecar.
type RiskModelNode struct {
	id     string
	params map[string]any
}

// NewRiskModelNode creates a new RiskModelNode.
func NewRiskModelNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &RiskModelNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *RiskModelNode) ID() string       { return n.id }
func (n *RiskModelNode) NodeType() string { return "risk_model" }
func (n *RiskModelNode) Category() string { return "risk" }

func (n *RiskModelNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "returns_data", Type: workflow.PortSeries, Required: true},
	}
}

func (n *RiskModelNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "volatility", Type: workflow.PortSeries},
		{Name: "covariance_matrix", Type: workflow.PortAny},
		{Name: "model_metrics", Type: workflow.PortAny},
	}
}

func (n *RiskModelNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "model_type", Type: "string", Default: "garch", Description: "risk model type: garch/gjr_garch/egarch/covariance"},
		{Name: "method", Type: "string", Default: "ledoit_wolf", Description: "covariance method: ledoit_wolf/sample"},
		{Name: "p", Type: "int", Default: "1", Description: "GARCH p (ARCH order)"},
		{Name: "q", Type: "int", Default: "1", Description: "GARCH q (GARCH order)"},
	}
}

func (n *RiskModelNode) Validate() error {
	modelType := getStringParam(n.params, "model_type", "garch")
	validTypes := map[string]bool{"garch": true, "gjr_garch": true, "egarch": true, "covariance": true}
	if !validTypes[modelType] {
		return fmt.Errorf("risk_model: invalid model_type '%s'", modelType)
	}
	return nil
}

func (n *RiskModelNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	_ = getStringParam(params, "model_type", "garch")
	_ = inputs["returns_data"]

	// TODO: Call Python sidecar RiskModel RPC (GARCH/covariance).
	// The gRPC endpoint is implemented (ml_grpc.pb.go RiskModel) and the Python
	// handler exists (engine.py RiskModel), but the Go MLClient wrapper is not yet
	// created. Once wired, this node delegates to the sidecar.
	return nil, fmt.Errorf("risk_model: not yet implemented — Python sidecar has the GARCH/covariance handler, but the Go→gRPC client method (MLClient.RiskModel) is not yet created")
}
