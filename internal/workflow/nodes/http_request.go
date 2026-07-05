package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"quantflow/internal/workflow"
)

type HTTPRequestNode struct {
	id     string
	params map[string]any
}

func NewHTTPRequestNode(id string, params map[string]any) (workflow.BaseNode, error) {
	return &HTTPRequestNode{id: id, params: params}, nil
}

func (n *HTTPRequestNode) ID() string       { return n.id }
func (n *HTTPRequestNode) NodeType() string { return "http_request" }
func (n *HTTPRequestNode) Category() string { return "utility" }

func (n *HTTPRequestNode) InputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "url", Type: workflow.PortString, Required: false},
		{Name: "method", Type: workflow.PortString, Required: false},
		{Name: "headers", Type: workflow.PortAny, Required: false},
		{Name: "body", Type: workflow.PortString, Required: false},
	}
}

func (n *HTTPRequestNode) OutputPorts() []workflow.PortDefinition {
	return []workflow.PortDefinition{
		{Name: "status_code", Type: workflow.PortNumber, Required: false},
		{Name: "body", Type: workflow.PortString, Required: false},
		{Name: "headers", Type: workflow.PortAny, Required: false},
	}
}

func (n *HTTPRequestNode) ParamSchema() []workflow.ParamDef {
	return []workflow.ParamDef{
		{Name: "url", Type: "string", Default: "", Description: "Request URL"},
		{Name: "method", Type: "string", Default: "GET", Description: "HTTP method"},
		{Name: "headers", Type: "string", Default: "", Description: "JSON-encoded request headers"},
		{Name: "body", Type: "string", Default: "", Description: "Request body"},
		{Name: "allow_private", Type: "bool", Default: "false", Description: "Allow requests to private IPs"},
		{Name: "timeout", Type: "number", Default: "10", Description: "Request timeout in seconds"},
	}
}

func (n *HTTPRequestNode) Execute(ctx context.Context, inputs map[string]any, params map[string]any, nctx *workflow.NodeContext) (map[string]any, error) {
	reqURL := firstString(inputs, params, "url")
	if reqURL == "" {
		return nil, fmt.Errorf("http_request: url is required")
	}

	method := firstString(inputs, params, "method")
	if method == "" {
		method = "GET"
	}

	reqHeaders := make(map[string]string)
	if h, err := firstStringMap(inputs, params, "headers"); err == nil {
		reqHeaders = h
	}

	reqBody := firstString(inputs, params, "body")

	allowPrivate := false
	if v, ok := params["allow_private"]; ok {
		allowPrivate, _ = toBool(v)
	} else if v, ok := inputs["allow_private"]; ok {
		allowPrivate, _ = toBool(v)
	}

	timeoutSec := 10.0
	if v := getFloatParam(params, "timeout", 10); v > 0 {
		timeoutSec = v
	} else if v, ok := inputs["timeout"]; ok {
		if f, ok := toFloat64(v); ok && f > 0 {
			timeoutSec = f
		}
	}

	if !allowPrivate {
		if err := ssrfGuard(reqURL); err != nil {
			return nil, fmt.Errorf("http_request: %w", err)
		}
	}

	client := &http.Client{Timeout: time.Duration(timeoutSec * float64(time.Second))}

	var bodyReader io.Reader
	if reqBody != "" {
		bodyReader = strings.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("http_request: failed to create request: %w", err)
	}
	for k, v := range reqHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http_request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http_request: failed to read response: %w", err)
	}

	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	return map[string]any{
		"status_code": float64(resp.StatusCode),
		"body":        string(respBody),
		"headers":     respHeaders,
	}, nil
}

func (n *HTTPRequestNode) Validate() error { return nil }

func ssrfGuard(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	host := parsed.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve %q: %w", host, err)
	}
	for _, ip := range ips {
		if ip.IsPrivate() {
			return fmt.Errorf("ssrf guard: refusing request to private IP %s", ip.String())
		}
	}
	return nil
}

func firstString(inputs, params map[string]any, key string) string {
	if v, ok := inputs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return getStringParam(params, key, "")
}

func firstStringMap(inputs, params map[string]any, key string) (map[string]string, error) {
	if v, ok := inputs[key]; ok {
		switch val := v.(type) {
		case map[string]string:
			return val, nil
		case map[string]any:
			m := make(map[string]string, len(val))
			for k, vv := range val {
				m[k] = fmt.Sprintf("%v", vv)
			}
			return m, nil
		case string:
			var m map[string]string
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				return m, nil
			}
		}
	}
	ps := getStringParam(params, key, "")
	if ps == "" {
		return map[string]string{}, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(ps), &m); err != nil {
		return nil, fmt.Errorf("http_request: invalid %q JSON: %w", key, err)
	}
	return m, nil
}

func toBool(v any) (bool, bool) {
	switch val := v.(type) {
	case bool:
		return val, true
	case string:
		return val == "true" || val == "1", true
	case int:
		return val != 0, true
	case float64:
		return val != 0, true
	default:
		return false, false
	}
}
