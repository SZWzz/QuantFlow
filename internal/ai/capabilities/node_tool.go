package capabilities

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/ai"
	"quantflow/internal/workflow"
)

// paramTypeToJSONSchema maps workflow ParamDef types to JSON Schema types.
func paramTypeToJSONSchema(ptype string) string {
	switch ptype {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "bool":
		return "boolean"
	case "string_array":
		return "array"
	default:
		return "string"
	}
}

// paramSchemaToJSONSchema converts a node's ParamSchema to a JSON Schema
// object suitable for LLM function-calling tool definitions.
func paramSchemaToJSONSchema(params []workflow.ParamDef) json.RawMessage {
	props := make(map[string]any, len(params))
	required := make([]string, 0, len(params))

	for _, p := range params {
		schema := map[string]any{
			"type":        paramTypeToJSONSchema(p.Type),
			"description": p.Description,
		}
		if p.Default != nil {
			schema["default"] = p.Default
		}
		// For string_array, add items type.
		if p.Type == "string_array" {
			schema["items"] = map[string]string{"type": "string"}
		}
		props[p.Name] = schema
		if p.Description == "" {
			// Parameters without explicit descriptions are likely required.
			// We treat all params as optional to avoid LLM validation failures.
		}
		// All params are currently treated as optional — LLMs handle missing ones gracefully.
		_ = required
	}

	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	data, _ := json.Marshal(schema)
	return data
}

// WhitelistedNodes defines which node types are exposed as LLM tools.
// We filter to ~25 most useful nodes — sending all 196 nodes would blow LLM context.
var WhitelistedNodes = map[string]bool{
	// Technical indicators
	"sma": true, "ema": true, "macd": true, "rsi": true,
	"bollinger": true, "atr": true, "kdj": true,
	// Data
	"data_loader": true, "merge": true, "filter": true,
	// Signals
	"cross_signal": true, "cross_over": true, "threshold_signal": true,
	// Arithmetic / Comparison
	"arithmetic": true, "compare": true,
	// Trading
	"place_order": true, "position_query": true, "order_query": true,
	// Portfolio / Risk
	"portfolio_summary": true, "risk_metrics": true,
	// Research
	"sentiment": true, "stock_research": true,
	// ML
	"feature_engineer": true, "predict": true,
}

// RegisterAllNodeCapabilities iterates the NodeRegistry and registers whitelisted
// node types as capabilities in the AI capability registry.
func RegisterAllNodeCapabilities(aiReg *ai.CapabilityRegistry, nodeReg *workflow.NodeRegistry) error {
	metas := nodeReg.ListAll()
	registered := 0
	for _, meta := range metas {
		if !WhitelistedNodes[meta.NodeType] {
			continue
		}
		// Create a temporary node to extract schema + get a constructor-like pattern.
		tmpNode, err := nodeReg.Create(meta.NodeType, "_tool", nil)
		if err != nil {
			fmt.Printf("[node_tool] skip %s: %v\n", meta.NodeType, err)
			continue
		}

		name := "node_" + meta.NodeType
		description := fmt.Sprintf("Workflow node: %s (category: %s)", meta.NodeType, meta.Category)
		params := paramSchemaToJSONSchema(tmpNode.ParamSchema())

		cap := &ai.Capability{
			Name:        name,
			Description: description,
			Parameters:  params,
			Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
				var paramsMap map[string]any
				if len(args) > 0 {
					if err := json.Unmarshal(args, &paramsMap); err != nil {
						return "", fmt.Errorf("parse args: %w", err)
					}
				}
				node, err := nodeReg.Create(meta.NodeType, "_tool_"+name, paramsMap)
				if err != nil {
					return "", fmt.Errorf("create node: %w", err)
				}
				outputs, err := node.Execute(ctx, nil, paramsMap, nil)
				if err != nil {
					return "", fmt.Errorf("execute: %w", err)
				}
				data, _ := json.Marshal(outputs)
				return string(data), nil
			},
		}

		if err := aiReg.Register(cap); err != nil {
			fmt.Printf("[node_tool] skip %s: %v\n", meta.NodeType, err)
			continue
		}
		registered++
	}
	fmt.Printf("[node_tool] registered %d node capabilities\n", registered)
	return nil
}
