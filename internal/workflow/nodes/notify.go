package nodes

import (
	"context"
	"fmt"
	"log/slog"
	"quantflow/internal/workflow"
)

// NotifyNode sends a notification or log message.
type NotifyNode struct {
	id     string
	params map[string]any
}

// NewNotifyNode creates a new NotifyNode.
func NewNotifyNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &NotifyNode{id: id, params: params}, nil
}

func (n *NotifyNode) ID() string       { return n.id }
func (n *NotifyNode) NodeType() string { return "notify" }
func (n *NotifyNode) Category() string { return "notify" }

func (n *NotifyNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "message", Type: workflow.PortString, Required: false},
	}
}

func (n *NotifyNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "success", Type: workflow.PortBoolean, Required: false},
	}
}

func (n *NotifyNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "level", Type: "string", Default: "info", Description: "info, warn, error, trade"},
		{Name: "title", Type: "string", Default: "", Description: "Notification title"},
		{Name: "body", Type: "string", Default: "", Description: "Notification body"},
	}
}

func (n *NotifyNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	title := getStringParam(params, "title", "")
	if title == "" {
		if v, ok := inputs["message"]; ok {
			title = fmt.Sprintf("%v", v)
		}
	}
	if title == "" {
		return nil, fmt.Errorf("notify: title is required")
	}
	slog.Info("notification", "title", title)
	return map[string]any{"success": true}, nil
}

func (n *NotifyNode) Validate() error { return nil }
