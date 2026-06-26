package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/ml"
	"quantflow/internal/python"
	"quantflow/internal/python/proto"
	"quantflow/internal/workflow"
)

// bridge is the package-level PythonBridge reference, set via SetPythonBridge.
var bridge *python.PythonBridge

// SetPythonBridge sets the Python bridge for ML nodes to use.
func SetPythonBridge(b *python.PythonBridge) {
	bridge = b
}

// modelRegistry is the package-level ModelRegistry reference, set via SetModelRegistry.
var modelRegistry *ml.ModelRegistry

// SetModelRegistry sets the ModelRegistry for ML nodes.
func SetModelRegistry(r *ml.ModelRegistry) {
	modelRegistry = r
}

// TrainModelNode trains an ML model via the Python sidecar.
// It consumes feature_matrix and target series inputs, and produces
// a model_id and train_metrics outputs.
type TrainModelNode struct {
	id     string
	params map[string]any
}

// NewTrainModelNode creates a new TrainModelNode.
func NewTrainModelNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &TrainModelNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *TrainModelNode) ID() string       { return n.id }
func (n *TrainModelNode) NodeType() string { return "train_model" }
func (n *TrainModelNode) Category() string { return "ml" }

func (n *TrainModelNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "feature_matrix", Type: workflow.PortSeries, Required: true},
		{Name: "target", Type: workflow.PortSeries, Required: true},
	}
}

func (n *TrainModelNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "model_id", Type: workflow.PortAny},
		{Name: "train_metrics", Type: workflow.PortSeries},
	}
}

func (n *TrainModelNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "model_type", Type: "string", Default: "xgboost", Description: "xgboost/lightgbm/lstm/transformer"},
		{Name: "target_type", Type: "string", Default: "regression", Description: "regression/classification"},
		{Name: "forecast_horizon", Type: "int", Default: "5", Description: "prediction horizon in bars"},
		{Name: "n_estimators", Type: "int", Default: "100", Description: "number of trees/epochs"},
		{Name: "max_depth", Type: "int", Default: "6", Description: "max tree depth"},
		{Name: "learning_rate", Type: "float", Default: "0.1", Description: "learning rate"},
		{Name: "timeout_seconds", Type: "int", Default: "300", Description: "training timeout"},
	}
}

func (n *TrainModelNode) Validate() error {
	modelType := getStringParam(n.params, "model_type", "xgboost")
	validTypes := ml.ValidModelTypes()
	for _, vt := range validTypes {
		if vt == modelType {
			return nil
		}
	}
	return fmt.Errorf("train_model: unsupported model_type '%s'", modelType)
}

func (n *TrainModelNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	if bridge == nil {
		return nil, fmt.Errorf("train_model: PythonBridge not set — call SetPythonBridge() first")
	}

	features := inputs["feature_matrix"].(map[string][]float64)
	target := inputs["target"].(map[string][]float64)

	// Convert to JSON bytes for the gRPC message
	featureJSON, _ := json.Marshal(features)
	targetJSON, _ := json.Marshal(target)

	hyperparams := map[string]string{
		"n_estimators":  fmt.Sprintf("%d", getIntParam(params, "n_estimators", 100)),
		"max_depth":     fmt.Sprintf("%d", getIntParam(params, "max_depth", 6)),
		"learning_rate": fmt.Sprintf("%f", getFloatParam(params, "learning_rate", 0.1)),
	}

	mlClient := python.NewMLClient(bridge)
	req := &proto.TrainRequest{
		ModelType:       getStringParam(params, "model_type", "xgboost"),
		Features:        featureJSON,
		Targets:         targetJSON,
		Hyperparams:     hyperparams,
		TargetType:      getStringParam(params, "target_type", "regression"),
		ForecastHorizon: int32(getIntParam(params, "forecast_horizon", 5)),
	}
	resp, err := mlClient.Train(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("train_model: %w", err)
	}

	// Register model in registry
	if modelRegistry != nil {
		m := &ml.MLModel{
			ID:          resp.ModelId,
			Name:        fmt.Sprintf("%s_%s", req.ModelType, resp.ModelId[:8]),
			ModelType:   ml.ModelType(req.ModelType),
			Category:    ml.CategoryPrediction,
			Hyperparams: req.Hyperparams,
			Metrics:     resp.Metrics,
			Status:      ml.ModelStatusReady,
		}
		_ = modelRegistry.Create(ctx, m)
	}

	metrics := make(map[string][]float64)
	for k, v := range resp.Metrics {
		metrics[k] = []float64{v}
	}

	return map[string]any{
		"model_id":      resp.ModelId,
		"train_metrics": metrics,
	}, nil
}
