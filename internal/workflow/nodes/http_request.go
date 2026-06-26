package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/workflow"
)

// HTTPRequestNode performs an HTTP request. Currently returns a mock
// response (status 200, body "{}") — real HTTP client will be wired
// when the network layer is available.
type HTTPRequestNode struct {
	id     string
	params map[string]any
}

// NewHTTPRequestNode creates a new HTTPRequestNode.
func NewHTTPRequestNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &HTTPRequestNode{id: id, params: params}, nil
}

func (n *HTTPRequestNode) ID() string       { return n.id }
func (n *HTTPRequestNode) NodeType() string { return "http_request" }
func (n *HTTPRequestNode) Category() string { return "utility" }

func (n *HTTPRequestNode) InputPorts() []workflow.PortDefinition {
	return nil // no dynamic inputs — params supply url/method/headers/body
}

func (n *HTTPRequestNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "response", Type: workflow.PortAny, Required: false},
		{Name: "status", Type: workflow.PortNumber, Required: false},
	}
}

func (n *HTTPRequestNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "url", Type: "string", Default: "", Description: "Request URL"},
		{Name: "method", Type: "string", Default: "GET", Description: "HTTP method: GET or POST"},
		{Name: "headers", Type: "string", Default: "", Description: "JSON-encoded request headers"},
		{Name: "body", Type: "string", Default: "", Description: "Request body (for POST)"},
	}
}

func (n *HTTPRequestNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	url := getStringParam(params, "url", "")
	if url == "" {
		return nil, fmt.Errorf("http_request: url parameter is required")
	}
	method := getStringParam(params, "method", "GET")
	if method != "GET" && method != "POST" {
		return nil, fmt.Errorf("http_request: method must be GET or POST, got %q", method)
	}

	// TODO: wire real HTTP client when network layer is available.
	// For now return a mock 200 with empty JSON body.
	_ = getStringParam(params, "headers", "")
	_ = getStringParam(params, "body", "")

	return map[string]any{
		"response": "{}",
		"status":   float64(200),
	}, nil
}

func (n *HTTPRequestNode) Validate() error {
	method := getStringParam(n.params, "method", "GET")
	if method != "GET" && method != "POST" {
		return fmt.Errorf("http_request: method must be GET or POST, got %q", method)
	}
	return nil
}
