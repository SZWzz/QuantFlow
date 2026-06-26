package nodes

import (
	"context"
	"fmt"
	"math"
	"sort"

	"quantflow/internal/workflow"
)

// FeatureEngineerNode preprocesses factor data: normalization, NA filling,
// and lag alignment to prevent look-ahead bias.
type FeatureEngineerNode struct {
	id     string
	params map[string]any
}

// NewFeatureEngineerNode creates a new FeatureEngineerNode.
func NewFeatureEngineerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &FeatureEngineerNode{id: id, params: params}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *FeatureEngineerNode) ID() string       { return n.id }
func (n *FeatureEngineerNode) NodeType() string { return "feature_engineer" }
func (n *FeatureEngineerNode) Category() string { return "ml" }

func (n *FeatureEngineerNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv_data", Type: workflow.PortSeries, Required: false},
		{Name: "factors", Type: workflow.PortSeries, Required: true},
	}
}

func (n *FeatureEngineerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "feature_matrix", Type: workflow.PortSeries},
	}
}

func (n *FeatureEngineerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "method", Type: "string", Default: "standardize", Description: "normalization method: standardize/minmax/none"},
		{Name: "fill_na", Type: "string", Default: "zero", Description: "missing value strategy: zero/mean/forward"},
		{Name: "lag_periods", Type: "int", Default: "1", Description: "lag periods for feature→target alignment"},
	}
}

func (n *FeatureEngineerNode) Validate() error {
	method := getStringParam(n.params, "method", "standardize")
	if method != "standardize" && method != "minmax" && method != "none" {
		return fmt.Errorf("feature_engineer: invalid method '%s'", method)
	}
	return nil
}

func (n *FeatureEngineerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	factors, ok := inputs["factors"].(map[string][]float64)
	if !ok {
		return nil, fmt.Errorf("feature_engineer: factors must be map[string][]float64")
	}

	method := getStringParam(params, "method", "standardize")
	fillNA := getStringParam(params, "fill_na", "zero")
	lagPeriods := getIntParam(params, "lag_periods", 1)

	// Collect factor names in sorted order for deterministic output
	names := make([]string, 0, len(factors))
	for name := range factors {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build feature matrix: each factor becomes a column, each row is one time point
	var nRows int
	for _, vals := range factors {
		if len(vals) > nRows {
			nRows = len(vals)
		}
	}

	matrix := make(map[string][]float64)
	for _, name := range names {
		col := make([]float64, nRows)
		vals := factors[name]
		copy(col, vals)

		// Fill NA
		for i := len(vals); i < nRows; i++ {
			col[i] = fillValue(vals, fillNA)
		}
		for i := 0; i < len(col); i++ {
			if math.IsNaN(col[i]) {
				col[i] = fillValue(vals, fillNA)
			}
		}

		// Normalize
		switch method {
		case "standardize":
			col = standardize(col)
		case "minmax":
			col = minmaxScale(col)
		}

		// Lag alignment: shift features forward so t uses ≤t data
		if lagPeriods > 0 {
			shifted := make([]float64, nRows)
			for i := 0; i < nRows; i++ {
				srcIdx := i - lagPeriods
				if srcIdx < 0 {
					shifted[i] = fillValue(vals, fillNA)
				} else {
					shifted[i] = col[srcIdx]
				}
			}
			col = shifted
		}

		matrix[name] = col
	}

	return map[string]any{
		"feature_matrix": matrix,
	}, nil
}

// fillValue returns a fill value based on the strategy.
func fillValue(vals []float64, strategy string) float64 {
	switch strategy {
	case "mean":
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		if len(vals) > 0 {
			return sum / float64(len(vals))
		}
		return 0.0
	case "forward":
		if len(vals) > 0 {
			return vals[len(vals)-1]
		}
		return 0.0
	default: // zero
		return 0.0
	}
}

// standardize normalizes values to zero mean and unit variance.
func standardize(vals []float64) []float64 {
	n := len(vals)
	if n == 0 {
		return vals
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(n)

	std := 0.0
	for _, v := range vals {
		std += (v - mean) * (v - mean)
	}
	std = math.Sqrt(std / float64(n))

	result := make([]float64, n)
	if std == 0 {
		copy(result, vals)
		return result
	}
	for i, v := range vals {
		result[i] = (v - mean) / std
	}
	return result
}

// minmaxScale normalizes values to the [0, 1] range.
func minmaxScale(vals []float64) []float64 {
	if len(vals) == 0 {
		return vals
	}
	minV, maxV := vals[0], vals[0]
	for _, v := range vals {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	denom := maxV - minV
	result := make([]float64, len(vals))
	if denom == 0 {
		copy(result, vals)
		return result
	}
	for i, v := range vals {
		result[i] = (v - minV) / denom
	}
	return result
}
