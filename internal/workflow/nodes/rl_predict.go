package nodes

import (
	"context"

	"quantflow/internal/workflow"
)

// RLPredictNode uses a trained RL model to predict an action from an observation.
type RLPredictNode struct {
	id     string
	params map[string]any
}

// NewRLPredictNode creates a new RLPredictNode.
func NewRLPredictNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &RLPredictNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *RLPredictNode) ID() string       { return n.id }
func (n *RLPredictNode) NodeType() string { return "rl_predict" }
func (n *RLPredictNode) Category() string { return "ml" }

func (n *RLPredictNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "model_id", Type: workflow.PortString, Required: true},
		{Name: "observation", Type: workflow.PortSeries, Required: true},
	}
}

func (n *RLPredictNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "action", Type: workflow.PortNumber},
		{Name: "action_value", Type: workflow.PortNumber},
	}
}

func (n *RLPredictNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "deterministic", Type: "bool", Default: "true", Description: "use deterministic policy (no exploration)"},
	}
}

func (n *RLPredictNode) Validate() error {
	return nil
}

func (n *RLPredictNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	_ = inputs["model_id"]
	_ = inputs["observation"]

	// For now, return hold action (1) as default.
	// Full implementation calls Python via gRPC for model inference.
	action := 1 // hold
	actionValue := 0.0

	return map[string]any{
		"action":       action,
		"action_value": actionValue,
	}, nil
}
