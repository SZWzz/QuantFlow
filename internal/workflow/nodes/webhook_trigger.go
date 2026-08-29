package nodes

import (
	"context"
	"fmt"
	"quantflow/internal/workflow"

	"github.com/google/uuid"
)

// WebhookTriggerNode is a trigger node that waits for an external HTTP request.
// When triggered, it passes the request body/payload as outputs.
type WebhookTriggerNode struct {
	id     string
	params map[string]any
}

func NewWebhookTriggerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &WebhookTriggerNode{id: id, params: params}, nil
}

func (n *WebhookTriggerNode) ID() string       { return n.id }
func (n *WebhookTriggerNode) NodeType() string { return "webhook_trigger" }
func (n *WebhookTriggerNode) Category() string { return "schedule" }

func (n *WebhookTriggerNode) InputPorts() []workflow.PortDefinition {
	return nil // trigger nodes have no inputs
}

func (n *WebhookTriggerNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "payload", Type: workflow.PortAny, Required: false},
		{Name: "headers", Type: workflow.PortAny, Required: false},
	}
}

func (n *WebhookTriggerNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{
			Name: "path", Type: "string", Default: "/webhook",
			Description: "URL path for this webhook",
		},
	}
}

func (n *WebhookTriggerNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	path := getStringParam(params, "path", "/webhook")

	// When executed by the engine (not via HTTP trigger), return a waiting state.
	// The actual execution happens when the webhook endpoint is hit.
	return map[string]any{
		"payload": nil,
		"status":  fmt.Sprintf("waiting for POST %s", path),
	}, nil
}

func (n *WebhookTriggerNode) Validate() error {
	return nil
}

// Ensure uuid import is used
var _ = uuid.New
