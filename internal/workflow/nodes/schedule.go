package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// ScheduleNode schedules a workflow to run on a cron expression.
type ScheduleNode struct {
	id     string
	params map[string]any
}

// NewScheduleNode creates a new ScheduleNode.
func NewScheduleNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &ScheduleNode{id: id, params: params}, nil
}

func (n *ScheduleNode) ID() string        { return n.id }
func (n *ScheduleNode) NodeType() string  { return "schedule" }
func (n *ScheduleNode) Category() string  { return "schedule" }

func (n *ScheduleNode) InputPorts() []workflow.PortDefinition { return nil }

func (n *ScheduleNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "task_id", Type: workflow.PortString, Required: false},
	}
}

func (n *ScheduleNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "cron_expr", Type: "string", Default: "0 9 * * 1-5", Description: "Cron expression"},
		{Name: "workflow_id", Type: "string", Default: "", Description: "Workflow ID to trigger"},
	}
}

func (n *ScheduleNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	wfID := getStringParam(params, "workflow_id", "")
	if wfID == "" {
		return nil, fmt.Errorf("schedule: workflow_id required")
	}
	return map[string]any{"task_id": fmt.Sprintf("sched-%s", n.id)}, nil
}

func (n *ScheduleNode) Validate() error { return nil }
