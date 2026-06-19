package nodes

import (
	"context"
	"fmt"
	"log/slog"

	"quantflow/internal/workflow"
)

// AlertNode evaluates a condition against an input value and fires an alert.
type AlertNode struct {
	id     string
	params map[string]any
}

// NewAlertNode creates a new AlertNode.
func NewAlertNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &AlertNode{id: id, params: params}, nil
}

func (n *AlertNode) ID() string        { return n.id }
func (n *AlertNode) NodeType() string  { return "alert" }
func (n *AlertNode) Category() string  { return "notify" }

func (n *AlertNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "value", Type: workflow.PortNumber, Required: true},
	}
}

func (n *AlertNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "triggered", Type: workflow.PortBoolean, Required: false},
		{Name: "value", Type: workflow.PortNumber, Required: false},
	}
}

func (n *AlertNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "condition", Type: "string", Default: "gt", Description: "gt, lt, gte, lte, eq"},
		{Name: "threshold", Type: "number", Default: "0", Description: "Threshold value"},
		{Name: "message", Type: "string", Default: "Alert triggered", Description: "Alert message"},
	}
}

func (n *AlertNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	var value float64
	if v, ok := inputs["value"]; ok {
		switch val := v.(type) {
		case float64:
			value = val
		case int:
			value = float64(val)
		default:
			fmt.Sscanf(fmt.Sprintf("%v", val), "%f", &value)
		}
	}
	cond := getStringParam(params, "condition", "gt")
	threshold := getFloatParam(params, "threshold", 0)
	triggered := false
	switch cond {
	case "gt":
		triggered = value > threshold
	case "lt":
		triggered = value < threshold
	case "gte":
		triggered = value >= threshold
	case "lte":
		triggered = value <= threshold
	case "eq":
		triggered = value == threshold
	}
	if triggered {
		slog.Warn("alert triggered", "value", value, "threshold", threshold)
	}
	return map[string]any{"triggered": triggered, "value": value}, nil
}

func (n *AlertNode) Validate() error { return nil }
