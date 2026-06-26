package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/python/proto"
	"quantflow/internal/workflow"
)

// PredictNode runs ML model inference via the Python sidecar.
// It consumes a model_id and feature_matrix, and produces predictions.
type PredictNode struct {
	id     string
	params map[string]any
}

// NewPredictNode creates a new PredictNode.
func NewPredictNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &PredictNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *PredictNode) ID() string       { return n.id }
func (n *PredictNode) NodeType() string { return "predict" }
func (n *PredictNode) Category() string { return "ml" }

func (n *PredictNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "model_id", Type: workflow.PortAny, Required: true},
		{Name: "feature_matrix", Type: workflow.PortSeries, Required: true},
	}
}

func (n *PredictNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "predictions", Type: workflow.PortSeries},
	}
}

func (n *PredictNode) ParamSchema() []workflow.ParamDef {
	return nil
}

func (n *PredictNode) Validate() error { return nil }

func (n *PredictNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var bridge *python.PythonBridge
	if nctx != nil {
		if nctx.Bridge != nil {
			bridge, _ = nctx.Bridge.(*python.PythonBridge)
		}
	}
	if bridge == nil {
		return nil, fmt.Errorf("predict: PythonBridge not set — call SetPythonBridge() first")
	}

	modelID, ok := inputs["model_id"].(string)
	if !ok {
		return nil, fmt.Errorf("predict: model_id must be string")
	}
	features := inputs["feature_matrix"].(map[string][]float64)
	featureJSON, _ := json.Marshal(features)

	mlClient := python.NewMLClient(bridge)
	req := &proto.PredictRequest{
		ModelId:  modelID,
		Features: featureJSON,
	}
	resp, err := mlClient.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("predict: %w", err)
	}

	return map[string]any{
		"predictions": map[string][]float64{"value": resp.Predictions},
	}, nil
}
