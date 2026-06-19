package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
)

// AgentResult holds the final output of an agent run.
type AgentResult struct {
	FinalContent string     `json:"final_content"`
	Steps        int        `json:"steps"`
	TokenUsage   TokenUsage `json:"token_usage"`
	ToolCalls    []ToolLog  `json:"tool_calls"`
}

// TokenUsage tracks token consumption for an agent run.
type TokenUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
}

// ToolLog records a tool call during agent execution.
type ToolLog struct {
	Tool   string `json:"tool"`
	Args   string `json:"args"`
	Result string `json:"result"`
}

// ErrMaxStepsExceeded is returned when the agent reaches its step limit.
var ErrMaxStepsExceeded = fmt.Errorf("max steps exceeded")

// AgentLoop executes a ReAct (Reasoning + Acting) agent loop.
//
// The loop: think -> act -> observe -> repeat.
// - think: sends messages to LLM, receives text + optional tool calls
// - act: executes tool calls via CapabilityRegistry
// - observe: appends tool results to message history
type AgentLoop struct {
	bridge  *python.PythonBridge
	reg     *CapabilityRegistry
	emitter *EventEmitter
}

// NewAgentLoop creates an AgentLoop with the given dependencies.
func NewAgentLoop(bridge *python.PythonBridge, reg *CapabilityRegistry, emitter *EventEmitter) *AgentLoop {
	return &AgentLoop{
		bridge:  bridge,
		reg:     reg,
		emitter: emitter,
	}
}

// Run executes the agent loop with the given configuration.
func (a *AgentLoop) Run(
	ctx context.Context,
	runID string,
	messages []*pb.ChatMessage,
	profile *AgentProfile,
	model string,
	temperature float32,
) (*AgentResult, error) {
	result := &AgentResult{}

	// Get tool definitions for LLM
	toolDefs := a.reg.ListForLLM(profile.Tools)
	tools := make([]*pb.LLMTool, len(toolDefs))
	for i, td := range toolDefs {
		tools[i] = &pb.LLMTool{
			Name:           td.Name,
			Description:    td.Description,
			ParametersJson: string(td.Parameters),
		}
	}

	modelID := model
	if modelID == "" {
		modelID = profile.DefaultLLM
	}

	for step := 0; step < profile.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		a.emit(runID, "step_start", map[string]any{"step": step + 1, "max_steps": profile.MaxSteps})

		req := &pb.LLMChatRequest{
			Model:        modelID,
			Messages:     messages,
			Tools:        tools,
			SystemPrompt: profile.SystemPrompt,
			Temperature:  temperature,
			StreamId:     runID,
		}

		// Call LLM (streaming)
		stream, err := a.bridge.Chat(ctx, req)
		if err != nil {
			a.emit(runID, "error", map[string]string{"error": err.Error()})
			return result, fmt.Errorf("agent step %d: %w", step, err)
		}

		// Accumulate streaming response
		var fullContent strings.Builder
		var toolCallDeltas map[int]*toolCallAccumulator
		var finishReason string

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				stream.Close()
				a.emit(runID, "error", map[string]string{"error": err.Error()})
				return result, fmt.Errorf("agent step %d recv: %w", step, err)
			}

			if chunk.DeltaContent != "" {
				fullContent.WriteString(chunk.DeltaContent)
				a.emit(runID, "think", map[string]string{"delta": chunk.DeltaContent})
			}

			if chunk.ToolCallDelta != nil {
				tcd := chunk.ToolCallDelta
				if toolCallDeltas == nil {
					toolCallDeltas = make(map[int]*toolCallAccumulator)
				}
				idx := int(tcd.Index)
				if _, ok := toolCallDeltas[idx]; !ok {
					toolCallDeltas[idx] = &toolCallAccumulator{}
				}
				acc := toolCallDeltas[idx]
				if tcd.Id != "" {
					acc.id = tcd.Id
				}
				if tcd.Name != "" {
					acc.name = tcd.Name
				}
				acc.argsBuilder.WriteString(tcd.ArgumentsDelta)
			}

			if chunk.FinishReason != "" {
				finishReason = chunk.FinishReason
			}
			result.TokenUsage.PromptTokens += chunk.PromptTokens
			result.TokenUsage.CompletionTokens += chunk.CompletionTokens
		}
		stream.Close()

		content := fullContent.String()

		// If no tool calls, agent is done
		if len(toolCallDeltas) == 0 || finishReason != "tool_calls" {
			result.FinalContent = content
			result.Steps = step + 1
			a.emit(runID, "finished", map[string]any{
				"steps":   result.Steps,
				"tokens":  result.TokenUsage,
				"content": result.FinalContent,
			})
			slog.Info("agent finished", "run_id", runID, "steps", result.Steps, "tokens", result.TokenUsage.PromptTokens)
			return result, nil
		}

		// Add assistant message with tool calls
		assistantMsg := &pb.ChatMessage{Role: "assistant", Content: content}
		for _, acc := range toolCallDeltas {
			tc := &pb.ToolCall{
				Id:        acc.id,
				Name:      acc.name,
				Arguments: acc.argsBuilder.String(),
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call
		for _, tc := range assistantMsg.ToolCalls {
			a.emit(runID, "tool_call", map[string]string{"tool": tc.Name, "args": tc.Arguments})

			toolResult, err := a.reg.Execute(ctx, tc.Name, json.RawMessage(tc.Arguments))
			if err != nil {
				toolResult = fmt.Sprintf("Error executing %s: %v", tc.Name, err)
				slog.Warn("tool_call failed", "run_id", runID, "tool", tc.Name, "error", err)
			}

			result.ToolCalls = append(result.ToolCalls, ToolLog{
				Tool:   tc.Name,
				Args:   tc.Arguments,
				Result: toolResult,
			})

			a.emit(runID, "tool_result", map[string]string{"tool": tc.Name, "result": toolResult})

			// Add tool result to messages
			messages = append(messages, &pb.ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallId: tc.Id,
			})
		}
	}

	return result, ErrMaxStepsExceeded
}

type toolCallAccumulator struct {
	id          string
	name        string
	argsBuilder strings.Builder
}

func (a *AgentLoop) emit(runID, eventType string, data interface{}) {
	if a.emitter == nil {
		return
	}
	a.emitter.Emit(AgentEvent{
		RunID:     runID,
		Timestamp: time.Now().UnixMilli(),
		Type:      eventType,
		Data:      data,
	})
}
