package python

import (
	"context"
	"fmt"
	"io"

	pb "quantflow/internal/python/proto"
)

// ChatStream represents a streaming chat response from the Python LLM sidecar.
// Call Recv() to get the next chunk, io.EOF when the stream ends.
type ChatStream struct {
	stream pb.LLMService_ChatClient
	cancel context.CancelFunc
}

// Recv returns the next chat response chunk from the stream.
// Returns io.EOF when the stream is complete.
func (s *ChatStream) Recv() (*pb.LLMChatResponse, error) {
	return s.stream.Recv()
}

// Close cancels the stream. Safe to call multiple times.
func (s *ChatStream) Close() error {
	s.cancel()
	return nil
}

// Chat starts a streaming chat with the Python LLM service.
// Returns a ChatStream that yields incremental LLMChatResponse chunks.
func (b *PythonBridge) Chat(ctx context.Context, req *pb.LLMChatRequest) (*ChatStream, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	stream, err := b.LLMClient.Chat(ctx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("llm chat: %w", err)
	}
	return &ChatStream{stream: stream, cancel: cancel}, nil
}

// ListModels returns available models from the Python sidecar.
func (b *PythonBridge) ListModels(ctx context.Context) ([]*pb.LLMModelInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	resp, err := b.LLMClient.ListModels(ctx, &pb.LLMListModelsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	return resp.Models, nil
}

// CountTokens estimates token count for a set of messages.
func (b *PythonBridge) CountTokens(ctx context.Context, model string, messages []*pb.ChatMessage, systemPrompt string) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, b.opts.RequestTimeout)
	defer cancel()

	req := &pb.CountTokensRequest{
		Model:        model,
		Messages:     messages,
		SystemPrompt: systemPrompt,
	}
	resp, err := b.LLMClient.CountTokens(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("count tokens: %w", err)
	}
	return resp.TokenCount, nil
}

// ensure import
var _ io.Closer = (*ChatStream)(nil)
