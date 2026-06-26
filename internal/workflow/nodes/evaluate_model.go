package nodes

import (
	"context"
	"fmt"
	"math"

	"quantflow/internal/workflow"
)

// EvaluateModelNode computes model performance metrics: MSE, MAE, RMSE, IC.
// It performs pure Go computation — no Python sidecar required.
type EvaluateModelNode struct {
	id     string
	params map[string]any
}

// NewEvaluateModelNode creates a new EvaluateModelNode.
func NewEvaluateModelNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &EvaluateModelNode{id: id, params: params}
	return n, n.Validate()
}

func (n *EvaluateModelNode) ID() string       { return n.id }
func (n *EvaluateModelNode) NodeType() string { return "evaluate_model" }
func (n *EvaluateModelNode) Category() string { return "ml" }

func (n *EvaluateModelNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "predictions", Type: workflow.PortSeries, Required: true},
		{Name: "actual", Type: workflow.PortSeries, Required: true},
		{Name: "model_id", Type: workflow.PortAny, Required: true},
	}
}

func (n *EvaluateModelNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "evaluation_report", Type: workflow.PortSeries},
	}
}

func (n *EvaluateModelNode) ParamSchema() []workflow.ParamDef { return nil }

func (n *EvaluateModelNode) Validate() error { return nil }

func (n *EvaluateModelNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	predMap, ok := inputs["predictions"].(map[string][]float64)
	if !ok {
		return nil, fmt.Errorf("evaluate_model: predictions must be map[string][]float64")
	}
	actualMap, ok := inputs["actual"].(map[string][]float64)
	if !ok {
		return nil, fmt.Errorf("evaluate_model: actual must be map[string][]float64")
	}

	// Extract first series from each map
	var preds, actuals []float64
	for _, v := range predMap {
		preds = v
		break
	}
	for _, v := range actualMap {
		actuals = v
		break
	}

	if len(preds) != len(actuals) || len(preds) == 0 {
		return nil, fmt.Errorf("evaluate_model: predictions and actuals must have same non-zero length (%d vs %d)", len(preds), len(actuals))
	}

	nObs := float64(len(preds))

	// MSE, MAE
	var mse, mae float64
	for i := range preds {
		diff := preds[i] - actuals[i]
		mse += diff * diff
		mae += math.Abs(diff)
	}
	mse /= nObs
	mae /= nObs
	rmse := math.Sqrt(mse)

	// IC (Pearson correlation)
	var sumP, sumA, sumPP, sumAA, sumPA float64
	for i := range preds {
		sumP += preds[i]
		sumA += actuals[i]
		sumPP += preds[i] * preds[i]
		sumAA += actuals[i] * actuals[i]
		sumPA += preds[i] * actuals[i]
	}
	ic := (nObs*sumPA - sumP*sumA) / math.Sqrt((nObs*sumPP-sumP*sumP)*(nObs*sumAA-sumA*sumA))
	if math.IsNaN(ic) {
		ic = 0
	}

	metrics := map[string][]float64{
		"mse":  {mse},
		"mae":  {mae},
		"rmse": {rmse},
		"ic":   {ic},
	}

	// Note: ModelRegistry.CreateEvaluation is a future API — not yet implemented.
	// When available, persistence of evaluation metrics will be wired here.
	_ = inputs["model_id"] // consumed for future persistence

	return map[string]any{
		"evaluation_report": metrics,
	}, nil
}
