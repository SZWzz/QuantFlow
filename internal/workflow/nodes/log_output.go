package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/workflow"
)

// LogOutputNode logs its input values via slog and passes them through.
type LogOutputNode struct {
	id     string
	params map[string]any
}

// NewLogOutputNode creates a new LogOutputNode.
func NewLogOutputNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &LogOutputNode{id: id, params: params}, nil
}

func (n *LogOutputNode) ID() string       { return n.id }
func (n *LogOutputNode) NodeType() string { return "log_output" }
func (n *LogOutputNode) Category() string { return "output" }

func (n *LogOutputNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "input", Type: workflow.PortAny, Required: true},
	}
}

func (n *LogOutputNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "output", Type: workflow.PortAny, Required: false},
	}
}

func (n *LogOutputNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "prefix", Type: "string", Default: "", Description: "Log message prefix"},
	}
}

func (n *LogOutputNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	prefix := ""
	if p, ok := n.params["prefix"]; ok {
		prefix = fmt.Sprint(p)
	}
	if p, ok := params["prefix"]; ok {
		prefix = fmt.Sprint(p)
	}
	for k, v := range inputs {
		slog.Info(fmt.Sprintf("%s%s", prefix, k), "value", fmt.Sprint(v))
	}
	return inputs, nil
}

func (n *LogOutputNode) Validate() error { return nil }
