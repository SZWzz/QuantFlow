package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/normalize"
	"quantflow/internal/workflow"
	"strings"
)

// DataNormalizeNode normalizes input data using a configured field mapper.
type DataNormalizeNode struct {
	id     string
	params map[string]any
}

// NewDataNormalizeNode creates a new DataNormalizeNode.
func NewDataNormalizeNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &DataNormalizeNode{id: id, params: params}, nil
}

func (n *DataNormalizeNode) ID() string       { return n.id }
func (n *DataNormalizeNode) NodeType() string { return "data_normalize" }
func (n *DataNormalizeNode) Category() string { return "data" }

func (n *DataNormalizeNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "raw", Type: workflow.PortAny, Required: true},
		{Name: "status", Type: workflow.PortString, Required: false},
		{Name: "order_type", Type: workflow.PortString, Required: false},
		{Name: "broker", Type: workflow.PortString, Required: false},
	}
}

func (n *DataNormalizeNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "ohlcv", Type: workflow.PortOHLCV, Required: false},
		{Name: "normalized_status", Type: workflow.PortString, Required: false},
		{Name: "normalized_type", Type: workflow.PortString, Required: false},
	}
}

func (n *DataNormalizeNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "source", Type: "string", Default: "", Description: "Data source name (e.g. eastmoney, ibkr, binance)"},
		{Name: "target", Type: "string", Default: "ohlcv", Description: "Normalization target: ohlcv, order_status, order_type"},
		{Name: "mapping", Type: "string", Default: `{}`, Description: "JSON field mapping: canonical->raw (e.g. {\"symbol\":\"code\",\"open\":\"opn\"})"},
	}
}

func (n *DataNormalizeNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	source := getStringParam(params, "source", "")
	target := getStringParam(params, "target", "ohlcv")

	switch target {
	case "ohlcv":
		return n.normalizeOHLCV(ctx, inputs, params, source)
	case "order_status":
		return n.normalizeOrderStatus(ctx, inputs, source)
	case "order_type":
		return n.normalizeOrderType(ctx, inputs, source)
	default:
		return nil, fmt.Errorf("data_normalize: unknown target %q", target)
	}
}

func (n *DataNormalizeNode) normalizeOHLCV(ctx context.Context, inputs map[string]any, params map[string]any, source string) (map[string]any, error) {
	raw, ok := inputs["raw"]
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing input 'raw'")
	}

	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("data_normalize: 'raw' must be a map[string]any")
	}

	mapping := getStringParam(params, "mapping", "{}")
	columns, err := parseJSONMapping(mapping)
	if err != nil {
		return nil, fmt.Errorf("data_normalize: invalid mapping: %w", err)
	}

	mapper := normalize.NewOHLCVMapper(source, columns)
	bar, err := mapper.Parse(rawMap)
	if err != nil {
		return nil, fmt.Errorf("data_normalize: %w", err)
	}

	return map[string]any{"ohlcv": bar}, nil
}

func (n *DataNormalizeNode) normalizeOrderStatus(ctx context.Context, inputs map[string]any, source string) (map[string]any, error) {
	status, ok := inputs["status"].(string)
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing or non-string input 'status'")
	}

	broker := source
	if b, ok := inputs["broker"].(string); ok && b != "" {
		broker = b
	}

	mapper := normalize.NewOrderStatusMapper(broker)
	return map[string]any{"normalized_status": mapper.Map(status)}, nil
}

func (n *DataNormalizeNode) normalizeOrderType(ctx context.Context, inputs map[string]any, source string) (map[string]any, error) {
	orderType, ok := inputs["order_type"].(string)
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing or non-string input 'order_type'")
	}

	broker := source
	if b, ok := inputs["broker"].(string); ok && b != "" {
		broker = b
	}

	mapper := normalize.NewOrderTypeMapper(broker)
	return map[string]any{"normalized_type": mapper.Map(orderType)}, nil
}

func (n *DataNormalizeNode) Validate() error {
	return nil
}

// parseJSONMapping parses a JSON string like {"symbol":"code"} into a Go map.
func parseJSONMapping(s string) (map[string]string, error) {
	result := make(map[string]string)
	if s == "" || s == "{}" {
		return result, nil
	}
	// Simple state-machine parser for flat string-string maps
	// Format: {"key1":"val1","key2":"val2"}
	body := s[1 : len(s)-1] // strip { }
	const (
		stExpectKey = iota
		stReadKey
		stExpectColon
		stExpectVal
		stReadVal
		stExpectNext
	)
	state := stExpectKey
	var key, val strings.Builder
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch state {
		case stExpectKey:
			if c == '"' {
				key.Reset()
				state = stReadKey
			}
		case stReadKey:
			if c == '"' {
				state = stExpectColon
			} else {
				key.WriteByte(c)
			}
		case stExpectColon:
			if c == ':' {
				state = stExpectVal
			}
		case stExpectVal:
			if c == '"' {
				val.Reset()
				state = stReadVal
			}
		case stReadVal:
			if c == '"' {
				result[key.String()] = val.String()
				state = stExpectNext
			} else {
				val.WriteByte(c)
			}
		case stExpectNext:
			if c == ',' {
				state = stExpectKey
			}
		}
	}
	return result, nil
}
