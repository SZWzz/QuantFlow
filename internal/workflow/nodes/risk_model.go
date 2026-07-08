package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"

	"quantflow/internal/normalize"
	pb "quantflow/internal/python/proto"
	"quantflow/internal/python"
	"quantflow/internal/workflow"
)

// RiskModelNode computes GARCH volatility or covariance matrix via Python sidecar.
// When the Python sidecar is unavailable, it falls back to historical volatility
// (standard deviation of log returns).
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
	modelType := getStringParam(params, "model_type", "garch")
	rawData := inputs["returns_data"]

	// 1. Compute log returns from OHLCV bars or use pre-computed returns.
	returns := computeReturns(rawData)
	if len(returns) < 2 {
		return nil, fmt.Errorf("risk_model: insufficient data points (%d), need at least 2", len(returns))
	}

	// 2. Try Python sidecar if available.
	if nctx != nil && nctx.MLClient != nil {
		if mlClient, ok := nctx.MLClient.(*python.MLClient); ok && mlClient != nil {
			result, err := n.callPythonRiskModel(ctx, mlClient, modelType, returns, params)
			if err == nil && result != nil {
				return result, nil
			}
			slog.Warn("risk_model: Python sidecar failed, falling back to historical volatility", "error", err)
		}
	}

	// 3. Fallback: compute historical volatility (stddev of returns).
	return n.fallbackVolatility(returns, modelType), nil
}

// callPythonRiskModel sends a RiskModel request to the Python sidecar.
func (n *RiskModelNode) callPythonRiskModel(ctx context.Context, client *python.MLClient, modelType string, returns []float64, params map[string]any) (map[string]any, error) {
	// Encode returns as JSON float array (TODO: replace with Arrow IPC encoding
	// when Apache Arrow Go library is added).
	returnsJSON, err := json.Marshal(returns)
	if err != nil {
		return nil, fmt.Errorf("risk_model: marshal returns: %w", err)
	}

	req := &pb.RiskModelRequest{
		ModelType:   modelType,
		ReturnsData: returnsJSON,
		Params: map[string]string{
			"p":      getStringParam(params, "p", "1"),
			"q":      getStringParam(params, "q", "1"),
			"method": getStringParam(params, "method", "ledoit_wolf"),
		},
	}

	resp, err := client.RiskModel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("risk_model: gRPC call failed: %w", err)
	}

	// Build volatility series from response (single value per model period).
	volatility := make([]float64, 0)
	if resp.ResultData != nil && len(resp.ResultData) > 0 {
		// If Python returned result data, extract the volatility column.
		volatility = append(volatility, 0) // placeholder
	}

	// Use annualized volatility from metrics if available.
	if annVol, ok := resp.Metrics["annualized_volatility"]; ok {
		volatility = []float64{annVol}
	}

	metrics := make(map[string]float64)
	for k, v := range resp.Metrics {
		metrics[k] = v
	}

	return map[string]any{
		"volatility":        volatility,
		"covariance_matrix": nil, // covariance matrix not available in fallback
		"model_metrics":     metrics,
	}, nil
}

// fallbackVolatility computes historical volatility as the standard deviation
// of log returns, annualized with sqrt(252) for daily data.
func (n *RiskModelNode) fallbackVolatility(returns []float64, modelType string) map[string]any {
	if modelType == "covariance" {
		return map[string]any{
			"volatility":        []float64{stdDev(returns) * math.Sqrt(252)},
			"covariance_matrix": nil,
			"model_metrics": map[string]float64{
				"historical_volatility": stdDev(returns),
				"annualized_volatility": stdDev(returns) * math.Sqrt(252),
				"data_points":           float64(len(returns)),
				"model":                 0, // 0 = fallback
			},
		}
	}

	return map[string]any{
		"volatility":        []float64{stdDev(returns) * math.Sqrt(252)},
		"covariance_matrix": nil,
		"model_metrics": map[string]float64{
			"historical_volatility": stdDev(returns),
			"annualized_volatility": stdDev(returns) * math.Sqrt(252),
			"data_points":           float64(len(returns)),
		},
	}
}

// computeReturns extracts log returns from OHLCV bars or float64 slices.
func computeReturns(data any) []float64 {
	// Case 1: pre-computed []float64 returns.
	if floats, ok := data.([]float64); ok {
		return floats
	}

	// Case 2: []normalize.OHLCVBar — compute returns from Close prices.
	if bars, ok := data.([]normalize.OHLCVBar); ok {
		return returnsFromBars(bars)
	}

	// Case 3: []any containing OHLCVBar.
	if rawSlice, ok := data.([]any); ok {
		bars := make([]normalize.OHLCVBar, 0, len(rawSlice))
		for _, item := range rawSlice {
			if bar, ok := item.(normalize.OHLCVBar); ok {
				bars = append(bars, bar)
			}
		}
		if len(bars) > 0 {
			return returnsFromBars(bars)
		}
	}

	return nil
}

// returnsFromBars computes log returns from OHLCV bar Close prices.
func returnsFromBars(bars []normalize.OHLCVBar) []float64 {
	if len(bars) < 2 {
		return nil
	}
	returns := make([]float64, 0, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			r := (bars[i].Close - bars[i-1].Close) / bars[i-1].Close
			returns = append(returns, r)
		}
	}
	return returns
}

// stdDev computes the population standard deviation.
func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	sumSq := 0.0
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)))
}
