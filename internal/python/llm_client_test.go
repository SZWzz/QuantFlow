package python

import (
	"testing"

	pb "quantflow/internal/python/proto"
)

func TestLLMClient_NoServer(t *testing.T) {
	// Test that creating a bridge without a running server returns an error
	opts := DefaultOptions()
	opts.DialTimeout = 100 // Very short timeout for test
	_, err := NewPythonBridge(opts)
	if err == nil {
		t.Skip("Python sidecar is running — skipping no-server test")
	}
	t.Logf("Expected error (no server): %v", err)
}

func TestChatMessages(t *testing.T) {
	// Verify protobuf message constructors work
	msg := &pb.ChatMessage{
		Role:    "user",
		Content: "test message",
	}
	if msg.Role != "user" {
		t.Errorf("role = %q", msg.Role)
	}

	req := &pb.LLMChatRequest{
		Model:        "ollama/llama3.1:8b",
		SystemPrompt: "You are helpful.",
		Messages:     []*pb.ChatMessage{msg},
		StreamId:     "test-123",
	}
	if req.StreamId != "test-123" {
		t.Errorf("stream_id = %q", req.StreamId)
	}

	tool := &pb.LLMTool{
		Name:           "test_tool",
		Description:    "A test tool",
		ParametersJson: `{"type":"object"}`,
	}
	if tool.Name != "test_tool" {
		t.Errorf("tool name = %q", tool.Name)
	}
}
