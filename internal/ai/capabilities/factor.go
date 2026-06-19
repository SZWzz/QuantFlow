package capabilities

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/ai"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// RegisterFactorCapabilities registers list_factors and compute_factor capabilities.
// These capabilities forward calls to the Python FactorService via gRPC.
func RegisterFactorCapabilities(reg *ai.CapabilityRegistry, bridge *python.PythonBridge) {
	reg.Register(&ai.Capability{
		Name:        "list_factors",
		Description: "List all available alpha factors with their categories and descriptions. Use this to discover what factors can be computed.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"category": {"type": "string", "description": "Optional filter: momentum, trend, volatility, volume, cross_sectional"}
			}
		}`,
		),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			if bridge == nil {
				return "Python sidecar is not connected. Cannot list factors.", nil
			}
			resp, err := bridge.FactorClient.ListFactors(ctx, &pb.ListFactorsRequest{})
			if err != nil {
				return "", fmt.Errorf("list_factors: %w", err)
			}

			var catFilter string
			if len(args) > 0 {
				var params struct {
					Category string `json:"category"`
				}
				json.Unmarshal(args, &params)
				catFilter = params.Category
			}

			var lines []string
			for _, fm := range resp.Factors {
				if catFilter != "" && fm.Category != catFilter {
					continue
				}
				lines = append(lines, fmt.Sprintf("- %s [%s]: %s", fm.Name, fm.Category, fm.Description))
			}
			if len(lines) == 0 {
				return "No factors found.", nil
			}
			result := "Available factors:\n" + join(lines, "\n")
			return result, nil
		},
	})

	reg.Register(&ai.Capability{
		Name:        "compute_factor",
		Description: "Compute an alpha factor for one or more symbols. Requires factor name and symbol. Returns factor values.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"factor_name": {"type": "string", "description": "Name of the factor, e.g. momentum_20d, rsi_14"},
				"symbol": {"type": "string", "description": "Stock symbol, e.g. 000001.SZ"}
			},
			"required": ["factor_name", "symbol"]
		}`,
		),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				FactorName string `json:"factor_name"`
				Symbol     string `json:"symbol"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("compute_factor: %w", err)
			}
			if bridge == nil {
				return "Python sidecar is not connected. Cannot compute factor.", nil
			}
			resp, err := bridge.FactorClient.ComputeFactor(ctx, &pb.ComputeFactorRequest{
				FactorName: params.FactorName,
				Symbols:    []string{params.Symbol},
			})
			if err != nil {
				return "", fmt.Errorf("compute_factor: %w", err)
			}
			if resp.Error != "" {
				return fmt.Sprintf("Error: %s", resp.Error), nil
			}
			result, _ := json.Marshal(resp.Results)
			return fmt.Sprintf("Factor %s computed in %dms. Results: %s", params.FactorName, resp.ComputeTimeMs, string(result)), nil
		},
	})
}

func join(lines []string, sep string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, l := range lines[1:] {
		result += sep + l
	}
	return result
}
