package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"

	"github.com/google/uuid"
)

// SubWorkflowNode executes a child workflow by ID.
// This is a stub — it returns a mock execution_id and status "pending".
// Will be wired to the actual WorkflowEngine later.
type SubWorkflowNode struct {
	id     string
	params map[string]any
}

// NewSubWorkflowNode creates a new SubWorkflowNode.
func NewSubWorkflowNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &SubWorkflowNode{id: id, params: params}, nil
}

func (n *SubWorkflowNode) ID() string       { return n.id }
func (n *SubWorkflowNode) NodeType() string { return "sub_workflow" }
func (n *SubWorkflowNode) Category() string { return "control" }

func (n *SubWorkflowNode) InputPorts() []workflow.PortDefinition {
	return nil
}

func (n *SubWorkflowNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "execution_id", Type: workflow.PortString, Required: false},
		{Name: "status", Type: workflow.PortString, Required: false},
	}
}

func (n *SubWorkflowNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "workflow_id", Type: "string", Default: "",
			Description: "ID of the sub-workflow to execute"},
	}
}

func (n *SubWorkflowNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any) (map[string]any, error) {
	workflowID := getStringParam(params, "workflow_id", "")
	if workflowID == "" {
		return nil, fmt.Errorf("sub_workflow: workflow_id param is required")
	}

	// Stub: return mock execution_id and "pending" status.
	// TODO: wire to actual WorkflowEngine.Submit(workflowID).
	executionID := uuid.New().String()

	return map[string]any{
		"execution_id": executionID,
		"status":       "pending",
	}, nil
}

func (n *SubWorkflowNode) Validate() error {
	workflowID := getStringParam(n.params, "workflow_id", "")
	if workflowID == "" {
		return fmt.Errorf("sub_workflow: workflow_id is required")
	}
	return nil
}
