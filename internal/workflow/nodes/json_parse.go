package nodes

import (
	"context"
	"encoding/json"
	"fmt"

	"quantflow/internal/workflow"
)

type JSONParseNode struct {
	id     string
	params map[string]any
}

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
		{Name: "parsed", Type: workflow.PortAny, Required: false},
		{Name: "value", Type: workflow.PortAny, Required: false},
	}
}

func (n *JSONParseNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "path", Type: "string", Default: "", Description: "Key to extract (empty returns entire parsed object)"},
	}
}

func (n *JSONParseNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	raw, ok := inputs["json_str"]
	if !ok || raw == nil {
		return nil, fmt.Errorf("json_parse: json_str input is required")
	}

	var jsonStr string
	switch v := raw.(type) {
	case string:
		jsonStr = v
	case []byte:
		jsonStr = string(v)
	default:
		jsonStr = fmt.Sprintf("%v", v)
	}

	var parsed any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("json_parse: invalid JSON: %w", err)
	}

	path := getStringParam(params, "path", "")
	if path == "" {
		return map[string]any{"parsed": parsed, "value": parsed}, nil
	}

	if m, ok := parsed.(map[string]any); ok {
		if val, exists := m[path]; exists {
			return map[string]any{"parsed": parsed, "value": val}, nil
		}
		return nil, fmt.Errorf("json_parse: key %q not found in parsed object", path)
	}

	return nil, fmt.Errorf("json_parse: path extraction requires a JSON object, got %T", parsed)
}

func (n *JSONParseNode) Validate() error {
	return nil
}
