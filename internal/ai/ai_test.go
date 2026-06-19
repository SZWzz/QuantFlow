package ai

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCapabilityRegistry_Register(t *testing.T) {
	reg := NewCapabilityRegistry()
	err := reg.Register(&Capability{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			return "ok", nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !reg.Has("test_tool") {
		t.Error("Has returned false for registered tool")
	}
}

func TestCapabilityRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{Name: "dup", Description: "first", Handler: func(ctx context.Context, args json.RawMessage) (string, error) { return "1", nil }})
	err := reg.Register(&Capability{Name: "dup", Description: "second", Handler: func(ctx context.Context, args json.RawMessage) (string, error) { return "2", nil }})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestCapabilityRegistry_Execute(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{
		Name:        "echo",
		Description: "Echoes input",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			return string(args), nil
		},
	})

	result, err := reg.Execute(context.Background(), "echo", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != `{"msg":"hello"}` {
		t.Errorf("result = %q, want %q", result, `{"msg":"hello"}`)
	}
}

func TestCapabilityRegistry_ExecuteUnknown(t *testing.T) {
	reg := NewCapabilityRegistry()
	_, err := reg.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for unknown capability")
	}
}

func TestCapabilityRegistry_ListForLLM(t *testing.T) {
	reg := NewCapabilityRegistry()
	reg.Register(&Capability{
		Name:        "tool_a",
		Description: "Tool A",
		Parameters:  json.RawMessage(`{}`),
		Handler:     func(ctx context.Context, args json.RawMessage) (string, error) { return "a", nil },
	})
	reg.Register(&Capability{
		Name:        "tool_b",
		Description: "Tool B",
		Parameters:  json.RawMessage(`{}`),
		Handler:     func(ctx context.Context, args json.RawMessage) (string, error) { return "b", nil },
	})

	// List all
	all := reg.ListForLLM(nil)
	if len(all) != 2 {
		t.Errorf("expected 2 tools, got %d", len(all))
	}

	// Filter by names
	filtered := reg.ListForLLM([]string{"tool_a"})
	if len(filtered) != 1 || filtered[0].Name != "tool_a" {
		t.Errorf("expected only tool_a, got %d tools", len(filtered))
	}
}

func TestEventEmitter_SubscribeAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	defer emitter.CloseRun("run1")

	ch := emitter.Subscribe("run1")
	emitter.Emit(AgentEvent{RunID: "run1", Type: "think", Data: "hello"})

	event := <-ch
	if event.Type != "think" {
		t.Errorf("event type = %q, want %q", event.Type, "think")
	}
	if event.Data != "hello" {
		t.Errorf("event data = %v, want %v", event.Data, "hello")
	}
}

func TestEventEmitter_DifferentRunIDs(t *testing.T) {
	emitter := NewEventEmitter()
	defer emitter.CloseRun("run1")
	defer emitter.CloseRun("run2")

	ch1 := emitter.Subscribe("run1")
	ch2 := emitter.Subscribe("run2")

	emitter.Emit(AgentEvent{RunID: "run1", Type: "think", Data: "one"})
	emitter.Emit(AgentEvent{RunID: "run2", Type: "think", Data: "two"})

	e1 := <-ch1
	e2 := <-ch2

	if e1.Data != "one" {
		t.Errorf("ch1 got %v", e1.Data)
	}
	if e2.Data != "two" {
		t.Errorf("ch2 got %v", e2.Data)
	}
}

func TestProfileManager_LoadFile(t *testing.T) {
	// Create temp profile and test loading
	pm := NewProfileManager()

	// Test that List returns empty initially
	list := pm.List()
	if len(list) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(list))
	}

	// Test Get on missing profile
	_, err := pm.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing profile")
	}
}

func TestProfileManager_LoadNonExistentFile(t *testing.T) {
	pm := NewProfileManager()
	err := pm.LoadFile("/nonexistent/path/profile.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
