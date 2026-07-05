package nodes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRequestNode_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer ts.Close()

	node, err := NewHTTPRequestNode("http1", nil)
	if err != nil {
		t.Fatalf("NewHTTPRequestNode() error = %v", err)
	}
	if node.NodeType() != "http_request" {
		t.Errorf("NodeType() = %q, want %q", node.NodeType(), "http_request")
	}
	outputs, err := node.Execute(context.Background(), map[string]any{"url": ts.URL, "method": "GET"}, map[string]any{"allow_private": true}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	statusCode, ok := outputs["status_code"].(float64)
	if !ok || statusCode != 200 {
		t.Errorf("status_code = %v, want 200", statusCode)
	}
	body, ok := outputs["body"].(string)
	if !ok || body != `{"status": "ok"}` {
		t.Errorf("body = %q, want `{\"status\": \"ok\"}`", body)
	}
}

func TestHTTPRequestNode_MissingURL(t *testing.T) {
	node, _err := NewHTTPRequestNode("http1", nil)
	if _err != nil {
		t.Fatalf("NewHTTPRequestNode() error = %v", _err)
	}
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing url")
	}
}

func TestHTTPRequestNode_WithHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	node, err := NewHTTPRequestNode("http1", nil)
	if err != nil {
		t.Fatalf("NewHTTPRequestNode() error = %v", err)
	}
	outputs, err := node.Execute(context.Background(), map[string]any{
		"url":     ts.URL,
		"method":  "GET",
		"headers": map[string]any{"Authorization": "Bearer token123"},
	}, map[string]any{"allow_private": true}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["status_code"].(float64) != 200 {
		t.Errorf("status_code = %v, want 200", outputs["status_code"])
	}
}

func TestHTTPRequestNode_POST(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	node2, err := NewHTTPRequestNode("http1", nil)
	if err != nil {
		t.Fatalf("NewHTTPRequestNode() error = %v", err)
	}
	outputs, err := node2.Execute(context.Background(), map[string]any{
		"url":    ts.URL,
		"method": "POST",
		"body":   `{"key": "value"}`,
	}, map[string]any{"allow_private": true}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if outputs["status_code"].(float64) != 201 {
		t.Errorf("status_code = %v, want 201", outputs["status_code"])
	}
}
