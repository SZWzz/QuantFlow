package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

func TestNewRiskModelNode(t *testing.T) {
	node, err := NewRiskModelNode("risk1", map[string]any{
		"model_type": "garch",
		"p":          "1",
		"q":          "1",
	})
	if err != nil {
		t.Fatalf("NewRiskModelNode: %v", err)
	}

	if node.ID() != "risk1" {
		t.Errorf("expected id 'risk1', got %q", node.ID())
	}
	if node.NodeType() != "risk_model" {
		t.Errorf("expected type 'risk_model', got %q", node.NodeType())
	}
	if node.Category() != "risk" {
		t.Errorf("expected category 'risk', got %q", node.Category())
	}

	ports := node.InputPorts()
	if len(ports) != 1 {
		t.Errorf("expected 1 input port, got %d", len(ports))
	}
	if ports[0].Name != "returns_data" {
		t.Errorf("expected input port 'returns_data', got %q", ports[0].Name)
	}

	outPorts := node.OutputPorts()
	if len(outPorts) != 3 {
		t.Errorf("expected 3 output ports, got %d", len(outPorts))
	}
}

func TestRiskModelNodeValidate(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]any
		wantError bool
	}{
		{"valid garch", map[string]any{"model_type": "garch"}, false},
		{"valid gjr_garch", map[string]any{"model_type": "gjr_garch"}, false},
		{"valid egarch", map[string]any{"model_type": "egarch"}, false},
		{"valid covariance", map[string]any{"model_type": "covariance"}, false},
		{"invalid type", map[string]any{"model_type": "unknown"}, true},
		{"empty params", map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRiskModelNode("test", tt.params)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError = %v", err, tt.wantError)
			}
		})
	}
}

func TestRiskModelNodeExecute(t *testing.T) {
	node, err := NewRiskModelNode("risk1", map[string]any{"model_type": "garch"})
	if err != nil {
		t.Fatalf("NewRiskModelNode: %v", err)
	}

	inputs := map[string]any{
		"returns_data": []float64{0.01, -0.02, 0.005, 0.015, -0.01},
	}
	_, err = node.Execute(context.Background(), inputs, map[string]any{
		"model_type": "garch",
	}, nil)
	// RiskModel is not yet wired to the Python sidecar — expect an honest error
	// instead of silently returning fake data.
	if err == nil {
		t.Error("expected 'not yet implemented' error, got nil")
	}
}

func TestRLEnvNode(t *testing.T) {
	node, err := NewRLEnvNode("env1", map[string]any{
		"action_type": "discrete",
		"window_size": "20",
	})
	if err != nil {
		t.Fatalf("NewRLEnvNode: %v", err)
	}

	if node.NodeType() != "rl_env" {
		t.Errorf("expected 'rl_env', got %q", node.NodeType())
	}
	if node.Category() != "ml" {
		t.Errorf("expected 'ml', got %q", node.Category())
	}

	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{
		"window_size": 10, "action_type": "discrete", "initial_cash": 50000,
	}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	config, ok := outputs["env_config"].(map[string]any)
	if !ok {
		t.Fatal("expected env_config to be map[string]any")
	}
	if config["window_size"].(int) != 10 {
		t.Errorf("expected window_size 10, got %v", config["window_size"])
	}
}

func TestRLTrainNode(t *testing.T) {
	node, err := NewRLTrainNode("train1", map[string]any{"algorithm": "ppo"})
	if err != nil {
		t.Fatalf("NewRLTrainNode: %v", err)
	}

	if node.NodeType() != "rl_train" {
		t.Errorf("expected 'rl_train', got %q", node.NodeType())
	}

	outputs, err := node.Execute(context.Background(), map[string]any{}, map[string]any{
		"algorithm": "ppo", "total_episodes": 50,
	}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, ok := outputs["model_id"]; !ok {
		t.Error("expected 'model_id' in outputs")
	}
	if _, ok := outputs["reward_curve"]; !ok {
		t.Error("expected 'reward_curve' in outputs")
	}
}

func TestRLPredictNode(t *testing.T) {
	node, err := NewRLPredictNode("pred1", nil)
	if err != nil {
		t.Fatalf("NewRLPredictNode: %v", err)
	}

	if node.NodeType() != "rl_predict" {
		t.Errorf("expected 'rl_predict', got %q", node.NodeType())
	}

	outputs, err := node.Execute(context.Background(), map[string]any{
		"model_id":    "rl_ppo_100",
		"observation": []float64{},
	}, map[string]any{"deterministic": "true"}, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	action, ok := outputs["action"].(int)
	if !ok {
		t.Fatal("expected action to be int")
	}
	if action < 0 || action > 2 {
		t.Errorf("expected action in [0,2], got %d", action)
	}
}

func TestRegisterAllHasNewNodes(t *testing.T) {
	r := workflow.NewRegistry()
	RegisterAll(r)

	expectedNodes := []string{"rl_env", "rl_train", "rl_predict", "risk_model"}
	for _, name := range expectedNodes {
		if !r.Has(name) {
			t.Errorf("expected %q to be registered, but it's not", name)
		}
	}
}
