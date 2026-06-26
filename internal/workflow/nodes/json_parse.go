package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// JSONParseNode parses a JSON string and optionally extracts a value
// at the given JSON path. Currently returns a mock parsed result.
type JSONParseNode struct {
	id     string
	params map[string]any
}

// NewJSONParseNode creates a new JSONParseNode.
func NewJSONParseNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &JSONParseNode{id: id, params: params}, nil
}

func (n *JSONParseNode) ID() string       { return n.id }
func (n *JSONParseNode) NodeType() string { return "json_parse" }
func (n *JSONParseNode) Category() string { return "utility" }

func (n *JSONParseNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "json_str", Type: workflow.PortAny, Required: true},
	}
}

func (n *JSONParseNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "parsed", Type: workflow.PortSeries, Required: false},
		{Name: "value", Type: workflow.PortAny, Required: false},
	}
}

func (n *JSONParseNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "path", Type: "string", Default: "", Description: "JSONPath or key to extract (empty returns entire parsed object)"},
	}
}

func (n *JSONParseNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	jsonStr, ok := inputs["json_str"]
	if !ok || jsonStr == nil {
		return nil, fmt.Errorf("json_parse: json_str input is required")
	}

	path := getStringParam(params, "path", "")
	_ = path
	_ = fmt.Sprint(jsonStr) // consume for when real parser is wired

	// TODO: wire real JSON parser with path extraction.
	// For now return mock parsed data.
	return map[string]any{
		"parsed": []float64{1.0, 2.0, 3.0},
		"value":  "{}",
	}, nil
}

func (n *JSONParseNode) Validate() error {
	return nil
}
