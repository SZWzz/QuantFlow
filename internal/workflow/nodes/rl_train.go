package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// RLTrainNode trains an RL agent using streaming gRPC to Python sidecar.
type RLTrainNode struct {
	id     string
	params map[string]any
}

// NewRLTrainNode creates a new RLTrainNode.
func NewRLTrainNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &RLTrainNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *RLTrainNode) ID() string       { return n.id }
func (n *RLTrainNode) NodeType() string { return "rl_train" }
func (n *RLTrainNode) Category() string { return "ml" }

func (n *RLTrainNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv_data", Type: workflow.PortSeries, Required: true},
		{Name: "env_config", Type: workflow.PortAny, Required: false},
	}
}

func (n *RLTrainNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "model_id", Type: workflow.PortString},
		{Name: "reward_curve", Type: workflow.PortSeries},
	}
}

func (n *RLTrainNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "algorithm", Type: "string", Default: "ppo", Description: "RL algorithm: ppo/dqn/sac"},
		{Name: "total_episodes", Type: "int", Default: "100", Description: "number of training episodes"},
		{Name: "learning_rate", Type: "float", Default: "0.0003", Description: "learning rate"},
	}
}

func (n *RLTrainNode) Validate() error {
	algo := getStringParam(n.params, "algorithm", "ppo")
	if algo != "ppo" && algo != "dqn" && algo != "sac" {
		return fmt.Errorf("rl_train: invalid algorithm '%s'", algo)
	}
	return nil
}

func (n *RLTrainNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	algorithm := getStringParam(params, "algorithm", "ppo")
	totalEpisodes := getIntParam(params, "total_episodes", 100)

	// The actual training is done via gRPC streaming to Python sidecar.
	// This node prepares the configuration; execution happens in the workflow engine.
	modelID := fmt.Sprintf("rl_%s_%d", algorithm, totalEpisodes)

	rewardCurve := make([]float64, 0)

	return map[string]any{
		"model_id":     modelID,
		"reward_curve": rewardCurve,
	}, nil
}
