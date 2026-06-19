package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// RLEnvNode configures a reinforcement learning trading environment.
type RLEnvNode struct {
	id     string
	params map[string]any
}

// NewRLEnvNode creates a new RLEnvNode.
func NewRLEnvNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &RLEnvNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *RLEnvNode) ID() string       { return n.id }
func (n *RLEnvNode) NodeType() string { return "rl_env" }
func (n *RLEnvNode) Category() string { return "ml" }

func (n *RLEnvNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv_data", Type: workflow.PortSeries, Required: true},
		{Name: "factors", Type: workflow.PortSeries, Required: false},
	}
}

func (n *RLEnvNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "env_config", Type: workflow.PortAny},
	}
}

func (n *RLEnvNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "window_size", Type: "int", Default: "20", Description: "observation window size"},
		{Name: "action_type", Type: "string", Default: "discrete", Description: "action space type: discrete/continuous"},
		{Name: "initial_cash", Type: "int", Default: "10000", Description: "initial cash for the environment"},
	}
}

func (n *RLEnvNode) Validate() error {
	actionType := getStringParam(n.params, "action_type", "discrete")
	if actionType != "discrete" && actionType != "continuous" {
		return fmt.Errorf("rl_env: invalid action_type '%s'", actionType)
	}
	return nil
}

func (n *RLEnvNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	windowSize := getIntParam(params, "window_size", 20)
	actionType := getStringParam(params, "action_type", "discrete")
	initialCash := getIntParam(params, "initial_cash", 10000)

	config := map[string]any{
		"window_size":  windowSize,
		"action_type":  actionType,
		"initial_cash": initialCash,
	}

	return map[string]any{
		"env_config": config,
	}, nil
}
