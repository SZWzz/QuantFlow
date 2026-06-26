package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/python"
	"quantflow/internal/python/proto"
	"quantflow/internal/workflow"
)

// AlphaMiningNode discovers new alpha factors via genetic programming.
// It consumes a pool of known factors and OHLCV data, then evolves new factor
// expressions optimized for the selected fitness metric (IC, IR, Sharpe, etc.).
type AlphaMiningNode struct {
	id     string
	params map[string]any
}

// NewAlphaMiningNode creates a new AlphaMiningNode.
func NewAlphaMiningNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &AlphaMiningNode{id: id, params: params}
	return n, n.Validate()
}

func (n *AlphaMiningNode) ID() string       { return n.id }
func (n *AlphaMiningNode) NodeType() string { return "alpha_mining" }
func (n *AlphaMiningNode) Category() string { return "ml" }

func (n *AlphaMiningNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "factor_pool", Type: workflow.PortSeries, Required: true},
		{Name: "ohlcv_data", Type: workflow.PortSeries, Required: true},
	}
}

func (n *AlphaMiningNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "new_factors", Type: workflow.PortSeries},
		{Name: "factor_scores", Type: workflow.PortSeries},
	}
}

func (n *AlphaMiningNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "population_size", Type: "int", Default: "200", Description: "GP population size"},
		{Name: "generations", Type: "int", Default: "50", Description: "GP generations"},
		{Name: "top_k", Type: "int", Default: "20", Description: "Number of top factors to return"},
		{Name: "crossover_rate", Type: "float", Default: "0.7", Description: "Crossover probability"},
		{Name: "mutation_rate", Type: "float", Default: "0.1", Description: "Mutation probability"},
		{Name: "fitness_metric", Type: "string", Default: "ic", Description: "ic/ir/sharpe/composite"},
	}
}

func (n *AlphaMiningNode) Validate() error { return nil }

func (n *AlphaMiningNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	var bridge *python.PythonBridge
	if nctx != nil {
		if nctx.Bridge != nil {
			bridge, _ = nctx.Bridge.(*python.PythonBridge)
		}
	}
	if bridge == nil {
		return nil, fmt.Errorf("alpha_mining: PythonBridge not set")
	}

	factorPool := inputs["factor_pool"].(map[string][]float64)
	ohlcv := inputs["ohlcv_data"].(map[string][]float64)

	// Encode factor data as Arrow-compatible JSON for the gRPC call
	factorJSON, _ := json.Marshal(factorPool)
	returnsJSON, _ := json.Marshal(ohlcv)

	mlClient := python.NewMLClient(bridge)
	req := &proto.AlphaMiningRequest{
		BaseFactorNames: getFactorNames(factorPool),
		FactorData:      factorJSON,
		ReturnsData:     returnsJSON,
		PopulationSize:  int32(getIntParam(params, "population_size", 200)),
		Generations:     int32(getIntParam(params, "generations", 50)),
		CrossoverRate:   getFloatParam(params, "crossover_rate", 0.7),
		MutationRate:    getFloatParam(params, "mutation_rate", 0.1),
		FitnessMetric:   getStringParam(params, "fitness_metric", "ic"),
		TopK:            int32(getIntParam(params, "top_k", 20)),
	}

	resp, err := mlClient.AlphaMining(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("alpha_mining: %w", err)
	}

	// Convert response to output format
	formulas := make(map[string][]float64)
	scores := make(map[string][]float64)
	for i, f := range resp.Factors {
		key := fmt.Sprintf("factor_%d", i)
		formulas[key] = nil // placeholder for formula string
		scores[key] = []float64{f.Ic, f.Ir, f.Sharpe}
	}

	return map[string]any{
		"new_factors":   formulas,
		"factor_scores": scores,
	}, nil
}

// getFactorNames extracts the factor names from a factor pool map.
func getFactorNames(factorPool map[string][]float64) []string {
	names := make([]string, 0, len(factorPool))
	for name := range factorPool {
		names = append(names, name)
	}
	return names
}
