package mcp

import (
	"context"
	"encoding/json"
	"quantflow/internal/ai"
)

// Handler dispatches MCP methods to the CapabilityRegistry.
type Handler struct {
	reg     *ai.CapabilityRegistry
	name    string
	version string
}

// NewHandler creates an MCP handler backed by a capability registry.
func NewHandler(reg *ai.CapabilityRegistry) *Handler {
	return &Handler{
		reg:     reg,
		name:    "quantflow",
		version: "0.1.0",
	}
}

// HandleMethod routes an MCP method name to its implementation.
// Returns JSON-encoded result or an MCP error code.
func (h *Handler) HandleMethod(method string, params json.RawMessage) (json.RawMessage, *Error) {
	switch method {
	case "initialize":
		return h.initialize(params)
	case "tools/list":
		return h.toolsList(params)
	case "tools/call":
		return h.toolsCall(params)
	case "notifications/initialized":
		return json.RawMessage("{}"), nil
	default:
		return nil, &Error{Code: -32601, Message: "method not found: " + method}
	}
}

// initialize responds with server capabilities.
func (h *Handler) initialize(_ json.RawMessage) (json.RawMessage, *Error) {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    h.name,
			"version": h.version,
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, &Error{Code: -32603, Message: "internal error: " + err.Error()}
	}
	return data, nil
}

// toolsList returns all registered capabilities formatted as MCP tools.
func (h *Handler) toolsList(_ json.RawMessage) (json.RawMessage, *Error) {
	defs := h.reg.ListForLLM(nil)
	tools := make([]map[string]any, 0, len(defs))
	for _, d := range defs {
		// Default to empty schema object if Parameters is empty/null.
		schema := json.RawMessage("{}")
		if len(d.Parameters) > 0 {
			schema = d.Parameters
		}
		tools = append(tools, map[string]any{
			"name":        d.Name,
			"description": d.Description,
			"inputSchema": schema,
		})
	}
	result := map[string]any{"tools": tools}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, &Error{Code: -32603, Message: "internal error: " + err.Error()}
	}
	return data, nil
}

// toolsCall executes a capability and returns the result in MCP content format.
func (h *Handler) toolsCall(params json.RawMessage) (json.RawMessage, *Error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &Error{Code: -32602, Message: "invalid params: " + err.Error()}
	}

	ctx := context.Background()
	result, err := h.reg.Execute(ctx, call.Name, call.Arguments)
	if err != nil {
		return nil, &Error{Code: -32000, Message: "tool execution failed: " + err.Error()}
	}

	content := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result,
			},
		},
	}
	data, err := json.Marshal(content)
	if err != nil {
		return nil, &Error{Code: -32603, Message: "internal error: " + err.Error()}
	}
	return data, nil
}
